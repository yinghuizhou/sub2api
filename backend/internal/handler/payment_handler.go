package handler

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	wechat "github.com/go-pay/gopay/wechat/v3"

	"github.com/gin-gonic/gin"
)

// sanitizeLogValue removes newlines and control characters from external input
// to prevent log injection attacks.
func sanitizeLogValue(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r < 0x20 {
			return '_'
		}
		return r
	}, s)
}

// PaymentHandler handles payment-related requests.
type PaymentHandler struct {
	paymentService *service.PaymentService
	settingService *service.SettingService
	cfg            *config.Config
	wxPubKey       *rsa.PublicKey    // cached WeChat public key for callback verification
	wxClient       *wechat.ClientV3 // WeChat V3 client for creating payments
	aliClient      *alipay.Client   // Alipay client for creating payments
}

// CreateOrderResponse is the response for creating a payment order.
type CreateOrderResponse struct {
	Order *service.PaymentOrder `json:"order"`
	QrURL string               `json:"qr_url"`
}

// NewPaymentHandler creates a new PaymentHandler.
// Loads and caches payment credentials at startup to avoid per-request disk I/O.
func NewPaymentHandler(svc *service.PaymentService, settingSvc *service.SettingService, cfg *config.Config) *PaymentHandler {
	h := &PaymentHandler{paymentService: svc, settingService: settingSvc, cfg: cfg}
	wxCfg := cfg.Payment.Wechat

	// Pre-load WeChat public key for callback verification
	if wxCfg.PublicKeyPath != "" {
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

	// Initialize WeChat V3 client for creating payments
	if wxCfg.MchID != "" && wxCfg.SerialNo != "" && wxCfg.APIV3Key != "" && wxCfg.PrivateKeyPath != "" {
		privBytes, err := os.ReadFile(wxCfg.PrivateKeyPath)
		if err != nil {
			logger.LegacyPrintf("handler.payment", "[PaymentHandler] WARNING: failed to read wechat private key: %v", err)
		} else {
			client, err := wechat.NewClientV3(wxCfg.MchID, wxCfg.SerialNo, wxCfg.APIV3Key, string(privBytes))
			if err != nil {
				logger.LegacyPrintf("handler.payment", "[PaymentHandler] WARNING: failed to create wechat V3 client: %v", err)
			} else {
				h.wxClient = client
			}
		}
	}

	// Initialize Alipay client for creating payments
	aliCfg := cfg.Payment.Alipay
	if aliCfg.AppID != "" && aliCfg.PrivateKey != "" {
		client, err := alipay.NewClient(aliCfg.AppID, aliCfg.PrivateKey, aliCfg.IsProd)
		if err != nil {
			logger.LegacyPrintf("handler.payment", "[PaymentHandler] WARNING: failed to create alipay client: %v", err)
		} else {
			if aliCfg.NotifyURL != "" {
				client.SetNotifyUrl(aliCfg.NotifyURL)
			}
			h.aliClient = client
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
	if h.settingService != nil && !h.settingService.IsPaymentEnabled(c.Request.Context()) {
		response.ErrorFrom(c, infraerrors.Forbidden("PAYMENT_DISABLED", "充值功能未开启"))
		return
	}
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

	// Call payment provider to get QR code URL
	var qrURL string
	switch req.Channel {
	case "wechat":
		qrURL, err = h.createWechatPayment(c.Request.Context(), order)
	case "alipay":
		qrURL, err = h.createAlipayPayment(c.Request.Context(), order)
	}
	if err != nil {
		// Cancel the orphaned pending order to prevent data pollution
		h.paymentService.CancelOrder(c.Request.Context(), order.OrderNo)
		logger.LegacyPrintf("handler.payment", "[CreateOrder] failed to create %s payment for order %s: %v", req.Channel, order.OrderNo, err)
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, CreateOrderResponse{Order: order, QrURL: qrURL})
}

// createWechatPayment calls WeChat V3 Native Pay API to get a QR code URL.
func (h *PaymentHandler) createWechatPayment(ctx context.Context, order *service.PaymentOrder) (string, error) {
	if h.wxClient == nil {
		return "", infraerrors.ServiceUnavailable("WECHAT_NOT_CONFIGURED", "微信支付未配置")
	}
	wxCfg := h.cfg.Payment.Wechat
	amountCents := int(math.Round(order.AmountCNY * 100))

	bm := make(gopay.BodyMap)
	bm.Set("appid", wxCfg.AppID).
		Set("mchid", wxCfg.MchID).
		Set("description", fmt.Sprintf("充值 ¥%.0f", order.AmountCNY)).
		Set("out_trade_no", order.OrderNo).
		Set("notify_url", wxCfg.NotifyURL).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("total", amountCents).
				Set("currency", "CNY")
		})

	rsp, err := h.wxClient.V3TransactionNative(ctx, bm)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[createWechatPayment] wechat native pay error: %v", err)
		return "", infraerrors.ServiceUnavailable("WECHAT_PAY_FAILED", "微信支付请求失败，请稍后重试")
	}
	if rsp == nil || rsp.Response == nil || rsp.Response.CodeUrl == "" {
		return "", infraerrors.ServiceUnavailable("WECHAT_EMPTY_RESPONSE", "微信支付返回异常，请稍后重试")
	}
	return rsp.Response.CodeUrl, nil
}

// createAlipayPayment calls Alipay TradePrecreate API to get a QR code URL.
func (h *PaymentHandler) createAlipayPayment(ctx context.Context, order *service.PaymentOrder) (string, error) {
	if h.aliClient == nil {
		return "", infraerrors.ServiceUnavailable("ALIPAY_NOT_CONFIGURED", "支付宝未配置")
	}

	bm := make(gopay.BodyMap)
	bm.Set("subject", fmt.Sprintf("充值 ¥%.0f", order.AmountCNY)).
		Set("out_trade_no", order.OrderNo).
		Set("total_amount", fmt.Sprintf("%.2f", order.AmountCNY))

	rsp, err := h.aliClient.TradePrecreate(ctx, bm)
	if err != nil {
		logger.LegacyPrintf("handler.payment", "[createAlipayPayment] alipay precreate error: %v", err)
		return "", infraerrors.ServiceUnavailable("ALIPAY_PAY_FAILED", "支付宝请求失败，请稍后重试")
	}
	if rsp == nil || rsp.Response == nil || rsp.Response.QrCode == "" {
		return "", infraerrors.ServiceUnavailable("ALIPAY_EMPTY_RESPONSE", "支付宝返回异常，请稍后重试")
	}
	return rsp.Response.QrCode, nil
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
		// M1: Parse failure is deterministic (malformed body) — return 200 to stop infinite retries.
		logger.LegacyPrintf("handler.payment", "[WechatCallback] parse notify failed: %v", err)
		c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: "FAIL", Message: "parse failed"})
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
		// M1: Decrypt failure is deterministic (wrong key or corrupt ciphertext) — return 200 to stop retries.
		logger.LegacyPrintf("handler.payment", "[WechatCallback] decrypt pay result failed: %v", err)
		c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: "FAIL", Message: "decrypt failed"})
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
		// M2: Use standard V3NotifyRsp format (not gin.H) for consistent WeChat protocol compliance.
		// 5xx triggers WeChat retry, which is desired for transient DB failures.
		c.JSON(http.StatusInternalServerError, &wechat.V3NotifyRsp{Code: "FAIL", Message: "process failed"})
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
		// M1: Parse failure is deterministic — return 200/"success" to stop Alipay retries.
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] parse notify failed: %v", err)
		c.String(http.StatusOK, "success")
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

	// M2: Sanitize external input to prevent log injection via newlines/control chars.
	orderNo := sanitizeLogValue(notifyReq.Get("out_trade_no"))
	tradeNo := sanitizeLogValue(notifyReq.Get("trade_no"))
	tradeStatus := notifyReq.Get("trade_status")

	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		c.String(http.StatusOK, "success")
		return
	}

	// B2: Verify payment amount matches order
	totalAmountStr := notifyReq.Get("total_amount")
	paidAmountCNY, err := strconv.ParseFloat(totalAmountStr, 64)
	if err != nil {
		// M1: Invalid amount format is deterministic — return 200 to stop retries.
		logger.LegacyPrintf("handler.payment", "[AlipayCallback] invalid total_amount: %s", totalAmountStr)
		c.String(http.StatusOK, "success")
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
