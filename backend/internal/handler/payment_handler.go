package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles payment-related requests.
type PaymentHandler struct {
	paymentService *service.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler.
func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: svc}
}

// CreateOrderRequest is the request body for creating a payment order.
type CreateOrderRequest struct {
	AmountCNY float64 `json:"amount_cny" binding:"required,gt=0"`
	PackageID *int64  `json:"package_id"`
	Channel   string  `json:"channel" binding:"required,oneof=wechat alipay"`
}

// CreateOrder handles POST /api/v1/payment/create
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	order, err := h.paymentService.CreateOrder(c.Request.Context(), &service.CreateOrderInput{
		UserID:    subject.UserID,
		AmountCNY: req.AmountCNY,
		PackageID: req.PackageID,
		Channel:   req.Channel,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, order)
}

// GetOrder handles GET /api/v1/payment/orders/:id
func (h *PaymentHandler) GetOrder(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid order id")
		return
	}
	order, err := h.paymentService.GetOrderByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// Prevent IDOR: ensure the order belongs to the requesting user
	if order.UserID != subject.UserID {
		response.NotFound(c, "order not found")
		return
	}
	response.Success(c, order)
}

// ListPackages handles GET /api/v1/payment/packages
func (h *PaymentHandler) ListPackages(c *gin.Context) {
	pkgs, err := h.paymentService.ListPackages(c.Request.Context(), true)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, pkgs)
}

// WechatCallback handles POST /api/v1/payment/callback/wechat
func (h *PaymentHandler) WechatCallback(c *gin.Context) {
	// TODO: parse wechat V3 notification, verify signature, call HandleCallback
	c.JSON(501, gin.H{"code": "NOT_IMPLEMENTED", "message": "wechat callback not implemented"})
}

// AlipayCallback handles POST /api/v1/payment/callback/alipay
func (h *PaymentHandler) AlipayCallback(c *gin.Context) {
	// TODO: parse alipay notification, verify signature, call HandleCallback
	c.JSON(501, gin.H{"code": "NOT_IMPLEMENTED", "message": "alipay callback not implemented"})
}
