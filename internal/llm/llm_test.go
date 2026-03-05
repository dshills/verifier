package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dshills/verifier/internal/domain"
)

type mockProvider struct {
	response string
	err      error
	calls    int
}

func (m *mockProvider) Name() string { return "mock" }

func (m *mockProvider) Complete(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
	m.calls++
	if m.err != nil {
		return CompletionResponse{}, m.err
	}
	return CompletionResponse{Content: m.response, Model: "mock-model"}, nil
}

func TestPromptHash(t *testing.T) {
	h1 := PromptHash("system", "user")
	h2 := PromptHash("system", "user")
	if h1 != h2 {
		t.Error("same inputs should produce same hash")
	}

	h3 := PromptHash("system", "different")
	if h1 == h3 {
		t.Error("different inputs should produce different hash")
	}
}

func TestNewProvider(t *testing.T) {
	p, err := NewProvider("openai", "gpt-4", "key")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "openai" {
		t.Errorf("name = %q, want openai", p.Name())
	}

	p, err = NewProvider("anthropic", "claude", "key")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "anthropic" {
		t.Errorf("name = %q, want anthropic", p.Name())
	}

	_, err = NewProvider("unknown", "model", "key")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestAssistExtractRequirements(t *testing.T) {
	response := `{"requirements": [{"id": "REQ-001", "text": "test req", "verifiability": "high"}]}`
	mock := &mockProvider{response: response}
	cfg := &domain.Config{LLM: domain.LLMConfig{Temperature: 0.2, MaxTokens: 4000}}

	reqs, hash, err := AssistExtractRequirements(context.Background(), mock, "spec text", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("reqs = %d, want 1", len(reqs))
	}
	if reqs[0].ID != "REQ-001" {
		t.Errorf("id = %q, want REQ-001", reqs[0].ID)
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
}

func TestAssistExtractRequirementsRetry(t *testing.T) {
	mock := &mockProvider{
		response: `{"requirements": [{"id": "REQ-001", "text": "test"}]}`,
	}
	cfg := &domain.Config{LLM: domain.LLMConfig{Temperature: 0.2, MaxTokens: 4000}}

	reqs, _, err := AssistExtractRequirements(context.Background(), mock, "spec", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 req, got %d", len(reqs))
	}
}

func TestAssistExtractRequirementsProviderError(t *testing.T) {
	mock := &mockProvider{err: fmt.Errorf("connection failed")}
	cfg := &domain.Config{LLM: domain.LLMConfig{Temperature: 0.2, MaxTokens: 4000}}

	_, _, err := AssistExtractRequirements(context.Background(), mock, "spec", cfg)
	if err == nil {
		t.Error("expected error")
	}
}

func TestAssistMapping(t *testing.T) {
	mappings := []domain.CoverageMapping{{RequirementID: "REQ-001", Symbols: []string{"Foo"}, Confidence: 0.8}}
	respJSON, _ := json.Marshal(map[string]any{"mappings": mappings})
	mock := &mockProvider{response: string(respJSON)}
	cfg := &domain.Config{LLM: domain.LLMConfig{Temperature: 0.2, MaxTokens: 4000}}

	result, _, err := AssistMapping(context.Background(), mock, nil, nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("mappings = %d, want 1", len(result))
	}
}

func TestAssistMappingConfidenceValidation(t *testing.T) {
	mappings := []domain.CoverageMapping{{RequirementID: "REQ-001", Symbols: []string{"Foo"}, Confidence: 5.0}}
	respJSON, _ := json.Marshal(map[string]any{"mappings": mappings})
	mock := &mockProvider{response: string(respJSON)}
	cfg := &domain.Config{LLM: domain.LLMConfig{Temperature: 0.2, MaxTokens: 4000}}

	result, _, err := AssistMapping(context.Background(), mock, nil, nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Confidence != 0.5 {
		t.Errorf("confidence = %f, want 0.5 (clamped)", result[0].Confidence)
	}
}

func TestAssistFallbackOnFailure(t *testing.T) {
	mock := &mockProvider{err: fmt.Errorf("timeout")}
	cfg := &domain.Config{LLM: domain.LLMConfig{Temperature: 0.2, MaxTokens: 4000}}

	_, _, err := AssistMapping(context.Background(), mock, nil, nil, cfg)
	if err == nil {
		t.Error("expected error for provider failure")
	}
}
