package handler

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/go-pay/gopay/alipay"
	wechat "github.com/go-pay/gopay/wechat/v3"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles payment-related requests.
type PaymentHandler struct {
	paymentService *service.PaymentService
	cfg            *config.Config
}

// NewPaymentHandler creates a new PaymentHandler.
func NewPaymentHandler(svc *service.PaymentService, cfg *config.Config) *PaymentHandler {
	return &PaymentHandler{paymentService: svc, cfg: cfg}
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
	wxCfg := h.cfg.Payment.Wechat
	if wxCfg.APIV3Key == "" || wxCfg.PublicKeyPath == "" {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] wechat payment not configured")
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "NOT_CONFIGURED"})
		return
	}

	notifyReq, err := wechat.V3ParseNotify(c.Request)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] parse notify failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": "PARSE_FAILED"})
		return
	}

	pemBytes, err := os.ReadFile(wxCfg.PublicKeyPath)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] read public key failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR"})
		return
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] invalid PEM data")
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR"})
		return
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] parse public key failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR"})
		return
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] not an RSA public key")
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR"})
		return
	}

	if err := notifyReq.VerifySignByPK(rsaPub); err != nil {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] signature verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"code": "SIGNATURE_INVALID"})
		return
	}

	payResult, err := notifyReq.DecryptPayCipherText(wxCfg.APIV3Key)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] decrypt pay result failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": "DECRYPT_FAILED"})
		return
	}

	if err := h.paymentService.HandleCallback(c.Request.Context(), payResult.OutTradeNo, payResult.TransactionId); err != nil {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] handle callback failed for order %s: %v", payResult.OutTradeNo, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "PROCESS_FAILED"})
		return
	}

	c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: "SUCCESS", Message: "成功"})
}

// AlipayCallback handles POST /api/v1/payment/callback/alipay
func (h *PaymentHandler) AlipayCallback(c *gin.Context) {
	aliCfg := h.cfg.Payment.Alipay
	if aliCfg.PublicKey == "" {
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] alipay payment not configured")
		c.String(http.StatusServiceUnavailable, "not configured")
		return
	}

	notifyReq, err := alipay.ParseNotifyToBodyMap(c.Request)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] parse notify failed: %v", err)
		c.String(http.StatusBadRequest, "parse failed")
		return
	}

	ok, err := alipay.VerifySign(aliCfg.PublicKey, notifyReq)
	if err != nil || !ok {
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] signature verification failed: %v, ok=%v", err, ok)
		c.String(http.StatusUnauthorized, "signature invalid")
		return
	}

	orderNo := notifyReq.Get("out_trade_no")
	tradeNo := notifyReq.Get("trade_no")
	tradeStatus := notifyReq.Get("trade_status")

	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		c.String(http.StatusOK, "success")
		return
	}

	if err := h.paymentService.HandleCallback(c.Request.Context(), orderNo, tradeNo); err != nil {
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] handle callback failed for order %s: %v", orderNo, err)
		c.String(http.StatusInternalServerError, "process failed")
		return
	}

	c.String(http.StatusOK, "success")
}
