package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

const (
	healthCheckInterval  = 30 * time.Second
	healthCheckTimeout   = 10 * time.Second
	healthCheckURL       = "https://api.anthropic.com"
	exitIPURL            = "https://api.ipify.org"
	unhealthyThreshold   = 3
	degradedThreshold    = 1
	maxReconnectAttempts = 2
	maxSwitchAttempts    = 3
)

// HealthChecker periodically checks tunnel health and triggers auto-failover.
type HealthChecker struct {
	tunnelMgr *TunnelManager
	store     *ConfigStore
	callback  *CallbackClient
	cfg       *AgentConfig
	scores    *StabilityScores
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(tm *TunnelManager, store *ConfigStore, cb *CallbackClient, cfg *AgentConfig) *HealthChecker {
	scores := loadScores(cfg.StateDir)
	return &HealthChecker{
		tunnelMgr: tm,
		store:     store,
		callback:  cb,
		cfg:       cfg,
		scores:    scores,
	}
}

// Start begins the periodic health check loop.
func (hc *HealthChecker) Start(ctx context.Context) {
	log.Println("HealthChecker started (interval: 30s)")
	// Initial delay to let tunnels stabilize
	select {
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		hc.checkAll()
		select {
		case <-ticker.C:
		case <-ctx.Done():
			log.Println("HealthChecker stopped")
			return
		}
	}
}

// checkAll checks all connected tunnels.
func (hc *HealthChecker) checkAll() {
	tunnels := hc.tunnelMgr.List()
	for _, t := range tunnels {
		if t.Status != "connected" {
			continue
		}
		hc.checkTunnel(&t)
	}
}

// checkTunnel checks a single tunnel's health via its SOCKS5 proxy.
func (hc *HealthChecker) checkTunnel(t *TunnelInfo) {
	start := time.Now()
	reachable := hc.testConnectivity(t.SocksPort)
	latency := int(time.Since(start).Milliseconds())

	hc.tunnelMgr.mu.Lock()
	rt, ok := hc.tunnelMgr.tunnels[t.Name]
	if !ok {
		hc.tunnelMgr.mu.Unlock()
		return
	}

	rt.LastCheck = time.Now()
	rt.LatencyMs = latency

	if reachable {
		rt.Failures = 0
		rt.Health = "healthy"
		// Detect exit IP in background (non-blocking)
		go hc.detectExitIP(t.Name, t.SocksPort)
		hc.scores.RecordSuccess(t.ConfigName)
	} else {
		rt.Failures++
		if rt.Failures >= unhealthyThreshold {
			rt.Health = "unhealthy"
		} else if rt.Failures >= degradedThreshold {
			rt.Health = "degraded"
		}
		hc.scores.RecordFailure(t.ConfigName)
		log.Printf("Health check failed for %q (failures: %d)", t.Name, rt.Failures)
	}

	// Copy state for callback
	info := rt.TunnelInfo
	hc.tunnelMgr.mu.Unlock()

	// Save scores periodically
	hc.scores.Save(hc.cfg.StateDir)

	// Notify Sub2API
	if info.ProxyID > 0 {
		go hc.callback.UpdateProxyStatus(info.ProxyID, info.ExitIP, info.Status, info.Health, info.LatencyMs)
	}

	// Trigger failover if unhealthy
	if info.Health == "unhealthy" {
		go hc.handleUnhealthy(&info)
	}
}

// testConnectivity tests if traffic can exit through the SOCKS5 proxy.
func (hc *HealthChecker) testConnectivity(socksPort int) bool {
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), nil, proxy.Direct)
	if err != nil {
		return false
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
	}
	client := &http.Client{Transport: transport, Timeout: healthCheckTimeout}

	req, _ := http.NewRequest(http.MethodHead, healthCheckURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	// 403 = Cloudflare blocks but proxy works, any response means reachable
	return true
}

// detectExitIP detects the public exit IP through a tunnel's SOCKS5 proxy.
func (hc *HealthChecker) detectExitIP(name string, socksPort int) {
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), nil, proxy.Direct)
	if err != nil {
		return
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
	}
	client := &http.Client{Transport: transport, Timeout: healthCheckTimeout}

	resp, err := client.Get(exitIPURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return
	}

	hc.tunnelMgr.mu.Lock()
	if rt, ok := hc.tunnelMgr.tunnels[name]; ok {
		rt.ExitIP = ip
	}
	hc.tunnelMgr.mu.Unlock()
}
