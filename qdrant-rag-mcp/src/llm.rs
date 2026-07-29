use anyhow::{Context, Result};
use reqwest::Client;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
struct ChatMessage {
    role: String,
    content: String,
}

#[derive(Serialize, Debug)]
struct ChatRequest {
    model: String,
    messages: Vec<ChatMessage>,
    temperature: f64,
    max_tokens: usize,
}

#[derive(Deserialize, Debug)]
struct ChatResponse {
    choices: Vec<Choice>,
}

#[derive(Deserialize, Debug)]
struct Choice {
    message: ChatMessageResponse,
}

#[derive(Deserialize, Debug)]
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_chat_request_serialization() {
        let req = ChatRequest {
            model: "test_model".to_string(),
            messages: vec![ChatMessage {
                role: "user".to_string(),
                content: "hello".to_string(),
            }],
            temperature: 0.2,
            max_tokens: 100,
        };
        let json_str = serde_json::to_string(&req).expect("Failed to serialize ChatRequest");
        assert!(json_str.contains("test_model"));
        assert!(json_str.contains("hello"));
    }
}
