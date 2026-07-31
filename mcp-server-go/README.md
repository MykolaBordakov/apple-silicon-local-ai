# Local LLM Go MCP Server (`local-llm-mcp`)

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Server Version](https://img.shields.io/badge/version-1.4.0-blue)](https://github.com/)
[![MCP Protocol](https://img.shields.io/badge/MCP%20Protocol-2024--11--05-green)](https://modelcontextprotocol.io/)

A high-performance **Model Context Protocol (MCP)** server written in Go that acts as a bridge between MCP-compliant clients (such as Google Antigravity IDE, Claude Code, Cursor, VS Code) and local LLM backends (e.g. `mlx_lm.server` executing **Devstral-Small-2-24B** on Apple Silicon).

---

## 🚀 Architecture & Overview

The server communicates via standard input/output (`stdio`) using JSON-RPC 2.0 messages. It enables developers and automated AI agents to execute local inference with **zero cloud API token costs**, total data privacy, and minimal response latency using Server-Sent Events (SSE) streaming.

```
┌────────────────────────────────┐          stdio          ┌─────────────────────────────────┐
│     Antigravity IDE / Agent    │ ◄─────────────────────► │   Local LLM Go MCP Server       │
│   (MCP Client - JSON-RPC 2.0)  │                         │     (local-llm-mcp v1.4.0)      │
└────────────────────────────────┘                         └────────────────┬────────────────┘
                                                                            │ HTTP / SSE
                                                                            ▼
                                                           ┌─────────────────────────────────┐
                                                           │   Apple Silicon MLX Backend     │
                                                           │   (mlx_lm.server:8080)          │
                                                           │   Devstral-Small-2-24B          │
                                                           └─────────────────────────────────┘
```

### Key Architectural Features
- **Zero API Token Costs**: Offloads context analysis, code reviews, and test generation entirely to local hardware.
- **SSE Streaming Support**: Parses `text/event-stream` chunks from `mlx_lm.server` real-time, eliminating HTTP connection timeouts during long token generations.
- **Strict Stdio Separation**: Operational logging uses `go.uber.org/zap` configured explicitly to stream to `os.Stderr`, guaranteeing clean, unpolluted JSON-RPC framing over `os.Stdout`.
- **Resilient Connection Pooling**: Built-in HTTP `Transport` reuse with active `/health` check validation before dispatching prompts.

---

## ⚙️ Configuration

The server is configured dynamically via environment variables with sensible defaults:

| Environment Variable | Default Value | Description |
| :--- | :--- | :--- |
| `LLM_SERVER_URL` | `http://127.0.0.1:8080/v1/chat/completions` | Full endpoint URL for OpenAI-compatible chat completion requests. |
| `LLM_HEALTH_URL` | `http://127.0.0.1:8080/health` | Healthcheck URL tested prior to executing requests. |
| `LLM_MODEL_PATH` | `./models/Devstral-Small-2-24B` | Target model path/name identifier sent to the LLM backend. |
| `LLM_TIMEOUT` | `300s` | Maximum HTTP client timeout duration (parsed via `time.ParseDuration`). |

---

## 🛠️ Complete Tool Specifications

### 1. `ask_local_llm`
Sends instructions or code prompts to the local Devstral-Small-24B model. Supports system personas, token limits, sampling parameters, structured JSON output, and native function declarations.

#### Input Schema Arguments:

| Argument | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `prompt` | `string` | **Yes** | — | The question, instruction, or code block to send to the LLM. |
| `system_prompt` | `string` | No | `""` | Optional system persona or background instructions. |
| `max_tokens` | `integer` | No | `2048` | Maximum tokens to generate (up to `4096`). |
| `temperature` | `number` | No | `0.2` | Sampling temperature (`0.0` for precise/code generation, `1.0` for creative). |
| `top_p` | `number` | No | `0.95` | Nucleus sampling probability parameter. |
| `json_mode` | `boolean` | No | `false` | When `true`, enforces strict `json_object` format response. |
| `stream` | `boolean` | No | `true` | Reads token stream via SSE to avoid timeouts. |
| `tools` | `string`/`array`| No | `nil` | Optional JSON string or schema array declaring functions for tool calling. |

#### JSON-RPC Tool Call Example:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "ask_local_llm",
    "arguments": {
      "prompt": "Explain Go concurrency channels and select statement with a quick example.",
      "system_prompt": "You are an expert Go backend architect.",
      "temperature": 0.1,
      "max_tokens": 1024
    }
  }
}
```

---

### 2. `summarize_local`
Concise summarization utility optimized for large log files, stack traces, or text documents over SSE streams.

#### Input Schema Arguments:

| Argument | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `text` | `string` | **Yes** | The raw text, log output, or code snippet to summarize. |

#### JSON-RPC Tool Call Example:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "summarize_local",
    "arguments": {
      "text": "goroutine 1 [running]:\nmain.processJob(0xc00005a080)\n\t/app/main.go:42 +0x3b\npanic: runtime error: invalid memory address or nil pointer dereference"
    }
  }
}
```

---

## 📋 MCP Prompts

The server exposes built-in prompts accessible via `prompts/list` and `prompts/get`:

| Prompt Name | Description | Required Arguments | Optional Arguments |
| :--- | :--- | :--- | :--- |
| `code-review` | Thorough Code Review analyzing logic, performance, safety, and best practices. | `code` | `language` |
| `generate-tests` | Automated unit test generator covering happy path, edge cases, and errors. | `code` | `framework` |
| `refactor-code` | Code refactoring focused on clean code, idiomatic Go patterns, and efficiency. | `code` | `goal` |
| `debug-error` | Error log and stack trace diagnostic helper providing root cause and fixes. | `error_log` | `context` |

### Prompt Invocation Example (`prompts/get`):
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "prompts/get",
  "params": {
    "name": "code-review",
    "arguments": {
      "code": "func calc(a, b int) int { return a / b }",
      "language": "Go"
    }
  }
}
```

---

## 📦 MCP Resources

Exposes real-time inspectable resources via `resources/list` and `resources/read`:

| URI | Resource Name | MIME Type | Description |
| :--- | :--- | :--- | :--- |
| `local-llm://status` | Local LLM Health & Status | `application/json` | Dynamic health check result, endpoint URLs, and current model path. |
| `local-llm://logs` | Local LLM Server Log History | `text/plain` | Operational log output and execution timestamps. |
| `local-llm://config` | Local LLM Runtime Config | `application/json` | Active runtime configuration parameter snapshot. |

### Resource Reading Example (`resources/read`):
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "resources/read",
  "params": {
    "uri": "local-llm://status"
  }
}
```

---

## 🛠️ Build, Testing & Quality Assurance

### 1. Building the Binary
```bash
# Build standalone executable binary
go build -ldflags="-s -w" -o local-llm-mcp main.go
```

### 2. Unit Testing
Run unit tests (includes mock HTTP server streaming tests for SSE & non-streaming responses):
```bash
# Run tests with verbose log output
go test -v ./...

# Run tests with race detection & coverage analysis
go test -race -cover ./...
```

### 3. Security Auditing (`gosec`)
```bash
# Install and run security analyzer
go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
```

### 4. Code Linting (`golangci-lint`)
```bash
# Run official Go linters
golangci-lint run
```

---

## ⚡ Antigravity IDE Integration

To connect `local-llm-mcp` with **Google Antigravity IDE**, edit your global MCP configuration file:

**File path:** `~/.gemini/config/mcp_config.json`

```json
{
  "mcpServers": {
    "local-llm": {
      "command": "/Users/Shared/LLM-Mykola/mcp-server-go/local-llm-mcp",
      "args": [],
      "env": {
        "LLM_SERVER_URL": "http://127.0.0.1:8080/v1/chat/completions",
        "LLM_HEALTH_URL": "http://127.0.0.1:8080/health",
        "LLM_MODEL_PATH": "./models/Devstral-Small-2-24B",
        "LLM_TIMEOUT": "300s"
      }
    }
  }
}
```

---

## 📄 License

Internal Development Utility — Free for local MLX AI acceleration.
