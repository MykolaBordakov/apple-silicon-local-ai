# Project TODO & Roadmap 📋

> **LLM Mykola & Apple Silicon Local AI Stack**

---

## 📌 Pending Tasks

### 1. 📚 Index ADR Docs & Sync RAG (Upon VPN Disconnect)
- [x] Connect to local Qdrant instance (`192.168.0.107:6333`).
- [x] Bulk-index new Architecture Decision Records:
  - `docs/adr/0001-qdrant-deterministic-point-ids.md`
  - `docs/adr/0002-git-repo-and-project-name-metadata.md`
  - `docs/adr/0003-hybrid-ai-architecture-senior-cloud-middle-local.md`
- [x] Run `qdrant_search` verification to confirm clean payload retrieval.

---

### 2. 💬 Chat History Indexing & Dedicated Collection (`chat_history`)
- [x] Create an isolated Qdrant collection `chat_history` (separated from `codebase_knowledge` to prevent vector noise).
- [x] Build a transcript log parser for Antigravity JSONL session logs (`transcript.jsonl`).
- [x] Add `qdrant_index_chat_history` tool to `qdrant-rag-mcp` for indexing dialogue sessions.
- [x] Allow searching past architectural discussions without polluting code RAG results.

---

### 3. 🚀 Go CLI Chat Native Tool Calling Loop
- [x] Add tool execution loop into `cli-chat/main.go`.
- [x] Enable `cli-chat` to call local MCP tools (`qdrant_search`, `read_file`, `exec_cmd`) directly via Devstral-24B for 100% zero-cloud-token local development.

---

## ✅ Completed Tasks
- [x] Implemented Rust `qdrant-rag-mcp` server with FastEmbed (BGE-Small).
- [x] Implemented line-aware text chunking (`line_start`, `line_end`) and unicode safety.
- [x] Implemented deterministic `point_id = hash(file_path:chunk_index)` for duplicate-free upserts.
- [x] Added `git_repo`, `project_name`, `relative_path`, and `language` payload metadata.
- [x] Added automatic `git remote get-url origin` detection in `qdrant-rag-mcp`.
- [x] Implemented Go `mcp-server-go` adapter for MLX Devstral-24B model.
- [x] Restructured project folders (`cli-chat/`, `mcp-server-go/`, `qdrant-rag-mcp/`).
- [x] Created clean `.gitignore` and removed temporary server binaries.
- [x] Published documentation in Ukrainian (`README.md`) and English (`README_EN.md`).
- [x] Published repository to GitHub (`git@github.com:MykolaBordakov/apple-silicon-local-ai.git`).
- [x] Created agent skills in `.agents/skills/` (`qdrant-rag`, `local-llm`, `codebase-indexer`, `local-code-reviewer`).
- [x] Established Knowledge Extraction ADR system (`docs/adr/0001` - `0003`).
