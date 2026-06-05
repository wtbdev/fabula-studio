#!/bin/bash

# AG-UI Protocol Test Script
# 用于测试 AG-UI 协议的各个端点

BASE_URL="http://localhost:8080"

echo "🧪 AG-UI Protocol Test Suite"
echo "=========================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试健康检查
echo -e "${YELLOW}1. 测试健康检查${NC}"
curl -s "$BASE_URL/api/health" | jq .
echo ""

# 创建会话
echo -e "${YELLOW}2. 创建会话${NC}"
SESSION=$(curl -s -X POST "$BASE_URL/api/sessions" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "test-user"}' | jq -r '.id')
echo "Session ID: $SESSION"
echo ""

# 列出会话
echo -e "${YELLOW}3. 列出所有会话${NC}"
curl -s "$BASE_URL/api/sessions" | jq .
echo ""

# 获取特定会话
echo -e "${YELLOW}4. 获取会话详情${NC}"
curl -s "$BASE_URL/api/sessions/$SESSION" | jq .
echo ""

# 普通聊天请求
echo -e "${YELLOW}5. 普通聊天请求${NC}"
RESPONSE=$(curl -s -X POST "$BASE_URL/api/chat" \
  -H "Content-Type: application/json" \
  -d "{
    \"session_id\": \"$SESSION\",
    \"message\": \"帮我分析一下这个故事的角色关系\"
  }")
echo "$RESPONSE" | jq .
echo ""

# SSE 流式请求
echo -e "${YELLOW}6. SSE 流式请求${NC}"
echo "发送流式请求，等待 10 秒..."
timeout 10 curl -N -s -X POST "$BASE_URL/api/chat/stream" \
  -H "Content-Type: application/json" \
  -d "{
    \"session_id\": \"$SESSION\",
    \"message\": \"简要介绍一下这个故事的主要冲突\"
  }" 2>&1 | while IFS= read -r line; do
    if [[ $line == data:* ]]; then
      echo -e "${GREEN}收到事件:${NC} ${line#data: }" | jq . 2>/dev/null || echo "$line"
    fi
  done
echo ""

# 测试错误情况
echo -e "${YELLOW}7. 测试错误处理 - 空消息${NC}"
curl -s -X POST "$BASE_URL/api/chat" \
  -H "Content-Type: application/json" \
  -d '{"message": ""}' | jq .
echo ""

echo -e "${YELLOW}8. 测试错误处理 - 无效会话${NC}"
curl -s "$BASE_URL/api/sessions/non-existent-id" | jq .
echo ""

echo -e "${GREEN}✅ 测试完成${NC}"
