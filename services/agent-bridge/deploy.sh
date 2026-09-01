#!/bin/bash
# deploy-agent-bridge.sh — 在 Agent 所在机器上运行
# 用法: ./deploy-agent-bridge.sh

set -e

# ═══════════════════════════════════════════
# 配置
# ═══════════════════════════════════════════
HUB_URL="${HUB_URL:-wss://47.107.172.201:8089/ws/agent}"
AGENT_NAME="${AGENT_NAME:-hermes-main}"
AGENT_TYPE="${AGENT_TYPE:-hermes}"
AGENT_TOKEN="${AGENT_TOKEN:-}"
INSTALL_DIR="${INSTALL_DIR:-/opt/agent-bridge}"

echo "======================================"
echo "  Agent Bridge 部署脚本"
echo "======================================"
echo "Hub URL:    $HUB_URL"
echo "Agent Name: $AGENT_NAME"
echo "Agent Type: $AGENT_TYPE"
echo "Install to: $INSTALL_DIR"
echo ""

# 1. 检查依赖
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go"
    echo "   wget https://golang.org/dl/go1.20.linux-amd64.tar.gz"
    echo "   sudo tar -C /usr/local -xzf go1.20.linux-amd64.tar.gz"
    exit 1
fi

# 2. 创建目录
mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

# 3. 编译 (如果有源码)
if [ -f "client.go" ]; then
    echo "📦 编译 agent-bridge..."
    go build -o agent-bridge .
elif [ -f "agent-bridge" ]; then
    echo "✅ agent-bridge 已存在"
else
    echo "❌ 未找到 client.go 或 agent-bridge"
    echo "   请先将 agent-bridge 目录复制到目标机器"
    exit 1
fi

# 4. 创建 systemd 服务
cat > /etc/systemd/system/agent-bridge.service << EOF
[Unit]
Description=AI Collab Hub Agent Bridge
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/agent-bridge --hub-url $HUB_URL --name $AGENT_NAME --type $AGENT_TYPE ${AGENT_TOKEN:+--token $AGENT_TOKEN}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 5. 启动服务
systemctl daemon-reload
systemctl enable agent-bridge
systemctl restart agent-bridge

echo ""
echo "✅ Agent Bridge 已部署并启动"
echo "   状态: systemctl status agent-bridge"
echo "   日志: journalctl -u agent-bridge -f"
echo ""
echo "   连接信息:"
echo "     URL: $HUB_URL"
echo "     Name: $AGENT_NAME"
echo "     Type: $AGENT_TYPE"
