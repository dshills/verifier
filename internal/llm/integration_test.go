//go:build integration

package llm

import (
	"context"
	"os"
	"testing"
	"time"
)

// Integration tests that call real LLM APIs.
// Run with: go test ./internal/llm/... -run TestIntegration -v

func TestIntegrationOpenAI(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	p := NewOpenAIProvider("gpt-4o-mini", key)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := p.Complete(ctx, CompletionRequest{
		SystemPrompt: "You are a helpful assistant. Reply in one short sentence.",
		UserPrompt:   "What is 2+2?",
		Temperature:  0.0,
		MaxTokens:    100,
	})
	if err != nil {
		t.Fatalf("OpenAI Complete failed: %v", err)
	}

	t.Logf("OpenAI response (model=%s): %s", resp.Model, resp.Content)
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
	if resp.Model == "" {
		t.Error("expected non-empty model")
	}
}

func TestIntegrationAnthropic(t *testing.T) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	p := NewAnthropicProvider("claude-haiku-4-5-20251001", key)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := p.Complete(ctx, CompletionRequest{
		SystemPrompt: "You are a helpful assistant. Reply in one short sentence.",
		UserPrompt:   "What is 2+2?",
		Temperature:  0.0,
		MaxTokens:    100,
	})
	if err != nil {
		t.Fatalf("Anthropic Complete failed: %v", err)
	}

	t.Logf("Anthropic response (model=%s): %s", resp.Model, resp.Content)
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
	if resp.Model == "" {
		t.Error("expected non-empty model")
	}
}

func TestIntegrationGemini(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	p := NewGeminiProvider("gemini-2.0-flash", key)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := p.Complete(ctx, CompletionRequest{
		SystemPrompt: "You are a helpful assistant. Reply in one short sentence.",
		UserPrompt:   "What is 2+2?",
		Temperature:  0.0,
		MaxTokens:    100,
	})
	if err != nil {
		t.Fatalf("Gemini Complete failed: %v", err)
	}

	t.Logf("Gemini response (model=%s): %s", resp.Model, resp.Content)
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}
