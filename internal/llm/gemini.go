package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

// GeminiProvider implements the Provider interface for Google Gemini.
type GeminiProvider struct {
	model  string
	apiKey string
	client *http.Client
}

// NewGeminiProvider creates a new Gemini provider.
func NewGeminiProvider(model, apiKey string) *GeminiProvider {
	return &GeminiProvider{
		model:  model,
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, p.model, p.apiKey)

	body := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": req.UserPrompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature": req.Temperature,
		},
	}

	if req.SystemPrompt != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{"text": req.SystemPrompt},
			},
		}
	}

	if req.MaxTokens > 0 {
		body["generationConfig"].(map[string]any)["maxOutputTokens"] = req.MaxTokens
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("gemini request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, respBody)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		ModelVersion string `json:"modelVersion"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return CompletionResponse{}, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return CompletionResponse{}, fmt.Errorf("gemini returned no content")
	}

	return CompletionResponse{
		Content: result.Candidates[0].Content.Parts[0].Text,
		Model:   result.ModelVersion,
	}, nil
}
