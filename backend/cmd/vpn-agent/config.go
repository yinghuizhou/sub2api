package main

import (
	"os"
	"strconv"
)

// AgentConfig holds all configuration for the VPN Agent.
type AgentConfig struct {
	ListenAddr    string
	Port          int
	OvpnConfigDir string
	StateDir      string
	Sub2APIURL    string
	Sub2APIKey    string
	LogDir        string
	APIKey        string
}

func loadConfig() *AgentConfig {
	return &AgentConfig{
		ListenAddr:    envOrDefault("VPN_AGENT_LISTEN", "127.0.0.1"),
		Port:          envIntOrDefault("VPN_AGENT_PORT", 9090),
		OvpnConfigDir: envOrDefault("OVPN_CONFIG_DIR", "/etc/vpn-agent/configs"),
		StateDir:      envOrDefault("STATE_DIR", "/etc/vpn-agent"),
		Sub2APIURL:    envOrDefault("SUB2API_URL", "http://localhost:8080"),
		Sub2APIKey:    os.Getenv("SUB2API_API_KEY"),
		LogDir:        envOrDefault("LOG_DIR", "/var/log/vpn-agent"),
		APIKey:        os.Getenv("VPN_AGENT_API_KEY"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
