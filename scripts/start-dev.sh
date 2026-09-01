#!/usr/bin/env bash
# 启动本地开发：mock-hub + frontend dev server
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MOCK_PORT="${MOCK_HUB_PORT:-8085}"
FE_PORT="${VITE_PORT:-5173}"

cleanup() {
  [ -n "${MOCK_PID:-}" ] && kill "$MOCK_PID" 2>/dev/null || true
  [ -n "${FE_PID:-}" ] && kill "$FE_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo "==> building mock-hub"
(cd services/mock-hub && go build -o mock-hub.exe .)
MOCK_BIN="$ROOT/services/mock-hub/mock-hub.exe"
[ -f "$MOCK_BIN" ] || MOCK_BIN="$ROOT/services/mock-hub/mock-hub"

echo "==> mock-hub :$MOCK_PORT"
MOCK_HUB_PORT="$MOCK_PORT" "$MOCK_BIN" &
MOCK_PID=$!
sleep 1

if ! curl -sf "http://localhost:$MOCK_PORT/health" >/dev/null; then
  echo "mock-hub failed to start"
  exit 1
fi
echo "    health OK"

echo "==> frontend dev :$FE_PORT"
cd apps/web
npm run dev -- --host 127.0.0.1 --port "$FE_PORT" &
FE_PID=$!
cd "$ROOT"

echo ""
echo "✅ Dev stack running"
echo "   Mock API:  http://localhost:$MOCK_PORT/health"
echo "   Frontend:  http://127.0.0.1:$FE_PORT/ai-collab-hub/"
echo "   Login:     admin / admin123 (mock accepts any)"
echo "   Real Hub:  set VITE_HUB_PROXY=http://localhost:8087 before npm run dev"
echo ""
echo "Press Ctrl+C to stop"

wait
