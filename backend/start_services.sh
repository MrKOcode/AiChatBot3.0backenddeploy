#!/bin/bash

# AI Chatbot Backend 完整服务启动脚本（包含ChatHistory）

set -e

RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
NC="\033[0m"

echo -e "${BLUE}🚀 AI 聊天机器人后端服务启动器 (完整版)${NC}"
echo "========================================="

# 检查端口占用并停止服务
echo -e "${BLUE}🔍 检查端口占用...${NC}"
for port in 8080 5001 5002 5004; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        PID=$(lsof -ti:$port)
        echo -e "${YELLOW}⚠️  端口 $port 被进程 $PID 占用，正在终止...${NC}"
        kill $PID 2>/dev/null || kill -9 $PID 2>/dev/null
        sleep 1
    fi
done

# 创建日志目录
mkdir -p logs

echo -e "${BLUE}🎯 启动所有服务...${NC}"

# 启动 Auth 微服务
echo -e "${GREEN}🔐 启动 Auth 微服务 (端口 5002)...${NC}"
cd components/Auth
nohup go run auth.go > ../../logs/auth.log 2>&1 &
AUTH_PID=$!
echo "Auth PID: $AUTH_PID"
cd ../..
sleep 2

# 启动 ChatHistory 微服务
echo -e "${GREEN}💬 启动 ChatHistory 微服务 (端口 5004)...${NC}"
cd components/ChatHistory
nohup go run chat_history.go > ../../logs/chathistory.log 2>&1 &
CHATHISTORY_PID=$!
echo "ChatHistory PID: $CHATHISTORY_PID"
cd ../..
sleep 2

# 启动 AI Chat 微服务
echo -e "${GREEN}🔧 启动 AI Chat 微服务 (端口 5001)...${NC}"
cd components/AIChat
nohup go run AIchat.go > ../../logs/aichat.log 2>&1 &
AICHAT_PID=$!
echo "AI Chat PID: $AICHAT_PID"
cd ../..
sleep 2

# 启动 API Gateway
echo -e "${GREEN}🌐 启动 API Gateway (端口 8080)...${NC}"
nohup go run ApiGateway/ApiGateway.go > logs/gateway.log 2>&1 &
GATEWAY_PID=$!
echo "API Gateway PID: $GATEWAY_PID"
sleep 3

# 检查服务状态
echo -e "${BLUE}🔍 检查服务状态...${NC}"
SUCCESS=true

if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✅ API Gateway (端口 8080) 运行正常${NC}"
else
    echo -e "${RED}❌ API Gateway (端口 8080) 启动失败${NC}"
    SUCCESS=false
fi

if lsof -Pi :5001 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✅ AI Chat 服务 (端口 5001) 运行正常${NC}"
else
    echo -e "${RED}❌ AI Chat 服务 (端口 5001) 启动失败${NC}"
    SUCCESS=false
fi

if lsof -Pi :5002 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Auth 服务 (端口 5002) 运行正常${NC}"
else
    echo -e "${RED}❌ Auth 服务 (端口 5002) 启动失败${NC}"
    SUCCESS=false
fi

if lsof -Pi :5004 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✅ ChatHistory 服务 (端口 5004) 运行正常${NC}"
else
    echo -e "${RED}❌ ChatHistory 服务 (端口 5004) 启动失败${NC}"
    SUCCESS=false
fi

if [ "$SUCCESS" = true ]; then
    echo -e "${GREEN}🎉 所有4个服务启动成功！${NC}"
    echo ""
    echo -e "${BLUE}📋 服务信息:${NC}"
    echo "API Gateway: http://localhost:8080"
    echo "AI Chat 服务: http://localhost:5001"
    echo "Auth 服务: http://localhost:5002"
    echo "ChatHistory 服务: http://localhost:5004"
    echo ""
    echo -e "${BLUE}🧪 测试ChatHistory API:${NC}"
    echo "curl http://localhost:8080/api/chat-history/admin/users"
else
    echo -e "${RED}❌ 部分服务启动失败，请检查日志文件${NC}"
    exit 1
fi