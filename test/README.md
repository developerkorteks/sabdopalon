# 🧪 SCHEDULER TESTING GUIDE

## 📋 Test Options

### **Option 1: Quick Info Check (No waiting)**
```bash
./test_scheduler.sh
```
Shows database state, active groups, and message counts.

---

### **Option 2: Manual 1h Summary Test**
```bash
# Check database state
./test_1h_summary.sh

# Run actual test
cd test
go run test_1h_summary.go
```

**What it does:**
- ✅ Gets all ACTIVE groups
- ✅ Gets messages from last 1 hour
- ✅ Generates 1h summary with hierarchical chunking
- ✅ Saves to database (type: "1h")
- ✅ Shows preview
- ✅ No need to wait!

---

### **Option 3: Manual Daily Summary Test**
```bash
# Check database state
./test_daily_summary.sh

# Run actual test
cd test
go run test_daily_summary.go
```

**What it does:**
- ✅ Gets all ACTIVE groups
- ✅ Gets all 1h summaries from today
- ✅ Aggregates them
- ✅ Generates daily summary with hierarchical chunking
- ✅ Saves to database (type: "daily-test")
- ✅ Shows preview
- ✅ No need to wait until 23:59!

---

### **Option 4: Quick Test Mode (1-minute intervals)**

**For rapid testing, modify scheduler:**

1. Edit `internal/scheduler/scheduler.go` line 75:
   ```go
   // Change from:
   ticker := time.NewTicker(1 * time.Hour)
   
   // To:
   ticker := time.NewTicker(1 * time.Minute)
   ```

2. Edit line 67 (wait duration):
   ```go
   // Change from:
   waitDuration := nextHour.Sub(now)
   
   // To:
   waitDuration := 10 * time.Second  // Start after 10 seconds
   ```

3. Rebuild:
   ```bash
   make build
   ```

4. Run bot:
   ```bash
   ./run.sh
   ```

5. **Results:**
   - 1h summaries will run every 1 minute (instead of 1 hour)
   - You can see multiple runs quickly
   - Good for testing, **NOT for production**

6. **Remember to revert changes after testing!**

---

## 🎯 RECOMMENDED TESTING FLOW

### **Step 1: Prepare Data**
```bash
# 1. Start bot to collect messages
./run.sh

# 2. Wait 10-15 minutes for messages to accumulate

# 3. Check message count
./test_1h_summary.sh
```

### **Step 2: Test 1h Summary**
```bash
# Run test
cd test
go run test_1h_summary.go
```

**Expected output:**
```
🧪 TESTING 1-HOUR SUMMARY
================================================
✅ Services initialized

📋 Found 4 active groups

⏰ Time range: 16:00 - 17:00

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Processing group 1/4: FXT Chat Recording
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📨 Messages found: 45
🤖 Generating summary...
   📝 Generating summary...
✅ Summary generated: 2543 chars
💾 Saved to database (ID: 123)

...

📊 TEST SUMMARY
✅ Success: 4/4 groups
📨 Total messages processed: 156

✅ Test complete!
```

### **Step 3: Test Daily Summary**
```bash
# First, make sure there are 1h summaries
./test_daily_summary.sh

# Run test
cd test
go run test_daily_summary.go
```

**Expected output:**
```
🧪 TESTING DAILY SUMMARY
================================================
✅ Services initialized

📋 Found 4 active groups

⏰ Time range: 2025-12-07 00:00 - 23:59

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Processing group 1/4: FXT Chat Recording
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📝 1h summaries found: 8
📊 Aggregated: 18543 chars from 340 messages
🤖 Generating daily summary...
   📝 Chat is large - using streaming...
   📦 Processing batch 1/1...
   🔄 Merging batch 1/1...
✅ Daily summary generated: 5234 chars
💾 Saved to database (ID: 456)

📄 Summary Preview (first 500 chars):
─────────────────────────────────────────────
╔════════════════════════════════╗
║  📊 SUMMARY PART 1/1          ║
╚════════════════════════════════╝

Group: `FXT Chat Recording`
Period: `00:00` - `23:59`
...
─────────────────────────────────────────────

📊 TEST SUMMARY
✅ Success: 4/4 groups

✅ Test complete!
```

---

## 📊 VERIFICATION QUERIES

### **Check 1h Summaries:**
```bash
sqlite3 telegram_bot.db "
SELECT 
    chat_id,
    datetime(period_start, 'localtime') as start,
    datetime(period_end, 'localtime') as end,
    message_count,
    LENGTH(summary_text) as chars
FROM summaries 
WHERE summary_type='1h' 
ORDER BY created_at DESC 
LIMIT 10
"
```

### **Check Daily Summaries:**
```bash
sqlite3 telegram_bot.db "
SELECT 
    chat_id,
    DATE(period_start) as date,
    message_count,
    LENGTH(summary_text) as chars
FROM summaries 
WHERE summary_type='daily' OR summary_type='daily-test'
ORDER BY created_at DESC 
LIMIT 5
"
```

### **Check Messages Per Hour:**
```bash
sqlite3 telegram_bot.db "
SELECT 
    strftime('%Y-%m-%d %H:00', timestamp) as hour,
    chat_id,
    COUNT(*) as count
FROM messages 
WHERE timestamp > datetime('now', '-24 hour')
GROUP BY hour, chat_id
ORDER BY hour DESC, count DESC
LIMIT 20
"
```

---

## ⏱️ TIMING TESTS

### **Test 1h Summary Timing:**
```bash
# Run and measure time
time go run test/test_1h_summary.go
```

**Expected timing:**
- Small group (30 msgs): ~5-10 seconds
- Medium group (90 msgs): ~15-20 seconds
- Large group (180 msgs): ~30-40 seconds

### **Test Daily Summary Timing:**
```bash
# Run and measure time
time go run test/test_daily_summary.go
```

**Expected timing:**
- Few 1h summaries (5): ~10-15 seconds
- Many 1h summaries (20): ~20-30 seconds

---

## 🔧 TROUBLESHOOTING

### **No Active Groups:**
```bash
# List all groups
sqlite3 telegram_bot.db "SELECT chat_id, group_name, is_active FROM tracked_groups"

# Activate a group manually
sqlite3 telegram_bot.db "UPDATE tracked_groups SET is_active=1 WHERE chat_id=2983014239"
```

### **No Messages:**
```bash
# Check if scraper is running
ps aux | grep telegram-summarizer

# Check message count
sqlite3 telegram_bot.db "SELECT COUNT(*) FROM messages"

# Check recent messages
sqlite3 telegram_bot.db "SELECT chat_id, COUNT(*) FROM messages WHERE timestamp > datetime('now', '-1 hour') GROUP BY chat_id"
```

### **No 1h Summaries for Daily Test:**
```bash
# Run 1h summary test first
go run test/test_1h_summary.go

# Then run daily test
go run test/test_daily_summary.go
```

---

## ✅ SUCCESS INDICATORS

### **1h Summary Test Success:**
- ✅ No errors during generation
- ✅ Summaries saved to database
- ✅ Summary length reasonable (500-5000 chars)
- ✅ All active groups processed

### **Daily Summary Test Success:**
- ✅ 1h summaries aggregated correctly
- ✅ Daily summary generated
- ✅ Elegant formatting applied
- ✅ Saved to database with type "daily-test"

---

## 🚀 PRODUCTION TESTING

### **Test Real Scheduler:**

1. **Start bot normally:**
   ```bash
   ./run.sh
   ```

2. **Watch logs in real-time:**
   ```bash
   tail -f logs/bot.log  # or wherever your logs are
   ```

3. **Wait for next hour mark** (e.g., 17:00, 18:00)

4. **You should see:**
   ```
   [INFO] 🕐 Generating 1-hour summaries...
   [INFO] Found 4 active groups
   [INFO] Processing group 1/4: FXT Chat Recording
   [INFO] Using hierarchical summarization for 1h summary
   ...
   [INFO] ✅ Success: 4/4 groups, 156 messages processed
   ```

5. **Verify in database:**
   ```bash
   sqlite3 telegram_bot.db "SELECT * FROM summaries WHERE summary_type='1h' ORDER BY created_at DESC LIMIT 1"
   ```

---

## 📝 NOTES

- Test scripts save with type "daily-test" (not "daily") to avoid confusion
- Real scheduler uses "1h" and "daily" types
- Test scripts don't send to Telegram (only save to DB)
- Use test scripts to verify logic without waiting

**Ready to test now!** 🧪✨
