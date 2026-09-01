#!/bin/bash
# Deploy Go MCP Server for AI Collab Hub
set -e

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
MCP_BIN="$ROOT/integrations/hub-mcp-go/hub-mcp"
CURSOR_CONFIG="$HOME/.cursor/mcp.json"
HUB_PORT="${HUB_PORT:-8087}"

echo "🔧 Deploying AI Collab Hub MCP Server (Go)..."

if [ ! -f "$MCP_BIN" ]; then
    echo "❌ MCP binary not found at $MCP_BIN"
    echo "   Run: cd $ROOT/integrations/hub-mcp-go && go build -o hub-mcp ."
    exit 1
fi

echo "📡 Fetching Hub token..."
TOKEN=$(curl -s -X POST "http://localhost:$HUB_PORT/ai-collab-hub/api/login" \
  -H "Content-Type: application/json" \
  -d '{"user":"admin","pass":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to get token. Is Hub running on :$HUB_PORT?"
    exit 1
fi

mkdir -p "$HOME/.cursor"

cat > "$CURSOR_CONFIG" << EOF
{
  "mcpServers": {
    "ai-collab-hub": {
      "command": "$MCP_BIN",
      "args": [],
      "env": {
        "HUB_URL": "http://localhost:$HUB_PORT",
        "HUB_TOKEN": "$TOKEN"
      }
    }
  }
}
EOF

echo "✅ MCP Server deployed → $CURSOR_CONFIG"
