#!/bin/bash
# 启动本地 Agent Bridge（连接 Hub WebSocket）
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

nohup ./services/agent-bridge/agent-bridge \
  --hub-url ws://127.0.0.1:8087/ws/agent \
  --name local-hermes \
  --type hermes \
  --token "${AGENT_TOKEN:-agt_hermes_dev}" \
  > /tmp/local-agent.log 2>&1 &

echo "Agent started. PID: $!"
echo "Log: /tmp/local-agent.log"
