---
name: codebase-indexer
description: "Scan and bulk-index an entire local repository into Qdrant vector database using qdrant-rag-mcp with automatic project_name and git_repo tagging."
---

# Codebase Indexer Skill 🚀

This skill automates scanning and bulk-indexing source code files from a workspace into the Qdrant vector database via `qdrant-rag-mcp`.

## Executable Location
- **Rust MCP Server**: `./qdrant-rag-mcp/target/release/qdrant-rag-mcp`

## Workflow

### 1. Automatic Repository Indexing
To index all key source files from a repository:
1. Detect project root and Git remote URL (`git remote get-url origin`).
2. Collect source files (excluding `.gitignore` paths, `.git/`, `node_modules/`, `target/`, `.venv/`, binaries).
3. Call `qdrant_index_path` for each file:
   - `path`: Absolute filepath
   - `collection_name`: `codebase_knowledge`
   - `project_name`: Repository folder name
   - `git_repo`: Remote Git URL

### 2. Overwrite Protection
Because `qdrant-rag-mcp` generates deterministic point IDs (`hash(file_path:chunk_index)`), re-running this indexer overwrites modified code chunks without creating duplicate points.
