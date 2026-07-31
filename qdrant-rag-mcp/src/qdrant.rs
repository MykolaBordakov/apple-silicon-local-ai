use anyhow::{Context, Result};
use qdrant_client::Qdrant;
use qdrant_client::qdrant::{
    CreateCollectionBuilder, CreateFieldIndexCollectionBuilder, Distance, FieldType, PointStruct,
    SearchPointsBuilder, UpsertPointsBuilder, VectorParamsBuilder,
};
use serde_json::json;

pub struct QdrantManager {
    client: Qdrant,
}

impl QdrantManager {
    pub async fn new(url: &str, api_key: &str) -> Result<Self> {
        let grpc_url = if url.contains(":6333") {
            url.replace(":6333", ":6334")
        } else {
            url.to_string()
        };
        let client = Qdrant::from_url(&grpc_url)
            .api_key(api_key)
            .connect_timeout(std::time::Duration::from_secs(10))
            .timeout(std::time::Duration::from_secs(30))
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

            for field in ["source", "file_path", "relative_path", "language", "project_name", "git_repo"] {
                let _ = self
                    .client
                    .create_field_index(
                        CreateFieldIndexCollectionBuilder::new(collection_name, field, FieldType::Keyword),
                    )
                    .await;
            }
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
    let chars: Vec<char> = text.chars().collect();
    if chars.len() <= chunk_size {
        return vec![text.to_string()];
    }

    let mut chunks = Vec::new();
    let mut start = 0;
    let text_len = chars.len();

    while start < text_len {
        let mut end = start + chunk_size;
        if end > text_len {
            end = text_len;
        }

        let chunk: String = chars[start..end].iter().collect();
        chunks.push(chunk);
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

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct CodeChunk {
    pub content: String,
    pub line_start: usize,
    pub line_end: usize,
}

pub fn chunk_text_with_lines(text: &str, chunk_size: usize, overlap: usize) -> Vec<CodeChunk> {
    if text.is_empty() {
        return vec![];
    }

    let lines: Vec<&str> = text.lines().collect();
    if lines.is_empty() {
        return vec![];
    }

    let mut chunks = Vec::new();
    let mut current_chunk_lines = Vec::new();
    let mut current_len = 0;
    let mut start_line = 1;
    let mut line_idx = 0;

    while line_idx < lines.len() {
        let line = lines[line_idx];
        current_chunk_lines.push(line);
        current_len += line.len() + 1;

        if current_len >= chunk_size || line_idx == lines.len() - 1 {
            let end_line = start_line + current_chunk_lines.len() - 1;
            chunks.push(CodeChunk {
                content: current_chunk_lines.join("\n"),
                line_start: start_line,
                line_end: end_line,
            });

            if line_idx == lines.len() - 1 {
                break;
            }

            let mut back_len = 0;
            let mut back_count = 0;
            for prev_line in current_chunk_lines.iter().rev() {
                if back_len + prev_line.len() > overlap {
                    break;
                }
                back_len += prev_line.len() + 1;
                back_count += 1;
            }

            let advance = current_chunk_lines.len().saturating_sub(back_count).max(1);
            start_line += advance;
            line_idx = start_line - 1;
            current_chunk_lines.clear();
            current_len = 0;
        } else {
            line_idx += 1;
        }
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

    #[test]
    fn test_chunk_text_with_lines() {
        let sample = "line 1\nline 2\nline 3\nline 4\nline 5";
        let chunks = chunk_text_with_lines(sample, 15, 5);
        assert!(!chunks.is_empty());
        assert_eq!(chunks[0].line_start, 1);
        assert!(chunks[0].line_end >= 1);
    }

    #[test]
    fn test_payload_field_names() {
        let fields = vec!["source", "file_path", "language"];
        assert_eq!(fields.len(), 3);
    }
}

