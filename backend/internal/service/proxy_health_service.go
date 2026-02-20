package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	healthCheckInterval   = 60 * time.Second
	healthCheckTimeout    = 15 * time.Second
	healthCheckURL        = "https://api.anthropic.com"
	exitIPCheckURL        = "https://api.ipify.org"
	unhealthyThreshold    = 3 // consecutive failures before marking unhealthy
	HealthStatusHealthy   = "healthy"
	HealthStatusDegraded  = "degraded"
	HealthStatusUnhealthy = "unhealthy"
	VpnStatusConnected    = "connected"
	VpnStatusDisconnected = "disconnected"
	VpnStatusError        = "error"
)

// ProxyHealthService periodically checks proxy health and updates status.
type ProxyHealthService struct {
	proxyRepo         ProxyRepository
	proxyGroupService *ProxyGroupService

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewProxyHealthService creates a new ProxyHealthService.
func NewProxyHealthService(
	proxyRepo ProxyRepository,
	proxyGroupService *ProxyGroupService,
) *ProxyHealthService {
	s := &ProxyHealthService{
		proxyRepo:         proxyRepo,
		proxyGroupService: proxyGroupService,
		stopCh:            make(chan struct{}),
	}
	s.wg.Add(1)
	go s.run()
	return s
}

// Stop gracefully stops the health check loop.
func (s *ProxyHealthService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

func (s *ProxyHealthService) run() {
	defer s.wg.Done()

	// Initial check after a short delay
	select {
	case <-time.After(10 * time.Second):
	case <-s.stopCh:
		return
	}

	s.checkAll()

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAll()
		case <-s.stopCh:
			return
		}
	}
}

func (s *ProxyHealthService) checkAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	proxies, err := s.proxyRepo.ListActive(ctx)
	if err != nil {
		slog.Error("proxy health: failed to list active proxies", "error", err)
		return
	}

	if len(proxies) == 0 {
		return
	}

	slog.Info("proxy health: starting check", "count", len(proxies))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // max 5 concurrent checks

	for i := range proxies {
		p := &proxies[i]
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			s.checkOne(ctx, p)
		}()
	}

	wg.Wait()
}

func (s *ProxyHealthService) checkOne(ctx context.Context, p *Proxy) {
	proxyURL := p.URL()
	start := time.Now()

	// Check connectivity through the proxy
	reachable, latencyMs := s.probeProxy(ctx, proxyURL)

	now := time.Now()
	p.LastHealthAt = &now

	if reachable {
		p.HealthCheckFailures = 0
		p.HealthStatus = HealthStatusHealthy
		p.LatencyMs = &latencyMs

		// Try to detect exit IP
		if exitIP := s.detectExitIP(ctx, proxyURL); exitIP != "" {
			p.VpnExitIP = exitIP
		}
	} else {
		p.HealthCheckFailures++
		if p.HealthCheckFailures >= unhealthyThreshold {
			p.HealthStatus = HealthStatusUnhealthy
			slog.Warn("proxy health: marked unhealthy",
				"proxy_id", p.ID, "name", p.Name,
				"failures", p.HealthCheckFailures,
				"group", p.GroupName)
		} else {
			p.HealthStatus = HealthStatusDegraded
		}
	}

	// Persist updated health status
	if err := s.proxyRepo.Update(ctx, p); err != nil {
		slog.Error("proxy health: failed to update proxy",
			"proxy_id", p.ID, "error", err)
		return
	}

	// Invalidate group cache so selection picks up new health status
	if p.GroupName != "" && s.proxyGroupService != nil {
		s.proxyGroupService.InvalidateGroupCache(p.GroupName)
	}

	slog.Debug("proxy health: check complete",
		"proxy_id", p.ID, "name", p.Name,
		"healthy", reachable, "latency_ms", latencyMs,
		"duration", time.Since(start).Round(time.Millisecond))
}

// probeProxy checks if the upstream API is reachable through the proxy.
func (s *ProxyHealthService) probeProxy(ctx context.Context, proxyURL string) (ok bool, latencyMs int) {
	transport, err := s.buildTransport(proxyURL)
	if err != nil {
		slog.Error("proxy health: failed to build transport", "error", err)
		return false, 0
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   healthCheckTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, healthCheckURL, nil)
	if err != nil {
		return false, 0
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		slog.Debug("proxy health: probe failed", "proxy", proxyURL, "error", err)
		return false, 0
	}
	_ = resp.Body.Close()

	ms := int(elapsed.Milliseconds())

	// 403 from Cloudflare means the proxy IP is blocked (GeoIP),
	// but the proxy itself is working. We still mark it as reachable
	// because the VPN tunnel is up; the 403 is an API-level issue.
	// Any response (even 403) means the proxy can reach the internet.
	return true, ms
}

// detectExitIP discovers the proxy's public exit IP.
func (s *ProxyHealthService) detectExitIP(ctx context.Context, proxyURL string) string {
	transport, err := s.buildTransport(proxyURL)
	if err != nil {
		return ""
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exitIPCheckURL, nil)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return ""
	}

	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) != nil {
		return ip
	}
	return ""
}

func (s *ProxyHealthService) buildTransport(proxyURL string) (*http.Transport, error) {
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy URL: %w", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(parsed),
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return transport, nil
}

// CheckOne triggers a health check for a single proxy by ID.
func (s *ProxyHealthService) CheckOne(ctx context.Context, proxyID int64) (*Proxy, error) {
	p, err := s.proxyRepo.GetByID(ctx, proxyID)
	if err != nil {
		return nil, fmt.Errorf("get proxy: %w", err)
	}
	s.checkOne(ctx, p)
	return p, nil
}

// CheckAll triggers health checks for all active proxies.
func (s *ProxyHealthService) CheckAll(ctx context.Context) {
	s.checkAll()
}

// CheckGroupHealth checks if an entire proxy group is healthy.
// Returns true if at least one proxy in the group is healthy.
func (s *ProxyHealthService) CheckGroupHealth(ctx context.Context, groupName string) (bool, error) {
	proxies, err := s.proxyRepo.ListByGroupName(ctx, groupName)
	if err != nil {
		return false, fmt.Errorf("list group proxies: %w", err)
	}

	for i := range proxies {
		if proxies[i].IsHealthy() {
			return true, nil
		}
	}

	return false, nil
}
