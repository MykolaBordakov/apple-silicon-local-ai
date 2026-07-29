# Rust Qdrant RAG MCP Server (`qdrant-rag-mcp`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a native, high-performance Rust MCP server that integrates an external Qdrant vector database (`192.168.0.107:6333`), in-memory FastEmbed embeddings (`bge-small-en-v1.5`), and the local Devstral-24B LLM (`127.0.0.1:8080`) over stdio JSON-RPC.

**Architecture:** A modular async Rust application using `tokio`, `qdrant-client`, `fastembed-rs`, `serde_json`, and `reqwest`. Logs are routed strictly to `stderr` via `tracing_subscriber` to maintain clean stdio JSON-RPC framing.

**Tech Stack:** Rust (Edition 2021), `tokio`, `qdrant-client` (v1.12+), `fastembed` (v4+), `reqwest`, `serde`, `serde_json`, `tracing`, `anyhow`.

---

### Task 1: Initialize Rust Cargo Project and Dependencies

**Files:**
- Create: `qdrant-rag-mcp/Cargo.toml`
- Create: `qdrant-rag-mcp/src/main.rs`

- [ ] **Step 1: Create Cargo.toml with dependencies**

```toml
[package]
name = "qdrant-rag-mcp"
version = "0.1.0"
edition = "2021"

[dependencies]
tokio = { version = "1.38", features = ["full"] }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
reqwest = { version = "0.12", features = ["json", "stream"] }
qdrant-client = "1.12"
fastembed = "4.0"
anyhow = "1.0"
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter"] }
```

- [ ] **Step 2: Create initial skeleton in main.rs**

```rust
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_writer(std::io::stderr)
        .init();

    tracing::info!("Starting qdrant-rag-mcp Rust server...");
    Ok(())
}
```

- [ ] **Step 3: Verify build**

Run: `cargo check` inside `qdrant-rag-mcp/`
Expected: `Finished check [unoptimized + debuginfo]`

- [ ] **Step 4: Commit**

```bash
git add qdrant-rag-mcp/Cargo.toml qdrant-rag-mcp/src/main.rs
git commit -m "feat: initialize qdrant-rag-mcp Rust project skeleton"
```

---

### Task 2: Environment Configuration Layer

**Files:**
- Create: `qdrant-rag-mcp/src/config.rs`
- Modify: `qdrant-rag-mcp/src/main.rs`

- [ ] **Step 1: Create config.rs module**

```rust
use std::env;

#[derive(Debug, Clone)]
pub struct Config {
    pub qdrant_url: String,
    pub qdrant_api_key: String,
    pub llm_url: String,
    pub default_collection: String,
}

impl Config {
    pub fn from_env() -> Self {
        Self {
            qdrant_url: env::var("QDRANT_URL")
                .unwrap_or_else(|_| "http://192.168.0.107:6333".to_string()),
            qdrant_api_key: env::var("QDRANT_API_KEY")
                .unwrap_or_else(|_| "4a3f5bad98ec768011aa8d8c2734a20f11a3f97899c0a3e0b52c949d5e482a24".to_string()),
            llm_url: env::var("LLM_URL")
                .unwrap_or_else(|_| "http://127.0.0.1:8080/v1/chat/completions".to_string()),
            default_collection: env::var("DEFAULT_COLLECTION")
                .unwrap_or_else(|_| "codebase_knowledge".to_string()),
        }
    }
}
```

- [ ] **Step 2: Test config initialization in main.rs**

Run: `cargo test`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add qdrant-rag-mcp/src/config.rs qdrant-rag-mcp/src/main.rs
git commit -m "feat: add environment configuration loader"
```

---

### Task 3: FastEmbed Embedding Engine Wrapper

**Files:**
- Create: `qdrant-rag-mcp/src/embed.rs`

- [ ] **Step 1: Write embed.rs wrapper around FastEmbed**

```rust
use anyhow::{Context, Result};
use fastembed::{InitOptions, EmbeddingModel, TextEmbedding};

pub struct EmbedEngine {
    model: TextEmbedding,
}

impl EmbedEngine {
    pub fn new() -> Result<Self> {
        let model = TextEmbedding::try_new(InitOptions {
            model_name: EmbeddingModel::BGESmallENV15,
            show_download_progress: false,
            ..Default::default()
        })
        .context("Failed to initialize FastEmbed model")?;

        Ok(Self { model })
    }

    pub fn embed_texts(&self, texts: &[String]) -> Result<Vec<Vec<f32>>> {
        let embeddings = self.model.embed(texts.to_vec(), None)?;
        Ok(embeddings)
    }
}
```

- [ ] **Step 2: Verify build and unit test embedding**

Run: `cargo test`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add qdrant-rag-mcp/src/embed.rs
git commit -m "feat: implement fastembed text embedding engine"
```

---

### Task 4: Qdrant Client API Wrapper & Chunking

**Files:**
- Create: `qdrant-rag-mcp/src/qdrant.rs`

- [ ] **Step 1: Write Qdrant client wrapper**

```rust
use anyhow::{Context, Result};
use qdrant_client::Qdrant;
use qdrant_client::qdrant::{
    CreateCollectionBuilder, Distance, PointStruct, SearchPointsBuilder, VectorParamsBuilder,
};
use serde_json::json;

pub struct QdrantManager {
    client: Qdrant,
}

impl QdrantManager {
    pub async fn new(url: &str, api_key: &str) -> Result<Self> {
        let client = Qdrant::from_url(url)
            .api_key(api_key)
            .build()
            .context("Failed to connect to Qdrant")?;
        Ok(Self { client })
    }

    pub async fn ensure_collection(&self, collection_name: &str, vector_size: u64) -> Result<()> {
        if !self.client.collection_exists(collection_name).await? {
            self.client
                .create_collection(
                    CreateCollectionBuilder::new(collection_name)
                        .vectors_config(VectorParamsBuilder::new(vector_size, Distance::Cosine)),
                )
                .await?;
        }
        Ok(())
    }

    pub async fn upsert_points(&self, collection_name: &str, points: Vec<PointStruct>) -> Result<()> {
        self.client.upsert_points(collection_name, points).await?;
        Ok(())
    }

    pub async fn search(&self, collection_name: &str, vector: Vec<f32>, limit: u64) -> Result<Vec<serde_json::Value>> {
        let response = self
            .client
            .search_points(SearchPointsBuilder::new(collection_name, vector, limit).with_payload(true))
            .await?;

        let mut results = Vec::new();
        for point in response.result {
            results.push(json!({
                "score": point.score,
                "payload": point.payload,
            }));
        }
        Ok(results)
    }

    pub async fn list_collections(&self) -> Result<Vec<String>> {
        let collections = self.client.list_collections().await?;
        Ok(collections.collections.into_iter().map(|c| c.name).collect())
    }
}
```

- [ ] **Step 2: Verify compilation**

Run: `cargo check`
Expected: `Finished check`

- [ ] **Step 3: Commit**

```bash
git add qdrant-rag-mcp/src/qdrant.rs
git commit -m "feat: implement Qdrant client collection management and search"
```

---

### Task 5: Local Devstral-24B HTTP Client Wrapper

**Files:**
- Create: `qdrant-rag-mcp/src/llm.rs`

- [ ] **Step 1: Write llm.rs for Devstral-24B query**

```rust
use anyhow::{Context, Result};
use reqwest::Client;
use serde::{Deserialize, Serialize};

#[derive(Serialize)]
struct ChatMessage {
    role: String,
    content: String,
}

#[derive(Serialize)]
struct ChatRequest {
    model: String,
    messages: Vec<ChatMessage>,
    temperature: f64,
    max_tokens: usize,
}

#[derive(Deserialize)]
struct ChatResponse {
    choices: Vec<Choice>,
}

#[derive(Deserialize)]
struct Choice {
    message: ChatMessageResponse,
}

#[derive(Deserialize)]
struct ChatMessageResponse {
    content: String,
}

pub struct LlmClient {
    client: Client,
    url: String,
}

impl LlmClient {
    pub fn new(url: String) -> Self {
        Self {
            client: Client::new(),
            url,
        }
    }

    pub async fn ask(&self, user_prompt: &str, system_prompt: &str) -> Result<String> {
        let req = ChatRequest {
            model: "./models/Devstral-Small-2-24B".to_string(),
            messages: vec![
                ChatMessage {
                    role: "system".to_string(),
                    content: system_prompt.to_string(),
                },
                ChatMessage {
                    role: "user".to_string(),
                    content: user_prompt.to_string(),
                },
            ],
            temperature: 0.2,
            max_tokens: 2048,
        };

        let res = self
            .client
            .post(&self.url)
            .json(&req)
            .send()
            .await
            .context("Failed to send request to local LLM")?
            .json::<ChatResponse>()
            .await
            .context("Failed to parse local LLM response")?;

        res.choices
            .first()
            .map(|c| c.message.content.clone())
            .ok_or_else(|| anyhow::anyhow!("No choices returned from local LLM"))
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add qdrant-rag-mcp/src/llm.rs
git commit -m "feat: add local LLM client for Devstral-24B"
```

---

### Task 6: MCP Stdio JSON-RPC Loop and Tool Dispatcher

**Files:**
- Create: `qdrant-rag-mcp/src/mcp.rs`
- Modify: `qdrant-rag-mcp/src/main.rs`

- [ ] **Step 1: Write MCP JSON-RPC protocol parser and handlers in mcp.rs**
- [ ] **Step 2: Connect main stdio event loop in main.rs**
- [ ] **Step 3: Test compilation and JSON-RPC initialization via echo pipe**

Run: `echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | cargo run`
Expected: `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05", ...}}`

- [ ] **Step 4: Commit**

```bash
git add qdrant-rag-mcp/src/mcp.rs qdrant-rag-mcp/src/main.rs
git commit -m "feat: implement stdio MCP server loop and tool routing"
```

---

### Task 7: Build Release Binary and Global Antigravity Config

**Files:**
- Modify: `/Users/mykola-nemo/.gemini/config/mcp_config.json`

- [ ] **Step 1: Compile release binary**

Run: `cargo build --release`
Expected: Executable created at `qdrant-rag-mcp/target/release/qdrant-rag-mcp`

- [ ] **Step 2: Update global Antigravity MCP config**

Add `qdrant-rag` to `~/.gemini/config/mcp_config.json`.

- [ ] **Step 3: Commit design and plan state**

```bash
git add docs/superpowers/plans/2026-07-29-qdrant-rag-mcp.md
git commit -m "docs: add implementation plan for Rust Qdrant RAG MCP server"
```
