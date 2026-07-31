---
name: qdrant-rag
description: "Use Qdrant vector database (at 192.168.0.107:6333 or localhost) with FastEmbed (BGE-Small) and local Devstral-24B for high-performance codebase vector search, line-aware text chunking, deterministic upserts, and autonomous RAG."
---

# Qdrant RAG Rust MCP Skill 🦀

This skill provides instructions for using the high-performance Rust MCP server `qdrant-rag-mcp` for codebase vectorization and Retrieval-Augmented Generation (RAG).

## Executable Location
- **Executable**: `./qdrant-rag-mcp/target/release/qdrant-rag-mcp`
- **Build command**: `cd qdrant-rag-mcp && cargo build --release`

## Key Architecture & Features
- **Deterministic Point IDs**: Point IDs are generated via `hash(file_path:chunk_index)`. Re-indexing a file automatically overwrites existing chunks without creating duplicates.
- **Line-Aware Text Chunking**: Code files are chunked into `CodeChunk` objects tracking 1-indexed `line_start` and `line_end`.
- **Automatic Payload Keyword Indexing**: Indexed collections automatically create keyword payload indexes for `source`, `file_path`, `relative_path`, `language`, `project_name`, and `git_repo`.
- **Clickable Context Anchors**: RAG context is formatted with file location anchors (`file://<file_path>#L<start>-L<end>`).

## Available MCP Tools

### 1. `qdrant_index_path`
Indexes a local file or text snippet into Qdrant.
- **Parameters**:
  - `path` (string, required): Absolute file path or text string to index.
  - `collection_name` (string, optional): Qdrant collection (default: `codebase_knowledge`).
  - `project_name` (string, optional): Name of project/repository (e.g. `apple-silicon-local-ai`).
  - `git_repo` (string, optional): Git remote URL (e.g. `https://github.com/MykolaBordakov/apple-silicon-local-ai.git`).
  - `chunk_size` (integer, optional): Character size per chunk (default: 500).

### 2. `qdrant_search`
Performs semantic vector similarity search in Qdrant.
- **Parameters**:
  - `query` (string, required): Semantic search prompt.
  - `collection_name` (string, optional): Target collection.
  - `limit` (integer, optional): Number of top results to return (default: 5).

### 3. `qdrant_rag_ask`
Executes end-to-end RAG: retrieves relevant codebase context from Qdrant with file location anchors and queries local Devstral-24B model for a grounded answer.
- **Parameters**:
  - `query` (string, required): Question about the codebase.
  - `collection_name` (string, optional): Target collection.

### 4. `qdrant_list_collections`
Lists all active vector collections available in the Qdrant instance.
