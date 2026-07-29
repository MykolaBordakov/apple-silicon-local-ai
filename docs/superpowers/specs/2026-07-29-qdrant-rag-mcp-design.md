# Design Specification: Rust Qdrant RAG MCP Server (`qdrant-rag-mcp`)

**Date**: 2026-07-29  
**Status**: Approved  
**Language**: Rust (Edition 2021)  
**Target Architecture**: Apple Silicon (macOS) / Native Async Rust  

---

## 1. Executive Summary

`qdrant-rag-mcp` is a high-performance, fully native Model Context Protocol (MCP) server written in Rust. It integrates an external vector database (**Qdrant** running at `http://192.168.0.107:6333`) with a local embedding engine (**FastEmbed Rust** running `bge-small-en-v1.5`) and a local LLM inference server (**Devstral-Small-2-24B** on `http://127.0.0.1:8080`).

It allows AI agents and developers to index codebases, search vector spaces, switch between project collections, and perform zero-cost autonomous Retrieval-Augmented Generation (RAG).

---

## 2. Architecture & Data Flow

```
┌─────────────────────────────────────────┐
│         Antigravity AI Agent / IDE      │
│         (MCP Client via stdio)          │
└────────────────────┬────────────────────┘
                     │ stdio (JSON-RPC 2.0)
                     ▼
┌─────────────────────────────────────────┐
│       Rust MCP Server (qdrant-rag-mcp)  │
│  ├── stdio JSON-RPC loop                │
│  ├── FastEmbed (bge-small-en-v1.5)      │ ──► [In-Memory Embeddings ~130MB]
│  └── Qdrant & LLM HTTP Clients          │
└──────────────┬──────────────────┬───────┘
               │                  │
   REST/gRPC   │                  │ HTTP POST /v1/chat/completions
               ▼                  ▼
┌──────────────────────────┐  ┌───────────────────────────────────┐
│   Qdrant Vector Server   │  │   Local MLX Backend (Devstral-24B)│
│  192.168.0.107:6333      │  │   127.0.0.1:8080                  │
└──────────────────────────┘  └───────────────────────────────────┘
```

---

## 3. Directory & File Structure

Path: `/Users/Shared/LLM-Mykola/qdrant-rag-mcp`

```
qdrant-rag-mcp/
├── Cargo.toml
└── src/
    ├── main.rs         # Stdio loop, tracing logger to stderr
    ├── config.rs       # Environment configuration
    ├── embed.rs        # FastEmbed text vectorizer wrapper
    ├── qdrant.rs       # Qdrant REST/gRPC client & collection management
    ├── llm.rs          # HTTP Client for local Devstral-24B
    └── mcp.rs          # MCP JSON-RPC protocol handlers & tool definitions
```

---

## 4. Configuration Parameters

Configured via environment variables with fallback defaults:

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `QDRANT_URL` | `http://192.168.0.107:6333` | Qdrant instance endpoint |
| `QDRANT_API_KEY` | `4a3f5bad98ec768011aa8d8c2734a20f11a3f97899c0a3e0b52c949d5e482a24` | Qdrant authentication API key |
| `LLM_URL` | `http://127.0.0.1:8080/v1/chat/completions` | Local Devstral-24B endpoint |
| `DEFAULT_COLLECTION` | `codebase_knowledge` | Fallback collection name if omitted |

---

## 5. MCP Tool Definitions

### 1. `qdrant_index_path`
- **Description**: Index a local file or entire folder into Qdrant.
- **Parameters**:
  - `path` (string, required): File or directory path to index.
  - `collection_name` (string, optional): Target collection (e.g. `go_patterns`, `project_alpha`).
  - `chunk_size` (integer, optional, default `500`): Chunk character length.
  - `overlap` (integer, optional, default `100`): Overlap character length.

### 2. `qdrant_search`
- **Description**: Perform semantic vector search over a specified or default collection.
- **Parameters**:
  - `query` (string, required): Search query.
  - `collection_name` (string, optional): Target collection.
  - `limit` (integer, optional, default `5`): Top-K results to return.

### 3. `qdrant_rag_ask`
- **Description**: End-to-end RAG pipeline: Query Qdrant vector context -> format system context -> query local Devstral-24B -> return answer.
- **Parameters**:
  - `query` (string, required): User question.
  - `collection_name` (string, optional): Target collection.
  - `limit` (integer, optional, default `5`): Context snippets limit.

### 4. `qdrant_list_collections`
- **Description**: List all collections stored in Qdrant, showing vector counts and status.
- **Parameters**: None.

---

## 6. Verification Plan

1. **Compilation**: `cargo build --release` cleanly without errors.
2. **Unit Tests**: `cargo test` verifying chunking, vector generation, and JSON-RPC parsing.
3. **Integration Test**:
   - Create a test collection `test_patterns`.
   - Index a sample Go file into `test_patterns`.
   - Run `qdrant_search` and verify cosine similarity matches.
   - Run `qdrant_rag_ask` to verify end-to-end local LLM response enriched with Qdrant context.
