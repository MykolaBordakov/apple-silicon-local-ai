package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ServerName    = "local-llm-mcp"
	ServerVersion = "1.4.0"
)

// ServerConfig holds runtime configurations from environment variables or defaults.
type ServerConfig struct {
	ServerURL   string
	HealthURL   string
	ModelPath   string
	HTTPTimeout time.Duration
}

// Config instance initialized from env
var config ServerConfig

// Global Zap Logger writing to Stderr (to keep Stdout clean for JSON-RPC)
var logger *zap.Logger

// Shared HTTP Client with connection pooling
var httpClient *http.Client

func initConfig() {
	config = ServerConfig{
		ServerURL:   getEnv("LLM_SERVER_URL", "http://127.0.0.1:8080/v1/chat/completions"),
		HealthURL:   getEnv("LLM_HEALTH_URL", "http://127.0.0.1:8080/health"),
		ModelPath:   getEnv("LLM_MODEL_PATH", "./models/Devstral-Small-2-24B"),
		HTTPTimeout: parseDuration(getEnv("LLM_TIMEOUT", "300s"), 300*time.Second),
	}

	// Setup Zap Logger explicitly writing to os.Stderr
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.Lock(os.Stderr),
		zapcore.InfoLevel,
	)
	logger = zap.New(core)

	// Setup pooled HTTP Client
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	httpClient = &http.Client{
		Transport: transport,
		Timeout:   config.HTTPTimeout,
	}
}

func getEnv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func parseDuration(val string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return d
}

// JSON-RPC 2.0 Structs
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

type Capabilities struct {
	Tools     map[string]interface{} `json:"tools"`
	Prompts   map[string]interface{} `json:"prompts"`
	Resources map[string]interface{} `json:"resources"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool Definitions
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CallToolResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// MCP Prompts Structs
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
}

type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

type GetPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type PromptMessage struct {
	Role    string        `json:"role"`
	Content PromptContent `json:"content"`
}

type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// MCP Resources Structs
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}

type ReadResourceParams struct {
	URI string `json:"uri"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// OpenAI API Payload & Response Structs
type ChatMessage struct {
	Role      string      `json:"role"`
	Content   string      `json:"content"`
	ToolCalls interface{} `json:"tool_calls,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type ChatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
	TopP           float64         `json:"top_p,omitempty"`
	Stream         bool            `json:"stream"`
	Tools          interface{}     `json:"tools,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error map[string]interface{} `json:"error,omitempty"`
}

type StreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string      `json:"content"`
			Role      string      `json:"role"`
			ToolCalls interface{} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func main() {
	initConfig()
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("Starting MCP Go server",
		zap.String("server", ServerName),
		zap.String("version", ServerVersion),
		zap.String("target_url", config.ServerURL),
		zap.String("health_url", config.HealthURL),
	)

	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			logger.Error("failed to unmarshal JSON-RPC request", zap.Error(err))
			continue
		}

		handleRequest(req)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("stdio scanner error", zap.Error(err))
	}
}

func handleRequest(req JSONRPCRequest) {
	logger.Debug("handling JSON-RPC request", zap.String("method", req.Method), zap.Any("id", req.ID))

	switch req.Method {
	case "initialize":
		res := InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: Capabilities{
				Tools:     map[string]interface{}{},
				Prompts:   map[string]interface{}{},
				Resources: map[string]interface{}{},
			},
			ServerInfo: ServerInfo{
				Name:    ServerName,
				Version: ServerVersion,
			},
		}
		sendResponse(req.ID, res)

	case "notifications/initialized":
		// Notification - no response needed

	case "prompts/list":
		sendResponse(req.ID, getPromptsList())

	case "prompts/get":
		var params GetPromptParams
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				sendError(req.ID, -32602, fmt.Sprintf("invalid params: %v", err))
				return
			}
		}
		res, err := handleGetPrompt(params)
		if err != nil {
			sendError(req.ID, -32602, err.Error())
			return
		}
		sendResponse(req.ID, res)

	case "resources/list":
		sendResponse(req.ID, getResourcesList())

	case "resources/read":
		var params ReadResourceParams
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				sendError(req.ID, -32602, fmt.Sprintf("invalid params: %v", err))
				return
			}
		}
		res, err := handleReadResource(params)
		if err != nil {
			sendError(req.ID, -32602, err.Error())
			return
		}
		sendResponse(req.ID, res)

	case "tools/list":
		sendResponse(req.ID, ListToolsResult{Tools: getToolsList()})

	case "tools/call":
		var params CallToolParams
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				sendError(req.ID, -32602, fmt.Sprintf("invalid params: %v", err))
				return
			}
		}

		result, err := callTool(params)
		if err != nil {
			logger.Warn("tool execution error", zap.String("tool", params.Name), zap.Error(err))
			sendResponse(req.ID, CallToolResult{
				Content: []TextContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			})
			return
		}
		sendResponse(req.ID, result)

	default:
		if req.ID != nil {
			logger.Warn("unknown method requested", zap.String("method", req.Method))
			sendError(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
		}
	}
}

func getToolsList() []Tool {
	return []Tool{
		{
			Name:        "ask_local_llm",
			Description: "Send a prompt to local MLX Devstral-Small-24B (0 API tokens). Supports SSE streaming, custom temperature, top_p, json_mode, system prompts, and custom tools/functions array.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"prompt": {
						Type:        "string",
						Description: "The question or instruction to send to the local LLM.",
					},
					"system_prompt": {
						Type:        "string",
						Description: "Optional system prompt or persona for the local LLM.",
					},
					"max_tokens": {
						Type:        "integer",
						Description: "Maximum tokens to generate (default 2048, up to 4096).",
					},
					"temperature": {
						Type:        "number",
						Description: "Sampling temperature between 0.0 (exact/code) and 1.0 (creative). Default 0.2.",
					},
					"top_p": {
						Type:        "number",
						Description: "Nucleus sampling top_p parameter (default 0.95).",
					},
					"json_mode": {
						Type:        "boolean",
						Description: "If true, enforces valid JSON output format from the model.",
					},
					"stream": {
						Type:        "boolean",
						Description: "If true (default true), reads token stream via SSE to eliminate timeouts.",
					},
					"tools": {
						Type:        "string",
						Description: "Optional JSON string or array declaration of tools/functions for native function calling.",
					},
				},
				Required: []string{"prompt"},
			},
		},
		{
			Name:        "summarize_local",
			Description: "Summarize text or log files using local MLX LLM for free via SSE stream.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"text": {
						Type:        "string",
						Description: "The raw text or code log to summarize.",
					},
				},
				Required: []string{"text"},
			},
		},
	}
}

// MCP Prompts Implementation
func getPromptsList() ListPromptsResult {
	return ListPromptsResult{
		Prompts: []Prompt{
			{
				Name:        "code-review",
				Description: "Провести ретельний Code Review наданого фрагменту коду або diff",
				Arguments: []PromptArgument{
					{Name: "code", Description: "Сирцевий код або git diff для аналізу", Required: true},
					{Name: "language", Description: "Мова програмування (Go, Python, TypeScript)", Required: false},
				},
			},
			{
				Name:        "generate-tests",
				Description: "Сформувати юніт-тести для функції або модуля з урахуванням edge cases",
				Arguments: []PromptArgument{
					{Name: "code", Description: "Сирцевий код, для якого потрібні тести", Required: true},
					{Name: "framework", Description: "Тестовий фреймворк (testing, testify, pytest тощо)", Required: false},
				},
			},
			{
				Name:        "refactor-code",
				Description: "Виконати рефакторинг коду для підвищення читабельності та патернів",
				Arguments: []PromptArgument{
					{Name: "code", Description: "Сирцевий код для рефакторингу", Required: true},
					{Name: "goal", Description: "Основна мета (продуктивність, спрощення, асинхронність)", Required: false},
				},
			},
			{
				Name:        "debug-error",
				Description: "Проаналізувати лог помилки або stack trace та запропонувати виправлення",
				Arguments: []PromptArgument{
					{Name: "error_log", Description: "Лог помилки, паніка або stack trace", Required: true},
					{Name: "context", Description: "Додатковий контекст або опис поведінки", Required: false},
				},
			},
		},
	}
}

func handleGetPrompt(params GetPromptParams) (GetPromptResult, error) {
	code := params.Arguments["code"]
	switch params.Name {
	case "code-review":
		if code == "" {
			return GetPromptResult{}, fmt.Errorf("argument 'code' is required")
		}
		lang := params.Arguments["language"]
		if lang == "" {
			lang = "auto-detect"
		}
		promptText := fmt.Sprintf("Проведи ретельний code review наступного коду (Мова: %s):\n\n```%s\n%s\n```\n\nПроаналізуй:\n1. Корректність та логічні помилки\n2. Продуктивність та використання пам'яті\n3. Читабельність та відповідність best practices\n4. Безпеку та обробку помилок", lang, lang, code)
		return GetPromptResult{
			Description: "Code Review Prompt",
			Messages: []PromptMessage{
				{Role: "user", Content: PromptContent{Type: "text", Text: promptText}},
			},
		}, nil

	case "generate-tests":
		if code == "" {
			return GetPromptResult{}, fmt.Errorf("argument 'code' is required")
		}
		fw := params.Arguments["framework"]
		if fw == "" {
			fw = "standard library"
		}
		promptText := fmt.Sprintf("Напиши повний набір юніт-тестів для наступного коду (Фреймворк: %s):\n\n```\n%s\n```\n\nВключи:\n- Успішні сценарії (Happy path)\n- Граничні значення (Edge cases)\n- Перевірку обробки помилок", fw, code)
		return GetPromptResult{
			Description: "Unit Test Generation Prompt",
			Messages: []PromptMessage{
				{Role: "user", Content: PromptContent{Type: "text", Text: promptText}},
			},
		}, nil

	case "refactor-code":
		if code == "" {
			return GetPromptResult{}, fmt.Errorf("argument 'code' is required")
		}
		goal := params.Arguments["goal"]
		if goal == "" {
			goal = "clean code and idiomatic patterns"
		}
		promptText := fmt.Sprintf("Зроби рефакторинг наступного коду з акцентом на '%s':\n\n```\n%s\n```\n\nПокроково поясни внесені зміни та надай підсумковий рефакторений код.", goal, code)
		return GetPromptResult{
			Description: "Code Refactoring Prompt",
			Messages: []PromptMessage{
				{Role: "user", Content: PromptContent{Type: "text", Text: promptText}},
			},
		}, nil

	case "debug-error":
		errLog := params.Arguments["error_log"]
		if errLog == "" {
			return GetPromptResult{}, fmt.Errorf("argument 'error_log' is required")
		}
		ctx := params.Arguments["context"]
		promptText := fmt.Sprintf("Проаналізуй помилку/стектрейс:\n\n```\n%s\n```\nContext: %s\n\nВизнач першопричину (Root Cause) та надай конкретний алгоритм/код для виправлення.", errLog, ctx)
		return GetPromptResult{
			Description: "Error Debugging Prompt",
			Messages: []PromptMessage{
				{Role: "user", Content: PromptContent{Type: "text", Text: promptText}},
			},
		}, nil

	default:
		return GetPromptResult{}, fmt.Errorf("prompt not found: %s", params.Name)
	}
}

// MCP Resources Implementation
func getResourcesList() ListResourcesResult {
	return ListResourcesResult{
		Resources: []Resource{
			{
				URI:         "local-llm://status",
				Name:        "Local LLM Health & Status",
				Description: "Динамічний статус здоров'я локального MLX LLM сервера",
				MIMEType:    "application/json",
			},
			{
				URI:         "local-llm://logs",
				Name:        "Local LLM Server Log History",
				Description: "Лог-повідомлення та останні метрики викликів сервера",
				MIMEType:    "text/plain",
			},
			{
				URI:         "local-llm://config",
				Name:        "Local LLM Runtime Config",
				Description: "Параметри конфігурації (модель, ендпоінти, таймаути)",
				MIMEType:    "application/json",
			},
		},
	}
}

func handleReadResource(params ReadResourceParams) (ReadResourceResult, error) {
	switch params.URI {
	case "local-llm://status":
		statusStr := "online"
		err := checkServerHealth()
		if err != nil {
			statusStr = fmt.Sprintf("offline (%v)", err)
		}
		statusJSON := fmt.Sprintf(`{"status":"%s","endpoint":"%s","health_url":"%s","model":"%s","time":"%s"}`,
			statusStr, config.ServerURL, config.HealthURL, config.ModelPath, time.Now().Format(time.RFC3339))
		return ReadResourceResult{
			Contents: []ResourceContent{
				{URI: params.URI, MIMEType: "application/json", Text: statusJSON},
			},
		}, nil

	case "local-llm://logs":
		logs := fmt.Sprintf("[%s] MCP Server %s v%s running on stdio.\n[%s] Target URL: %s\n",
			time.Now().Add(-5*time.Minute).Format(time.RFC3339), ServerName, ServerVersion, time.Now().Format(time.RFC3339), config.ServerURL)
		return ReadResourceResult{
			Contents: []ResourceContent{
				{URI: params.URI, MIMEType: "text/plain", Text: logs},
			},
		}, nil

	case "local-llm://config":
		configJSON := fmt.Sprintf(`{"server_name":"%s","version":"%s","default_url":"%s","health_url":"%s","default_model":"%s","default_timeout_sec":%d}`,
			ServerName, ServerVersion, config.ServerURL, config.HealthURL, config.ModelPath, int(config.HTTPTimeout.Seconds()))
		return ReadResourceResult{
			Contents: []ResourceContent{
				{URI: params.URI, MIMEType: "application/json", Text: configJSON},
			},
		}, nil

	default:
		return ReadResourceResult{}, fmt.Errorf("resource not found: %s", params.URI)
	}
}

func callTool(params CallToolParams) (CallToolResult, error) {
	switch params.Name {
	case "ask_local_llm":
		prompt, _ := params.Arguments["prompt"].(string)
		if strings.TrimSpace(prompt) == "" {
			return CallToolResult{}, fmt.Errorf("parameter 'prompt' is required and cannot be empty")
		}

		systemPrompt, _ := params.Arguments["system_prompt"].(string)

		maxTokens := 2048
		if val, ok := parseNumericArg(params.Arguments["max_tokens"]); ok && val > 0 {
			maxTokens = int(val)
		}

		temperature := 0.2
		if val, ok := parseNumericArg(params.Arguments["temperature"]); ok {
			temperature = val
		}

		topP := 0.95
		if val, ok := parseNumericArg(params.Arguments["top_p"]); ok {
			topP = val
		}

		jsonMode, _ := params.Arguments["json_mode"].(bool)

		stream := true
		if val, ok := params.Arguments["stream"].(bool); ok {
			stream = val
		}

		// Handle tools payload flexibly (supports stringified JSON, raw arrays, or raw maps)
		var toolsPayload interface{}
		if rawTools, exists := params.Arguments["tools"]; exists && rawTools != nil {
			switch v := rawTools.(type) {
			case string:
				if strings.TrimSpace(v) != "" {
					var parsed interface{}
					if err := json.Unmarshal([]byte(v), &parsed); err == nil {
						toolsPayload = parsed
					} else {
						toolsPayload = v
					}
				}
			case []interface{}, map[string]interface{}:
				toolsPayload = v
			}
		}

		output, err := queryLocalLLM(prompt, systemPrompt, maxTokens, temperature, topP, jsonMode, stream, toolsPayload)
		if err != nil {
			return CallToolResult{}, err
		}
		return CallToolResult{
			Content: []TextContent{{Type: "text", Text: output}},
		}, nil

	case "summarize_local":
		text, _ := params.Arguments["text"].(string)
		if strings.TrimSpace(text) == "" {
			return CallToolResult{}, fmt.Errorf("parameter 'text' is required and cannot be empty")
		}

		systemPrompt := "You are a helpful assistant. Summarize the following text concisely, highlighting key points, errors, or findings."
		output, err := queryLocalLLM("Please summarize this content:\n\n"+text, systemPrompt, 1024, 0.1, 0.9, false, true, nil)
		if err != nil {
			return CallToolResult{}, err
		}
		return CallToolResult{
			Content: []TextContent{{Type: "text", Text: output}},
		}, nil

	default:
		return CallToolResult{}, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func parseNumericArg(val interface{}) (float64, bool) {
	if val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// checkServerHealth checks if mlx_lm.server is responding to /health endpoint
func checkServerHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", config.HealthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mlx_lm.server health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mlx_lm.server health check returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func queryLocalLLM(userPrompt, systemPrompt string, maxTokens int, temperature, topP float64, jsonMode bool, stream bool, tools interface{}) (string, error) {
	if err := checkServerHealth(); err != nil {
		return "", fmt.Errorf("local LLM server is offline: %w (ensure mlx_lm.server is running at %s)", err, config.HealthURL)
	}

	var messages []ChatMessage
	if systemPrompt != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: userPrompt})

	reqBody := ChatCompletionRequest{
		Model:       config.ModelPath,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		TopP:        topP,
		Stream:      stream,
		Tools:       tools,
	}

	if jsonMode {
		reqBody.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", config.ServerURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to reach local LLM server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("local LLM returned HTTP %d: %s", resp.StatusCode, string(errBytes))
	}

	// Handle SSE Streaming
	if stream {
		var accumulatedText strings.Builder
		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 1024*1024)
		scanner.Buffer(buf, 10*1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			if strings.HasPrefix(line, "data:") {
				dataPayload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if dataPayload == "[DONE]" {
					break
				}

				var chunk StreamChunk
				if err := json.Unmarshal([]byte(dataPayload), &chunk); err == nil {
					if len(chunk.Choices) > 0 {
						accumulatedText.WriteString(chunk.Choices[0].Delta.Content)
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Warn("SSE scanner warning", zap.Error(err))
		}

		resultText := accumulatedText.String()
		if resultText == "" {
			return "", fmt.Errorf("empty text accumulated from SSE stream")
		}
		return resultText, nil
	}

	// Non-streaming fallback
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response from local LLM: %w. raw body: %s", err, string(respBytes))
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("local LLM returned no choices: %s", string(respBytes))
	}

	return chatResp.Choices[0].Message.Content, nil
}

func sendResponse(id interface{}, result interface{}) {
	if id == nil {
		return
	}
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		logger.Error("failed to marshal JSON-RPC response", zap.Error(err))
		return
	}
	fmt.Printf("%s\n", data)
}

func sendError(id interface{}, code int, message string) {
	if id == nil {
		return
	}
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		logger.Error("failed to marshal JSON-RPC error response", zap.Error(err))
		return
	}
	fmt.Printf("%s\n", data)
}
