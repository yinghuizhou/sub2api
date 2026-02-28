package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// CallbackClient sends status updates back to the Sub2API server.
type CallbackClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewCallbackClient creates a new CallbackClient.
func NewCallbackClient(baseURL, apiKey string) *CallbackClient {
	return &CallbackClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// proxyUpdatePayload is the JSON body sent to Sub2API.
type proxyUpdatePayload struct {
	VpnExitIP    string `json:"vpn_exit_ip,omitempty"`
	VpnStatus    string `json:"vpn_status,omitempty"`
	HealthStatus string `json:"health_status,omitempty"`
	LatencyMs    *int   `json:"latency_ms,omitempty"`
}

// UpdateProxyStatus updates a proxy record in Sub2API.
// PUT /api/v1/admin/proxies/{id}
func (c *CallbackClient) UpdateProxyStatus(proxyID int, exitIP, vpnStatus, healthStatus string, latencyMs int) error {
	if c.baseURL == "" || c.apiKey == "" || proxyID == 0 {
		return nil // no callback configured or no proxy ID
	}

	payload := proxyUpdatePayload{
		VpnExitIP:    exitIP,
		VpnStatus:    vpnStatus,
		HealthStatus: healthStatus,
	}
	if latencyMs > 0 {
		payload.LatencyMs = &latencyMs
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/admin/proxies/%d", c.baseURL, proxyID)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Sub2API returned %d", resp.StatusCode)
	}
	return nil
}

// ReportStatus sends tunnel status updates to Sub2API for all tunnels with a ProxyID.
func (c *CallbackClient) ReportStatus(tunnels []TunnelInfo) error {
	for _, t := range tunnels {
		if t.ProxyID == 0 {
			continue
		}
		if err := c.UpdateProxyStatus(t.ProxyID, t.ExitIP, t.Status, t.Health, t.LatencyMs); err != nil {
			log.Printf("Callback error for tunnel %q (proxy %d): %v", t.Name, t.ProxyID, err)
		}
	}
	return nil
}
