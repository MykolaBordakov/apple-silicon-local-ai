---
name: local-llm
description: "Use the local MLX Devstral-Small-2-24B model via local-llm-mcp Go server for free, offline code analysis, summarization, code reviews, and reasoning without using cloud API tokens."
---

# Local LLM MCP Skill

This skill provides instructions for utilizing the local `Devstral-Small-2-24B` LLM running on Apple Silicon via `local-llm-mcp`.

## MCP Executable Location
- **Executable**: `/Users/Shared/LLM-Mykola/mcp-server-go/local-llm-mcp`
- **Server URL**: `http://127.0.0.1:8080/v1/chat/completions`
- **Health URL**: `http://127.0.0.1:8080/health`

## Available MCP Tools
- `ask_local_llm(prompt, system_prompt, max_tokens, temperature, top_p, json_mode, stream, tools)`
  - Use to send queries to the local 24B model for code generation, review, or zero-cost token tasks.
- `summarize_local(text)`
  - Use to quickly summarize large log files or code output locally.

## Start Local Server Command
If the local server is offline, start it with:
```bash
/Users/Shared/LLM-Mykola/run_devstral.sh
```
