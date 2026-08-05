# ADR 0001: Qdrant Deterministic Point IDs for Duplicate-Free Upserts

## Status
Accepted

## Context
Previously, `qdrant-rag-mcp` generated point IDs based on a microsecond UNIX timestamp (`as_micros() as u64 + idx`). When a file was re-indexed multiple times, Qdrant assigned new unique point IDs to each chunk, resulting in duplicated stale chunks and payload noise in vector search results.

## Decision
We implemented deterministic point ID hashing using Rust's `std::collections::hash_map::DefaultHasher`:
```rust
let mut hasher = std::collections::hash_map::DefaultHasher::new();
format!("{}:{}", path_or_text, idx).hash(&mut hasher);
let point_id = hasher.finish();
```

## Consequences
- **Upsert Overwriting**: Re-indexing an existing file now generates the exact same point IDs, causing Qdrant to overwrite existing points and payloads.
- **Zero Duplicates**: Total collection point count remains stable when re-running indexers.
- **Clean Vector Search**: Prevents stale code snippets from appearing alongside fresh code snippets in RAG results.
