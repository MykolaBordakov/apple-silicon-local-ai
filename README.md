# LLM Mykola & Local MCP Stack 🚀

> **Високоефективний автономний локальний AI-стек для Apple Silicon (M1-M4).**

Цей репозиторій містить повну екосистему для роботи з локальною великою мовною моделлю `Devstral-Small-2-24B`, швидкісними MCP-серверами (Model Context Protocol) на Go та Rust, а також інтегрованою векторною базою даних **Qdrant** для автономного Codebase RAG.

---

## 🏛 Архітектура екосистеми

```
                               ┌───────────────────────────┐
                               │     AI Client / Agent     │
                               │ (Antigravity / IDE / CLI) │
                               └─────────────┬─────────────┘
                                             │ (MCP Protocol / JSON-RPC over stdio)
                      ┌──────────────────────┴──────────────────────┐
                      ▼                                             ▼
        ┌───────────────────────────┐                 ┌───────────────────────────┐
        │     mcp-server-go (Go)    │                 │   qdrant-rag-mcp (Rust)   │
        │  Local LLM Proxy Server   │                 │   Vector RAG MCP Server   │
        └─────────────┬─────────────┘                 └─────────────┬─────────────┘
                      │ (HTTP API / v1/completions)                 │ (FastEmbed BGE-Small)
                      ▼                                             ▼
        ┌───────────────────────────┐                 ┌───────────────────────────┐
        │   mlx_lm.server (Python)  │                 │    Qdrant Vector DB      │
        │ Devstral-Small-2-24B 4-bit│                 │   (192.168.0.107:6333)    │
        └───────────────────────────┘                 └───────────────────────────┘
```

Екосистема складається з чотирьох основних модулів:

1. **Devstral-Small-2-24B MLX Server** — локальний інференс 24B моделі з 4-bit квантизацією через бібліотеку MLX, оптимізований під Metal & Unified Memory.
2. **Go MCP Server (`mcp-server-go/`)** — протокольний адаптер на Go, що зв'язує AI-агентів із локальною моделлю (підтримує SSE-стрімінг, JSON-Mode, Health-check).
3. **Rust Qdrant RAG MCP Server (`qdrant-rag-mcp/`)** — автономний RAG-сервер на Rust з вбудованим `FastEmbed` (BGE-Small). Робить векторний пошук коду, детерміноване оновлення чанків (`point_id`), відстежує номери рядків (`line_start`/`line_end`), мову (`language`), назву проєкту (`project_name`) та Git-репозиторій (`git_repo`).
4. **CLI Interactive Chat (`cli-chat/`)** — термінальний чат на Go з SSE-стрімінгом токенів у реальному часі.

---

## 📂 Структура репозиторію

```text
LLM-Mykola/
├── README.md               # Повна документація українською мовою
├── README_EN.md            # English documentation
├── .gitignore              # Налаштування виключень (моделі, venv, бінарники)
├── run_devstral.sh         # Швидкий скрипт запуску локального сервера моделі
├── cli-chat/               # 💬 Консольний чат для спілкування з Devstral
│   └── main.go
├── mcp-server-go/          # 🛠 Go MCP сервер (адаптер моделі)
│   ├── main.go
│   └── go.mod
└── qdrant-rag-mcp/         # 🦀 Rust Qdrant RAG MCP сервер (векторний пошук)
    ├── Cargo.toml
    └── src/
        ├── main.rs         # Точка входу MCP-сервера
        ├── mcp.rs          # Обробник інструментів MCP JSON-RPC
        ├── qdrant.rs       # Клієнт Qdrant, нарізка коду (chunking) та індекси
        ├── embed.rs        # Генератор векторів FastEmbed (BGE-Small)
        ├── llm.rs          # Клієнт зв'язку з Devstral LLM
        └── config.rs       # Зчитування конфігурацій середовища
```

---

## 🛠 1. Системні вимоги та Встановлення

### Вимоги:
- **ОС**: macOS (Apple Silicon M1/M2/M3/M4, Unified Memory 16GB+).
- **Go**: 1.21+
- **Rust**: 1.75+ (cargo)
- **Python**: 3.10+
- **Qdrant**: векторний сервер (за замовчуванням `192.168.0.107:6333` або `localhost:6333`).

### Кроки з налаштування:

1. **Створення virtualenv та встановлення MLX**:
   ```bash
   python3 -m venv .venv
   source .venv/bin/activate
   pip install --upgrade pip setuptools wheel
   pip install mlx mlx-lm huggingface_hub
   ```

2. **Завантаження ваг моделі Devstral-Small-2-24B**:
   ```bash
   huggingface-cli download mlx-community/Devstral-Small-2-24B-4bit --local-dir ./models/Devstral-Small-2-24B
   ```

3. **Збірка бінарників**:
   ```bash
   # Збірка Go MCP Сервера
   cd mcp-server-go && go build -o local-llm-mcp main.go && cd ..

   # Збірка Rust Qdrant RAG MCP Сервера
   cd qdrant-rag-mcp && cargo build --release && cd ..

   # Збірка CLI-чату
   cd cli-chat && go build -o cli-chat main.go && cd ..
   ```

---

## 🚀 2. Запуск локального стеку

### Крок 1: Запуск сервера моделі Devstral-24B
Запустіть скрипт швдкодії:
```bash
./run_devstral.sh
```
*Сервер підніметься на порту `8080` з Prompt Cache 4GB та лімітом 4096 токенів.*

### Крок 2: Налаштування `mcp_config.json` для IDE / Antigravity
Додайте обидва сервери у свій конфігуратор MCP (наприклад, `~/.gemini/config/mcp_config.json`):

```json
{
  "mcpServers": {
    "local-llm": {
      "command": "/Users/Shared/LLM-Mykola/mcp-server-go/local-llm-mcp",
      "env": {
        "LLM_SERVER_URL": "http://127.0.0.1:8080/v1/chat/completions"
      }
    },
    "qdrant-rag": {
      "command": "/Users/Shared/LLM-Mykola/qdrant-rag-mcp/target/release/qdrant-rag-mcp",
      "env": {
        "QDRANT_URL": "http://192.168.0.107:6333",
        "QDRANT_API_KEY": "your-api-key",
        "LLM_URL": "http://127.0.0.1:8080/v1/chat/completions",
        "DEFAULT_COLLECTION": "codebase_knowledge"
      }
    }
  }
}
```

---

## 🧰 3. Можливості та Інструменти MCP

### 🦀 Rust Qdrant RAG MCP (`qdrant-rag-mcp`)
Сервер надає 4 нативні інструменти для векторизації та RAG:
* **`qdrant_index_path`**: індексує файл чи текст у Qdrant з розрахунком детермінованих ID (`hash(file_path:chunk_index)`). Записує `file_path`, `relative_path`, `language`, `line_start`, `line_end`, `project_name` та `git_repo`. При повторному запуску старі чанки переписуються без створення дублікатів.
* **`qdrant_search`**: виконує векторний семантичний пошук у Qdrant.
* **`qdrant_rag_ask`**: робить векторний пошук, будує промпт з клікабельними лінками на файли (`file://...#L10-L45`) та генерує відповідь через локальний Devstral-24B.
* **`qdrant_list_collections`**: показує список усіх векторних колекцій Qdrant.

### 🛠 Go Local LLM MCP (`mcp-server-go`)
* **`ask_local_llm`**: відправляє промпт до Devstral-24B з підтримкою параметрів `temperature`, `top_p`, `json_mode` та передачею інструментів (Function Calling).
* **`summarize_local`**: самаризує великі тексти локальною моделлю без витрати хмарних токенів.

### 💬 CLI Chat (`cli-chat`)
Запустіть термінальний чат для прямого спілкування з моделлю у реальному часі:
```bash
cd cli-chat && ./cli-chat
```

---

## 📄 Ліцензія
Проєкт розповсюджується під ліцензією [MIT](LICENSE).
