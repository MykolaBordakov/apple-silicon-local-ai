# ADR 0003: Hybrid AI Architecture (Senior Cloud + Middle Local via MCP)

## Status
Accepted

## Context
Running full multi-file codebase reasoning exclusively on cloud LLMs (Gemini/Claude) consumes massive token quotas when sending entire source trees. Conversely, running a smaller local model exclusively can lead to hallucinations in complex multi-step architectural planning.

## Decision
We established a two-tier hybrid AI architecture:
- **Senior Architect (Cloud LLM)**: Handles high-level reasoning, strategy, complex refactoring plans, and quality control with minimal token expenditure.
- **Middle Developer (Local Devstral-24B on Apple Silicon)**: Executes heavy, zero-cost tasks (Qdrant RAG searches, code parsing, AST metadata generation, log summarization) via Go and Rust MCP servers (`mcp-server-go`, `qdrant-rag-mcp`).

## Consequences
- **90-95% Reduction in Cloud Token Expenditure**: Bulk context processing is offloaded to Apple Silicon (Metal MLX).
- **Zero API Cost for Heavy Tasks**: Log parsing, code reviews, and RAG lookups run locally at 30-50+ tokens/second.
- **Strict Privacy**: Local codebase embeddings and sensitive source code remain within the local Mac environment.
