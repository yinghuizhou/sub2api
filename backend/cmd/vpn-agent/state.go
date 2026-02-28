package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// TunnelState represents the persisted state of a single tunnel.
type TunnelState struct {
	ConfigName string `json:"config_name"`
	SocksPort  int    `json:"socks_port"`
	Region     string `json:"region"`
	ServerIP   string `json:"server_ip"`
}

// AgentState is the top-level persisted state.
type AgentState struct {
	Tunnels []TunnelState `json:"tunnels"`
}

// StateStore manages persisting and loading agent state to/from disk.
type StateStore struct {
	dir  string
	file string
	mu   sync.Mutex
}

// NewStateStore creates a StateStore that persists to dir/state.json.
func NewStateStore(dir string) *StateStore {
	return &StateStore{
		dir:  dir,
		file: filepath.Join(dir, "state.json"),
	}
}

// Load reads the persisted state from disk. Returns empty state if file
// does not exist.
func (s *StateStore) Load() (*AgentState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return &AgentState{}, nil
		}
		return nil, err
	}

	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// Save writes the current state to disk atomically.
func (s *StateStore) Save(state *AgentState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.file + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.file)
}
