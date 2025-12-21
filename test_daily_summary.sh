#!/bin/bash

# ================================================
# TEST DAILY SUMMARY
# ================================================
# Manually trigger daily summary generation

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "================================================"
echo "🧪 TESTING DAILY SUMMARY"
echo "================================================"

# Load env
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)
fi

echo ""
echo "📊 Database Info:"
echo "  Database: ${DATABASE_PATH:-telegram_bot.db}"
echo ""

# Check database
if [ ! -f "${DATABASE_PATH:-telegram_bot.db}" ]; then
    echo "❌ Database not found. Please run bot first."
    exit 1
fi

# Show 1h summaries from today
echo "📋 1h Summaries from today:"
SUMMARIES_TODAY=$(sqlite3 ${DATABASE_PATH:-telegram_bot.db} "SELECT COUNT(*) FROM summaries WHERE summary_type='1h' AND DATE(period_start) = DATE('now')")
echo "  Count: $SUMMARIES_TODAY"

if [ "$SUMMARIES_TODAY" = "0" ]; then
    echo ""
    echo -e "${YELLOW}⚠️  No 1h summaries from today!${NC}"
    echo "   Daily summary requires 1h summaries to aggregate."
    echo ""
    echo "   Check recent 1h summaries:"
    sqlite3 -header -column ${DATABASE_PATH:-telegram_bot.db} "
    SELECT 
        chat_id,
        DATE(period_start) as date,
        COUNT(*) as count
    FROM summaries 
    WHERE summary_type='1h' 
    GROUP BY chat_id, DATE(period_start)
    ORDER BY date DESC
    LIMIT 10
    "
else
    echo ""
    sqlite3 -header -column ${DATABASE_PATH:-telegram_bot.db} "
    SELECT 
        chat_id,
        COUNT(*) as summaries,
        SUM(message_count) as total_msgs,
        MIN(datetime(period_start)) as first,
        MAX(datetime(period_end)) as last
    FROM summaries 
    WHERE summary_type='1h' AND DATE(period_start) = DATE('now')
    GROUP BY chat_id
    "
fi

echo ""
echo "📝 Existing daily summaries:"
DAILY_COUNT=$(sqlite3 ${DATABASE_PATH:-telegram_bot.db} "SELECT COUNT(*) FROM summaries WHERE summary_type='daily'")
echo "  Count: $DAILY_COUNT"

if [ "$DAILY_COUNT" != "0" ]; then
    echo ""
    sqlite3 -header -column ${DATABASE_PATH:-telegram_bot.db} "
    SELECT 
        chat_id,
        datetime(period_start) as date,
        message_count,
        LENGTH(summary_text) as summary_len,
        datetime(created_at) as created
    FROM summaries 
    WHERE summary_type='daily' 
    ORDER BY created_at DESC 
    LIMIT 5
    "
fi

echo ""
echo "================================================"
echo "🚀 To manually test daily summary:"
echo "================================================"
echo ""
echo "Option 1: Use Go test (create test file)"
echo "  go run test/test_daily_summary.go"
echo ""
echo "Option 2: Wait until ${DAILY_SUMMARY_TIME:-23:59}"
echo "  The scheduler will run automatically"
echo ""
echo "Option 3: Temporarily change schedule time"
echo "  1. Edit .env: DAILY_SUMMARY_TIME=17:00 (next hour)"
echo "  2. Restart bot: ./run.sh"
echo "  3. Wait until 17:00"
echo ""

echo -e "${GREEN}✅ Daily summary test info displayed${NC}"
