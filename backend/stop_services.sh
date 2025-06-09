#!/bin/bash

# AI Chatbot Backend 停止服务脚本

RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
NC="\033[0m"

echo -e "${BLUE}🛑 停止 AI 聊天机器人服务...${NC}"

# 停止所有相关进程
echo -e "${YELLOW}🔄 停止 Go 进程...${NC}"
pkill -f "go run.*AIchat.go" 2>/dev/null
pkill -f "go run.*ApiGateway.go" 2>/dev/null

# 停止编译后的进程
echo -e "${YELLOW}🔄 停止编译进程...${NC}"
pkill -f "AIchat" 2>/dev/null
pkill -f "ApiGateway" 2>/dev/null

# 按端口停止进程
echo -e "${YELLOW}🔄 按端口停止进程...${NC}"
for port in 8080 5001; do
    PID=$(lsof -ti:$port 2>/dev/null)
    if [ ! -z "$PID" ]; then
        echo "停止端口 $port 上的进程 (PID: $PID)"
        kill $PID 2>/dev/null
    fi
done

sleep 2

# 强制停止
echo -e "${YELLOW}🔄 强制停止残留进程...${NC}"
for port in 8080 5001; do
    PID=$(lsof -ti:$port 2>/dev/null)
    if [ ! -z "$PID" ]; then
        echo "强制停止端口 $port 上的进程 (PID: $PID)"
        kill -9 $PID 2>/dev/null
    fi
done

echo -e "${GREEN}✅ 所有服务已停止${NC}"

# 检查状态
echo ""
echo -e "${BLUE}🔍 检查服务状态...${NC}"
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${RED}⚠️  端口 8080 仍有进程在运行${NC}"
else
    echo -e "${GREEN}✅ 端口 8080 已释放${NC}"
fi

if lsof -Pi :5001 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${RED}⚠️  端口 5001 仍有进程在运行${NC}"
else
    echo -e "${GREEN}✅ 端口 5001 已释放${NC}"
fi
