package handler

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
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
	wxPubKey       *rsa.PublicKey // cached WeChat public key, loaded once at startup
}

// NewPaymentHandler creates a new PaymentHandler.
// Loads and caches the WeChat public key at startup to avoid per-request disk I/O.
func NewPaymentHandler(svc *service.PaymentService, cfg *config.Config) *PaymentHandler {
	h := &PaymentHandler{paymentService: svc, cfg: cfg}
	// Pre-load WeChat public key if configured
	if wxCfg := cfg.Payment.Wechat; wxCfg.PublicKeyPath != "" {
		pemBytes, err := os.ReadFile(wxCfg.PublicKeyPath)
		if err != nil {
			logger.LegacyPrintf("handler.payment", "[PaymentHandler] WARNING: failed to read wechat public key: %v", err)
		} else if block, _ := pem.Decode(pemBytes); block == nil {
			logger.LegacyPrintf("handler.payment", "[PaymentHandler] WARNING: invalid PEM data in %s", wxCfg.PublicKeyPath)
		} else if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err != nil {
			logger.LegacyPrintf("handler.payment", "[PaymentHandler] WARNING: parse public key failed: %v", err)
		} else if rsaPub, ok := pub.(*rsa.PublicKey); !ok {
			logger.LegacyPrintf("handler.payment", "[PaymentHandler] WARNING: not an RSA public key")
		} else {
			h.wxPubKey = rsaPub
		}
	}
	return h
}

// CreateOrderRequest is the request body for creating a payment order.
type CreateOrderRequest struct {
	AmountCNY float64 `json:"amount_cny" binding:"required,gte=1,lte=100000"`
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
	if wxCfg.APIV3Key == "" || h.wxPubKey == nil {
		// M1: Return 200 for deterministic "not configured" — stops WeChat from retrying indefinitely.
		logger.LegacyPrintf("handler.payment", "[WechatCallback] ERROR: wechat payment not configured, acking to stop retries")
		c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: "FAIL", Message: "not configured"})
		return
	}

	notifyReq, err := wechat.V3ParseNotify(c.Request)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] parse notify failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": "PARSE_FAILED"})
		return
	}

	// Verify signature using cached public key (C4: no disk I/O per request)
	if err := notifyReq.VerifySignByPK(h.wxPubKey); err != nil {
		// M1: Signature failure is deterministic — return 200 to stop retries for forged/invalid requests.
		logger.LegacyPrintf("handler.payment", "[WechatCallback] signature verification failed: %v", err)
		c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: "FAIL", Message: "signature invalid"})
		return
	}

	payResult, err := notifyReq.DecryptPayCipherText(wxCfg.APIV3Key)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] decrypt pay result failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": "DECRYPT_FAILED"})
		return
	}

	// B1: Check trade state — only process successful payments
	if payResult.TradeState != "SUCCESS" {
		logger.LegacyPrintf("handler.payment", "[WechatCallback] non-success trade state: %s for order %s", payResult.TradeState, payResult.OutTradeNo)
		c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: "SUCCESS", Message: "成功"})
		return
	}

	// B2: Verify payment amount matches order (WeChat amount is in cents)
	paidAmountCNY := float64(payResult.Amount.Total) / 100.0
	if err := h.paymentService.HandleCallback(c.Request.Context(), payResult.OutTradeNo, payResult.TransactionId, paidAmountCNY); err != nil {
		// B1: Amount mismatch is a deterministic error — return 200 to stop WeChat retries.
		// The order has been marked as "failed" by the service layer.
		if errors.Is(err, service.ErrPaymentAmountMismatch) {
			logger.LegacyPrintf("handler.payment", "[WechatCallback] amount mismatch for order %s, marked as failed", payResult.OutTradeNo)
			c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: "SUCCESS", Message: "成功"})
			return
		}
		logger.LegacyPrintf("handler.payment", "[WechatCallback] handle callback failed for order %s: %v", payResult.OutTradeNo, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "PROCESS_FAILED"})
		return
	}

	c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: "SUCCESS", Message: "成功"})
}

// AlipayCallback handles POST /api/v1/payment/callback/alipay
func (h *PaymentHandler) AlipayCallback(c *gin.Context) {
	aliCfg := h.cfg.Payment.Alipay
	if aliCfg.PublicKey == "" || aliCfg.AppID == "" {
		// M1: Return 200/"success" for deterministic "not configured" — stops Alipay from retrying.
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] ERROR: alipay payment not configured, acking to stop retries")
		c.String(http.StatusOK, "success")
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
		// M1: Signature failure is deterministic — return 200/"success" to stop Alipay retries.
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] signature verification failed: %v, ok=%v", err, ok)
		c.String(http.StatusOK, "success")
		return
	}

	// B3: Verify app_id to prevent cross-merchant notification replay
	if notifyReq.Get("app_id") != aliCfg.AppID {
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] app_id mismatch: got %s, expected %s", notifyReq.Get("app_id"), aliCfg.AppID)
		c.String(http.StatusOK, "success")
		return
	}

	orderNo := notifyReq.Get("out_trade_no")
	tradeNo := notifyReq.Get("trade_no")
	tradeStatus := notifyReq.Get("trade_status")

	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		c.String(http.StatusOK, "success")
		return
	}

	// B2: Verify payment amount matches order
	totalAmountStr := notifyReq.Get("total_amount")
	paidAmountCNY, err := strconv.ParseFloat(totalAmountStr, 64)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] invalid total_amount: %s", totalAmountStr)
		c.String(http.StatusBadRequest, "invalid amount")
		return
	}

	if err := h.paymentService.HandleCallback(c.Request.Context(), orderNo, tradeNo, paidAmountCNY); err != nil {
		// B1: Amount mismatch is deterministic — return 200 to stop Alipay retries.
		if errors.Is(err, service.ErrPaymentAmountMismatch) {
			logger.LegacyPrintf("handler.payment", "[AlipayCallback] amount mismatch for order %s, marked as failed", orderNo)
			c.String(http.StatusOK, "success")
			return
		}
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] handle callback failed for order %s: %v", orderNo, err)
		c.String(http.StatusInternalServerError, "process failed")
		return
	}

	c.String(http.StatusOK, "success")
}
