mod config;
mod embed;
mod llm;
mod qdrant;

use anyhow::Result;
use config::Config;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_writer(std::io::stderr)
        .init();

    let _config = Config::from_env();

    tracing::info!("Starting qdrant-rag-mcp Rust server...");
    Ok(())
}
