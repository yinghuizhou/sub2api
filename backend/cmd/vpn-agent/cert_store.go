package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// Certificate represents a client certificate extracted from an .ovpn file.
type Certificate struct {
	ID           string    `json:"id"`
	Label        string    `json:"label"`
	CreatedAt    time.Time `json:"created_at"`
	InUse        bool      `json:"in_use"`
	ActiveTunnel string    `json:"active_tunnel,omitempty"`
}

// CertStore manages client certificates on disk.
type CertStore struct {
	dir string
	mu  sync.RWMutex
}

// NewCertStore creates a CertStore rooted at the given directory.
func NewCertStore(dir string) *CertStore {
	return &CertStore{dir: dir}
}

func (s *CertStore) metaPath() string { return filepath.Join(s.dir, "certs.json") }

func (s *CertStore) load() ([]Certificate, error) {
	data, err := os.ReadFile(s.metaPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var certs []Certificate
	return certs, json.Unmarshal(data, &certs)
}

func (s *CertStore) save(certs []Certificate) error {
	data, err := json.MarshalIndent(certs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(), data, 0600)
}

// List returns all certificates from metadata.
func (s *CertStore) List() []Certificate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	certs, _ := s.load()
	return certs
}

// Get returns a single certificate by ID.
func (s *CertStore) Get(id string) (*Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	certs, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range certs {
		if certs[i].ID == id {
			return &certs[i], nil
		}
	}
	return nil, fmt.Errorf("certificate %s not found", id)
}

// Import extracts cert and key from .ovpn data, saves them to disk.
func (s *CertStore) Import(label string, ovpnData []byte) (*Certificate, error) {
	certPEM, err := extractInlineBlock(ovpnData, "cert")
	if err != nil {
		return nil, fmt.Errorf("extract cert: %w", err)
	}
	keyPEM, err := extractInlineBlock(ovpnData, "key")
	if err != nil {
		return nil, fmt.Errorf("extract key: %w", err)
	}

	id := generateUUID()
	certDir := filepath.Join(s.dir, id)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(certDir, "client.crt"), []byte(certPEM), 0600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(certDir, "client.key"), []byte(keyPEM), 0600); err != nil {
		return nil, err
	}

	cert := Certificate{ID: id, Label: label, CreatedAt: time.Now()}
	certs, _ := s.load()
	certs = append(certs, cert)
	if err := s.save(certs); err != nil {
		return nil, err
	}
	return &cert, nil
}

// Delete removes a certificate directory and its metadata.
func (s *CertStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	certs, err := s.load()
	if err != nil {
		return err
	}
	idx := -1
	for i := range certs {
		if certs[i].ID == id {
			if certs[i].InUse {
				return fmt.Errorf("certificate %s is in use by tunnel %s", id, certs[i].ActiveTunnel)
			}
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("certificate %s not found", id)
	}

	os.RemoveAll(filepath.Join(s.dir, id))
	certs = append(certs[:idx], certs[idx+1:]...)
	return s.save(certs)
}

// GetCertPEM reads the certificate PEM file.
func (s *CertStore) GetCertPEM(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(filepath.Join(s.dir, id, "client.crt"))
	return string(data), err
}

// GetKeyPEM reads the private key PEM file.
func (s *CertStore) GetKeyPEM(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(filepath.Join(s.dir, id, "client.key"))
	return string(data), err
}

// MarkInUse marks a certificate as being used by a tunnel.
func (s *CertStore) MarkInUse(id, tunnelName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCert(id, func(c *Certificate) { c.InUse = true; c.ActiveTunnel = tunnelName })
}

// MarkFree marks a certificate as available.
func (s *CertStore) MarkFree(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCert(id, func(c *Certificate) { c.InUse = false; c.ActiveTunnel = "" })
}

func (s *CertStore) updateCert(id string, fn func(*Certificate)) {
	certs, err := s.load()
	if err != nil {
		return
	}
	for i := range certs {
		if certs[i].ID == id {
			fn(&certs[i])
			s.save(certs)
			return
		}
	}
}

// GetPEM returns both cert and key PEM for a certificate.
func (s *CertStore) GetPEM(id string) (certPEM, keyPEM string, err error) {
	certPEM, err = s.GetCertPEM(id)
	if err != nil {
		return
	}
	keyPEM, err = s.GetKeyPEM(id)
	return
}

// IsAvailable checks if a certificate is free to use.
func (s *CertStore) IsAvailable(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	certs, _ := s.load()
	for _, c := range certs {
		if c.ID == id {
			return !c.InUse
		}
	}
	return false
}

func extractInlineBlock(data []byte, blockName string) (string, error) {
	re := regexp.MustCompile(`(?s)<` + regexp.QuoteMeta(blockName) + `>\s*(.*?)\s*</` + regexp.QuoteMeta(blockName) + `>`)
	m := re.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("block <%s> not found", blockName)
	}
	return string(m[1]), nil
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
