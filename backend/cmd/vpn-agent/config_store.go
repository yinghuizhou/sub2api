package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ConfigMeta holds metadata for a single .ovpn config file.
type ConfigMeta struct {
	UploadedAt    time.Time  `json:"uploaded_at"`
	ContentHash   string     `json:"content_hash"`
	Version       int        `json:"version"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	Stale         bool       `json:"stale"`
}

// ConfigStoreMeta is the on-disk metadata for all configs.
type ConfigStoreMeta struct {
	Configs map[string]*ConfigMeta `json:"configs"`
}

// OvpnConfig represents a parsed .ovpn configuration file.
type OvpnConfig struct {
	Name     string `json:"name"`
	Region   string `json:"region"`
	ServerIP string `json:"server_ip"`
	FilePath string `json:"file_path"`
}

// OvpnConfigWithMeta extends OvpnConfig with metadata fields.
type OvpnConfigWithMeta struct {
	OvpnConfig
	UploadedAt    time.Time  `json:"uploaded_at"`
	Version       int        `json:"version"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	Stale         bool       `json:"stale"`
}

// ConfigStore manages .ovpn configuration files on disk.
type ConfigStore struct {
	dir string
	mu  sync.RWMutex
}

// NewConfigStore creates a ConfigStore rooted at the given directory.
func NewConfigStore(dir string) *ConfigStore {
	return &ConfigStore{dir: dir}
}

// List reads the config directory and returns all parsed .ovpn configs.
func (s *ConfigStore) List() ([]OvpnConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read config dir: %w", err)
	}

	var configs []OvpnConfig
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ovpn") {
			continue
		}
		name := e.Name()
		fp := filepath.Join(s.dir, name)

		content, err := os.ReadFile(fp)
		if err != nil {
			continue
		}

		configs = append(configs, OvpnConfig{
			Name:     name,
			Region:   parseRegion(name),
			ServerIP: parseServerIP(string(content)),
			FilePath: fp,
		})
	}
	return configs, nil
}

// Get reads and returns the content of a named .ovpn file.
func (s *ConfigStore) Get(name string) ([]byte, error) {
	if err := validateConfigName(name); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return os.ReadFile(filepath.Join(s.dir, name))
}

// Save writes an .ovpn file to disk and tracks metadata.
func (s *ConfigStore) Save(fileName string, content []byte) error {
	_, _, err := s.SaveWithMeta(fileName, content)
	return err
}

// SaveWithMeta saves the file, computes sha256 hash, and returns metadata.
// Returns (meta, isNew, error). isNew=false if content hash matches existing.
func (s *ConfigStore) SaveWithMeta(fileName string, content []byte) (*ConfigMeta, bool, error) {
	if err := validateConfigName(fileName); err != nil {
		return nil, false, err
	}

	// Compute content hash (first 16 chars of sha256).
	hash := sha256.Sum256(content)
	contentHash := hex.EncodeToString(hash[:])[:16]

	s.mu.Lock()
	defer s.mu.Unlock()

	meta := s.loadMeta()
	existing, found := meta.Configs[fileName]

	// If hash matches, skip writing — content is identical.
	if found && existing.ContentHash == contentHash {
		return existing, false, nil
	}

	// Write file to disk.
	fp := filepath.Join(s.dir, filepath.Base(fileName))
	if err := os.WriteFile(fp, content, 0600); err != nil {
		return nil, false, err
	}

	// Build new metadata entry.
	now := time.Now()
	newMeta := &ConfigMeta{
		UploadedAt:  now,
		ContentHash: contentHash,
		Version:     1,
		Stale:       false,
	}
	if found {
		newMeta.Version = existing.Version + 1
		newMeta.LastSuccessAt = existing.LastSuccessAt
	}
	meta.Configs[fileName] = newMeta
	s.saveMeta(meta)

	return newMeta, true, nil
}

// GetMeta returns metadata for a config file, nil if not found.
func (s *ConfigStore) GetMeta(name string) *ConfigMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta := s.loadMeta()
	return meta.Configs[name]
}

// ListWithMeta returns all configs with their metadata.
func (s *ConfigStore) ListWithMeta() ([]OvpnConfigWithMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read config dir: %w", err)
	}

	meta := s.loadMeta()
	var configs []OvpnConfigWithMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ovpn") {
			continue
		}
		name := e.Name()
		fp := filepath.Join(s.dir, name)

		content, err := os.ReadFile(fp)
		if err != nil {
			continue
		}

		cwm := OvpnConfigWithMeta{
			OvpnConfig: OvpnConfig{
				Name:     name,
				Region:   parseRegion(name),
				ServerIP: parseServerIP(string(content)),
				FilePath: fp,
			},
		}

		if m, ok := meta.Configs[name]; ok {
			cwm.UploadedAt = m.UploadedAt
			cwm.Version = m.Version
			cwm.LastSuccessAt = m.LastSuccessAt
			cwm.Stale = m.Stale
		}

		configs = append(configs, cwm)
	}
	return configs, nil
}

// RecordSuccess updates last_success_at for a config.
func (s *ConfigStore) RecordSuccess(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta := s.loadMeta()
	if m, ok := meta.Configs[name]; ok {
		now := time.Now()
		m.LastSuccessAt = &now
		s.saveMeta(meta)
	}
}

const (
	configsMetaFile = "configs_meta.json"
	staleThreshold  = 7 * 24 * time.Hour // 7 days
)

// CheckAndMarkStale marks configs as stale if they haven't succeeded recently.
func (s *ConfigStore) CheckAndMarkStale() {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta := s.loadMeta()
	changed := false
	for name, m := range meta.Configs {
		wasStale := m.Stale
		if m.LastSuccessAt == nil {
			// Never succeeded — stale if uploaded > 7 days ago
			m.Stale = time.Since(m.UploadedAt) > staleThreshold
		} else {
			m.Stale = time.Since(*m.LastSuccessAt) > staleThreshold
		}
		if m.Stale != wasStale {
			changed = true
		}
		meta.Configs[name] = m
	}
	if changed {
		s.saveMeta(meta)
	}
}

// loadMeta loads metadata from configs_meta.json. Caller must hold at least RLock.
func (s *ConfigStore) loadMeta() *ConfigStoreMeta {
	fp := filepath.Join(s.dir, configsMetaFile)
	data, err := os.ReadFile(fp)
	if err != nil {
		return &ConfigStoreMeta{Configs: make(map[string]*ConfigMeta)}
	}
	var meta ConfigStoreMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		log.Printf("Warning: failed to parse %s: %v", fp, err)
		return &ConfigStoreMeta{Configs: make(map[string]*ConfigMeta)}
	}
	if meta.Configs == nil {
		meta.Configs = make(map[string]*ConfigMeta)
	}
	return &meta
}

// saveMeta writes metadata to configs_meta.json. Caller must hold Lock.
func (s *ConfigStore) saveMeta(meta *ConfigStoreMeta) {
	fp := filepath.Join(s.dir, configsMetaFile)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		log.Printf("Error marshaling config metadata: %v", err)
		return
	}
	if err := os.WriteFile(fp, data, 0644); err != nil {
		log.Printf("Error saving config metadata to %s: %v", fp, err)
	}
}

// Delete removes an .ovpn file from disk.
func (s *ConfigStore) Delete(name string) error {
	if err := validateConfigName(name); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return os.Remove(filepath.Join(s.dir, name))
}

// FilePath returns the full path for a named config file.
func (s *ConfigStore) FilePath(name string) string {
	return filepath.Join(s.dir, name)
}

// regionPatterns maps filename prefixes/patterns to region codes.
var regionPatterns = []struct {
	prefix string
	region string
}{
	{"TCP-USA-", "us"},
	{"US-", "us"},
	{"TCP-UK-", "uk"},
	{"UK-", "uk"},
	{"TCP-Australia-", "au"},
	{"AU-", "au"},
	{"TCP-Japan-", "jp"},
	{"JP-", "jp"},
	{"TCP-Singapore-", "sg"},
	{"SG-", "sg"},
	{"TCP-HongKong-", "hk"},
	{"HK-", "hk"},
	{"TCP-Germany-", "de"},
	{"DE-", "de"},
	{"TCP-Canada-", "ca"},
	{"CA-", "ca"},
	{"TCP-France-", "fr"},
	{"FR-", "fr"},
	{"TCP-Netherlands-", "nl"},
	{"NL-", "nl"},
	{"TCP-Korea-", "kr"},
	{"KR-", "kr"},
	{"TCP-Taiwan-", "tw"},
	{"TW-", "tw"},
	{"TCP-India-", "in"},
	{"IN-", "in"},
}

// parseRegion extracts a region code from an .ovpn filename.
func parseRegion(filename string) string {
	upper := strings.ToUpper(filename)
	for _, p := range regionPatterns {
		if strings.HasPrefix(upper, strings.ToUpper(p.prefix)) {
			return p.region
		}
	}
	return "unknown"
}

var remoteRe = regexp.MustCompile(`^remote\s+(\S+)`)

// parseServerIP extracts the server IP/hostname from .ovpn content.
func parseServerIP(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if m := remoteRe.FindStringSubmatch(line); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}
