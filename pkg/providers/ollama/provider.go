// Package ollama provides a PicoClaw-compatible LLM provider that targets
// an Ollama instance running on the EdgeClaw brain node. The brain is
// accessed transparently over the Tailscale VPN (e.g., http://qwen-brain.tailscale:11434/v1).
//
// When the brain node is unreachable (network partition, maintenance), the
// provider automatically falls back to a lighter local model running on the
// edge device itself, ensuring the agent remains operational in degraded mode.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Config configures the brain node provider and optional edge-local fallback.
type Config struct {
	// Name is a human-readable label (e.g., "qwen_brain").
	Name string `json:"name"`

	// APIBase is the Ollama-compatible API endpoint on the brain node.
	APIBase string `json:"api_base"`

	// Model is the primary model for inference (e.g., "qwen3-30b-a3b-instruct").
	Model string `json:"model"`

	// TimeoutSeconds for brain node requests.
	TimeoutSeconds int `json:"timeout_seconds"`

	// FallbackAPIBase is the local Ollama endpoint for edge-local inference.
	FallbackAPIBase string `json:"fallback_api_base,omitempty"`

	// FallbackModel is the lighter model for fallback (e.g., "qwen3-0.6b").
	FallbackModel string `json:"fallback_model,omitempty"`
}

// Provider implements an Ollama-compatible LLM provider with brain/edge fallback.
type Provider struct {
	cfg    Config
	client *http.Client
}

// New creates a brain node provider.
func New(cfg Config) *Provider {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &Provider{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the OpenAI-compatible chat completions request body.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

// ChatResponse is the OpenAI-compatible chat completions response.
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Complete sends a chat completion request to the brain node, falling back
// to the local edge model if the brain is unreachable.
func (p *Provider) Complete(ctx context.Context, messages []Message, temperature float64, maxTokens int) (*ChatResponse, error) {
	resp, err := p.doComplete(ctx, p.cfg.APIBase, p.cfg.Model, messages, temperature, maxTokens)
	if err != nil && p.cfg.FallbackAPIBase != "" && p.cfg.FallbackModel != "" {
		log.Printf("[ollama] brain unreachable (%v), falling back to local %s", err, p.cfg.FallbackModel)
		return p.doComplete(ctx, p.cfg.FallbackAPIBase, p.cfg.FallbackModel, messages, temperature, maxTokens)
	}
	return resp, err
}

func (p *Provider) doComplete(ctx context.Context, apiBase, model string, messages []Message, temperature float64, maxTokens int) (*ChatResponse, error) {
	body := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := apiBase + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request to %s: %w", endpoint, err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API returned %d: %s", httpResp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &chatResp, nil
}

// Healthy checks if the brain node is reachable.
func (p *Provider) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.APIBase+"/models", nil)
	if err != nil {
		return false
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
