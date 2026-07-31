---
name: local-llm
description: "Use the local MLX Devstral-Small-2-24B model via local-llm-mcp Go server for free, offline code analysis, summarization, code reviews, and reasoning without using cloud API tokens."
---

# Local LLM MCP Skill 🛠

This skill provides instructions for using the local `Devstral-Small-2-24B` LLM running on Apple Silicon via `local-llm-mcp`.

## MCP Executable Location
- **Executable**: `/Users/Shared/LLM-Mykola/mcp-server-go/local-llm-mcp`
- **Build command**: `cd mcp-server-go && go build -o local-llm-mcp main.go`
- **Model Server URL**: `http://127.0.0.1:8080/v1/chat/completions`
- **Health Check**: `http://127.0.0.1:8080/health`

## Available MCP Tools

### 1. `ask_local_llm`
Sends a prompt to the local Devstral-24B model.
- **Parameters**:
  - `prompt` (string, required): The prompt or instruction for the local model.
  - `system_prompt` (string, optional): System prompt/persona for model guidance.
  - `temperature` (number, optional): Sampling temperature (0.0 to 1.0, default: 0.2).
  - `top_p` (number, optional): Nucleus sampling top_p (default: 0.95).
  - `max_tokens` (integer, optional): Maximum tokens to generate (default: 2048).
  - `json_mode` (boolean, optional): Enforces strict JSON output format.
  - `stream` (boolean, optional): Enables SSE token streaming.
  - `tools` (string/array, optional): Native tool definitions for function calling.

### 2. `summarize_local`
Quickly summarizes large text or log output using local Devstral-24B model with 0 cloud tokens.
- **Parameters**:
  - `text` (string, required): Text content to summarize.

## Startup Script
If the local model server is offline, start it with:
```bash
/Users/Shared/LLM-Mykola/run_devstral.sh
```

## CLI Terminal Chat
For direct interactive chat with the model in terminal:
```bash
cd /Users/Shared/LLM-Mykola/cli-chat && ./cli-chat
```
