package admin

import (
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// VpnHandler handles VPN management admin API endpoints.
type VpnHandler struct {
	vpnAgentService *service.VpnAgentService
}

// NewVpnHandler creates a new VpnHandler.
func NewVpnHandler(vpnAgentService *service.VpnAgentService) *VpnHandler {
	return &VpnHandler{vpnAgentService: vpnAgentService}
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

// CreateTunnel deploys a new VPN tunnel.
func (h *VpnHandler) CreateTunnel(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	var req service.CreateTunnelInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	tunnel, err := h.vpnAgentService.CreateTunnel(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
	}
	response.Created(c, tunnel)
}

// RemoveTunnel deletes a VPN tunnel.
func (h *VpnHandler) RemoveTunnel(c *gin.Context) {
	if !h.requireEnabled(c) {
		return
	}
	name := c.Param("name")
	if err := h.vpnAgentService.RemoveTunnel(c.Request.Context(), name); err != nil {
		response.Error(c, http.StatusBadGateway, "Agent error: "+err.Error())
		return
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
