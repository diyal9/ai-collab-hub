#!/bin/bash
cd /root/aispace/hermesshare
nohup ./agent-bridge/agent-bridge --hub-url ws://127.0.0.1:8087/ws/agent --name local-hermes --type hermes --token agt_hermes_1778853513_edc7 > /tmp/local-agent.log 2>&1 &
echo "Agent started. PID: $!"
echo "Log: /tmp/local-agent.log"
