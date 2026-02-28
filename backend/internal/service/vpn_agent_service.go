package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// VpnTunnel represents a VPN tunnel from the Agent API.
type VpnTunnel struct {
	Name       string    `json:"name"`
	TunnelType string    `json:"tunnel_type"`
	ConfigName string    `json:"config_name"`
	Region     string    `json:"region"`
	ServerIP   string    `json:"server_ip"`
	SocksPort  int       `json:"socks_port"`
	Status     string    `json:"status"`
	TunDevice  string    `json:"tun_device"`
	LocalIP    string    `json:"local_ip"`
	ExitIP     string    `json:"exit_ip"`
	Health     string    `json:"health"`
	LatencyMs  int       `json:"latency_ms"`
	Uptime     time.Time `json:"uptime"`
	Failures   int       `json:"consecutive_failures"`
	LastCheck  time.Time `json:"last_check"`
	ProxyID    int       `json:"proxy_id,omitempty"`
	CertID     string    `json:"cert_id,omitempty"`
}

// VpnOvpnConfig represents an .ovpn config file from the Agent.
type VpnOvpnConfig struct {
	Name     string `json:"name"`
	Region   string `json:"region"`
	ServerIP string `json:"server_ip"`
	FilePath string `json:"file_path"`
}

// CreateTunnelInput is the request body for creating a tunnel.
type CreateTunnelInput struct {
	Name       string `json:"name"`
	ConfigName string `json:"config_name"`
	Region     string `json:"region"`
	SocksPort  int    `json:"socks_port"`
	ProxyID    int    `json:"proxy_id"`
	CertID     string `json:"cert_id,omitempty"`
	TunnelType string `json:"tunnel_type,omitempty"` // "openvpn" (default) or "wireguard"
}

// AgentHealth represents the Agent's health status.
type AgentHealth struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime"`
	Tunnels int    `json:"tunnels"`
}

// VpnCertificate represents a client certificate from the Agent.
type VpnCertificate struct {
	ID           string    `json:"id"`
	Label        string    `json:"label"`
	CreatedAt    time.Time `json:"created_at"`
	InUse        bool      `json:"in_use"`
	ActiveTunnel string    `json:"active_tunnel,omitempty"`
}

// ImportCertInput is the request body for importing a certificate.
type ImportCertInput struct {
	Label    string `json:"label"`
	OvpnData string `json:"ovpn_data"`
}

// agentAPIResponse is the envelope returned by the VPN Agent API.
type agentAPIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// VpnAgentService communicates with the VPN Agent running on the host.
type VpnAgentService struct {
	agentURL   string
	apiKey     string
	proxyHost  string
	httpClient *http.Client
	enabled    bool
}

// NewVpnAgentService creates a new VpnAgentService.
func NewVpnAgentService(cfg *config.Config) *VpnAgentService {
	proxyHost := cfg.VpnAgent.ProxyHost
	if proxyHost == "" {
		if u, err := url.Parse(cfg.VpnAgent.URL); err == nil {
			proxyHost = u.Hostname()
		}
	}
	return &VpnAgentService{
		agentURL:  cfg.VpnAgent.URL,
		apiKey:    cfg.VpnAgent.APIKey,
		proxyHost: proxyHost,
		enabled:   cfg.VpnAgent.Enabled,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // long timeout for tunnel creation
		},
	}
}

// GetAgentHost returns the external host/IP for the VPN Agent, used in proxy records.
func (s *VpnAgentService) GetAgentHost() string {
	return s.proxyHost
}

// IsEnabled returns whether VPN Agent integration is enabled.
func (s *VpnAgentService) IsEnabled() bool {
	return s.enabled
}

// ListTunnels returns all tunnels from the Agent.
func (s *VpnAgentService) ListTunnels(ctx context.Context) ([]VpnTunnel, error) {
	var tunnels []VpnTunnel
	if err := s.doGet(ctx, "/api/tunnels", &tunnels); err != nil {
		return nil, err
	}
	return tunnels, nil
}

// CreateTunnel creates a new tunnel via the Agent.
func (s *VpnAgentService) CreateTunnel(ctx context.Context, input CreateTunnelInput) (*VpnTunnel, error) {
	var tunnel VpnTunnel
	if err := s.doPost(ctx, "/api/tunnels", input, &tunnel); err != nil {
		return nil, err
	}
	return &tunnel, nil
}

// RemoveTunnel removes a tunnel by name.
func (s *VpnAgentService) RemoveTunnel(ctx context.Context, name string) error {
	return s.doDelete(ctx, "/api/tunnels/"+url.PathEscape(name))
}

// RestartTunnel restarts a tunnel by name.
func (s *VpnAgentService) RestartTunnel(ctx context.Context, name string) error {
	return s.doPostEmpty(ctx, "/api/tunnels/"+url.PathEscape(name)+"/restart")
}

// StartTunnel starts a stopped tunnel.
func (s *VpnAgentService) StartTunnel(ctx context.Context, name string) error {
	return s.doPostEmpty(ctx, "/api/tunnels/"+url.PathEscape(name)+"/start")
}

// StopTunnel stops a running tunnel.
func (s *VpnAgentService) StopTunnel(ctx context.Context, name string) error {
	return s.doPostEmpty(ctx, "/api/tunnels/"+url.PathEscape(name)+"/stop")
}

// GetTunnelStatus returns detailed status for a single tunnel.
func (s *VpnAgentService) GetTunnelStatus(ctx context.Context, name string) (*VpnTunnel, error) {
	var tunnel VpnTunnel
	if err := s.doGet(ctx, "/api/tunnels/"+url.PathEscape(name)+"/status", &tunnel); err != nil {
		return nil, err
	}
	return &tunnel, nil
}

// ListConfigs returns all .ovpn config files from the Agent.
func (s *VpnAgentService) ListConfigs(ctx context.Context) ([]VpnOvpnConfig, error) {
	var configs []VpnOvpnConfig
	if err := s.doGet(ctx, "/api/configs", &configs); err != nil {
		return nil, err
	}
	return configs, nil
}

// DeleteConfig deletes an .ovpn config file.
func (s *VpnAgentService) DeleteConfig(ctx context.Context, name string) error {
	return s.doDelete(ctx, "/api/configs/"+url.PathEscape(name))
}

// GetAgentHealth returns the Agent's health status.
func (s *VpnAgentService) GetAgentHealth(ctx context.Context) (*AgentHealth, error) {
	var health AgentHealth
	if err := s.doGet(ctx, "/api/health", &health); err != nil {
		return nil, err
	}
	return &health, nil
}

// GetNextPort returns the next available SOCKS port from the Agent.
func (s *VpnAgentService) GetNextPort(ctx context.Context) (int, error) {
	var result struct {
		Port int `json:"port"`
	}
	if err := s.doGet(ctx, "/api/ports/next", &result); err != nil {
		return 0, err
	}
	return result.Port, nil
}

// UploadConfigs uploads .ovpn files to the Agent via multipart form.
func (s *VpnAgentService) UploadConfigs(ctx context.Context, files map[string][]byte) ([]string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for name, data := range files {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			return nil, fmt.Errorf("create form file: %w", err)
		}
		if _, err := part.Write(data); err != nil {
			return nil, fmt.Errorf("write form file: %w", err)
		}
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.agentURL+"/api/configs/upload", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if s.apiKey != "" {
		req.Header.Set("X-Api-Key", s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agent request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp agentAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("agent error: %s", apiResp.Message)
	}

	var result struct {
		Uploaded []string `json:"uploaded"`
	}
	if err := json.Unmarshal(apiResp.Data, &result); err != nil {
		return nil, fmt.Errorf("decode upload result: %w", err)
	}
	return result.Uploaded, nil
}

// ListCertificates returns all certificates from the Agent.
func (s *VpnAgentService) ListCertificates(ctx context.Context) ([]VpnCertificate, error) {
	var certs []VpnCertificate
	if err := s.doGet(ctx, "/api/certs", &certs); err != nil {
		return nil, err
	}
	return certs, nil
}

// ImportCertificate imports a certificate via the Agent.
func (s *VpnAgentService) ImportCertificate(ctx context.Context, input ImportCertInput) (*VpnCertificate, error) {
	var cert VpnCertificate
	if err := s.doPost(ctx, "/api/certs/import", input, &cert); err != nil {
		return nil, err
	}
	return &cert, nil
}

// DeleteCertificate deletes a certificate by ID.
func (s *VpnAgentService) DeleteCertificate(ctx context.Context, id string) error {
	return s.doDelete(ctx, "/api/certs/"+url.PathEscape(id))
}
