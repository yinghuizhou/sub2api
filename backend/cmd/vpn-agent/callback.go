package main

import "log"

// CallbackClient sends status updates back to the Sub2API server.
// This is a stub implementation; full logic will be added in a later task.
type CallbackClient struct {
	baseURL string
	apiKey  string
}

// NewCallbackClient creates a new CallbackClient.
func NewCallbackClient(baseURL, apiKey string) *CallbackClient {
	return &CallbackClient{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// ReportStatus sends a tunnel status update to Sub2API. Stub.
func (c *CallbackClient) ReportStatus(tunnels []TunnelInfo) error {
	log.Printf("CallbackClient.ReportStatus: stub, %d tunnels", len(tunnels))
	return nil
}
