#!/bin/bash

# ================================================
# TEST 1-HOUR SUMMARY
# ================================================
# Manually trigger 1h summary generation

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "================================================"
echo "🧪 TESTING 1-HOUR SUMMARY"
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

# Show active groups
echo "📋 Active Groups:"
sqlite3 ${DATABASE_PATH:-telegram_bot.db} "SELECT chat_id, group_name, is_active FROM tracked_groups WHERE is_active=1" | head -10
ACTIVE_COUNT=$(sqlite3 ${DATABASE_PATH:-telegram_bot.db} "SELECT COUNT(*) FROM tracked_groups WHERE is_active=1")
echo ""
echo "Total active groups: $ACTIVE_COUNT"

if [ "$ACTIVE_COUNT" = "0" ]; then
    echo ""
    echo "⚠️  No active groups found!"
    echo "   Please activate groups first:"
    echo "   /enable <chat_id>"
    exit 1
fi

echo ""
echo "📨 Messages in last hour:"
sqlite3 ${DATABASE_PATH:-telegram_bot.db} "
SELECT 
    chat_id, 
    COUNT(*) as count 
FROM messages 
WHERE timestamp > datetime('now', '-1 hour') 
GROUP BY chat_id
ORDER BY count DESC
LIMIT 10
"

TOTAL_MSGS=$(sqlite3 ${DATABASE_PATH:-telegram_bot.db} "SELECT COUNT(*) FROM messages WHERE timestamp > datetime('now', '-1 hour')")
echo ""
echo "Total messages (last 1h): $TOTAL_MSGS"

if [ "$TOTAL_MSGS" = "0" ]; then
    echo ""
    echo "⚠️  No messages in last hour. Using last 24h for testing..."
    echo ""
    sqlite3 ${DATABASE_PATH:-telegram_bot.db} "
    SELECT 
        chat_id, 
        COUNT(*) as count 
    FROM messages 
    WHERE timestamp > datetime('now', '-24 hour') 
    GROUP BY chat_id
    ORDER BY count DESC
    LIMIT 5
    "
fi

echo ""
echo "================================================"
echo "🚀 To manually test 1h summary:"
echo "================================================"
echo ""
echo "Option 1: Use Go test"
echo "  go run test/test_1h_summary.go"
echo ""
echo "Option 2: Wait for next hour mark"
echo "  The scheduler will run automatically at next :00"
echo ""
echo "Option 3: Check existing 1h summaries"
echo "  sqlite3 ${DATABASE_PATH:-telegram_bot.db} \"SELECT * FROM summaries WHERE summary_type='1h' ORDER BY created_at DESC LIMIT 5\""
echo ""

# Show existing 1h summaries
SUMMARY_COUNT=$(sqlite3 ${DATABASE_PATH:-telegram_bot.db} "SELECT COUNT(*) FROM summaries WHERE summary_type='1h'")
if [ "$SUMMARY_COUNT" != "0" ]; then
    echo "📝 Existing 1h summaries: $SUMMARY_COUNT"
    echo ""
    sqlite3 -header -column ${DATABASE_PATH:-telegram_bot.db} "
    SELECT 
        chat_id,
        datetime(period_start) as start,
        datetime(period_end) as end,
        message_count,
        LENGTH(summary_text) as summary_len,
        datetime(created_at) as created
    FROM summaries 
    WHERE summary_type='1h' 
    ORDER BY created_at DESC 
    LIMIT 5
    "
fi

echo ""
echo -e "${GREEN}✅ 1h summary test info displayed${NC}"
