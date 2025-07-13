package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OpenAIClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
}

type OpenAIChoice struct {
	Message OpenAIMessage `json:"message"`
}

func NewOpenAIClient(configs map[string]string) (*OpenAIClient, error) {
	apiKey := configs["openai_api_key"]
	if apiKey == "" {
		return nil, fmt.Errorf("openai_api_key is required")
	}

	baseURL := configs["openai_base_url"]
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := configs["model"]
	if model == "" {
		model = "gpt-4"
	}

	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}, nil
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []Message) (string, error) {
	// 转换消息格式
	openaiMessages := make([]OpenAIMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	request := OpenAIRequest{
		Model:    c.model,
		Messages: openaiMessages,
		Stream:   false,
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error: %d %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}
