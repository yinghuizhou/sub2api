package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// VendorBackgroundService 供应商后台定时任务服务
type VendorBackgroundService struct {
	healthService  *VendorHealthService
	balanceService *VendorBalanceService
	settingService *SettingService

	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewVendorBackgroundService 创建后台服务
func NewVendorBackgroundService(
	healthService *VendorHealthService,
	balanceService *VendorBalanceService,
	settingService *SettingService,
) *VendorBackgroundService {
	return &VendorBackgroundService{
		healthService:  healthService,
		balanceService: balanceService,
		settingService: settingService,
		stopCh:         make(chan struct{}),
	}
}

// Start 启动后台任务
func (s *VendorBackgroundService) Start() {
	go s.runHealthCheckLoop()
	go s.runBalanceAlertLoop()
	go s.runAutoSuspendLoop()
	slog.Info("[VendorBackground] started")
}

// Stop 停止后台任务
func (s *VendorBackgroundService) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		slog.Info("[VendorBackground] stopped")
	})
}

func (s *VendorBackgroundService) isEnabled() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	val, err := s.settingService.GetValue(ctx, "vendor_system_enabled")
	if err != nil {
		return false
	}
	return val == "true"
}

func (s *VendorBackgroundService) runHealthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if !s.isEnabled() {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			results, err := s.healthService.RunAllDueHealthChecks(ctx)
			cancel()
			if err != nil {
				slog.Error("[VendorBackground] health check failed", "error", err)
			} else if len(results) > 0 {
				slog.Info("[VendorBackground] health checks completed", "count", len(results))
			}
		}
	}
}

func (s *VendorBackgroundService) runBalanceAlertLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if !s.isEnabled() {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			alerts, err := s.balanceService.CheckBalanceAlerts(ctx)
			cancel()
			if err != nil {
				slog.Error("[VendorBackground] balance alert check failed", "error", err)
			} else if len(alerts) > 0 {
				slog.Warn("[VendorBackground] balance alerts triggered", "count", len(alerts))
			}
		}
	}
}

func (s *VendorBackgroundService) runAutoSuspendLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if !s.isEnabled() {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			suspended, err := s.healthService.AutoSuspendUnhealthy(ctx, 5)
			if err != nil {
				slog.Error("[VendorBackground] auto-suspend unhealthy failed", "error", err)
			} else if suspended > 0 {
				slog.Warn("[VendorBackground] auto-suspended unhealthy vendors", "count", suspended)
			}
			depleted, err := s.balanceService.AutoSuspendDepleted(ctx)
			if err != nil {
				slog.Error("[VendorBackground] auto-suspend depleted failed", "error", err)
			} else if depleted > 0 {
				slog.Warn("[VendorBackground] auto-suspended depleted vendors", "count", depleted)
			}
			cancel()
		}
	}
}
