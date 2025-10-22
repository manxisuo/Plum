#!/bin/bash
# Agent快速编译脚本

set -e

echo "Building Go Agent..."
cd "$(dirname "$0")/agent-go"
go build -o plum-agent

echo "✅ Agent built successfully: agent-go/plum-agent"
echo ""
echo "Run with:"
echo "  AGENT_NODE_ID=nodeA CONTROLLER_BASE=http://127.0.0.1:8080 ./agent-go/plum-agent"

