package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// VendorHealthService 供应商健康检查服务
type VendorHealthService struct {
	vendorRepo VendorRepository
	httpClient *http.Client
}

// NewVendorHealthService 创建健康检查服务
func NewVendorHealthService(vendorRepo VendorRepository) *VendorHealthService {
	return &VendorHealthService{
		vendorRepo: vendorRepo,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// RunHealthCheck 对单个供应商执行健康检查
func (s *VendorHealthService) RunHealthCheck(ctx context.Context, vendorID int64) (*VendorHealthResult, error) {
	vendor, err := s.vendorRepo.GetByID(ctx, vendorID)
	if err != nil {
		return nil, err
	}
	return s.checkVendor(ctx, vendor)
}

// VendorHealthResult 健康检查结果
type VendorHealthResult struct {
	VendorID int64  `json:"vendor_id"`
	Status   string `json:"status"`
	Latency  int    `json:"latency_ms"`
	Error    string `json:"error,omitempty"`
}

// checkVendor 执行实际的健康检查
func (s *VendorHealthService) checkVendor(ctx context.Context, vendor *Vendor) (*VendorHealthResult, error) {
	result := &VendorHealthResult{VendorID: vendor.ID}

	// 构建测试请求
	apiPath := vendor.GetAPIPath()
	targetURL := vendor.BaseURL + apiPath

	var reqBody []byte
	if vendor.APIFormat == VendorAPIFormatOpenAI {
		reqBody, _ = json.Marshal(map[string]any{
			"model":      vendor.HealthCheckModel,
			"messages":   []map[string]string{{"role": "user", "content": "hi"}},
			"max_tokens": 1,
		})
	} else {
		reqBody, _ = json.Marshal(map[string]any{
			"model":      vendor.HealthCheckModel,
			"messages":   []map[string]any{{"role": "user", "content": []map[string]string{{"type": "text", "text": "hi"}}}},
			"max_tokens": 1,
		})
	}

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(reqBody))
	if err != nil {
		result.Status = VendorHealthError
		result.Error = fmt.Sprintf("build request: %v", err)
		return result, nil
	}
	req.Header.Set("Content-Type", "application/json")

	// Anthropic 格式必须携带 anthropic-version 头
	if vendor.APIFormat == VendorAPIFormatAnthropic {
		if _, hasVersion := vendor.ExtraHeaders["anthropic-version"]; !hasVersion {
			req.Header.Set("anthropic-version", "2023-06-01")
		}
	}

	// 应用供应商配置的认证头（管理员在 extra_headers 中设置，如 x-api-key 或 Authorization）
	for k, v := range vendor.ExtraHeaders {
		req.Header.Set(k, v)
	}

	// 对于 reseller 类型的供应商，如果有 reseller_api_key 但 extra_headers 中没有认证信息，自动添加
	if vendor.VendorType == "reseller" && vendor.ResellerAPIKey != nil && *vendor.ResellerAPIKey != "" {
		// 检查是否已经有认证头
		hasAuth := false
		for k := range vendor.ExtraHeaders {
			lowerKey := strings.ToLower(k)
			if lowerKey == "authorization" || lowerKey == "x-api-key" {
				hasAuth = true
				break
			}
		}
		// 如果没有认证头，根据 API 格式自动添加
		if !hasAuth {
			if vendor.APIFormat == VendorAPIFormatAnthropic {
				req.Header.Set("x-api-key", *vendor.ResellerAPIKey)
			} else {
				req.Header.Set("Authorization", "Bearer "+*vendor.ResellerAPIKey)
			}
		}
	}

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	latency := int(time.Since(start).Milliseconds())
	result.Latency = latency

	if err != nil {
		result.Status = VendorHealthTimeout
		result.Error = fmt.Sprintf("request failed: %v", err)
	} else {
		defer resp.Body.Close() //nolint:errcheck
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if latency > 10000 {
				result.Status = VendorHealthSlow
			} else {
				result.Status = VendorHealthOK
			}
		} else {
			result.Status = VendorHealthError
			errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
			// 尝试从响应体提取错误详情
			if len(body) > 0 {
				var errResp struct {
					Error struct {
						Message string `json:"message"`
						Type    string `json:"type"`
					} `json:"error"`
					Message string `json:"message"`
				}
				if json.Unmarshal(body, &errResp) == nil {
					if errResp.Error.Message != "" {
						errMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, errResp.Error.Message)
					} else if errResp.Message != "" {
						errMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, errResp.Message)
					}
				}
			}
			result.Error = errMsg
		}
	}

	// 更新供应商健康状态
	consecutiveFailures := 0
	var errMsg *string
	if result.Status == VendorHealthError || result.Status == VendorHealthTimeout {
		consecutiveFailures = vendor.ConsecutiveFailures + 1
		errMsg = &result.Error
	}

	if updateErr := s.vendorRepo.UpdateHealthStatus(ctx, vendor.ID, result.Status, &result.Latency, errMsg, consecutiveFailures); updateErr != nil {
		slog.Error("failed to update vendor health status", "vendor_id", vendor.ID, "error", updateErr)
	}

	return result, nil
}

// RunAllDueHealthChecks 并行执行所有到期的健康检查（最多 5 个并发）
func (s *VendorHealthService) RunAllDueHealthChecks(ctx context.Context) ([]VendorHealthResult, error) {
	vendors, err := s.vendorRepo.ListHealthCheckDue(ctx)
	if err != nil {
		return nil, fmt.Errorf("list health check due: %w", err)
	}

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	var mu sync.Mutex
	var wg sync.WaitGroup
	var results []VendorHealthResult

	for i := range vendors {
		wg.Add(1)
		go func(v *Vendor) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := s.checkVendor(ctx, v)
			if err != nil {
				slog.Error("health check failed", "vendor_id", v.ID, "error", err)
				return
			}
			mu.Lock()
			results = append(results, *result)
			mu.Unlock()
		}(&vendors[i])
	}

	wg.Wait()
	return results, nil
}

// AutoSuspendUnhealthy 自动暂停连续失败超过阈值的供应商
func (s *VendorHealthService) AutoSuspendUnhealthy(ctx context.Context, maxFailures int) (int, error) {
	if maxFailures <= 0 {
		maxFailures = 5
	}

	vendors, err := s.vendorRepo.ListActive(ctx)
	if err != nil {
		return 0, err
	}

	suspended := 0
	for _, v := range vendors {
		if v.HealthCheckEnabled && v.ConsecutiveFailures >= maxFailures {
			if err := s.vendorRepo.UpdateStatus(ctx, v.ID, VendorStatusError); err != nil {
				slog.Error("failed to suspend unhealthy vendor", "vendor_id", v.ID, "error", err)
				continue
			}
			suspended++
			slog.Warn("auto-suspended unhealthy vendor", "vendor_id", v.ID, "name", v.Name, "consecutive_failures", v.ConsecutiveFailures)
		}
	}
	return suspended, nil
}
