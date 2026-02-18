package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew_DefaultTimeout(t *testing.T) {
	p := New(Config{
		Name:           "test",
		APIBase:        "http://localhost:11434/v1",
		Model:          "test-model",
		TimeoutSeconds: 0,
	})

	if p.client.Timeout != 120*time.Second {
		t.Errorf("Timeout = %v, want %v", p.client.Timeout, 120*time.Second)
	}
}

func TestNew_ExplicitTimeout(t *testing.T) {
	p := New(Config{
		Name:           "test",
		APIBase:        "http://localhost:11434/v1",
		Model:          "test-model",
		TimeoutSeconds: 30,
	})

	if p.client.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", p.client.Timeout, 30*time.Second)
	}
}

func TestNew_StoresConfig(t *testing.T) {
	cfg := Config{
		Name:            "brain",
		APIBase:         "http://brain:11434/v1",
		Model:           "qwen3-30b",
		FallbackAPIBase: "http://localhost:11434/v1",
		FallbackModel:   "qwen3-0.6b",
	}
	p := New(cfg)

	if p.cfg.Name != "brain" {
		t.Errorf("cfg.Name = %q, want %q", p.cfg.Name, "brain")
	}
	if p.cfg.FallbackModel != "qwen3-0.6b" {
		t.Errorf("cfg.FallbackModel = %q, want %q", p.cfg.FallbackModel, "qwen3-0.6b")
	}
}

func TestChatRequest_JSONMarshal(t *testing.T) {
	req := ChatRequest{
		Model: "qwen3-30b",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   1024,
		Stream:      false,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Verify required fields are present
	for _, key := range []string{"model", "messages", "stream"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing JSON key %q", key)
		}
	}

	// Verify model value
	var model string
	json.Unmarshal(raw["model"], &model)
	if model != "qwen3-30b" {
		t.Errorf("model = %q, want %q", model, "qwen3-30b")
	}

	// Verify messages array length
	var msgs []Message
	json.Unmarshal(raw["messages"], &msgs)
	if len(msgs) != 2 {
		t.Errorf("messages length = %d, want %d", len(msgs), 2)
	}
}

func TestChatRequest_OmitsZeroOptional(t *testing.T) {
	req := ChatRequest{
		Model:    "test",
		Messages: []Message{{Role: "user", Content: "hi"}},
		Stream:   false,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	json.Unmarshal(data, &raw)

	if _, ok := raw["temperature"]; ok {
		t.Error("expected 'temperature' to be omitted when zero")
	}
	if _, ok := raw["max_tokens"]; ok {
		t.Error("expected 'max_tokens' to be omitted when zero")
	}
}

func TestChatResponse_JSONUnmarshal(t *testing.T) {
	respJSON := `{
		"choices": [
			{
				"message": {
					"content": "Hello! How can I help you?"
				}
			}
		],
		"model": "qwen3-30b",
		"usage": {
			"prompt_tokens": 10,
			"total_tokens": 25
		}
	}`

	var resp ChatResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("Choices length = %d, want 1", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello! How can I help you?" {
		t.Errorf("Content = %q, want %q", resp.Choices[0].Message.Content, "Hello! How can I help you?")
	}
	if resp.Model != "qwen3-30b" {
		t.Errorf("Model = %q, want %q", resp.Model, "qwen3-30b")
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want %d", resp.Usage.PromptTokens, 10)
	}
	if resp.Usage.TotalTokens != 25 {
		t.Errorf("TotalTokens = %d, want %d", resp.Usage.TotalTokens, 25)
	}
}

func TestChatResponse_EmptyChoices(t *testing.T) {
	respJSON := `{"choices": [], "model": "test"}`
	var resp ChatResponse
	if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(resp.Choices) != 0 {
		t.Errorf("Choices length = %d, want 0", len(resp.Choices))
	}
}

func TestHealthy_ReturnsTrue_WhenServerOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"models": []}`)
	}))
	defer srv.Close()

	p := New(Config{
		Name:    "test",
		APIBase: srv.URL,
		Model:   "test-model",
	})

	if !p.Healthy(context.Background()) {
		t.Error("Healthy() = false, want true when server returns 200")
	}
}

func TestHealthy_ReturnsFalse_WhenServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := New(Config{
		Name:    "test",
		APIBase: srv.URL,
		Model:   "test-model",
	})

	if p.Healthy(context.Background()) {
		t.Error("Healthy() = true, want false when server returns 500")
	}
}

func TestHealthy_ReturnsFalse_WhenUnreachable(t *testing.T) {
	p := New(Config{
		Name:           "test",
		APIBase:        "http://127.0.0.1:1", // port 1 is unlikely to be listening
		Model:          "test-model",
		TimeoutSeconds: 1,
	})

	if p.Healthy(context.Background()) {
		t.Error("Healthy() = true, want false when server is unreachable")
	}
}

func TestComplete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %q", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"choices": [{"message": {"content": "test response"}}],
			"model": "test-model",
			"usage": {"prompt_tokens": 5, "total_tokens": 15}
		}`)
	}))
	defer srv.Close()

	p := New(Config{
		Name:    "test",
		APIBase: srv.URL,
		Model:   "test-model",
	})

	resp, err := p.Complete(context.Background(), []Message{
		{Role: "user", Content: "hello"},
	}, 0.7, 100)

	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("Choices length = %d, want 1", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "test response" {
		t.Errorf("Content = %q, want %q", resp.Choices[0].Message.Content, "test response")
	}
}

func TestComplete_ReturnsError_WhenNoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices": [], "model": "test-model"}`)
	}))
	defer srv.Close()

	p := New(Config{
		Name:    "test",
		APIBase: srv.URL,
		Model:   "test-model",
	})

	_, err := p.Complete(context.Background(), []Message{
		{Role: "user", Content: "hello"},
	}, 0.7, 100)

	if err == nil {
		t.Error("expected error for empty choices, got nil")
	}
}

func TestComplete_FallsBackOnPrimaryFailure(t *testing.T) {
	// Primary server returns an error
	primaryCalled := false
	primarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryCalled = true
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"error": "service unavailable"}`)
	}))
	defer primarySrv.Close()

	// Fallback server returns a valid response
	fallbackCalled := false
	fallbackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"choices": [{"message": {"content": "fallback response"}}],
			"model": "small-model",
			"usage": {"prompt_tokens": 5, "total_tokens": 10}
		}`)
	}))
	defer fallbackSrv.Close()

	p := New(Config{
		Name:            "test",
		APIBase:         primarySrv.URL,
		Model:           "big-model",
		FallbackAPIBase: fallbackSrv.URL,
		FallbackModel:   "small-model",
	})

	resp, err := p.Complete(context.Background(), []Message{
		{Role: "user", Content: "hello"},
	}, 0.7, 100)

	if err != nil {
		t.Fatalf("Complete with fallback: %v", err)
	}
	if !primaryCalled {
		t.Error("expected primary server to be called")
	}
	if !fallbackCalled {
		t.Error("expected fallback server to be called")
	}
	if resp.Choices[0].Message.Content != "fallback response" {
		t.Errorf("Content = %q, want %q", resp.Choices[0].Message.Content, "fallback response")
	}
}

func TestComplete_NoFallback_WhenFallbackNotConfigured(t *testing.T) {
	primarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"error": "service unavailable"}`)
	}))
	defer primarySrv.Close()

	p := New(Config{
		Name:    "test",
		APIBase: primarySrv.URL,
		Model:   "big-model",
		// No fallback configured
	})

	_, err := p.Complete(context.Background(), []Message{
		{Role: "user", Content: "hello"},
	}, 0.7, 100)

	if err == nil {
		t.Error("expected error when primary fails and no fallback configured, got nil")
	}
}

func TestComplete_FallbackAlsoFails(t *testing.T) {
	primarySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer primarySrv.Close()

	fallbackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer fallbackSrv.Close()

	p := New(Config{
		Name:            "test",
		APIBase:         primarySrv.URL,
		Model:           "big-model",
		FallbackAPIBase: fallbackSrv.URL,
		FallbackModel:   "small-model",
	})

	_, err := p.Complete(context.Background(), []Message{
		{Role: "user", Content: "hello"},
	}, 0.7, 100)

	if err == nil {
		t.Error("expected error when both primary and fallback fail, got nil")
	}
}
