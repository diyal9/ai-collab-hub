#!/bin/bash
# Deploy Python MCP Server for AI Collab Hub
set -e

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
MCP_DIR="$ROOT/integrations/hub-mcp-py"
MCP_SCRIPT="$MCP_DIR/hub_mcp_server.py"
CURSOR_CONFIG="$HOME/.cursor/mcp.json"
HUB_PORT="${HUB_PORT:-8087}"

echo "🔧 Deploying AI Collab Hub MCP Server (Python)..."

if [ ! -f "$MCP_SCRIPT" ]; then
    echo "❌ Not found: $MCP_SCRIPT"
    exit 1
fi

TOKEN=$(curl -s -X POST "http://localhost:$HUB_PORT/ai-collab-hub/api/login" \
  -H "Content-Type: application/json" \
  -d '{"user":"admin","pass":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

mkdir -p "$HOME/.cursor"

cat > "$CURSOR_CONFIG" << EOF
{
  "mcpServers": {
    "ai-collab-hub": {
      "command": "python3",
      "args": ["$MCP_SCRIPT"],
      "env": {
        "HUB_URL": "http://localhost:$HUB_PORT",
        "HUB_TOKEN": "$TOKEN"
      }
    }
  }
}
EOF

echo "✅ Python MCP deployed → $CURSOR_CONFIG"
