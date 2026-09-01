#!/bin/bash

# 获取脚本所在目录
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

BINARY="$DIR/agent-bridge-linux-amd64"
CONFIG="$DIR/config.yaml"

# 检查二进制文件
if [ ! -f "$BINARY" ]; then
    echo "Error: Binary not found at $BINARY"
    exit 1
fi

# 检查配置文件
if [ ! -f "$CONFIG" ]; then
    echo "Error: Config not found at $CONFIG"
    exit 1
fi

echo "Starting Agent Bridge..."
echo "Binary: $BINARY"
echo "Config: $CONFIG"

exec "$BINARY" -config "$CONFIG"
