package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// doGet performs a GET request to the Agent and decodes the response data.
func (s *VpnAgentService) doGet(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.agentURL+path, nil)
	if err != nil {
		return err
	}
	return s.doRequest(req, out)
}

// doPost performs a POST request with a JSON body.
func (s *VpnAgentService) doPost(ctx context.Context, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.agentURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return s.doRequest(req, out)
}

// doPostEmpty performs a POST request with no body.
func (s *VpnAgentService) doPostEmpty(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.agentURL+path, nil)
	if err != nil {
		return err
	}
	return s.doRequest(req, nil)
}

// doDelete performs a DELETE request.
func (s *VpnAgentService) doDelete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.agentURL+path, nil)
	if err != nil {
		return err
	}
	return s.doRequest(req, nil)
}

// doRequest executes the request, checks the envelope, and decodes data.
func (s *VpnAgentService) doRequest(req *http.Request, out any) error {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vpn agent request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("vpn agent error (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp agentAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if apiResp.Code != 0 {
		return fmt.Errorf("agent: %s", apiResp.Message)
	}

	if out != nil && len(apiResp.Data) > 0 {
		if err := json.Unmarshal(apiResp.Data, out); err != nil {
			return fmt.Errorf("decode data: %w", err)
		}
	}
	return nil
}
