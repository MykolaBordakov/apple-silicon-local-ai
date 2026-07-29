---
name: qdrant-rag
description: "Use Qdrant vector database (at 192.168.0.107:6333) with FastEmbed and local Devstral-24B for semantic vector search, indexing, collection management, and autonomous RAG."
---

# Qdrant RAG Rust MCP Skill

## Executable Location
`/Users/Shared/LLM-Mykola/qdrant-rag-mcp/target/release/qdrant-rag-mcp`

## Tools Available
- `qdrant_index_path`: Index local file or content into Qdrant collection.
- `qdrant_search`: Semantic vector search in specified collection.
- `qdrant_rag_ask`: End-to-end vector search + local Devstral-24B LLM response.
- `qdrant_list_collections`: List all active vector collections in Qdrant.
