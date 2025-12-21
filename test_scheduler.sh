#!/bin/bash

# ================================================
# SCHEDULER TEST SCRIPT
# ================================================
# Test 1h and daily summaries without waiting

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "================================================"
echo "🧪 SCHEDULER TESTING SCRIPT"
echo "================================================"

# Load .env
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)
fi

# Check if bot is running
BOT_PID=$(pgrep -f "telegram-summarizer" || echo "")
if [ -n "$BOT_PID" ]; then
    echo -e "${YELLOW}⚠️  Bot is running (PID: $BOT_PID)${NC}"
    echo "   Stop it before running tests? (y/n)"
    read -r answer
    if [ "$answer" = "y" ]; then
        kill $BOT_PID
        sleep 2
        echo -e "${GREEN}✅ Bot stopped${NC}"
    fi
fi

echo ""
echo "📋 Test Options:"
echo "  1. Test 1-hour summary (immediate)"
echo "  2. Test daily summary (immediate)"
echo "  3. Test both schedulers"
echo "  4. Test with 1-minute intervals (quick test)"
echo ""
read -p "Choose option [1-4]: " option

case $option in
    1)
        echo -e "${BLUE}🧪 Testing 1-hour summary...${NC}"
        ./test_1h_summary.sh
        ;;
    2)
        echo -e "${BLUE}🧪 Testing daily summary...${NC}"
        ./test_daily_summary.sh
        ;;
    3)
        echo -e "${BLUE}🧪 Testing both schedulers...${NC}"
        ./test_1h_summary.sh
        echo ""
        ./test_daily_summary.sh
        ;;
    4)
        echo -e "${BLUE}🧪 Testing with 1-minute intervals...${NC}"
        echo ""
        echo "This will:"
        echo "  - Run 1h summary every 1 minute (instead of 1 hour)"
        echo "  - Run daily summary after 3 minutes"
        echo ""
        echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
        echo ""
        
        # Modify scheduler temporarily for testing
        echo "Starting test mode..."
        # Note: This requires code modification, see instructions below
        echo ""
        echo -e "${RED}❌ Quick test mode requires code modification${NC}"
        echo ""
        echo "To enable quick test mode:"
        echo "1. Edit internal/scheduler/scheduler.go"
        echo "2. Change line 75: time.NewTicker(1 * time.Hour)"
        echo "   To: time.NewTicker(1 * time.Minute)"
        echo "3. Rebuild: make build"
        echo "4. Run: ./bin/telegram-summarizer --phone +6287742028130"
        echo ""
        ;;
    *)
        echo -e "${RED}Invalid option${NC}"
        exit 1
        ;;
esac
