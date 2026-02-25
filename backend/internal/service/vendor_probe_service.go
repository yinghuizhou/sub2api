package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

// VendorProbeRequest 探测请求
type VendorProbeRequest struct {
	URL    string `json:"url" binding:"required"`
	APIKey string `json:"api_key" binding:"required"`
}

// VendorProbeResult 探测结果
type VendorProbeResult struct {
	// 探测状态
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`

	// 自动填充建议
	APIFormat    string `json:"api_format"` // "openai" | "anthropic"
	BaseURL      string `json:"base_url"`
	PlatformType string `json:"platform_type"` // "new-api" | "one-api" | "openai" | "anthropic" | "unknown"

	// 可用模型
	Models []string `json:"models"`

	// 余额信息（无管理员权限时可能为 nil）
	BalanceUSD *float64 `json:"balance_usd,omitempty"`
	BalanceRaw string   `json:"balance_raw,omitempty"` // 原始余额字符串（如 "¥10.00"、"500000 tokens"）

	// 延迟
	LatencyMS int `json:"latency_ms"`

	DetectedAt time.Time `json:"detected_at"`
}

// VendorProbeService 供应商探测服务
type VendorProbeService struct {
	httpClient *http.Client
	sfGroup    singleflight.Group
}

// NewVendorProbeService 创建探测服务
func NewVendorProbeService() *VendorProbeService {
	return &VendorProbeService{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Probe 探测供应商 URL，返回平台信息和余额
func (s *VendorProbeService) Probe(ctx context.Context, req *VendorProbeRequest) (*VendorProbeResult, error) {
	baseURL := strings.TrimRight(req.URL, "/")
	// singleflight: 对同一 URL 的并发探测请求合并为一次
	sfKey := baseURL
	v, err, _ := s.sfGroup.Do(sfKey, func() (any, error) {
		return s.doProbe(ctx, baseURL, req.APIKey)
	})
	if err != nil {
		return nil, err
	}
	result, ok := v.(*VendorProbeResult)
	if !ok {
		return nil, fmt.Errorf("unexpected probe result type: %T", v)
	}
	return result, nil
}

func (s *VendorProbeService) doProbe(ctx context.Context, baseURL, apiKey string) (*VendorProbeResult, error) {
	result := &VendorProbeResult{
		BaseURL:    baseURL,
		DetectedAt: time.Now(),
	}

	start := time.Now()

	// Step 1: 探测平台类型并获取模型列表（始终使用 OpenAI 格式的 /v1/models）
	models, platformType, err := s.probeModels(ctx, baseURL, apiKey)
	result.LatencyMS = int(time.Since(start).Milliseconds())

	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("API Key 验证失败: %v", err)
		return result, nil
	}

	result.Success = true
	result.Models = models
	result.PlatformType = platformType
	// anthropic-proxy 说明 x-api-key 认证成功，使用 anthropic 格式
	if platformType == "anthropic-proxy" {
		result.APIFormat = "anthropic"
	} else {
		result.APIFormat = "openai"
	}

	// Step 2: 尝试获取余额（多种策略，取第一个成功的）
	balance, balanceRaw := s.probeBalance(ctx, baseURL, apiKey, platformType)
	result.BalanceUSD = balance
	result.BalanceRaw = balanceRaw

	if balance != nil {
		result.Message = fmt.Sprintf("检测成功，发现 %d 个模型，余额 $%.2f", len(models), *balance)
	} else {
		msg := fmt.Sprintf("检测成功，发现 %d 个模型", len(models))
		if balanceRaw != "" {
			msg += fmt.Sprintf("，余额: %s", balanceRaw)
		} else {
			msg += "（余额需手动填写）"
		}
		result.Message = msg
	}

	return result, nil
}

// probeModels 探测模型列表并检测平台类型
// 自动尝试 OpenAI 格式（Authorization: Bearer）和 Anthropic 格式（x-api-key）
func (s *VendorProbeService) probeModels(ctx context.Context, baseURL, apiKey string) ([]string, string, error) {
	// 策略列表：优先试 OpenAI Bearer 格式，失败再试 Anthropic x-api-key 格式
	type authStrategy struct {
		header string
		value  string
		name   string
	}
	strategies := []authStrategy{
		{"Authorization", "Bearer " + apiKey, "openai"},
		{"x-api-key", apiKey, "anthropic"},
	}

	var lastErr error
	for _, strategy := range strategies {
		models, platformType, err := s.probeModelsWithAuth(ctx, baseURL, apiKey, strategy.header, strategy.value, strategy.name)
		if err == nil {
			return models, platformType, nil
		}
		lastErr = err
	}
	return nil, "unknown", lastErr
}

// probeModelsWithAuth 使用指定认证头探测模型列表
func (s *VendorProbeService) probeModelsWithAuth(ctx context.Context, baseURL, apiKey, authHeader, authValue, authStyle string) ([]string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, "unknown", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set(authHeader, authValue)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "unknown", fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 最多读 1MB
	if err != nil {
		return nil, "unknown", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, "unknown", fmt.Errorf("API Key 无效（HTTP %d）", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "unknown", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	// 解析模型列表
	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Object string `json:"object"`
	}
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, "unknown", fmt.Errorf("parse models: %w", err)
	}

	models := make([]string, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}

	// 检测平台类型（通过响应头、内容和认证方式）
	platformType := detectPlatformType(resp, body)
	// 如果是 x-api-key 认证成功，说明是 Anthropic 格式的代理
	if authStyle == "anthropic" && platformType == "openai-compatible" {
		platformType = "anthropic-proxy"
	}

	return models, platformType, nil
}

// detectPlatformType 通过响应头和内容检测平台类型
func detectPlatformType(resp *http.Response, body []byte) string {
	// 检查响应头特征
	server := strings.ToLower(resp.Header.Get("Server"))
	xVersion := resp.Header.Get("X-Oneapi-Version")
	if xVersion != "" {
		return "one-api"
	}

	// New API 特有的响应头
	if strings.Contains(server, "new-api") || resp.Header.Get("X-New-Api-Version") != "" {
		return "new-api"
	}

	// 通过响应内容检测
	bodyStr := string(body)
	if strings.Contains(bodyStr, `"owned_by":"platform"`) ||
		strings.Contains(bodyStr, `"owned_by": "platform"`) {
		return "new-api"
	}

	return "openai-compatible"
}

// probeBalance 尝试多种策略获取余额
func (s *VendorProbeService) probeBalance(ctx context.Context, baseURL, apiKey, platformType string) (*float64, string) {
	type strategy func() (*float64, string)

	strategies := []strategy{
		// 策略 1: New API / One API 用户信息（部分实现接受 Bearer API Key）
		func() (*float64, string) {
			return s.probeNewAPIUserSelf(ctx, baseURL, apiKey)
		},
		// 策略 2: OpenAI 官方 billing 端点（部分中转平台实现了此接口）
		func() (*float64, string) {
			return s.probeOpenAIBilling(ctx, baseURL, apiKey)
		},
		// 策略 3: New API token 搜索端点（通用 API Key 有时被当作 token key 处理）
		func() (*float64, string) {
			return s.probeNewAPITokenSearch(ctx, baseURL, apiKey)
		},
	}

	for _, fn := range strategies {
		balance, raw := fn()
		if balance != nil || raw != "" {
			return balance, raw
		}
	}
	return nil, ""
}

// probeNewAPIUserSelf 尝试 New API / One API 的 /api/user/self 端点（同时试两种认证头）
func (s *VendorProbeService) probeNewAPIUserSelf(ctx context.Context, baseURL, apiKey string) (*float64, string) {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, baseURL+"/api/user/self", nil)
	if err != nil {
		return nil, ""
	}
	// New API / One API 通常使用 Bearer token 认证
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, ""
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Quota     int64   `json:"quota"`
			UsedQuota int64   `json:"used_quota"`
			Balance   float64 `json:"balance"` // 有些实现用美元余额
		} `json:"data"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, ""
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Success {
		return nil, ""
	}

	// quota 单位通常是 token 数，换算为 USD 不可靠，直接返回原始值
	if result.Data.Balance > 0 {
		b := result.Data.Balance
		return &b, ""
	}
	if result.Data.Quota > 0 {
		remaining := result.Data.Quota - result.Data.UsedQuota
		raw := fmt.Sprintf("%d tokens remaining", remaining)
		return nil, raw
	}
	return nil, ""
}

// probeOpenAIBilling 尝试 OpenAI billing 端点
func (s *VendorProbeService) probeOpenAIBilling(ctx context.Context, baseURL, apiKey string) (*float64, string) {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, baseURL+"/v1/dashboard/billing/subscription", nil)
	if err != nil {
		return nil, ""
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, ""
	}

	var result struct {
		HardLimitUSD    float64 `json:"hard_limit_usd"`
		SoftLimitUSD    float64 `json:"soft_limit_usd"`
		SystemHardLimit float64 `json:"system_hard_limit_usd"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, ""
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, ""
	}
	if result.HardLimitUSD > 0 {
		b := result.HardLimitUSD
		return &b, ""
	}
	return nil, ""
}

// probeNewAPITokenSearch 尝试 New API 的 token 搜索端点
func (s *VendorProbeService) probeNewAPITokenSearch(ctx context.Context, baseURL, apiKey string) (*float64, string) {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 用 API Key 本身搜索 token 信息（某些部署允许通过 key 值查询）
	// SECURITY: key 通过 query param 传递是该 upstream API 端点的强制要求，
	// 无法使用 header-based 替代。确保不在 debug 日志中打印完整 URL。
	url := fmt.Sprintf("%s/api/token/search?key=%s", baseURL, apiKey)
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ""
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, ""
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			RemainQuota    int64 `json:"remain_quota"`
			UnlimitedQuota bool  `json:"unlimited_quota"`
		} `json:"data"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, ""
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Success {
		return nil, ""
	}

	if result.Data.UnlimitedQuota {
		return nil, "unlimited"
	}
	if result.Data.RemainQuota > 0 {
		raw := fmt.Sprintf("%d tokens remaining", result.Data.RemainQuota)
		return nil, raw
	}
	return nil, ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
