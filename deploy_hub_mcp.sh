#!/bin/bash
# Deploy Go MCP Server for AI Collab Hub
# Usage: ./deploy_hub_mcp.sh

set -e

MCP_BIN="/root/aispace/hermesshare/hub-mcp/hub-mcp"
CURSOR_CONFIG="$HOME/.cursor/mcp.json"

echo "🔧 Deploying AI Collab Hub MCP Server (Go)..."

# Verify binary
if [ ! -f "$MCP_BIN" ]; then
    echo "❌ MCP binary not found at $MCP_BIN"
    echo "   Run: cd /root/aispace/hermesshare/hub-mcp && go build -o hub-mcp ."
    exit 1
fi

# Get current token
echo "📡 Fetching Hub token..."
TOKEN=$(curl -s -X POST http://localhost:8087/api/login \
  -H "Content-Type: application/json" \
  -d '{"user":"admin","pass":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to get token. Is Hub running on :8087?"
    exit 1
fi

# Create Cursor config
mkdir -p "$HOME/.cursor"

cat > "$CURSOR_CONFIG" << EOF
{
  "mcpServers": {
    "ai-collab-hub": {
      "command": "$MCP_BIN",
      "args": [],
      "env": {
        "HUB_URL": "http://localhost:8087",
        "HUB_TOKEN": "$TOKEN"
      }
    }
  }
}
EOF

echo ""
echo "✅ MCP Server deployed!"
echo ""
echo "📋 Configuration:"
echo "  - Binary: $MCP_BIN"
echo "  - Cursor Config: $CURSOR_CONFIG"
echo ""
echo "🚀 Next steps:"
echo "  1. Restart Cursor"
echo "  2. Open any file in Cursor"
echo "  3. Type @ai-collab-hub or use the MCP tools panel"
echo ""
echo "📦 Available Tools:"
echo "  • list_tasks         - 查看任务列表"
echo "  • get_task           - 查看任务详情"
echo "  • create_task        - 创建新任务"
echo "  • complete_task      - 标记任务完成"
echo "  • list_flows         - 查看流程列表"
echo "  • execute_flow       - 执行流程"
echo "  • get_agent_instances - 查看 Agent 状态"
echo ""
echo "⚠️  Token expires! To refresh, run this script again."
