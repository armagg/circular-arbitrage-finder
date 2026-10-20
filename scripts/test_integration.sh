#!/bin/bash


set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="$PROJECT_ROOT/scripts"
GO_BINARY="$PROJECT_ROOT/bin/arb-finder"
CONFIG_FILE="$PROJECT_ROOT/config.yaml"

cd "$PROJECT_ROOT"

echo -e "${BLUE}🚀 Starting Arbitrage Finder Integration Test${NC}"
echo -e "${BLUE}================================================${NC}"

echo -e "${YELLOW}📋 Checking dependencies...${NC}"

if [ ! -f "$GO_BINARY" ]; then
    echo -e "${RED}❌ Go binary not found at: $GO_BINARY${NC}"
    echo -e "${YELLOW}💡 Build it with: go build -o bin/arb-finder ./cmd/arb-finder${NC}"
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}❌ Config file not found at: $CONFIG_FILE${NC}"
    exit 1
fi

if [ ! -f "$SCRIPTS_DIR/market_data_feeder.py" ]; then
    echo -e "${RED}❌ Market data feeder script not found${NC}"
    exit 1
fi

if [ ! -f "$SCRIPTS_DIR/strategy_logger.py" ]; then
    echo -e "${RED}❌ Strategy logger script not found${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All dependencies found${NC}"

cleanup() {
    echo -e "\n${YELLOW}🧹 Cleaning up processes...${NC}"
    kill $(jobs -p) 2>/dev/null || true
    wait 2>/dev/null || true
}

trap cleanup EXIT

echo -e "${YELLOW}🔧 Starting Go arbitrage finder service...${NC}"
export INGRESS_ADDR="127.0.0.1:50051"
export EXECUTOR_ADDR="127.0.0.1:60051"

"$GO_BINARY" &
GO_PID=$!

sleep 3

if ! kill -0 $GO_PID 2>/dev/null; then
    echo -e "${RED}❌ Go service failed to start${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go service started (PID: $GO_PID)${NC}"

echo -e "${YELLOW}📊 Starting strategy logger...${NC}"
python3 "$SCRIPTS_DIR/strategy_logger.py" --summary-interval 0 &
LOGGER_PID=$!

sleep 2

if ! kill -0 $LOGGER_PID 2>/dev/null; then
    echo -e "${RED}❌ Strategy logger failed to start${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Strategy logger started (PID: $LOGGER_PID)${NC}"

echo -e "${YELLOW}📈 Running market data feeder...${NC}"
python3 "$SCRIPTS_DIR/market_data_feeder.py" --mode arbitrage --interval 0.5 &
FEEDER_PID=$!

sleep 10

kill $FEEDER_PID 2>/dev/null || true
wait $FEEDER_PID 2>/dev/null || true

echo -e "${GREEN}✅ Market data feeder completed${NC}"

echo -e "${YELLOW}⏹️  Stopping strategy logger...${NC}"
kill $LOGGER_PID 2>/dev/null || true
wait $LOGGER_PID 2>/dev/null || true

echo -e "${GREEN}✅ Strategy logger stopped${NC}"

echo -e "${YELLOW}⏹️  Stopping Go service...${NC}"
kill $GO_PID 2>/dev/null || true
wait $GO_PID 2>/dev/null || true

echo -e "${GREEN}✅ Go service stopped${NC}"

echo -e "\n${GREEN}🎉 Integration test completed successfully!${NC}"
echo -e "${BLUE}================================================${NC}"
echo -e "${YELLOW}💡 Next steps:${NC}"
echo -e "  • Check the strategy logger output above for detected arbitrage opportunities"
echo -e "  • Run with --verbose flag for detailed logging:"
echo -e "    python3 scripts/market_data_feeder.py --verbose --mode arbitrage"
echo -e "    python3 scripts/strategy_logger.py --verbose"
echo -e "  • Use the integration test script for automated testing:"
echo -e "    python3 scripts/integration_test.py --duration 30"
