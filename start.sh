#!/bin/bash

# Fabula Studio - Start Script
# Starts observability stack and backend server

set -e

echo "🎬 Fabula Studio - Starting..."
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Start observability stack
echo -e "${BLUE}Starting observability stack...${NC}"
docker compose up -d

echo ""
echo -e "${GREEN}✅ Observability stack started:${NC}"
echo "   - Jaeger UI: http://localhost:16686"
echo "   - Grafana:   http://localhost:3000"
echo ""

# Wait for services to be ready
echo -e "${YELLOW}Waiting for services to start...${NC}"
sleep 5

# Check if port 8080 is available
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo "⚠️  Port 8080 is already in use. Using port 8081 instead."
    export LISTEN_ADDR=":8081"
    API_PORT=8081
else
    API_PORT=8080
fi

# Start backend
echo ""
echo -e "${BLUE}Starting backend server...${NC}"
cd apps/backend

export OPENAI_API_KEY="${OPENAI_API_KEY:-tp-c8htfkee8pzb1i5x9sdde7metqtl6q9sw6c2ls3gxzpiug4v}"
export OPENAI_BASE_URL="${OPENAI_BASE_URL:-https://token-plan-cn.xiaomimimo.com/v1}"
export LLM_MODEL="${LLM_MODEL:-mimo-v2.5}"
export OTLP_ENDPOINT="localhost:4317"

go run ./cmd/server/ &
BACKEND_PID=$!

echo ""
echo -e "${GREEN}✅ Backend server started (PID: $BACKEND_PID)${NC}"
echo "   - API: http://localhost:${API_PORT}"
echo "   - Events: http://localhost:${API_PORT}/api/events"
echo "   - SSE Stream: http://localhost:${API_PORT}/api/events/stream"
echo ""
echo "📊 Monitor page: file://$(pwd)/../test/monitor.html"
echo ""
echo "Press Ctrl+C to stop all services"

# Trap to cleanup
trap "echo ''; echo 'Stopping services...'; kill $BACKEND_PID 2>/dev/null; docker compose down; echo 'Done.'; exit" INT TERM

# Wait for backend
wait $BACKEND_PID
