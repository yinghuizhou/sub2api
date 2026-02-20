package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// VendorHandler handles admin vendor management
type VendorHandler struct {
	vendorService  *service.VendorService
	probeService   *service.VendorProbeService
	healthService  *service.VendorHealthService
	balanceService *service.VendorBalanceService
}

// NewVendorHandler creates a new vendor handler
func NewVendorHandler(vendorService *service.VendorService, healthService *service.VendorHealthService, balanceService *service.VendorBalanceService) *VendorHandler {
	return &VendorHandler{
		vendorService:  vendorService,
		probeService:   service.NewVendorProbeService(),
		healthService:  healthService,
		balanceService: balanceService,
	}
}

// List GET /api/v1/admin/vendors
func (h *VendorHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := strings.TrimSpace(c.Query("status"))
	apiFormat := strings.TrimSpace(c.Query("api_format"))
	billingType := strings.TrimSpace(c.Query("billing_type"))
	search := strings.TrimSpace(c.Query("search"))

	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	vendors, paginationResult, err := h.vendorService.List(c.Request.Context(), params, status, apiFormat, billingType, search)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.PaginatedWithResult(c, vendors, &response.PaginationResult{
		Total:    paginationResult.Total,
		Page:     paginationResult.Page,
		PageSize: paginationResult.PageSize,
		Pages:    paginationResult.Pages,
	})
}

// GetByID GET /api/v1/admin/vendors/:id
func (h *VendorHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid vendor id")
		return
	}
	vendor, err := h.vendorService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, vendor)
}

// Create POST /api/v1/admin/vendors
func (h *VendorHandler) Create(c *gin.Context) {
	var input service.CreateVendorInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}
	vendor, err := h.vendorService.Create(c.Request.Context(), &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, vendor)
}

// Update PUT /api/v1/admin/vendors/:id
func (h *VendorHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid vendor id")
		return
	}
	var input service.UpdateVendorInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}
	vendor, err := h.vendorService.Update(c.Request.Context(), id, &input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, vendor)
}

// Delete DELETE /api/v1/admin/vendors/:id
func (h *VendorHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid vendor id")
		return
	}
	if err := h.vendorService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

// Test POST /api/v1/admin/vendors/:id/test
func (h *VendorHandler) Test(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid vendor id")
		return
	}
	result, err := h.healthService.RunHealthCheck(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// RefreshBalance POST /api/v1/admin/vendors/:id/refresh-balance
func (h *VendorHandler) RefreshBalance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid vendor id")
		return
	}
	vendor, err := h.vendorService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// 从 ExtraHeaders 中提取 API Key
	apiKey := extractAPIKey(vendor.ExtraHeaders)
	if apiKey == "" {
		response.BadRequest(c, "no API key found in vendor extra headers")
		return
	}
	result, err := h.probeService.Probe(c.Request.Context(), &service.VendorProbeRequest{
		URL:    vendor.BaseURL,
		APIKey: apiKey,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if result.BalanceUSD != nil {
		updateInput := &service.UpdateVendorInput{BalanceUSD: result.BalanceUSD}
		if _, updateErr := h.vendorService.Update(c.Request.Context(), id, updateInput); updateErr != nil {
			response.ErrorFrom(c, updateErr)
			return
		}
	}
	response.Success(c, result)
}

// extractAPIKey 从 ExtraHeaders 中提取 API Key
func extractAPIKey(headers map[string]string) string {
	for k, v := range headers {
		lower := strings.ToLower(k)
		if lower == "x-api-key" {
			return v
		}
		if lower == "authorization" {
			if strings.HasPrefix(v, "Bearer ") {
				return strings.TrimPrefix(v, "Bearer ")
			}
			return v
		}
	}
	return ""
}

// GetStats GET /api/v1/admin/vendors/:id/stats
func (h *VendorHandler) GetStats(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid vendor id")
		return
	}
	analysis, err := h.balanceService.GetCostAnalysis(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, analysis)
}

// Dashboard GET /api/v1/admin/vendors/dashboard
func (h *VendorHandler) Dashboard(c *gin.Context) {
	response.Success(c, gin.H{"status": "ok"})
}

// TriggerHealthCheck POST /api/v1/admin/vendors/:id/health-check
func (h *VendorHandler) TriggerHealthCheck(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid vendor id")
		return
	}
	result, err := h.healthService.RunHealthCheck(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetVendorAccounts GET /api/v1/admin/vendors/:id/accounts
func (h *VendorHandler) GetVendorAccounts(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid vendor id")
		return
	}
	// TODO: implement get vendor accounts
	_ = id
	response.Success(c, []any{})
}

// BatchCreateAccounts POST /api/v1/admin/vendors/:id/accounts
func (h *VendorHandler) BatchCreateAccounts(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid vendor id")
		return
	}
	// TODO: implement batch create accounts for vendor
	_ = id
	response.Success(c, []any{})
}

// Detect POST /api/v1/admin/vendors/detect
func (h *VendorHandler) Detect(c *gin.Context) {
	var req service.VendorProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}
	result, err := h.probeService.Probe(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
