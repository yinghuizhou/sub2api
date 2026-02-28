package main

import "net/http"

// Server handles HTTP requests for the VPN Agent API.
// This is a stub implementation; full routes will be added in a later task.
type Server struct {
	cfg           *AgentConfig
	store         *ConfigStore
	tunnelMgr     *TunnelManager
	healthChecker *HealthChecker
}

// NewServer creates a new Server.
func NewServer(cfg *AgentConfig, store *ConfigStore, tm *TunnelManager, hc *HealthChecker) *Server {
	return &Server{
		cfg:           cfg,
		store:         store,
		tunnelMgr:     tm,
		healthChecker: hc,
	}
}

// Router returns the HTTP handler with all routes registered. Stub.
func (s *Server) Router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	return mux
}
