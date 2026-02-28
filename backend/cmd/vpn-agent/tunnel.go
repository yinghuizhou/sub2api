package main

import "log"

// TunnelInfo represents the runtime state of a tunnel.
type TunnelInfo struct {
	ConfigName string `json:"config_name"`
	Region     string `json:"region"`
	ServerIP   string `json:"server_ip"`
	SocksPort  int    `json:"socks_port"`
	Status     string `json:"status"` // "running", "stopped", "error"
	PID        int    `json:"pid,omitempty"`
}

// TunnelManager manages OpenVPN tunnels and 3proxy SOCKS5 proxies.
// This is a stub implementation; full logic will be added in a later task.
type TunnelManager struct {
	cfg        *AgentConfig
	store      *ConfigStore
	stateStore *StateStore
	callback   *CallbackClient
}

// NewTunnelManager creates a new TunnelManager.
func NewTunnelManager(cfg *AgentConfig, store *ConfigStore, stateStore *StateStore, callback *CallbackClient) *TunnelManager {
	return &TunnelManager{
		cfg:        cfg,
		store:      store,
		stateStore: stateStore,
		callback:   callback,
	}
}

// RestoreFromState restores tunnels from persisted state. Stub.
func (tm *TunnelManager) RestoreFromState() error {
	log.Println("TunnelManager.RestoreFromState: stub, no-op")
	return nil
}

// StopAll stops all running tunnels. Stub.
func (tm *TunnelManager) StopAll() {
	log.Println("TunnelManager.StopAll: stub, no-op")
}

// List returns all tunnel info. Stub.
func (tm *TunnelManager) List() []TunnelInfo {
	return nil
}

// Start starts a tunnel for the given config. Stub.
func (tm *TunnelManager) Start(configName string, socksPort int) (*TunnelInfo, error) {
	return nil, nil
}

// Stop stops a tunnel by config name. Stub.
func (tm *TunnelManager) Stop(configName string) error {
	return nil
}
