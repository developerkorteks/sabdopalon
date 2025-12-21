# 📅 AUTO-SUMMARY SYSTEM - COMPLETE GUIDE

Berdasarkan analisis source code `internal/scheduler/scheduler.go`

---

## 🎯 OVERVIEW

Auto-summary adalah fitur **OPTIONAL** yang berjalan di background untuk generate summary secara otomatis tanpa user harus kirim command `/summary`.

**Status Saat Ini:** ❌ **DISABLED** (karena `SUMMARY_TARGET_CHAT_ID` belum di-set)

---

## 🏗️ ARSITEKTUR

```
┌─────────────────────────────────────────────────────────┐
│              SCHEDULER (Optional)                       │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  IF SUMMARY_TARGET_CHAT_ID is set:                     │
│                                                         │
│  ┌──────────────────────────────────────────┐          │
│  │  TWO INDEPENDENT SCHEDULERS:             │          │
│  ├──────────────────────────────────────────┤          │
│  │                                          │          │
│  │  1. HOURLY SCHEDULER                     │          │
│  │     • Runs every hour (00:00-23:00)     │          │
│  │     • Generate 1h summaries             │          │
│  │     • Save to database                  │          │
│  │     • For ALL active groups             │          │
│  │                                          │          │
│  │  2. DAILY SCHEDULER                      │          │
│  │     • Runs once at 20:00 WIB            │          │
│  │     • Aggregate 1h summaries            │          │
│  │     • Generate daily summary            │          │
│  │     • Send to target chat               │          │
│  │     • Cleanup old messages (>24h)       │          │
│  │                                          │          │
│  └──────────────────────────────────────────┘          │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## 🔄 FLOW AUTO-SUMMARY

### **STARTUP FLOW:**

```
./bot startup
  ↓
Load config (TELEGRAM_TOKEN, GEMINI_API_KEY, etc)
  ↓
Initialize Database, Summarizer, Bot
  ↓
Check: SUMMARY_TARGET_CHAT_ID environment variable
  ↓
┌─────────────────────────────────────────────┐
│ IF SUMMARY_TARGET_CHAT_ID is set:          │
├─────────────────────────────────────────────┤
│                                             │
│ targetChatID = parse(SUMMARY_TARGET_CHAT_ID)│
│   ↓                                         │
│ Create Scheduler:                           │
│   scheduler.NewScheduler(                   │
│     database,                               │
│     summarizer,                             │
│     botAPI,                                 │
│     targetChatID                            │
│   )                                         │
│   ↓                                         │
│ Start Scheduler:                            │
│   scheduler.Start("20:00")  // daily time   │
│   ↓                                         │
│ Launches TWO goroutines:                    │
│   1. go run1HourScheduler()                 │
│   2. go runDailyScheduler("20:00")          │
│   ↓                                         │
│ Log: "✅ Scheduler ready (Daily at 20:00)"  │
│                                             │
└─────────────────────────────────────────────┘
  ↓
┌─────────────────────────────────────────────┐
│ ELSE (not set):                             │
├─────────────────────────────────────────────┤
│                                             │
│ Log: "⚠️ SUMMARY_TARGET_CHAT_ID not set"    │
│ Log: "Auto-summary scheduler disabled"     │
│ Log: "You can still use /summary manually" │
│                                             │
│ Scheduler = nil (not created)               │
│                                             │
└─────────────────────────────────────────────┘
  ↓
Bot continues normally (manual commands still work)
```

---

## ⏰ HOURLY SCHEDULER (1-Hour Summaries)

### **Flow:**

```
Startup
  ↓
Calculate time until next hour mark
  (e.g., now=10:23, next=11:00, wait=37 minutes)
  ↓
Log: "⏰ Next 1h summary in: 37m"
  ↓
Wait (sleep) until 11:00
  ↓
┌────────────────────────────────────────────────┐
│ EVERY HOUR (00:00, 01:00, 02:00, ... 23:00)  │
├────────────────────────────────────────────────┤
│                                                │
│ generate1HourSummaries()                       │
│   ↓                                            │
│ 1. Get all active groups from database        │
│    groups = database.GetActiveGroups()        │
│    ↓                                           │
│ 2. For each active group:                     │
│    ↓                                           │
│    a. Get messages (last 1 hour)              │
│       startTime = now - 1h                     │
│       endTime = now                            │
│       messages = db.GetMessagesByTimeRange()  │
│       ↓                                        │
│    b. Skip if < 3 messages                     │
│       ↓                                        │
│    c. Format messages:                         │
│       [15:30] User1: text                      │
│       [15:45] User2: text                      │
│       ...                                      │
│       ↓                                        │
│    d. Build prompt (Indonesian):               │
│       promptManager.Get1HourPrompt(...)        │
│       ↓                                        │
│    e. Generate summary with AI FALLBACK:       │
│       summarizer.GenerateSummary(prompt, "1h") │
│       → Tries 18 AI providers                  │
│       ↓                                        │
│    f. Parse metadata:                          │
│       • sentiment                              │
│       • credibility_score                      │
│       • products_mentioned                     │
│       • red_flags                              │
│       ↓                                        │
│    g. Save summary to database:                │
│       db.SaveSummary(summary)                  │
│       db.SaveProductMention(products)          │
│       ↓                                        │
│    h. Log: "✅ 1h summary saved for GroupName" │
│       ↓                                        │
│ 3. Log: "✅ 1-hour summaries complete: X/Y"   │
│                                                │
└────────────────────────────────────────────────┘
  ↓
Sleep 1 hour
  ↓
REPEAT (next hour)
```

### **Key Points:**

- ✅ Runs **24 times per day** (every hour)
- ✅ Processes **ALL active groups** (currently 4 groups)
- ✅ Saves to database (NOT sent to chat automatically)
- ✅ Skips groups with < 3 messages in the hour
- ✅ Uses same **18 AI providers fallback** system
- ✅ Extracts metadata (sentiment, products, etc)

### **Example Log Output:**

```
[INFO] 🕐 Generating 1-hour summaries...
[INFO] Processing 4 active groups
[INFO] 📝 1h summary for: FXT Chat Recording (ID: 2983014239)
[INFO] ✅ 1h summary saved for FXT Chat Recording (23 messages, 2 products)
[INFO] 📝 1h summary for: AnooooMali Engsellllll (ID: 3103764752)
[INFO] ⏭️  Skipping (λ)³: only 2 messages (need at least 3)
[INFO] ✅ 1h summary saved for AnooooMali Engsellllll (18 messages, 1 products)
[INFO] ✅ 1-hour summaries complete: 2/4 groups, 41 messages processed
```

---

## 🌅 DAILY SCHEDULER (Daily Summary)

### **Flow:**

```
Startup
  ↓
Parse daily time: "20:00"
  ↓
Calculate next run time:
  - If now < 20:00 today → run today at 20:00
  - If now > 20:00 today → run tomorrow at 20:00
  ↓
Log: "⏰ Next daily summary at: 2024-12-07 20:00:00 (in 3h 37m)"
  ↓
Wait (sleep) until 20:00
  ↓
┌──────────────────────────────────────────────────┐
│ EVERY DAY AT 20:00 WIB                          │
├──────────────────────────────────────────────────┤
│                                                  │
│ runDailySummaryForAllGroups()                    │
│   ↓                                              │
│ 1. Get all active groups                        │
│    groups = database.GetTrackedGroups()         │
│    filter: is_active = 1                        │
│    ↓                                             │
│ 2. For each active group:                       │
│    ↓                                             │
│    generateAndSendDailySummary(group)            │
│      ↓                                           │
│    a. Get time range (today 00:00 - now)        │
│       startTime = today 00:00:00                 │
│       endTime = now                              │
│       ↓                                          │
│    b. Get all 1h summaries from today           │
│       summaries = db.GetSummariesByTimeRange(   │
│         chatID, "1h", startTime, endTime        │
│       )                                          │
│       ↓                                          │
│    c. Skip if no 1h summaries                   │
│       ↓                                          │
│    d. Combine all 1h summaries:                 │
│       === Periode 1: 00:00 - 01:00 ===          │
│       [1h summary text]                          │
│                                                  │
│       === Periode 2: 01:00 - 02:00 ===          │
│       [1h summary text]                          │
│       ...                                        │
│       ↓                                          │
│    e. Build daily prompt:                        │
│       promptManager.GetDailyPrompt(              │
│         combinedSummaries, groupName, date      │
│       )                                          │
│       ↓                                          │
│    f. Generate daily summary with AI:            │
│       summarizer.GenerateSummary(prompt, "daily")│
│       → Uses 18 AI providers fallback            │
│       ↓                                          │
│    g. Parse metadata (sentiment, products, etc)  │
│       ↓                                          │
│    h. Save daily summary to database            │
│       ↓                                          │
│    i. Format message:                            │
│       📝 Daily Summary for GroupName             │
│       📅 Date: 2024-12-06                        │
│       💬 Total Messages: 234                     │
│       📊 Based on 18 one-hour summaries          │
│                                                  │
│       [Daily summary text]                       │
│       ↓                                          │
│    j. Send to SUMMARY_TARGET_CHAT_ID             │
│       bot.Send(targetChatID, message)            │
│       ↓                                          │
│    k. Cleanup old messages (>24h old)            │
│       db.DeleteMessagesOlderThan(chatID, 24h)   │
│       ↓                                          │
│    l. Log: "✅ Daily summary sent for GroupName" │
│       ↓                                          │
│ 3. Send completion report to target chat:       │
│    "📊 Daily Summary Report                      │
│     ✅ Successfully: 4 groups                    │
│     ❌ Failed: 0 groups                          │
│     📅 Date: 2024-12-06"                         │
│                                                  │
└──────────────────────────────────────────────────┘
  ↓
Wait 24 hours
  ↓
REPEAT (next day at 20:00)
```

### **Key Points:**

- ✅ Runs **once per day** at 20:00 WIB
- ✅ **Aggregates 1h summaries** (not raw messages)
- ✅ Sends summary to **SUMMARY_TARGET_CHAT_ID**
- ✅ Auto-cleanup old messages (>24h) after successful summary
- ✅ Sends completion report
- ✅ Uses **18 AI providers fallback**
- ✅ Auto-split long messages (>4000 chars)

### **Example Output (in target chat):**

```
📝 Daily Summary for FXT Chat Recording 💠

📅 Date: 2024-12-06
💬 Total Messages: 234
📊 Based on 18 one-hour summaries

━━━━━━━━━━━━━━━━━━━━━━━

**Ringkasan Harian:**

Hari ini di grup FXT Chat Recording terdapat diskusi aktif 
mengenai beberapa topik utama:

1. **Trading Signals** (08:00-12:00)
   - Diskusi intensif tentang EUR/USD signals
   - Banyak member sharing hasil profit
   - Beberapa strategi baru dibahas

2. **Market Analysis** (13:00-17:00)
   - Analisis fundamental ekonomi global
   - Prediksi pergerakan harga
   - Tips risk management

3. **Social & Chitchat** (18:00-20:00)
   - Obrolan santai antar member
   - Sharing pengalaman trading
   - Q&A session

**Produk yang disebutkan:**
- MetaTrader 5
- TradingView Pro
- Signal Provider XYZ

**Sentimen:** Positive
**Kredibilitas:** 4/5

━━━━━━━━━━━━━━━━━━━━━━━
Generated by AI ✨

---

📊 Daily Summary Report

✅ Successfully summarized: 4 groups
❌ Failed: 0 groups
📅 Date: 2024-12-06
```

---

## 🔧 CARA ENABLE AUTO-SUMMARY

### **Method 1: Environment Variable (Recommended)**

```bash
# Set target chat ID
export SUMMARY_TARGET_CHAT_ID=6491485169

# Restart bot
./bot
```

### **Method 2: .env File**

```bash
# Edit .env atau config file
echo "SUMMARY_TARGET_CHAT_ID=6491485169" >> .env

# Restart bot
./bot
```

### **Method 3: Command Line (Temporary)**

```bash
SUMMARY_TARGET_CHAT_ID=6491485169 ./bot
```

### **Cara Get Chat ID:**

1. Buka Telegram bot
2. Kirim `/start` ke bot
3. Bot akan reply dengan:
   ```
   🤖 Telegram Chat Summarizer Bot
   
   Your Chat ID: 6491485169
   
   To enable auto-summaries, set:
   export SUMMARY_TARGET_CHAT_ID=6491485169
   ```
4. Copy chat ID tersebut

---

## 📊 CONFIGURATION OPTIONS

```bash
# Required
TELEGRAM_TOKEN=your_bot_token
GEMINI_API_KEY=your_gemini_key

# Optional (for auto-summary)
SUMMARY_TARGET_CHAT_ID=6491485169     # Your chat ID
DAILY_SUMMARY_TIME=20:00              # Default: 20:00 WIB
SUMMARY_INTERVAL=1                     # Hours for hourly summary (default: 1)

# Database & Debug
DATABASE_PATH=telegram_bot.db         # Default
DEBUG_MODE=false                      # Default
```

---

## 📈 STATISTICS & MONITORING

### **Check Scheduler Status:**

```bash
# In bot logs
grep "Scheduler" bot.log

# Should see:
# [INFO] 📅 Starting schedulers...
# [INFO]   ⏰ 1-hour summaries: Every hour
# [INFO]   🌅 Daily summary: 20:00
# [INFO] ✅ Scheduler ready (Daily summary at 20:00)
```

### **Monitor 1h Summaries:**

```bash
# Check database
sqlite3 telegram_bot.db "SELECT COUNT(*) FROM summaries WHERE summary_type='1h';"

# Check logs
grep "1h summary" bot.log | tail -20
```

### **Monitor Daily Summaries:**

```bash
# Check database
sqlite3 telegram_bot.db "SELECT * FROM summaries WHERE summary_type='daily' ORDER BY created_at DESC LIMIT 5;"

# Check logs
grep "Daily summary" bot.log | tail -20
```

---

## 🎯 BENEFITS

### **1. Automatic Summarization**
- ❌ No need manual `/summary` command
- ✅ Automatic every hour + daily
- ✅ Always up-to-date summaries

### **2. Two-Level Hierarchy**
```
Raw Messages (real-time)
     ↓
1-Hour Summaries (24x per day)
     ↓
Daily Summary (1x per day)
```
- Better organization
- Less data to process for daily
- Faster daily summary generation

### **3. Automatic Cleanup**
- ✅ Old messages deleted after 24h
- ✅ Database stays small
- ✅ Only summaries kept for history

### **4. Centralized Delivery**
- ✅ All summaries sent to one chat
- ✅ Easy to review
- ✅ Notification for each group

### **5. Production-Grade**
- ✅ Graceful shutdown handling
- ✅ Error recovery (continues if one group fails)
- ✅ Rate limiting (delays between groups)
- ✅ Auto-split long messages

---

## 🚨 CURRENT STATUS

```
Auto-Summary: ❌ DISABLED

Reason:
  SUMMARY_TARGET_CHAT_ID environment variable not set

To Enable:
  1. Get your chat ID: send /start to bot
  2. Set environment: export SUMMARY_TARGET_CHAT_ID=your_chat_id
  3. Restart bot: pkill bot && ./bot &

Note:
  Manual /summary command still works even without scheduler!
```

---

## 🔄 COMPARISON: Manual vs Auto-Summary

| Feature | Manual `/summary` | Auto-Summary |
|---------|------------------|--------------|
| **Trigger** | User command | Automatic (scheduler) |
| **Frequency** | On-demand | Every hour + daily |
| **Target** | User who sent command | SUMMARY_TARGET_CHAT_ID |
| **Time Range** | Last 24h | 1h (hourly) / 1 day (daily) |
| **Source** | Raw messages | Raw (1h) / Aggregated (daily) |
| **Database Save** | ✅ Yes | ✅ Yes |
| **Cleanup** | ❌ No | ✅ Yes (daily only) |
| **Requires Setup** | ❌ No | ✅ Yes (SUMMARY_TARGET_CHAT_ID) |
| **Works Now** | ✅ Yes | ❌ No (not configured) |

---

## 💡 RECOMMENDATIONS

### **For Testing:**
```bash
# Start with short interval for testing
export SUMMARY_TARGET_CHAT_ID=your_chat_id
export DAILY_SUMMARY_TIME=23:00  # Or any time soon

# Start bot
./bot
```

### **For Production:**
```bash
# Use proper time
export SUMMARY_TARGET_CHAT_ID=your_private_chat_id
export DAILY_SUMMARY_TIME=20:00

# Run in background with logs
nohup ./bot > bot.log 2>&1 &
```

### **Multiple Target Chats:**
Currently supports only **ONE target chat**. To send to multiple chats:
- Option 1: Modify code to support array of chat IDs
- Option 2: Use Telegram channel/group as target
- Option 3: Create Telegram bot that forwards summaries

---

## 🎉 CONCLUSION

Auto-summary adalah fitur yang **sangat powerful** untuk:
- ✅ Automatic monitoring 125 groups
- ✅ Hourly summaries (24x per day)
- ✅ Daily aggregated summary
- ✅ Automatic cleanup
- ✅ Centralized delivery

**Status:** Ready to use, hanya perlu set `SUMMARY_TARGET_CHAT_ID`!

**To Enable:**
```bash
export SUMMARY_TARGET_CHAT_ID=6491485169
pkill bot
./bot > bot.log 2>&1 &
```

---

*Last Updated: 2024-12-06*  
*Auto-Summary Version: 1.0*  
*Based on: internal/scheduler/scheduler.go*
