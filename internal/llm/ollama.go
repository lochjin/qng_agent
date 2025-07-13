package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OllamaClient struct {
	baseURL string
	model   string
	client  *http.Client
}

type OllamaRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *OllamaOptions  `json:"options,omitempty"`
}

type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
}

type OllamaResponse struct {
	Message OllamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

func NewOllamaClient(configs map[string]string) (*OllamaClient, error) {
	baseURL := configs["ollama_base_url"]
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := configs["model"]
	if model == "" {
		model = "llama2"
	}

	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}, nil
}

func (c *OllamaClient) Chat(ctx context.Context, messages []Message) (string, error) {
	// 转换消息格式
	ollamaMessages := make([]OllamaMessage, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = OllamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// 构建请求
	request := OllamaRequest{
		Model:    c.model,
		Messages: ollamaMessages,
		Stream:   false,
		Options: &OllamaOptions{
			Temperature: 0.7,
			TopP:        0.9,
		},
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/api/chat", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 解析响应
	var response OllamaResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Message.Content, nil
}
