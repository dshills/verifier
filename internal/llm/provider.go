package llm

import (
	"context"
	"crypto/sha256"
	"fmt"
)

// CompletionRequest holds the input for an LLM completion.
type CompletionRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
	Seed         *int
}

// CompletionResponse holds the LLM output.
type CompletionResponse struct {
	Content string
	Model   string
}

// Provider is the interface for LLM providers.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	Name() string
}

// PromptHash computes SHA-256 hash of the full prompt for traceability.
func PromptHash(system, user string) string {
	input := system + "\x00" + user
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h)
}

// NewProvider creates a provider based on config.
func NewProvider(providerName, model, apiKey string) (Provider, error) {
	switch providerName {
	case "openai":
		return NewOpenAIProvider(model, apiKey), nil
	case "anthropic":
		return NewAnthropicProvider(model, apiKey), nil
	case "gemini":
		return NewGeminiProvider(model, apiKey), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider %q", providerName)
	}
}
