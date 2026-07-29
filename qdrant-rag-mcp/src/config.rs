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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config_defaults() {
        let config = Config::from_env();
        assert!(!config.qdrant_url.is_empty());
        assert!(!config.qdrant_api_key.is_empty());
        assert!(!config.llm_url.is_empty());
        assert_eq!(config.default_collection, "codebase_knowledge");
    }
}
