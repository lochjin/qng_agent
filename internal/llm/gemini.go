package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type GeminiClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

func NewGeminiClient(configs map[string]string) (*GeminiClient, error) {
	apiKey := configs["gemini_api_key"]
	if apiKey == "" {
		return nil, fmt.Errorf("gemini_api_key is required")
	}

	baseURL := configs["gemini_base_url"]
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	model := configs["model"]
	if model == "" {
		model = "gemini-pro"
	}

	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}, nil
}

func (c *GeminiClient) Chat(ctx context.Context, messages []Message) (string, error) {
	// 转换消息格式
	contents := make([]GeminiContent, 0, len(messages))
	for _, msg := range messages {
		// Gemini使用不同的角色映射
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		contents = append(contents, GeminiContent{
			Role: role,
			Parts: []GeminiPart{
				{Text: msg.Content},
			},
		})
	}

	// 构建请求
	request := GeminiRequest{
		Contents: contents,
		GenerationConfig: &GeminiGenerationConfig{
			Temperature:     0.7,
			TopP:            0.9,
			TopK:            40,
			MaxOutputTokens: 2048,
		},
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
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
		return "", fmt.Errorf("Gemini API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 解析响应
	var response GeminiResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 提取响应内容
	if len(response.Candidates) > 0 && len(response.Candidates[0].Content.Parts) > 0 {
		return response.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("no response content from Gemini")
}
