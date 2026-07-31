# LLM Mykola & Local MCP Stack 🚀

> **High-Performance Standalone Local AI Ecosystem for Apple Silicon (M1-M4).**

This repository contains a complete ecosystem for running the local `Devstral-Small-2-24B` Large Language Model, high-speed MCP (Model Context Protocol) servers in Go and Rust, and an integrated **Qdrant** vector database for autonomous Codebase RAG.

---

## 🏛 Ecosystem Architecture

```
                               ┌───────────────────────────┐
                               │     AI Client / Agent     │
                               │ (Antigravity / IDE / CLI) │
                               └─────────────┬─────────────┘
                                             │ (MCP Protocol / JSON-RPC over stdio)
                      ┌──────────────────────┴──────────────────────┐
                      ▼                                             ▼
        ┌───────────────────────────┐                 ┌───────────────────────────┐
        │     mcp-server-go (Go)    │                 │   qdrant-rag-mcp (Rust)   │
        │  Local LLM Proxy Server   │                 │   Vector RAG MCP Server   │
        └─────────────┬─────────────┘                 └─────────────┬─────────────┘
                      │ (HTTP API / v1/completions)                 │ (FastEmbed BGE-Small)
                      ▼                                             ▼
        ┌───────────────────────────┐                 ┌───────────────────────────┐
        │   mlx_lm.server (Python)  │                 │    Qdrant Vector DB      │
        │ Devstral-Small-2-24B 4-bit│                 │   (192.168.0.107:6333)    │
        └───────────────────────────┘                 └───────────────────────────┘
```

The ecosystem consists of four core modules:

1. **Devstral-Small-2-24B MLX Server** — Local inference for the 24B model with 4-bit quantization powered by the MLX framework, optimized for Metal & Unified Memory.
2. **Go MCP Server (`mcp-server-go/`)** — Protocol adapter written in Go connecting AI agents to the local model (supports SSE streaming, JSON-Mode, Health-check).
3. **Rust Qdrant RAG MCP Server (`qdrant-rag-mcp/`)** — High-performance standalone RAG server in Rust with embedded `FastEmbed` (BGE-Small). Performs code vectorization, deterministic chunk updating (`point_id`), line number tracking (`line_start`/`line_end`), language detection (`language`), project name (`project_name`), and Git repository tagging (`git_repo`).
4. **CLI Interactive Chat (`cli-chat/`)** — Go-based terminal utility with real-time SSE token streaming.

---

## 📂 Repository Structure

```text
LLM-Mykola/
├── README.md               # Complete Ukrainian documentation
├── README_EN.md            # Complete English documentation
├── .gitignore              # Git exclusions (models, venv, build binaries)
├── run_devstral.sh         # Fast startup script for model server
├── cli-chat/               # 💬 Interactive terminal chat for Devstral
│   └── main.go
├── mcp-server-go/          # 🛠 Go MCP Server (Model Adapter)
│   ├── main.go
│   └── go.mod
└── qdrant-rag-mcp/         # 🦀 Rust Qdrant RAG MCP Server (Vector Search)
    ├── Cargo.toml
    └── src/
        ├── main.rs         # MCP Server entrypoint
        ├── mcp.rs          # MCP JSON-RPC tool dispatchers
        ├── qdrant.rs       # Qdrant client, line-aware text chunker & indexes
        ├── embed.rs        # FastEmbed vector generator (BGE-Small)
        ├── llm.rs          # Devstral LLM HTTP client
        └── config.rs       # Environment configuration parser
```

---

## 🛠 1. Prerequisites & Setup

### Requirements:
- **OS**: macOS (Apple Silicon M1/M2/M3/M4, Unified Memory 16GB+).
- **Go**: 1.21+
- **Rust**: 1.75+ (cargo)
- **Python**: 3.10+
- **Qdrant**: Vector Database (default: `192.168.0.107:6333` or `localhost:6333`).

### Setup Steps:

1. **Create Virtual Environment & Install MLX**:
   ```bash
   python3 -m venv .venv
   source .venv/bin/activate
   pip install --upgrade pip setuptools wheel
   pip install mlx mlx-lm huggingface_hub
   ```

2. **Download Devstral-Small-2-24B Weights**:
   ```bash
   huggingface-cli download mlx-community/Devstral-Small-2-24B-4bit --local-dir ./models/Devstral-Small-2-24B
   ```

3. **Build Binaries**:
   ```bash
   # Build Go MCP Server
   cd mcp-server-go && go build -o local-llm-mcp main.go && cd ..

   # Build Rust Qdrant RAG MCP Server
   cd qdrant-rag-mcp && cargo build --release && cd ..

   # Build CLI Chat
   cd cli-chat && go build -o cli-chat main.go && cd ..
   ```

---

## 🚀 2. Launching the Local Stack

### Step 1: Start Devstral-24B Model Server
Execute the optimized startup script:
```bash
./run_devstral.sh
```
*The server will start at port `8080` with a 4GB Prompt Cache and 4096 tokens limit.*

### Step 2: Configure `mcp_config.json` for IDE / Antigravity
Add both servers to your MCP configuration (e.g. `~/.gemini/config/mcp_config.json`):

```json
{
  "mcpServers": {
    "local-llm": {
      "command": "/Users/Shared/LLM-Mykola/mcp-server-go/local-llm-mcp",
      "env": {
        "LLM_SERVER_URL": "http://127.0.0.1:8080/v1/chat/completions"
      }
    },
    "qdrant-rag": {
      "command": "/Users/Shared/LLM-Mykola/qdrant-rag-mcp/target/release/qdrant-rag-mcp",
      "env": {
        "QDRANT_URL": "http://192.168.0.107:6333",
        "QDRANT_API_KEY": "your-api-key",
        "LLM_URL": "http://127.0.0.1:8080/v1/chat/completions",
        "DEFAULT_COLLECTION": "codebase_knowledge"
      }
    }
  }
}
```

---

## 🧰 3. Features & MCP Tools

### 🦀 Rust Qdrant RAG MCP (`qdrant-rag-mcp`)
Exposes 4 native tools for code vectorization and RAG:
* **`qdrant_index_path`**: Indexes a file or text snippet into Qdrant using deterministic IDs (`hash(file_path:chunk_index)`). Stores `file_path`, `relative_path`, `language`, `line_start`, `line_end`, `project_name`, and `git_repo`. Re-indexing overwrites old chunks without creating duplicates.
* **`qdrant_search`**: Performs semantic vector search in Qdrant collections.
* **`qdrant_rag_ask`**: Executes vector search, constructs context prompts with clickable file anchors (`file://...#L10-L45`), and queries the local Devstral-24B model for grounded answers.
* **`qdrant_list_collections`**: Lists all active Qdrant vector collections.

### 🛠 Go Local LLM MCP (`mcp-server-go`)
* **`ask_local_llm`**: Sends prompts to Devstral-24B with support for `temperature`, `top_p`, `json_mode`, and Native Function Calling.
* **`summarize_local`**: Summarizes large texts using the local model with 0 cloud token usage.

### 💬 CLI Chat (`cli-chat`)
Launch terminal interface for real-time interactive chat with the model:
```bash
cd cli-chat && ./cli-chat
```

---

## 📄 License
This project is released under the [MIT License](LICENSE).
