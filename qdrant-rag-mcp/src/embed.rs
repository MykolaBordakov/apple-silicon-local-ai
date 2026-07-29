use anyhow::{Context, Result};
use fastembed::{EmbeddingModel, InitOptions, TextEmbedding};

pub struct EmbedEngine {
    model: TextEmbedding,
}

impl EmbedEngine {
    pub fn new() -> Result<Self> {
        let cache_dir = std::env::var("FASTEMBED_CACHE_DIR")
            .map(std::path::PathBuf::from)
            .unwrap_or_else(|_| {
                let home = std::env::var("HOME").unwrap_or_else(|_| "/tmp".to_string());
                std::path::PathBuf::from(home).join(".cache").join("fastembed")
            });

        if let Err(_) = std::fs::create_dir_all(&cache_dir) {
            let tmp_dir = std::path::PathBuf::from("/tmp/fastembed_cache");
            let _ = std::fs::create_dir_all(&tmp_dir);
        }

        let model = TextEmbedding::try_new(
            InitOptions::new(EmbeddingModel::BGESmallENV15)
                .with_show_download_progress(false)
                .with_cache_dir(cache_dir),
        )
        .context("Failed to initialize FastEmbed BGESmallENV15 model")?;

        Ok(Self { model })
    }

    pub fn embed_texts(&self, texts: &[String]) -> Result<Vec<Vec<f32>>> {
        let embeddings = self.model.embed(texts.to_vec(), None)
            .map_err(|e| anyhow::anyhow!("FastEmbed generation error: {:?}", e))?;
        Ok(embeddings)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_embed_texts() {
        let engine = EmbedEngine::new().expect("Failed to create EmbedEngine");
        let texts = vec!["Hello world from Rust MCP".to_string()];
        let vectors = engine.embed_texts(&texts).expect("Failed to embed text");

        assert_eq!(vectors.len(), 1);
        assert_eq!(vectors[0].len(), 384, "BGESmallENV15 vector length should be 384");
    }
}
