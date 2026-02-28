package main

import (
	"encoding/json"
	"net/http"
)

// apiResponse is the standard JSON response envelope.
type apiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// createTunnelRequest is the JSON body for POST /api/tunnels.
type createTunnelRequest struct {
	Name       string `json:"name"`
	ConfigName string `json:"config_name"`
	Region     string `json:"region"`
	SocksPort  int    `json:"socks_port"`
	ProxyID    int    `json:"proxy_id"`
}

// Server handles HTTP requests for the VPN Agent API.
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

// Router returns the HTTP handler with all routes registered.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// Tunnels
	mux.HandleFunc("GET /api/tunnels", s.handleListTunnels)
	mux.HandleFunc("POST /api/tunnels", s.handleCreateTunnel)
	mux.HandleFunc("DELETE /api/tunnels/{name}", s.handleRemoveTunnel)
	mux.HandleFunc("POST /api/tunnels/{name}/start", s.handleStartTunnel)
	mux.HandleFunc("POST /api/tunnels/{name}/stop", s.handleStopTunnel)
	mux.HandleFunc("POST /api/tunnels/{name}/restart", s.handleRestartTunnel)
	mux.HandleFunc("GET /api/tunnels/{name}/status", s.handleTunnelStatus)

	// Configs
	mux.HandleFunc("POST /api/configs/upload", s.handleUploadConfigs)
	mux.HandleFunc("GET /api/configs", s.handleListConfigs)
	mux.HandleFunc("DELETE /api/configs/{name}", s.handleDeleteConfig)

	// Ports
	mux.HandleFunc("GET /api/ports/next", s.handleNextPort)

	// Wrap with auth middleware if API key is configured
	if s.cfg.APIKey != "" {
		return s.authMiddleware(mux)
	}
	return mux
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoint
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		key := r.Header.Get("X-Api-Key")
		if key != s.cfg.APIKey {
			writeError(w, http.StatusUnauthorized, "invalid or missing API key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// writeJSON writes a success JSON response.
func writeJSON(w http.ResponseWriter, httpCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	json.NewEncoder(w).Encode(apiResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// writeError writes an error JSON response.
func writeError(w http.ResponseWriter, httpCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	json.NewEncoder(w).Encode(apiResponse{
		Code:    1,
		Message: msg,
	})
}
