# План реалізації системи з локальною LLM та Qdrant RAG

Двофазний план побудови екосистеми з локальною моделлю `Devstral-24B` на Mac та векторною БД `Qdrant` на міні-сервері.

---

```
                       [ Твій Mac (Apple Silicon) ]                        [ Сервер у шафі ]
 ┌──────────────────────────────────────────────────────────────────┐      ┌───────────────┐
 │                                                                  │      │               │
 │  ┌───────────────────┐    (HTTP:8080)   ┌─────────────────────┐  │      │  ┌─────────┐  │
 │  │ Antigravity Agent ├─────────────────►│   MLX LM Server     │  │      │  │ Qdrant  │  │
 │  └─────────┬─────────┘                  │ (Devstral-Small-24B)│  │      │  │ (Docker)│  │
 │            │                            └─────────────────────┘  │      │  └────▲────┘  │
 │            │ (MCP Tool Calls)                                    │      │       │       │
 │  ┌─────────▼──────────────────────────────────────────────────┐  │      │       │       │
 │  │ Local MCP Tools (ask_local_llm & qdrant_rag_search)        ┼──┼──────┼───────┘       │
 │  └────────────────────────────────────────────────────────────┘  │ (REST / gRPC:6333)   │
 └──────────────────────────────────────────────────────────────────┘      └───────────────┘
```

---

## 🛑 Фаза 1. Локальна модель MLX & Інтеграція Local-LLM MCP (На Mac)

### **Крок 1.1. Дочекатися завантаження та перевірити права**
- Завершити завантаження ваг моделі у `models/Devstral-Small-2-24B`.
- Забезпечити спільні права читання для всіх користувачів системи:
  ```bash
  chmod -R a+rX /Users/Shared/LLM-Mykola/models
  ```

### **Крок 1.2. Запуск та автозапуск `mlx_lm.server`**
- Запустити сервер на порту `8080`:
  ```bash
  /Users/Shared/LLM-Mykola/.venv/bin/mlx_lm.server \
    --model /Users/Shared/LLM-Mykola/models/Devstral-Small-2-24B \
    --port 8080 \
    --max-tokens 4096
  ```
- *(За бажанням)* Налаштувати `launchd` plist-файл на Mac для автоматичного фонового запуску сервера під час старту системи.

### **Крок 1.3. Створення локального MCP-сервера `local-llm-mcp`**
- Створити легкий Python MCP-сервер (використовуючи `mcp` SDK), який реалізує тули:
  - `ask_local_llm(prompt, system_prompt, max_tokens)` — звернення до `http://localhost:8080/v1/chat/completions`.
  - `summarize_code_local(files_content)` — швидка суммаризація без витрати хмарних токенів.
- Зареєструвати MCP-сервер у конфігу Antigravity.

---

## 📦 Фаза 2. Qdrant RAG (Сервер у шафі + Mac Клієнт)

### **Крок 2.1. Розгортання Qdrant на міні-сервері**
- На міні-сервері підняти Qdrant у Docker (`docker-compose.yml`):
  ```yaml
  version: '3.8'
  services:
    qdrant:
      image: qdrant/qdrant:latest
      restart: always
      ports:
        - "6333:6333" # REST API & Dashboard
        - "6334:6334" # gRPC API
      volumes:
        - ./qdrant_storage:/qdrant/storage
  ```
- Перевірити доступність Web UI Qdrant з Mac: `http://<IP_СЕРВЕРА>:6333/dashboard`.

### **Крок 2.2. Створення Qdrant RAG MCP-сервера на Mac**
- Встановити `qdrant-client` та `fastembed` (надшвидка генерація векторних ембеддінгів):
  ```bash
  /Users/Shared/LLM-Mykola/.venv/bin/pip install qdrant-client fastembed
  ```
- Написати MCP-сервер `qdrant-rag-mcp` із тулами:
  1. `index_directory(path, collection_name)` — розбиває файли коду/документації на чанки, генерує ембеддінги та відправляє у Qdrant на сервер.
  2. `search_knowledge_base(query, collection_name, top_k)` — шукає найбільш релевантні фрагменти коду в Qdrant.
  3. `rag_ask_local(query, collection_name)` — автономна зв'язка: знаходить контекст у Qdrant -> відправляє в `Devstral-24B` -> повертає мені готову відповідь.

### **Крок 2.3. Тестування та перевірка економії**
- Проіндексувати перший проект чи паку документів.
- Зробити кілька тестових семантичних запитів через Antigravity та порівняти точність і швидкість.
