#!/bin/bash
echo "=== 停止所有 Go 进程 ==="
taskkill //F //IM go.exe //T 2>/dev/null || echo "没有 Go 进程"
taskkill //F //IM record_v2.exe //T 2>/dev/null || echo "没有 record_v2 进程"
taskkill //F //IM record-v2.exe //T 2>/dev/null || echo "没有 record-v2 进程"

echo ""
echo "=== 清理并重新编译 ==="
cd /d/CODE/ClaudeCode/record_V2
go clean -cache
go build -o bin/record_v2.exe ./cmd/server

echo ""
echo "=== 启动服务 ==="
./bin/record_v2.exe
