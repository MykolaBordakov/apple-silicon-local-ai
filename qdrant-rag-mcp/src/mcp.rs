use anyhow::Result;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::sync::Arc;
use tokio::sync::Mutex;
use qdrant_client::qdrant::PointStruct;

use crate::config::Config;
use crate::embed::EmbedEngine;
use crate::llm::LlmClient;
use crate::qdrant::{chunk_text_with_lines, QdrantManager};

#[derive(Deserialize, Debug)]
pub struct JSONRPCRequest {
    pub jsonrpc: String,
    pub id: Option<Value>,
    pub method: String,
    pub params: Option<Value>,
}

#[derive(Serialize, Debug)]
pub struct JSONRPCResponse {
    pub jsonrpc: String,
    pub id: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub result: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<RPCError>,
}

#[derive(Serialize, Debug)]
pub struct RPCError {
    pub code: i32,
    pub message: String,
}

pub struct ServerState {
    pub config: Config,
    pub embed: EmbedEngine,
    pub qdrant: Option<QdrantManager>,
    pub llm: LlmClient,
}

impl ServerState {
    pub async fn get_qdrant(&mut self) -> Result<&QdrantManager> {
        if self.qdrant.is_none() {
            tracing::info!("Lazily connecting to Qdrant at {}...", self.config.qdrant_url);
            let mgr = QdrantManager::new(&self.config.qdrant_url, &self.config.qdrant_api_key).await?;
            self.qdrant = Some(mgr);
        }
        Ok(self.qdrant.as_ref().unwrap())
    }
}

pub async fn handle_request(req: JSONRPCRequest, state: Arc<Mutex<ServerState>>) -> Option<JSONRPCResponse> {
    let id = req.id.clone();
    match req.method.as_str() {
        "initialize" => {
            let res = json!({
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "tools": {}
                },
                "serverInfo": {
                    "name": "qdrant-rag-mcp",
                    "version": "0.1.0"
                }
            });
            Some(JSONRPCResponse {
                jsonrpc: "2.0".to_string(),
                id,
                result: Some(res),
                error: None,
            })
        }
        "notifications/initialized" => None,
        "tools/list" => {
            let tools = json!({
                "tools": [
                    {
                        "name": "qdrant_index_path",
                        "description": "Index a local file or text content into Qdrant vector database.",
                        "inputSchema": {
                            "type": "object",
                            "properties": {
                                "path": { "type": "string", "description": "File path or content to index" },
                                "collection_name": { "type": "string", "description": "Optional Qdrant collection name (default: codebase_knowledge)" },
                                "project_name": { "type": "string", "description": "Optional project/repository name" },
                                "git_repo": { "type": "string", "description": "Optional Git remote URL" },
                                "chunk_size": { "type": "integer", "description": "Chunk character length (default 500)" }
                            },
                            "required": ["path"]
                        }
                    },
                    {
                        "name": "qdrant_search",
                        "description": "Perform semantic vector search in Qdrant.",
                        "inputSchema": {
                            "type": "object",
                            "properties": {
                                "query": { "type": "string", "description": "Semantic query text" },
                                "collection_name": { "type": "string", "description": "Optional Qdrant collection name" },
                                "limit": { "type": "integer", "description": "Top-K results count (default 5)" }
                            },
                            "required": ["query"]
                        }
                    },
                    {
                        "name": "qdrant_rag_ask",
                        "description": "Perform RAG query: Search Qdrant vector context and query local Devstral-24B model for answer.",
                        "inputSchema": {
                            "type": "object",
                            "properties": {
                                "query": { "type": "string", "description": "User question" },
                                "collection_name": { "type": "string", "description": "Optional Qdrant collection name" }
                            },
                            "required": ["query"]
                        }
                    },
                    {
                        "name": "qdrant_list_collections",
                        "description": "List all vector collections available in Qdrant.",
                        "inputSchema": {
                            "type": "object",
                            "properties": {}
                        }
                    }
                ]
            });
            Some(JSONRPCResponse {
                jsonrpc: "2.0".to_string(),
                id,
                result: Some(tools),
                error: None,
            })
        }
        "tools/call" => {
            let params = req.params.unwrap_or(json!({}));
            let tool_name = params["name"].as_str().unwrap_or("");
            let arguments = &params["arguments"];

            let res = match call_tool(tool_name, arguments, state).await {
                Ok(content) => JSONRPCResponse {
                    jsonrpc: "2.0".to_string(),
                    id,
                    result: Some(json!({
                        "content": [{ "type": "text", "text": content }]
                    })),
                    error: None,
                },
                Err(err) => JSONRPCResponse {
                    jsonrpc: "2.0".to_string(),
                    id,
                    result: Some(json!({
                        "content": [{ "type": "text", "text": format!("Error: {}", err) }],
                        "isError": true
                    })),
                    error: None,
                },
            };
            Some(res)
        }
        _ => {
            if id.is_some() {
                Some(JSONRPCResponse {
                    jsonrpc: "2.0".to_string(),
                    id,
                    result: None,
                    error: Some(RPCError {
                        code: -32601,
                        message: format!("Method not found: {}", req.method),
                    }),
                })
            } else {
                None
            }
        }
    }
}

async fn call_tool(name: &str, args: &Value, state: Arc<Mutex<ServerState>>) -> Result<String> {
    let mut st = state.lock().await;

    match name {
        "qdrant_list_collections" => {
            let qdrant = st.get_qdrant().await?;
            let collections = qdrant.list_collections().await?;
            Ok(serde_json::to_string_pretty(&collections)?)
        }
        "qdrant_search" => {
            let query = args["query"].as_str().ok_or_else(|| anyhow::anyhow!("'query' is required"))?;
            let collection = args["collection_name"]
                .as_str()
                .unwrap_or(&st.config.default_collection)
                .to_string();
            let limit = args["limit"].as_u64().unwrap_or(5);

            let vectors = st.embed.embed_texts(&[query.to_string()])?;
            let vector = vectors[0].clone();

            let qdrant = st.get_qdrant().await?;
            qdrant.ensure_collection(&collection, 384).await?;
            let results = qdrant.search(&collection, vector, limit).await?;
            Ok(serde_json::to_string_pretty(&results)?)
        }
        "qdrant_index_path" => {
            let path_or_text = args["path"].as_str().ok_or_else(|| anyhow::anyhow!("'path' is required"))?;
            let collection = args["collection_name"]
                .as_str()
                .unwrap_or(&st.config.default_collection)
                .to_string();
            let chunk_size = args["chunk_size"].as_u64().unwrap_or(500) as usize;

            let content = if std::path::Path::new(path_or_text).exists() {
                tokio::fs::read_to_string(path_or_text).await?
            } else {
                path_or_text.to_string()
            };

            let project_name = args["project_name"]
                .as_str()
                .map(|s| s.to_string())
                .unwrap_or_else(|| {
                    let path = std::path::Path::new(path_or_text);
                    if path.is_absolute() {
                        path.components()
                            .rev()
                            .nth(1)
                            .and_then(|c| c.as_os_str().to_str())
                            .unwrap_or("unknown_project")
                            .to_string()
                    } else {
                        "unknown_project".to_string()
                    }
                });

            let git_repo = args["git_repo"]
                .as_str()
                .unwrap_or("")
                .to_string();

            let relative_path = {
                let path = std::path::Path::new(path_or_text);
                if path.is_absolute() {
                    let components: Vec<_> = path.components().map(|c| c.as_os_str().to_str().unwrap_or("")).collect();
                    if components.len() >= 2 {
                        components[components.len()-2..].join("/")
                    } else {
                        path_or_text.to_string()
                    }
                } else {
                    path_or_text.to_string()
                }
            };

            let ext = std::path::Path::new(path_or_text)
                .extension()
                .and_then(|e| e.to_str())
                .unwrap_or("txt");

            let chunks = chunk_text_with_lines(&content, chunk_size, 100);
            if chunks.is_empty() {
                return Ok("No text content to index.".to_string());
            }

            let texts: Vec<String> = chunks.iter().map(|c| c.content.clone()).collect();
            let vectors = st.embed.embed_texts(&texts)?;

            let mut points = Vec::new();
            for (idx, (chunk, vec)) in chunks.into_iter().zip(vectors.into_iter()).enumerate() {
                use std::hash::{Hash, Hasher};
                let mut hasher = std::collections::hash_map::DefaultHasher::new();
                format!("{}:{}", path_or_text, idx).hash(&mut hasher);
                let point_id = hasher.finish();

                let mut payload = std::collections::HashMap::new();
                payload.insert("content".to_string(), json!(chunk.content));
                payload.insert("source".to_string(), json!(path_or_text));
                payload.insert("file_path".to_string(), json!(path_or_text));
                payload.insert("relative_path".to_string(), json!(relative_path));
                payload.insert("language".to_string(), json!(ext));
                payload.insert("project_name".to_string(), json!(project_name));
                payload.insert("git_repo".to_string(), json!(git_repo));
                payload.insert("line_start".to_string(), json!(chunk.line_start));
                payload.insert("line_end".to_string(), json!(chunk.line_end));

                points.push(PointStruct::new(point_id, vec, payload));
            }

            let qdrant = st.get_qdrant().await?;
            qdrant.ensure_collection(&collection, 384).await?;
            qdrant.upsert_points(&collection, points).await?;
            Ok(format!("Indexed content into collection '{}' successfully.", collection))
        }
        "qdrant_rag_ask" => {
            let query = args["query"].as_str().ok_or_else(|| anyhow::anyhow!("'query' is required"))?;
            let collection = args["collection_name"]
                .as_str()
                .unwrap_or(&st.config.default_collection)
                .to_string();

            let vectors = st.embed.embed_texts(&[query.to_string()])?;
            let vector = vectors[0].clone();

            let qdrant = st.get_qdrant().await?;
            qdrant.ensure_collection(&collection, 384).await?;
            let results = qdrant.search(&collection, vector, 5).await?;

            let mut context_str = String::new();
            for res in &results {
                if let Some(payload) = res["payload"].as_object() {
                    let content_text = payload.get("content")
                        .or_else(|| payload.get("text"))
                        .and_then(|v| v.as_str())
                        .unwrap_or("");

                    let file_path = payload.get("file_path")
                        .or_else(|| payload.get("source"))
                        .and_then(|v| v.as_str())
                        .unwrap_or("unknown");

                    let project = payload.get("project_name").and_then(|v| v.as_str()).unwrap_or("");
                    let git = payload.get("git_repo").and_then(|v| v.as_str()).unwrap_or("");
                    let l_start = payload.get("line_start").and_then(|v| v.as_u64());
                    let l_end = payload.get("line_end").and_then(|v| v.as_u64());

                    let mut header = format!("### Project: {} | File: file://{}", project, file_path);
                    if let (Some(s), Some(e)) = (l_start, l_end) {
                        header.push_str(&format!("#L{}-L{}", s, e));
                    }
                    if !git.is_empty() {
                        header.push_str(&format!(" (git: {})", git));
                    }
                    header.push('\n');
                    context_str.push_str(&header);
                    context_str.push_str(content_text);
                    context_str.push_str("\n---\n");
                }
            }

            let system_prompt = format!(
                "You are an expert AI software assistant. Use the following context retrieved from vector search to answer the user's question accurately.\n\nContext:\n{}",
                context_str
            );

            let answer = st.llm.ask(query, &system_prompt).await?;
            Ok(answer)
        }
        _ => Err(anyhow::anyhow!("Unknown tool: {}", name)),
    }
}
