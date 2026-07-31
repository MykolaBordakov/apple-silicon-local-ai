---
name: local-code-reviewer
description: "Perform zero-token-cost local code reviews, security audits, and performance profiling using Devstral-24B via local-llm-mcp."
---

# Local Code Reviewer Skill 🛡

This skill uses the local `Devstral-Small-2-24B` model to perform deep, privacy-preserving, zero-cost code reviews.

## Executable Location
- **Go MCP Server**: `./mcp-server-go/local-llm-mcp`

## Workflow

### 1. Code Review Checklist
When analyzing code, pass the target code snippet or diff to `ask_local_llm` with instructions to review:
1. **Security & Vulnerabilities**: SQL injection, unhandled errors, secret leaks, unsafe pointers.
2. **Performance & Memory**: Allocation hotspots, unnecessary loops, missing context deadlines.
3. **Idiomatic Go/Rust Style**: Error handling conventions, interface design, conciseness.

### 2. Execution Call Example
Call `ask_local_llm` via MCP with `temperature: 0.1` and `system_prompt`:
> "You are a Principal Code Reviewer. Analyze the provided code snippet for bugs, security risks, and optimization opportunities. Return structured markdown with code diff suggestions."
