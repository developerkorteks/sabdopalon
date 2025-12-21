#!/bin/bash

# ================================================
# TELEGRAM SUMMARIZER - RUN SCRIPT
# ================================================
# This script loads .env file and runs the bot

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "================================================"
echo "🤖 TELEGRAM SUMMARIZER LAUNCHER"
echo "================================================"

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${RED}❌ Error: .env file not found!${NC}"
    echo ""
    echo "Please create .env file first:"
    echo "  cp .env.example .env"
    echo "  nano .env"
    echo ""
    exit 1
fi

# Load .env file
echo -e "${GREEN}✅ Loading environment variables from .env...${NC}"
export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)

# Check if binary exists
if [ ! -f bin/telegram-summarizer ]; then
    echo -e "${YELLOW}⚠️  Binary not found. Building...${NC}"
    make build
fi

# Display configuration (without showing full API keys)
echo ""
echo "📋 Configuration:"
echo "  • Bot Token: ${TELEGRAM_BOT_TOKEN:0:20}..."
echo "  • Phone: ${PHONE_NUMBER}"
echo "  • Gemini Key: ${GEMINI_API_KEY:0:20}..."
echo "  • Database: ${DATABASE_PATH}"
echo "  • Debug Mode: ${DEBUG_MODE}"
echo ""

# Parse command line arguments
MODE="${RUN_MODE:-all}"
PHONE="${PHONE_NUMBER}"

# Override with command line args if provided
while [[ $# -gt 0 ]]; do
    case $1 in
        --mode)
            MODE="$2"
            shift 2
            ;;
        --phone)
            PHONE="$2"
            shift 2
            ;;
        --help)
            echo "Usage: ./run.sh [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --mode <mode>    Run mode: all, bot, scraper (default: all)"
            echo "  --phone <number> Phone number for scraper"
            echo "  --help           Show this help message"
            echo ""
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Run the bot
echo -e "${GREEN}🚀 Starting Telegram Summarizer...${NC}"
echo "  Mode: $MODE"
if [ "$MODE" != "bot" ]; then
    echo "  Phone: $PHONE"
fi
echo ""

if [ "$MODE" == "bot" ]; then
    ./bin/telegram-summarizer --mode bot
elif [ "$MODE" == "scraper" ]; then
    ./bin/telegram-summarizer --mode scraper --phone "$PHONE"
else
    ./bin/telegram-summarizer --phone "$PHONE"
fi
