package main

import (
	"context"
	"log"
)

// HealthChecker periodically checks tunnel health.
// This is a stub implementation; full logic will be added in a later task.
type HealthChecker struct {
	tunnelMgr *TunnelManager
	store     *ConfigStore
	callback  *CallbackClient
	cfg       *AgentConfig
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(tm *TunnelManager, store *ConfigStore, callback *CallbackClient, cfg *AgentConfig) *HealthChecker {
	return &HealthChecker{
		tunnelMgr: tm,
		store:     store,
		callback:  callback,
		cfg:       cfg,
	}
}

// Start begins periodic health checking. Stub: blocks until context is done.
func (hc *HealthChecker) Start(ctx context.Context) {
	log.Println("HealthChecker.Start: stub, waiting for context cancellation")
	<-ctx.Done()
}
