package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitConfigDefaults(t *testing.T) {
	initConfig()
	if config.ServerURL == "" {
		t.Error("expected non-empty ServerURL default")
	}
	if config.HealthURL == "" {
		t.Error("expected non-empty HealthURL default")
	}
	if httpClient == nil {
		t.Error("expected httpClient to be initialized")
	}
}

func TestGetToolsList(t *testing.T) {
	tools := getToolsList()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "ask_local_llm" {
		t.Errorf("expected first tool 'ask_local_llm', got %s", tools[0].Name)
	}
	if tools[1].Name != "summarize_local" {
		t.Errorf("expected second tool 'summarize_local', got %s", tools[1].Name)
	}
}

func TestGetPromptsList(t *testing.T) {
	prompts := getPromptsList()
	if len(prompts.Prompts) == 0 {
		t.Error("expected non-empty prompts list")
	}
}

func TestGetResourcesList(t *testing.T) {
	resources := getResourcesList()
	if len(resources.Resources) == 0 {
		t.Error("expected non-empty resources list")
	}
}

func TestCallToolValidation(t *testing.T) {
	initConfig()

	// Test missing prompt in ask_local_llm
	_, err := callTool(CallToolParams{
		Name:      "ask_local_llm",
		Arguments: map[string]interface{}{},
	})
	if err == nil {
		t.Error("expected error for missing 'prompt', got nil")
	}

	// Test missing text in summarize_local
	_, err = callTool(CallToolParams{
		Name:      "summarize_local",
		Arguments: map[string]interface{}{},
	})
	if err == nil {
		t.Error("expected error for missing 'text', got nil")
	}

	// Test unknown tool
	_, err = callTool(CallToolParams{
		Name:      "non_existent_tool",
		Arguments: map[string]interface{}{},
	})
	if err == nil {
		t.Error("expected error for unknown tool, got nil")
	}
}

func TestParseNumericArg(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
		ok       bool
	}{
		{float64(42.5), 42.5, true},
		{int(10), 10.0, true},
		{int64(100), 100.0, true},
		{"0.8", 0.8, true},
		{"invalid", 0, false},
		{nil, 0, false},
	}

	for _, tt := range tests {
		val, ok := parseNumericArg(tt.input)
		if ok != tt.ok || (ok && val != tt.expected) {
			t.Errorf("parseNumericArg(%v) = (%v, %v); want (%v, %v)", tt.input, val, ok, tt.expected, tt.ok)
		}
	}
}

func TestMockLLMServerStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/v1/chat/completions" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("data: {\"choices\": [{\"delta\": {\"content\": \"Hello \"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\": [{\"delta\": {\"content\": \"World!\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config.ServerURL = server.URL + "/v1/chat/completions"
	config.HealthURL = server.URL + "/health"

	output, err := queryLocalLLM("Hi", "System prompt", 100, 0.2, 0.9, false, true, nil)
	if err != nil {
		t.Fatalf("unexpected error querying mock LLM: %v", err)
	}

	if output != "Hello World!" {
		t.Errorf("expected 'Hello World!', got %q", output)
	}
}

func TestMockLLMServerNonStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/v1/chat/completions" {
			w.Header().Set("Content-Type", "application/json")
			resp := ChatCompletionResponse{
				Choices: []struct {
					Message ChatMessage `json:"message"`
				}{
					{Message: ChatMessage{Role: "assistant", Content: "Static response"}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config.ServerURL = server.URL + "/v1/chat/completions"
	config.HealthURL = server.URL + "/health"

	output, err := queryLocalLLM("Hi", "", 100, 0.2, 0.9, false, false, nil)
	if err != nil {
		t.Fatalf("unexpected error querying mock LLM non-streaming: %v", err)
	}

	if output != "Static response" {
		t.Errorf("expected 'Static response', got %q", output)
	}
}
