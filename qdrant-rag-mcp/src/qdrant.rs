use anyhow::{Context, Result};
use qdrant_client::Qdrant;
use qdrant_client::qdrant::{
    CreateCollectionBuilder, Distance, PointStruct, SearchPointsBuilder, UpsertPointsBuilder, VectorParamsBuilder,
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
            .context("Failed to connect to Qdrant server")?;
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
        self.client
            .upsert_points(UpsertPointsBuilder::new(collection_name, points))
            .await?;
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

pub fn chunk_text(text: &str, chunk_size: usize, overlap: usize) -> Vec<String> {
    if text.is_empty() {
        return vec![];
    }
    if text.len() <= chunk_size {
        return vec![text.to_string()];
    }

    let mut chunks = Vec::new();
    let mut start = 0;
    let text_len = text.len();

    while start < text_len {
        let mut end = start + chunk_size;
        if end > text_len {
            end = text_len;
        }

        chunks.push(text[start..end].to_string());
        if end == text_len {
            break;
        }
        start = if start + chunk_size > overlap {
            start + chunk_size - overlap
        } else {
            end
        };
    }

    chunks
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_chunk_text() {
        let sample = "Hello World! This is a test chunking function for Qdrant RAG indexer.";
        let chunks = chunk_text(sample, 20, 5);
        assert!(!chunks.is_empty());
        assert!(chunks[0].len() <= 20);
    }
}
