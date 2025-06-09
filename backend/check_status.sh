#!/bin/bash

# AI Chatbot Backend 状态检查脚本

RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
NC="\033[0m"

echo -e "${BLUE}📊 AI 聊天机器人服务状态${NC}"
echo "================================"

# 检查服务状态
echo -e "${BLUE}🔍 检查服务运行状态...${NC}"

GATEWAY_RUNNING=false
AICHAT_RUNNING=false

if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    GATEWAY_PID=$(lsof -ti:8080)
    echo -e "${GREEN}✅ API Gateway (端口 8080) 运行中 - PID: $GATEWAY_PID${NC}"
    GATEWAY_RUNNING=true
else
    echo -e "${RED}❌ API Gateway (端口 8080) 未运行${NC}"
fi

if lsof -Pi :5001 -sTCP:LISTEN -t >/dev/null 2>&1; then
    AICHAT_PID=$(lsof -ti:5001)
    echo -e "${GREEN}✅ AI Chat 服务 (端口 5001) 运行中 - PID: $AICHAT_PID${NC}"
    AICHAT_RUNNING=true
else
    echo -e "${RED}❌ AI Chat 服务 (端口 5001) 未运行${NC}"
fi

# 检查数据库文件
echo ""
echo -e "${BLUE}💾 检查数据库状态...${NC}"
if [ -f "components/AIChat/chatbot.db" ]; then
    DB_SIZE=$(ls -lh components/AIChat/chatbot.db | awk '{print $5}')
    echo -e "${GREEN}✅ SQLite 数据库存在 - 大小: $DB_SIZE${NC}"
else
    echo -e "${YELLOW}⚠️  SQLite 数据库文件不存在（首次运行时会自动创建）${NC}"
fi

# 检查环境变量配置
echo ""
echo -e "${BLUE}⚙️  检查配置文件...${NC}"
if [ -f ".env" ]; then
    echo -e "${GREEN}✅ API Gateway .env 文件存在${NC}"
else
    echo -e "${RED}❌ API Gateway .env 文件缺失${NC}"
fi

if [ -f "components/AIChat/.env" ]; then
    echo -e "${GREEN}✅ AI Chat .env 文件存在${NC}"
    # 检查 API Key
    if grep -q "your_openai_api_key_here" components/AIChat/.env; then
        echo -e "${RED}⚠️  请设置正确的 OPENAI_API_KEY${NC}"
    else
        echo -e "${GREEN}✅ OPENAI_API_KEY 已配置${NC}"
    fi
else
    echo -e "${RED}❌ AI Chat .env 文件缺失${NC}"
fi

# 检查日志文件
echo ""
echo -e "${BLUE}📋 检查日志文件...${NC}"
if [ -d "logs" ]; then
    if [ -f "logs/gateway.log" ]; then
        LOG_SIZE=$(ls -lh logs/gateway.log 2>/dev/null | awk '{print $5}' || echo "0")
        echo -e "${GREEN}✅ API Gateway 日志存在 - 大小: $LOG_SIZE${NC}"
    fi
    if [ -f "logs/aichat.log" ]; then
        LOG_SIZE=$(ls -lh logs/aichat.log 2>/dev/null | awk '{print $5}' || echo "0")
        echo -e "${GREEN}✅ AI Chat 日志存在 - 大小: $LOG_SIZE${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  日志目录不存在${NC}"
fi

# 整体状态
echo ""
echo -e "${BLUE}📊 整体状态...${NC}"
if [ "$GATEWAY_RUNNING" = true ] && [ "$AICHAT_RUNNING" = true ]; then
    echo -e "${GREEN}🎉 所有服务运行正常！${NC}"
    echo ""
    echo -e "${BLUE}🌐 服务地址:${NC}"
    echo "API Gateway: http://localhost:8080"
    echo "AI Chat 直接访问: http://localhost:5001"
else
    echo -e "${YELLOW}⚠️  部分或全部服务未运行${NC}"
    echo -e "${BLUE}💡 启动服务: ./start_services.sh${NC}"
fi

echo ""
echo -e "${BLUE}🔧 管理命令:${NC}"
echo "启动服务: ./start_services.sh"
echo "停止服务: ./stop_services.sh"
echo "查看状态: ./check_status.sh"
echo "查看日志: tail -f logs/aichat.log"
echo "         tail -f logs/gateway.log"
