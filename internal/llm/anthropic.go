package llm

import (
	"context"
	"fmt"
)

type AnthropicClient struct {
	apiKey string
	model  string
}

func NewAnthropicClient(configs map[string]string) (*AnthropicClient, error) {
	apiKey := configs["anthropic_api_key"]
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic_api_key is required")
	}

	model := configs["model"]
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}

	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
	}, nil
}

func (c *AnthropicClient) Chat(ctx context.Context, messages []Message) (string, error) {
	// 简化实现，实际使用时需要调用Anthropic API
	return "Anthropic client not implemented yet", nil
}
