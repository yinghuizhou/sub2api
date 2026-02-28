package admin

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// VpnHandler handles VPN management admin API endpoints.
type VpnHandler struct {
	vpnAgentService *service.VpnAgentService
	adminService    service.AdminService
}

// NewVpnHandler creates a new VpnHandler.
func NewVpnHandler(vpnAgentService *service.VpnAgentService, adminService service.AdminService) *VpnHandler {
	return &VpnHandler{vpnAgentService: vpnAgentService, adminService: adminService}
}

func (h *VpnHandler) requireEnabled(c *gin.Context) bool {
	if !h.vpnAgentService.IsEnabled() {
		response.Error(c, http.StatusServiceUnavailable, "VPN Agent integration is not enabled")
		return false
	}
	return true
}

// ListTunnels returns all VPN tunnels.
func (h *VpnHandler) ListTunnels(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	tunnels, err := h.vpnAgentService.ListTunnels(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, tunnels)
}

// CreateTunnel deploys a new VPN tunnel and auto-creates a matching SOCKS5 proxy.
func (h *VpnHandler) CreateTunnel(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	var req service.CreateTunnelInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()

	// Auto-create proxy if no proxy_id was provided.
	if req.ProxyID == 0 {
		// Pre-assign socks port if not specified.
		if req.SocksPort == 0 {
			port, err := h.vpnAgentService.GetNextPort(ctx)
			if err != nil {
				response.Error(c, http.StatusBadGateway, "Agent error (get port): "+err.Error())
				return
			}
			req.SocksPort = port
		}

		proxy, err := h.adminService.CreateProxy(ctx, &service.CreateProxyInput{
			Name:      "vpn-" + req.Name,
			Protocol:  "socks5",
			Host:      h.vpnAgentService.GetAgentHost(),
			Port:      req.SocksPort,
			Region:    req.Region,
			GroupName: "vpn-auto",
		})
		if err != nil {
			log.Printf("Warning: auto-create proxy for tunnel %q failed: %v", req.Name, err)
		} else {
			req.ProxyID = int(proxy.ID)
		}
	}

	tunnel, err := h.vpnAgentService.CreateTunnel(ctx, req)
	if err != nil {
		// Clean up auto-created proxy on tunnel failure.
		if req.ProxyID > 0 {
			if delErr := h.adminService.DeleteProxy(ctx, int64(req.ProxyID)); delErr != nil {
				log.Printf("Warning: cleanup proxy %d after tunnel failure: %v", req.ProxyID, delErr)
			}
		}
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Created(c, tunnel)
}

// RemoveTunnel deletes a VPN tunnel and its auto-created proxy.
func (h *VpnHandler) RemoveTunnel(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	ctx := c.Request.Context()
	name := c.Param("name")

	// Fetch proxy_id before deletion.
	var proxyID int
	if tunnel, err := h.vpnAgentService.GetTunnelStatus(ctx, name); err == nil {
		proxyID = tunnel.ProxyID
	}

	if err := h.vpnAgentService.RemoveTunnel(ctx, name); err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}

	// Auto-delete the linked proxy.
	if proxyID > 0 {
		if err := h.adminService.DeleteProxy(ctx, int64(proxyID)); err != nil {
			log.Printf("Warning: auto-delete proxy %d for tunnel %q failed: %v", proxyID, name, err)
		}
	}

	response.Success(c, nil)
}

// RestartTunnel restarts a VPN tunnel.
func (h *VpnHandler) RestartTunnel(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	name := c.Param("name")
	if err := h.vpnAgentService.RestartTunnel(c.Request.Context(), name); err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, nil)
}

// StartTunnel starts a stopped VPN tunnel.
func (h *VpnHandler) StartTunnel(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	name := c.Param("name")
	if err := h.vpnAgentService.StartTunnel(c.Request.Context(), name); err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, nil)
}

// StopTunnel stops a running VPN tunnel.
func (h *VpnHandler) StopTunnel(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	name := c.Param("name")
	if err := h.vpnAgentService.StopTunnel(c.Request.Context(), name); err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, nil)
}

// GetTunnelStatus returns detailed status for a tunnel.
func (h *VpnHandler) GetTunnelStatus(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	name := c.Param("name")
	tunnel, err := h.vpnAgentService.GetTunnelStatus(c.Request.Context(), name)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, tunnel)
}

// UploadConfigs handles multipart upload of .ovpn files.
func (h *VpnHandler) UploadConfigs(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		response.BadRequest(c, "parse form: "+err.Error())
		return
	}
	fileHeaders := form.File["files"]
	if len(fileHeaders) == 0 {
		response.BadRequest(c, "no files provided")
		return
	}

	files := make(map[string][]byte)
	for _, fh := range fileHeaders {
		if !strings.HasSuffix(fh.Filename, ".ovpn") {
			continue
		}
		f, err := fh.Open()
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "open file: "+err.Error())
			return
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "read file: "+err.Error())
			return
		}
		files[fh.Filename] = data
	}

	uploaded, err := h.vpnAgentService.UploadConfigs(c.Request.Context(), files)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, map[string]any{"uploaded": uploaded, "count": len(uploaded)})
}

// ListConfigs returns all .ovpn config files.
func (h *VpnHandler) ListConfigs(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	configs, err := h.vpnAgentService.ListConfigs(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, configs)
}

// DeleteConfig deletes an .ovpn config file.
func (h *VpnHandler) DeleteConfig(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	name := c.Param("name")
	if err := h.vpnAgentService.DeleteConfig(c.Request.Context(), name); err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, nil)
}

// GetAgentHealth returns the VPN Agent's health status.
func (h *VpnHandler) GetAgentHealth(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	health, err := h.vpnAgentService.GetAgentHealth(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, health)
}

// ListCertificates returns all VPN certificates.
func (h *VpnHandler) ListCertificates(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	certs, err := h.vpnAgentService.ListCertificates(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, certs)
}

// ImportCertificate imports a certificate from .ovpn data.
func (h *VpnHandler) ImportCertificate(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	var req service.ImportCertInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	cert, err := h.vpnAgentService.ImportCertificate(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Created(c, cert)
}

// DeleteCertificate deletes a certificate by ID.
func (h *VpnHandler) DeleteCertificate(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	id := c.Param("id")
	if err := h.vpnAgentService.DeleteCertificate(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Success(c, nil)
}

// SyncTunnelProxies reads tunnel data from the Agent and updates matching proxy records
// so that each proxy's host:port matches its tunnel's actual SOCKS address.
// POST /api/v1/admin/vpn/sync-proxies
func (h *VpnHandler) SyncTunnelProxies(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	ctx := c.Request.Context()

	tunnels, err := h.vpnAgentService.ListTunnels(ctx)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}

	agentHost := h.vpnAgentService.GetAgentHost()
	var updated, skipped, failed int

	for _, t := range tunnels {
		if t.ProxyID == 0 || t.SocksPort == 0 {
			skipped++
			continue
		}

		proxy, err := h.adminService.GetProxy(ctx, int64(t.ProxyID))
		if err != nil {
			log.Printf("sync-proxies: proxy %d not found for tunnel %q: %v", t.ProxyID, t.Name, err)
			failed++
			continue
		}

		// Check if update is needed.
		if proxy.Host == agentHost && proxy.Port == t.SocksPort {
			skipped++
			continue
		}

		_, err = h.adminService.UpdateProxy(ctx, int64(t.ProxyID), &service.UpdateProxyInput{
			Host: agentHost,
			Port: t.SocksPort,
		})
		if err != nil {
			log.Printf("sync-proxies: failed to update proxy %d: %v", t.ProxyID, err)
			failed++
			continue
		}
		updated++
	}

	response.Success(c, map[string]int{
		"updated": updated,
		"skipped": skipped,
		"failed":  failed,
	})
}
