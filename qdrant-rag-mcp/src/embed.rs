use anyhow::{Context, Result};
use fastembed::{EmbeddingModel, InitOptions, TextEmbedding};

pub struct EmbedEngine {
    model: TextEmbedding,
}

impl EmbedEngine {
    pub fn new() -> Result<Self> {
        let model = TextEmbedding::try_new(
            InitOptions::new(EmbeddingModel::BGESmallENV15)
                .with_show_download_progress(false),
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
