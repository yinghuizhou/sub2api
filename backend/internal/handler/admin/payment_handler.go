package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// PaymentHandler handles admin payment management endpoints.
type PaymentHandler struct {
	paymentService *service.PaymentService
}

// NewAdminPaymentHandler creates a new admin PaymentHandler.
func NewAdminPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: svc}
}

type UpsertPackageRequest struct {
	AmountCNY  float64 `json:"amount_cny" binding:"required,gt=0"`
	BonusRate  float64 `json:"bonus_rate" binding:"min=0,max=1"`
	BonusFixed float64 `json:"bonus_fixed" binding:"min=0"`
	Label      *string `json:"label"`
	IsActive   bool    `json:"is_active"`
	SortOrder  int     `json:"sort_order"`
}

func (h *PaymentHandler) ListPackages(c *gin.Context) {
	pkgs, err := h.paymentService.ListPackages(c.Request.Context(), false)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, pkgs)
}

func (h *PaymentHandler) CreatePackage(c *gin.Context) {
	var req UpsertPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	pkg, err := h.paymentService.CreatePackage(c.Request.Context(), &service.RechargePackage{
		AmountCNY:  req.AmountCNY,
		BonusRate:  req.BonusRate,
		BonusFixed: req.BonusFixed,
		Label:      req.Label,
		IsActive:   req.IsActive,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, pkg)
}

func (h *PaymentHandler) UpdatePackage(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid package id")
		return
	}
	var req UpsertPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	pkg, err := h.paymentService.UpdatePackage(c.Request.Context(), id, &service.RechargePackage{
		AmountCNY:  req.AmountCNY,
		BonusRate:  req.BonusRate,
		BonusFixed: req.BonusFixed,
		Label:      req.Label,
		IsActive:   req.IsActive,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, pkg)
}

func (h *PaymentHandler) DeletePackage(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid package id")
		return
	}
	if err := h.paymentService.DeletePackage(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *PaymentHandler) ListOrders(c *gin.Context) {
	response.Success(c, []any{})
}
