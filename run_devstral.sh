#!/bin/bash
# Script to start mlx_lm.server with Devstral-Small-2-24B and optimal Apple Silicon settings

MODEL_PATH="./models/Devstral-Small-2-24B"
MLX_SERVER="./.venv/bin/mlx_lm.server"
PORT=8080

echo "🚀 Starting Devstral-Small-2-24B local LLM server on port ${PORT}..."
echo "⚙️ Settings: --prompt-cache-size 10, --prompt-cache-bytes 4GB, --max-tokens 4096"

exec ${MLX_SERVER} \
  --model "${MODEL_PATH}" \
  --port ${PORT} \
  --max-tokens 4096 \
  --prompt-cache-size 10 \
  --prompt-cache-bytes 4GB \
  --chat-template-args '{"fix_mistral_regex": true}'
