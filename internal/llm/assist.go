package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dshills/verifier/internal/domain"
)

// AssistExtractRequirements uses LLM to extract requirements from spec text.
func AssistExtractRequirements(ctx context.Context, provider Provider, specText string, cfg *domain.Config) ([]domain.Requirement, string, error) {
	systemPrompt := `You are a requirements analyst. Extract testable requirements from the given specification.
Return a JSON object with a "requirements" array. Each requirement should have:
- "id": string (use existing IDs like REQ-001 if present, otherwise generate SYN-XXX-NNN)
- "text": string (the requirement text)
- "verifiability": string ("high", "medium", or "low")
- "acceptance_criteria": array of strings (if any)
- "heading_context": string (the section heading this was found under)

Return ONLY valid JSON, no markdown formatting.`

	req := CompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   specText,
		Temperature:  cfg.LLM.Temperature,
		MaxTokens:    cfg.LLM.MaxTokens,
		Seed:         cfg.Seed,
	}

	hash := PromptHash(systemPrompt, specText)

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		return nil, hash, fmt.Errorf("llm extract requirements: %w", err)
	}

	var result struct {
		Requirements []domain.Requirement `json:"requirements"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// Retry once on malformed JSON
		slog.Warn("malformed LLM response, retrying", "err", err)
		resp, err = provider.Complete(ctx, req)
		if err != nil {
			return nil, hash, fmt.Errorf("llm retry: %w", err)
		}
		if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
			return nil, hash, fmt.Errorf("llm parse retry: %w", err)
		}
	}

	return result.Requirements, hash, nil
}

// AssistMapping uses LLM to map requirements to symbols.
func AssistMapping(ctx context.Context, provider Provider, reqs []domain.Requirement, symbols []domain.Symbol, cfg *domain.Config) ([]domain.CoverageMapping, string, error) {
	systemPrompt := `You are a code analysis expert. Map requirements to code symbols.
Return a JSON object with a "mappings" array. Each mapping should have:
- "requirement_id": string
- "symbols": array of symbol name strings
- "confidence": float (0.0 to 1.0)

Return ONLY valid JSON, no markdown formatting.`

	reqJSON, _ := json.Marshal(reqs)
	symJSON, _ := json.Marshal(symbols)
	userPrompt := fmt.Sprintf("Requirements:\n%s\n\nSymbols:\n%s", reqJSON, symJSON)

	hash := PromptHash(systemPrompt, userPrompt)

	resp, err := provider.Complete(ctx, CompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  cfg.LLM.Temperature,
		MaxTokens:    cfg.LLM.MaxTokens,
		Seed:         cfg.Seed,
	})
	if err != nil {
		return nil, hash, err
	}

	var result struct {
		Mappings []domain.CoverageMapping `json:"mappings"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, hash, fmt.Errorf("parse mapping response: %w", err)
	}

	// Validate confidence range
	for i := range result.Mappings {
		if result.Mappings[i].Confidence < 0 || result.Mappings[i].Confidence > 1.0 {
			result.Mappings[i].Confidence = 0.5
		}
	}

	return result.Mappings, hash, nil
}

// AssistStrategy uses LLM to suggest test strategies.
func AssistStrategy(ctx context.Context, provider Provider, mappings []domain.CoverageMapping, risks []domain.RiskSignal, cfg *domain.Config) ([]domain.Recommendation, string, error) {
	systemPrompt := `You are a test strategy expert. For each requirement-to-symbol mapping, suggest test categories and approaches.
Return a JSON object with a "recommendations" array. Each recommendation should have:
- "category": string (unit, integration, contract, property, fuzz, concurrency, perf, security)
- "target": {"kind": "function"|"method", "name": string}
- "proposal": {"title": string, "approach": string, "assertions": [string]}
- "covers": {"requirements": [string], "risks": [string]}

Return ONLY valid JSON, no markdown formatting.`

	mappingsJSON, _ := json.Marshal(mappings)
	risksJSON, _ := json.Marshal(risks)
	userPrompt := fmt.Sprintf("Mappings:\n%s\n\nRisk Signals:\n%s", mappingsJSON, risksJSON)

	hash := PromptHash(systemPrompt, userPrompt)

	resp, err := provider.Complete(ctx, CompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  cfg.LLM.Temperature,
		MaxTokens:    cfg.LLM.MaxTokens,
		Seed:         cfg.Seed,
	})
	if err != nil {
		return nil, hash, err
	}

	var result struct {
		Recommendations []domain.Recommendation `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, hash, fmt.Errorf("parse strategy response: %w", err)
	}

	return result.Recommendations, hash, nil
}
