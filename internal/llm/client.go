package llm

import (
	"context"
	"qng_agent/internal/config"
)

type Client interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewClient(config config.LLMConfig) (Client, error) {
	switch config.Provider {
	case "openai":
		return NewOpenAIClient(config.Configs)
	case "anthropic":
		return NewAnthropicClient(config.Configs)
	case "ollama":
		return NewOllamaClient(config.Configs)
	case "gemini":
		return NewGeminiClient(config.Configs)
	default:
		return NewOpenAIClient(config.Configs)
	}
}
