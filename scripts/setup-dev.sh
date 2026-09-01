#!/usr/bin/env bash
# AI Collab Hub — 一键安装开发依赖
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> 1/6 Frontend (npm)"
cd apps/web && npm install && cd "$ROOT"

echo "==> 2/6 cursor-mcp-bridge (npm)"
cd integrations/cursor-mcp-bridge && npm install && cd "$ROOT"

echo "==> 3/6 Go modules build"
(cd services/mock-hub && go build -o mock-hub.exe . && go test ./...)
(cd services/agent-bridge && go build -o agent-bridge.exe . && go test ./...)
(cd integrations/hub-mcp-go && go build -o hub-mcp.exe .)
if (cd apps/hub && go build -o hub.exe .); then
  echo "    apps/hub build OK"
else
  echo "    [warn] apps/hub build skipped (missing internal/llm — 待补全)"
fi

echo "==> 4/6 Superpowers skills (project-local)"
SUP="$ROOT/.cursor/skills/superpowers"
if [ ! -d "$SUP/using-superpowers" ]; then
  TMP=$(mktemp -d)
  git clone --depth 1 https://github.com/obra/superpowers.git "$TMP"
  mkdir -p "$SUP"
  for s in using-superpowers brainstorming writing-plans executing-plans test-driven-development requesting-code-review receiving-code-review; do
    [ -d "$TMP/skills/$s" ] && cp -r "$TMP/skills/$s" "$SUP/"
  done
  rm -rf "$TMP"
  echo "    Superpowers skills installed to $SUP"
else
  echo "    Superpowers skills already present"
fi

echo "==> 5/6 Python MCP (optional)"
if python3 -c "import mcp" 2>/dev/null; then
  echo "    mcp package OK"
else
  echo "    [optional] pip install mcp requests"
fi

echo "==> 6/6 Structure check"
test -d apps/hub && test -d apps/web && test -d brain/wiki && echo "    directory layout OK"

echo ""
echo "✅ Setup complete. Run: bash scripts/start-dev.sh"
