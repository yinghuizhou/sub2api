package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// OvpnConfig represents a parsed .ovpn configuration file.
type OvpnConfig struct {
	Name     string `json:"name"`
	Region   string `json:"region"`
	ServerIP string `json:"server_ip"`
	FilePath string `json:"file_path"`
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

// Save writes an .ovpn file to disk.
func (s *ConfigStore) Save(fileName string, content []byte) error {
	if err := validateConfigName(fileName); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fp := filepath.Join(s.dir, filepath.Base(fileName))
	return os.WriteFile(fp, content, 0600)
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
