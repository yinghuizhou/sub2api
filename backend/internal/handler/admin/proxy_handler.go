package admin

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

var validGroupName = regexp.MustCompile(`^[a-zA-Z0-9_\-]{1,100}$`)

// ProxyHandler handles admin proxy management
type ProxyHandler struct {
	adminService       service.AdminService
	proxyHealthService *service.ProxyHealthService
}

// NewProxyHandler creates a new admin proxy handler
func NewProxyHandler(adminService service.AdminService, proxyHealthService *service.ProxyHealthService) *ProxyHandler {
	return &ProxyHandler{
		adminService:       adminService,
		proxyHealthService: proxyHealthService,
	}
}

// CreateProxyRequest represents create proxy request
type CreateProxyRequest struct {
	Name         string `json:"name" binding:"required"`
	Protocol     string `json:"protocol" binding:"required,oneof=http https socks5 socks5h"`
	Host         string `json:"host" binding:"required"`
	Port         int    `json:"port" binding:"required,min=1,max=65535"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Region       string `json:"region"`
	GroupName    string `json:"group_name"`
	OvpnConfig   string `json:"ovpn_config"`
	OvpnUsername string `json:"ovpn_username"`
	OvpnPassword string `json:"ovpn_password"`
	IsDedicated  bool   `json:"is_dedicated"`
}

// UpdateProxyRequest represents update proxy request
type UpdateProxyRequest struct {
	Name         string  `json:"name"`
	Protocol     string  `json:"protocol" binding:"omitempty,oneof=http https socks5 socks5h"`
	Host         string  `json:"host"`
	Port         int     `json:"port" binding:"omitempty,min=1,max=65535"`
	Username     string  `json:"username"`
	Password     string  `json:"password"`
	Status       string  `json:"status" binding:"omitempty,oneof=active inactive"`
	Region       *string `json:"region"`
	GroupName    *string `json:"group_name"`
	OvpnConfig   *string `json:"ovpn_config"`
	OvpnUsername *string `json:"ovpn_username"`
	OvpnPassword *string `json:"ovpn_password"`
	IsDedicated  *bool   `json:"is_dedicated"`
}

// List handles listing all proxies with pagination
// GET /api/v1/admin/proxies
func (h *ProxyHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	protocol := c.Query("protocol")
	status := c.Query("status")
	search := c.Query("search")
	// 标准化和验证 search 参数
	search = strings.TrimSpace(search)
	if len(search) > 100 {
		search = search[:100]
	}

	proxies, total, err := h.adminService.ListProxiesWithAccountCount(c.Request.Context(), page, pageSize, protocol, status, search)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.ProxyWithAccountCount, 0, len(proxies))
	for i := range proxies {
		out = append(out, *dto.ProxyWithAccountCountFromService(&proxies[i]))
	}
	response.Paginated(c, out, total, page, pageSize)
}

// GetAll handles getting all active proxies without pagination
// GET /api/v1/admin/proxies/all
// Optional query param: with_count=true to include account count per proxy
func (h *ProxyHandler) GetAll(c *gin.Context) {
	withCount := c.Query("with_count") == "true"

	if withCount {
		proxies, err := h.adminService.GetAllProxiesWithAccountCount(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		out := make([]dto.ProxyWithAccountCount, 0, len(proxies))
		for i := range proxies {
			out = append(out, *dto.ProxyWithAccountCountFromService(&proxies[i]))
		}
		response.Success(c, out)
		return
	}

	proxies, err := h.adminService.GetAllProxies(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.Proxy, 0, len(proxies))
	for i := range proxies {
		out = append(out, *dto.ProxyFromService(&proxies[i]))
	}
	response.Success(c, out)
}

// GetByID handles getting a proxy by ID
// GET /api/v1/admin/proxies/:id
func (h *ProxyHandler) GetByID(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	proxy, err := h.adminService.GetProxy(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.ProxyFromService(proxy))
}

// Create handles creating a new proxy
// POST /api/v1/admin/proxies
func (h *ProxyHandler) Create(c *gin.Context) {
	var req CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	executeAdminIdempotentJSON(c, "admin.proxies.create", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		proxy, err := h.adminService.CreateProxy(ctx, &service.CreateProxyInput{
			Name:         strings.TrimSpace(req.Name),
			Protocol:     strings.TrimSpace(req.Protocol),
			Host:         strings.TrimSpace(req.Host),
			Port:         req.Port,
			Username:     strings.TrimSpace(req.Username),
			Password:     strings.TrimSpace(req.Password),
			Region:       strings.TrimSpace(req.Region),
			GroupName:    strings.TrimSpace(req.GroupName),
			OvpnConfig:   strings.TrimSpace(req.OvpnConfig),
			OvpnUsername: strings.TrimSpace(req.OvpnUsername),
			OvpnPassword: strings.TrimSpace(req.OvpnPassword),
			IsDedicated:  req.IsDedicated,
		})
		if err != nil {
			return nil, err
		}
		return dto.ProxyFromService(proxy), nil
	})
}

// Update handles updating a proxy
// PUT /api/v1/admin/proxies/:id
func (h *ProxyHandler) Update(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	var req UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	updateInput := &service.UpdateProxyInput{
		Name:         strings.TrimSpace(req.Name),
		Protocol:     strings.TrimSpace(req.Protocol),
		Host:         strings.TrimSpace(req.Host),
		Port:         req.Port,
		Username:     strings.TrimSpace(req.Username),
		Password:     strings.TrimSpace(req.Password),
		Status:       strings.TrimSpace(req.Status),
		Region:       req.Region,
		GroupName:    req.GroupName,
		OvpnConfig:   req.OvpnConfig,
		OvpnUsername: req.OvpnUsername,
		OvpnPassword: req.OvpnPassword,
		IsDedicated:  req.IsDedicated,
	}
	proxy, err := h.adminService.UpdateProxy(c.Request.Context(), proxyID, updateInput)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.ProxyFromService(proxy))
}

// Delete handles deleting a proxy
// DELETE /api/v1/admin/proxies/:id
func (h *ProxyHandler) Delete(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	err = h.adminService.DeleteProxy(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Proxy deleted successfully"})
}

// BatchDelete handles batch deleting proxies
// POST /api/v1/admin/proxies/batch-delete
func (h *ProxyHandler) BatchDelete(c *gin.Context) {
	type BatchDeleteRequest struct {
		IDs []int64 `json:"ids" binding:"required,min=1"`
	}

	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.adminService.BatchDeleteProxies(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// Test handles testing proxy connectivity
// POST /api/v1/admin/proxies/:id/test
func (h *ProxyHandler) Test(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	result, err := h.adminService.TestProxy(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// CheckQuality handles checking proxy quality across common AI targets.
// POST /api/v1/admin/proxies/:id/quality-check
func (h *ProxyHandler) CheckQuality(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	result, err := h.adminService.CheckProxyQuality(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// GetStats handles getting proxy statistics
// GET /api/v1/admin/proxies/:id/stats
func (h *ProxyHandler) GetStats(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	// Return mock data for now
	_ = proxyID
	response.Success(c, gin.H{
		"total_accounts":  0,
		"active_accounts": 0,
		"total_requests":  0,
		"success_rate":    100.0,
		"average_latency": 0,
	})
}

// GetProxyAccounts handles getting accounts using a proxy
// GET /api/v1/admin/proxies/:id/accounts
func (h *ProxyHandler) GetProxyAccounts(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	accounts, err := h.adminService.GetProxyAccounts(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.ProxyAccountSummary, 0, len(accounts))
	for i := range accounts {
		out = append(out, *dto.ProxyAccountSummaryFromService(&accounts[i]))
	}
	response.Success(c, out)
}

// GetGroupNames handles listing all distinct proxy group names
// GET /api/v1/admin/proxies/group-names
func (h *ProxyHandler) GetGroupNames(c *gin.Context) {
	names, err := h.adminService.ListProxyGroupNames(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, names)
}

// BatchCreateProxyItem represents a single proxy in batch create request
type BatchCreateProxyItem struct {
	Protocol string `json:"protocol" binding:"required,oneof=http https socks5 socks5h"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required,min=1,max=65535"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// BatchCreateRequest represents batch create proxies request
type BatchCreateRequest struct {
	Proxies []BatchCreateProxyItem `json:"proxies" binding:"required,min=1"`
}

// BatchCreate handles batch creating proxies
// POST /api/v1/admin/proxies/batch
func (h *ProxyHandler) BatchCreate(c *gin.Context) {
	var req BatchCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	created := 0
	skipped := 0

	for _, item := range req.Proxies {
		// Trim all string fields
		host := strings.TrimSpace(item.Host)
		protocol := strings.TrimSpace(item.Protocol)
		username := strings.TrimSpace(item.Username)
		password := strings.TrimSpace(item.Password)

		// Check for duplicates (same host, port, username, password)
		exists, err := h.adminService.CheckProxyExists(c.Request.Context(), host, item.Port, username, password)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}

		if exists {
			skipped++
			continue
		}

		// Create proxy with default name
		_, err = h.adminService.CreateProxy(c.Request.Context(), &service.CreateProxyInput{
			Name:     "default",
			Protocol: protocol,
			Host:     host,
			Port:     item.Port,
			Username: username,
			Password: password,
		})
		if err != nil {
			// If creation fails due to duplicate, count as skipped
			skipped++
			continue
		}

		created++
	}

	response.Success(c, gin.H{
		"created": created,
		"skipped": skipped,
	})
}

// HealthCheck triggers an immediate health check for a single proxy
// POST /api/v1/admin/proxies/:id/health-check
func (h *ProxyHandler) HealthCheck(c *gin.Context) {
	proxyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	proxy, err := h.proxyHealthService.CheckOne(c.Request.Context(), proxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.ProxyFromService(proxy))
}

// HealthCheckAll triggers a full health check cycle for all active proxies
// POST /api/v1/admin/proxies/health-check-all
func (h *ProxyHandler) HealthCheckAll(c *gin.Context) {
	h.proxyHealthService.CheckAll(c.Request.Context())
	response.Success(c, gin.H{"message": "Health check completed for all active proxies"})
}

// GetGroupSummary returns summary info for a proxy group (proxy list + assignment counts)
// GET /api/v1/admin/proxies/groups/:name/summary
func (h *ProxyHandler) GetGroupSummary(c *gin.Context) {
	name := c.Param("name")
	if !validGroupName.MatchString(name) {
		response.BadRequest(c, "Invalid group name")
		return
	}

	summary, err := h.adminService.GetProxyGroupSummary(c.Request.Context(), name)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

// GetGroupAssignments returns the list of account-proxy assignments for a group
// GET /api/v1/admin/proxies/groups/:name/assignments
func (h *ProxyHandler) GetGroupAssignments(c *gin.Context) {
	name := c.Param("name")
	if !validGroupName.MatchString(name) {
		response.BadRequest(c, "Invalid group name")
		return
	}

	assignments, err := h.adminService.ListProxyAssignments(c.Request.Context(), name)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, assignments)
}

// DistributeAccounts evenly distributes accounts to proxies within a group
// POST /api/v1/admin/proxies/groups/:name/distribute
func (h *ProxyHandler) DistributeAccounts(c *gin.Context) {
	name := c.Param("name")
	if !validGroupName.MatchString(name) {
		response.BadRequest(c, "Invalid group name")
		return
	}

	var req struct {
		Strategy string `json:"strategy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Strategy = "round-robin"
	}

	result, err := h.adminService.DistributeAccountsToGroup(c.Request.Context(), name, req.Strategy)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ReassignAccount reassigns a single account to a different proxy
// PUT /api/v1/admin/proxies/assignments/:accountId
func (h *ProxyHandler) ReassignAccount(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("accountId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	var req struct {
		ProxyID int64 `json:"proxy_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.adminService.ReassignAccountProxy(c.Request.Context(), accountID, req.ProxyID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Account reassigned successfully"})
}
