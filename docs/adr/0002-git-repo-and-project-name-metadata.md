# ADR 0002: Explicit Payload Metadata (git_repo, project_name, relative_path)

## Status
Accepted

## Context
Initial Qdrant point payloads relied strictly on machine-specific absolute file paths (`file_path: "/Users/Shared/LLM-Mykola/src/..."`). When cloning the repository onto another machine or into a different folder, RAG context lost project boundary awareness and remote repository traceability.

## Decision
We enhanced `qdrant-rag-mcp` payload metadata and automatic index generation:
1. **Automatic Git Detection**: If `git_repo` is omitted during indexing, `qdrant-rag-mcp` automatically runs `git remote get-url origin` in the target file directory to detect the remote URL.
2. **Payload Keyword Indexes**: Created payload indexes in Qdrant for `git_repo`, `project_name`, `relative_path`, `language`, `source`, and `file_path`.
3. **Location Headers**: RAG context headers format clickable line anchors with project & git metadata:
   ```markdown
   ### Project: apple-silicon-local-ai | File: file:///Users/Shared/LLM-Mykola/src/mcp.rs#L80-L95 (git: git@github.com:MykolaBordakov/apple-silicon-local-ai.git)
   ```

## Consequences
- Multi-repository isolation within a single Qdrant collection.
- Portability across different machines and folder structures.
- Direct clickable line-level anchors (`file://...#L<start>-L<end>`) in IDEs.
