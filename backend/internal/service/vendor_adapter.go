package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// VendorProtocolAdapter Anthropic ↔ OpenAI 协议转换适配器
type VendorProtocolAdapter struct{}

// NewVendorProtocolAdapter 创建协议转换适配器
func NewVendorProtocolAdapter() *VendorProtocolAdapter {
	return &VendorProtocolAdapter{}
}

// --- 请求转换：Anthropic → OpenAI ---

// ConvertAnthropicRequestToOpenAI 将 Anthropic 格式请求体转换为 OpenAI 格式
func (a *VendorProtocolAdapter) ConvertAnthropicRequestToOpenAI(anthropicBody []byte) ([]byte, error) {
	var req map[string]any
	if err := json.Unmarshal(anthropicBody, &req); err != nil {
		return nil, fmt.Errorf("unmarshal anthropic request: %w", err)
	}

	openaiReq := map[string]any{}

	// model
	if model, ok := req["model"].(string); ok {
		openaiReq["model"] = model
	}

	// max_tokens
	if maxTokens, ok := req["max_tokens"]; ok {
		openaiReq["max_tokens"] = maxTokens
	}

	// temperature
	if temp, ok := req["temperature"]; ok {
		openaiReq["temperature"] = temp
	}

	// top_p
	if topP, ok := req["top_p"]; ok {
		openaiReq["top_p"] = topP
	}

	// stream
	if stream, ok := req["stream"]; ok {
		openaiReq["stream"] = stream
	}

	// stop_sequences → stop
	if stop, ok := req["stop_sequences"]; ok {
		openaiReq["stop"] = stop
	}

	// messages 转换
	var openaiMessages []map[string]any

	// system prompt → system message
	if system, ok := req["system"]; ok {
		switch s := system.(type) {
		case string:
			if s != "" {
				openaiMessages = append(openaiMessages, map[string]any{
					"role":    "system",
					"content": s,
				})
			}
		case []any:
			// Anthropic system 可以是 content block 数组
			var parts []string
			for _, block := range s {
				if b, ok := block.(map[string]any); ok {
					if text, ok := b["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
			if len(parts) > 0 {
				openaiMessages = append(openaiMessages, map[string]any{
					"role":    "system",
					"content": strings.Join(parts, "\n"),
				})
			}
		}
	}

	// messages 转换
	if messages, ok := req["messages"].([]any); ok {
		for _, msg := range messages {
			m, ok := msg.(map[string]any)
			if !ok {
				continue
			}
			openaiMsg := map[string]any{
				"role": m["role"],
			}

			// content 转换
			switch content := m["content"].(type) {
			case string:
				openaiMsg["content"] = content
			case []any:
				// Anthropic content blocks → OpenAI content string
				var parts []string
				for _, block := range content {
					if b, ok := block.(map[string]any); ok {
						if text, ok := b["text"].(string); ok {
							parts = append(parts, text)
						}
					}
				}
				openaiMsg["content"] = strings.Join(parts, "")
			}

			openaiMessages = append(openaiMessages, openaiMsg)
		}
	}

	openaiReq["messages"] = openaiMessages

	return json.Marshal(openaiReq)
}

// --- 响应转换：OpenAI → Anthropic ---

// ConvertOpenAIResponseToAnthropic 将 OpenAI 非流式响应转换为 Anthropic 格式
func (a *VendorProtocolAdapter) ConvertOpenAIResponseToAnthropic(openaiBody []byte) ([]byte, error) {
	var resp map[string]any
	if err := json.Unmarshal(openaiBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal openai response: %w", err)
	}

	anthropicResp := map[string]any{
		"id":   resp["id"],
		"type": "message",
		"role": "assistant",
	}

	// model
	if model, ok := resp["model"].(string); ok {
		anthropicResp["model"] = model
	}

	// choices → content
	var contentBlocks []map[string]any
	stopReason := "end_turn"

	if choices, ok := resp["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if message, ok := choice["message"].(map[string]any); ok {
				if content, ok := message["content"].(string); ok {
					contentBlocks = append(contentBlocks, map[string]any{
						"type": "text",
						"text": content,
					})
				}
			}
			if reason, ok := choice["finish_reason"].(string); ok {
				stopReason = convertFinishReason(reason)
			}
		}
	}

	anthropicResp["content"] = contentBlocks
	anthropicResp["stop_reason"] = stopReason
	anthropicResp["stop_sequence"] = nil

	// usage 转换
	if usage, ok := resp["usage"].(map[string]any); ok {
		anthropicUsage := map[string]any{}
		if promptTokens, ok := usage["prompt_tokens"]; ok {
			anthropicUsage["input_tokens"] = promptTokens
		}
		if completionTokens, ok := usage["completion_tokens"]; ok {
			anthropicUsage["output_tokens"] = completionTokens
		}
		anthropicResp["usage"] = anthropicUsage
	}

	return json.Marshal(anthropicResp)
}

// ConvertOpenAISSEToAnthropic 将 OpenAI SSE data 转换为 Anthropic SSE 事件
// 返回转换后的 Anthropic SSE 行列表
func (a *VendorProtocolAdapter) ConvertOpenAISSEToAnthropic(openaiData []byte) ([][]byte, error) {
	dataStr := strings.TrimSpace(string(openaiData))
	if dataStr == "[DONE]" {
		// 生成 message_stop 事件
		stopEvent, _ := json.Marshal(map[string]any{
			"type": "message_stop",
		})
		return [][]byte{[]byte("event: message_stop\ndata: " + string(stopEvent))}, nil
	}

	var chunk map[string]any
	if err := json.Unmarshal(openaiData, &chunk); err != nil {
		return nil, nil // 跳过无法解析的行
	}

	var events [][]byte

	choices, ok := chunk["choices"].([]any)
	if !ok || len(choices) == 0 {
		return nil, nil
	}

	choice, ok := choices[0].(map[string]any)
	if !ok {
		return nil, nil
	}

	// delta → content_block_delta
	if delta, ok := choice["delta"].(map[string]any); ok {
		if content, ok := delta["content"].(string); ok && content != "" {
			deltaEvent, _ := json.Marshal(map[string]any{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]any{
					"type": "text_delta",
					"text": content,
				},
			})
			events = append(events, []byte("event: content_block_delta\ndata: "+string(deltaEvent)))
		}
	}

	// finish_reason → message_delta
	if reason, ok := choice["finish_reason"].(string); ok && reason != "" {
		// usage from chunk (if present)
		var outputTokens any
		if usage, ok := chunk["usage"].(map[string]any); ok {
			outputTokens = usage["completion_tokens"]
		}

		deltaEvent, _ := json.Marshal(map[string]any{
			"type": "message_delta",
			"delta": map[string]any{
				"stop_reason":   convertFinishReason(reason),
				"stop_sequence": nil,
			},
			"usage": map[string]any{
				"output_tokens": outputTokens,
			},
		})
		events = append(events, []byte("event: message_delta\ndata: "+string(deltaEvent)))
	}

	return events, nil
}

// GenerateMessageStartEvent 生成 Anthropic message_start SSE 事件
func (a *VendorProtocolAdapter) GenerateMessageStartEvent(model string) []byte {
	event, _ := json.Marshal(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			"type":          "message",
			"role":          "assistant",
			"content":       []any{},
			"model":         model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	})
	return []byte("event: message_start\ndata: " + string(event))
}

// GenerateContentBlockStartEvent 生成 content_block_start SSE 事件
func (a *VendorProtocolAdapter) GenerateContentBlockStartEvent() []byte {
	event, _ := json.Marshal(map[string]any{
		"type":          "content_block_start",
		"index":         0,
		"content_block": map[string]any{"type": "text", "text": ""},
	})
	return []byte("event: content_block_start\ndata: " + string(event))
}

// convertFinishReason OpenAI finish_reason → Anthropic stop_reason
func convertFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "content_filter":
		return "end_turn"
	case "tool_calls", "function_call":
		return "tool_use"
	default:
		return "end_turn"
	}
}
