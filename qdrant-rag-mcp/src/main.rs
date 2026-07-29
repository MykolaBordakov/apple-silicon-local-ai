use anyhow::Result;
use std::sync::Arc;
use tokio::io::{AsyncBufReadExt, BufReader};
use tokio::sync::Mutex;

mod config;
mod embed;
mod llm;
mod mcp;
mod qdrant;

use config::Config;
use embed::EmbedEngine;
use llm::LlmClient;
use mcp::{handle_request, JSONRPCRequest, ServerState};

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_writer(std::io::stderr)
        .init();

    tracing::info!("Starting qdrant-rag-mcp Rust server v0.1.0...");

    let cfg = Config::from_env();
    tracing::info!("Configured Qdrant target at {}", cfg.qdrant_url);

    let embed = EmbedEngine::new()?;
    let llm = LlmClient::new(cfg.llm_url.clone());

    let state = Arc::new(Mutex::new(ServerState {
        config: cfg,
        embed,
        qdrant: None,
        llm,
    }));

    let stdin = tokio::io::stdin();
    let mut reader = BufReader::new(stdin).lines();

    while let Ok(Some(line)) = reader.next_line().await {
        let trimmed = line.trim();
        if trimmed.is_empty() {
            continue;
        }

        if let Ok(req) = serde_json::from_str::<JSONRPCRequest>(trimmed) {
            if let Some(resp) = handle_request(req, state.clone()).await {
                if let Ok(json_out) = serde_json::to_string(&resp) {
                    println!("{}", json_out);
                }
            }
        }
    }

    Ok(())
}
