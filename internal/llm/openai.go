package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"qng_agent/internal/config"
	"strings"
	"time"
)

type OpenAIClient struct {
	config config.OpenAIConfig
	client *http.Client
}

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewOpenAIClient(config config.OpenAIConfig) (Client, error) {
	if config.APIKey == "" {
		return NewMockClient(), nil
	}

	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	return &OpenAIClient{
		config: config,
		client: client,
	}, nil
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []Message) (string, error) {
	log.Printf("🔍 OpenAI客户端诊断信息:")
	log.Printf("  - API密钥长度: %d", len(c.config.APIKey))
	log.Printf("  - BaseURL: %s", c.config.BaseURL)
	log.Printf("  - Model: %s", c.config.Model)
	log.Printf("  - Timeout: %d", c.config.Timeout)
	
	if c.config.APIKey == "" || c.config.BaseURL == "" {
		log.Printf("⚠️  使用模拟客户端 (API密钥或BaseURL为空)")
		mockClient := NewMockClient()
		return mockClient.Chat(ctx, messages)
	}

	requestBody := OpenAIRequest{
		Model:     c.config.Model,
		Messages:  messages,
		MaxTokens: c.config.MaxTokens,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.config.BaseURL + "/chat/completions"
	log.Printf("🌐 请求URL: %s", url)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return response.Choices[0].Message.Content, nil
}
