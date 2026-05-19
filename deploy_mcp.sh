#!/bin/bash
# Deploy MCP Server for AI Collab Hub
# Usage: ./deploy_mcp.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MCP_DIR="/root/aispace/hermesshare/mcp"

echo "🔧 Deploying AI Collab Hub MCP Server..."

# Create directory
mkdir -p "$MCP_DIR"

# Copy MCP server
cp "$SCRIPT_DIR/hub_mcp_server.py" "$MCP_DIR/"

# Install dependencies if needed
if ! python3 -c "import mcp" 2>/dev/null; then
    echo "📦 Installing MCP package..."
    pip3 install mcp requests
fi

# Generate Cursor config
CURSOR_CONFIG_DIR="$HOME/.cursor"
mkdir -p "$CURSOR_CONFIG_DIR"

# Get current token
TOKEN=$(curl -s -X POST http://localhost:8087/api/login \
  -H "Content-Type: application/json" \
  -d '{"user":"admin","pass":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to get token. Is Hub running on :8087?"
    exit 1
fi

# Create Cursor MCP config
cat > "$CURSOR_CONFIG_DIR/mcp.json" << EOF
{
  "mcpServers": {
    "ai-collab-hub": {
      "command": "python3",
      "args": ["$MCP_DIR/hub_mcp_server.py"],
      "env": {
        "HUB_URL": "http://localhost:8087",
        "HUB_TOKEN": "$TOKEN"
      }
    }
  }
}
EOF

echo "✅ MCP Server deployed!"
echo ""
echo "📋 Configuration:"
echo "  - MCP Server: $MCP_DIR/hub_mcp_server.py"
echo "  - Cursor Config: $CURSOR_CONFIG_DIR/mcp.json"
echo ""
echo "🚀 Next steps:"
echo "  1. Restart Cursor"
echo "  2. In Cursor, type @hub to interact with AI Collab Hub"
echo "  3. Available commands:"
echo "     - @hub list_tasks"
echo "     - @hub create_task"
echo "     - @hub get_task <task_id>"
echo "     - @hub complete_task <task_id> <output>"
echo "     - @hub list_flows"
echo "     - @hub execute_flow <flow_id>"
echo ""
echo "⚠️  Token expires! To refresh, run:"
echo "   $SCRIPT_DIR/deploy_mcp.sh"
