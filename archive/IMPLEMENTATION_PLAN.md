# 🤖 Telegram Bot Chat Summarizer - Implementation Plan

## 📋 Project Overview
Bot Telegram yang merekam chat di group dan membuat summary harian menggunakan Gemini API.

## 🔑 Credentials
- **Telegram Bot Token**: `8255703783:AAG4Vq8itkxsoUw4Nx03wb0H8DAIeVzFSy0`
- **Gemini API Key**: `AIzaSyAJY8DSWZlpeUidWC_T7z6zR7MLXC1DTDE`
- **Gemini Model**: `gemini-2.0-flash`

## 🎯 Strategy
- **Approach**: Incremental Summarization (setiap 4 jam) + Pre-filtering
- **Database**: SQLite
- **Language**: Go
- **Development**: Step-by-step dengan testing di setiap step

---

## 📦 Phase 0: Project Setup & Dependencies
**Goal**: Setup project structure dan install dependencies

### Tasks:
- [x] Create project structure
- [ ] Install Go dependencies:
  - `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Telegram Bot
  - `github.com/mattn/go-sqlite3` - SQLite driver
  - `database/sql` - Standard library
  - `net/http` - For Gemini API calls
- [ ] Create basic project structure:
  ```
  /cmd/bot/main.go          - Entry point
  /internal/bot/handler.go  - Telegram handlers
  /internal/db/sqlite.go    - Database operations
  /internal/gemini/client.go - Gemini API client
  /internal/summarizer/summarizer.go - Summarization logic
  /internal/config/config.go - Configuration
  /internal/logger/logger.go - Logging utility
  ```

### Testing:
```bash
go mod init telegram-summarizer
go mod tidy
go run cmd/bot/main.go --version
```

**Status**: ⏳ Pending

---

## 📦 Phase 1: Logger & Config Setup
**Goal**: Setup logging dan configuration yang solid

### Tasks:
- [ ] Create logger with debug mode
  - Console output dengan timestamp
  - Different log levels (DEBUG, INFO, WARN, ERROR)
  - File logging optional
- [ ] Create config structure
  - Read from environment variables
  - Hardcoded fallback untuk development

### Testing:
```go
// Test logging
logger.Debug("Test debug message")
logger.Info("Test info message")
logger.Error("Test error message")

// Test config
cfg := config.Load()
fmt.Printf("Bot Token: %s\n", cfg.TelegramToken[:10]+"...")
```

**Expected Output**:
```
[DEBUG] 2024-01-15 10:00:00 - Test debug message
[INFO]  2024-01-15 10:00:00 - Test info message
[ERROR] 2024-01-15 10:00:00 - Test error message
Bot Token: 8255703783...
```

**Status**: ⏳ Pending

---

## 📦 Phase 2: SQLite Database Setup
**Goal**: Create database schema dan basic CRUD operations

### Tasks:
- [ ] Create database schema:
  ```sql
  CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    username TEXT,
    message_text TEXT NOT NULL,
    message_length INTEGER,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE summaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id INTEGER NOT NULL,
    summary_type TEXT, -- 'incremental' or 'daily'
    period_start DATETIME,
    period_end DATETIME,
    summary_text TEXT,
    message_count INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );

  CREATE INDEX idx_messages_chat_time ON messages(chat_id, timestamp);
  CREATE INDEX idx_summaries_chat_time ON summaries(chat_id, period_start);
  ```

- [ ] Implement database functions:
  - `InitDB()` - Initialize database
  - `SaveMessage()` - Save single message
  - `GetMessagesByTimeRange()` - Get messages for summarization
  - `SaveSummary()` - Save summary result
  - `GetLastSummaryTime()` - Get last summary timestamp

### Testing:
```go
// Test 1: Initialize DB
db, err := InitDB("test.db")
if err != nil {
    log.Fatal(err)
}

// Test 2: Save message
err = SaveMessage(db, Message{
    ChatID: 123456,
    UserID: 789,
    Username: "testuser",
    MessageText: "Hello, this is a test message",
})

// Test 3: Retrieve messages
messages, err := GetMessagesByTimeRange(db, 123456, time.Now().Add(-1*time.Hour), time.Now())
fmt.Printf("Found %d messages\n", len(messages))

// Test 4: Save summary
err = SaveSummary(db, Summary{
    ChatID: 123456,
    SummaryType: "incremental",
    SummaryText: "Test summary",
    MessageCount: 10,
})
```

**Expected Output**:
```
[INFO] Database initialized: test.db
[INFO] Message saved: ID=1, ChatID=123456
[INFO] Found 1 messages
[INFO] Summary saved: ID=1
```

**Status**: ⏳ Pending

---

## 📦 Phase 3: Telegram Bot Basic Connection
**Goal**: Connect bot ke Telegram dan test basic message receiving

### Tasks:
- [ ] Initialize Telegram bot
- [ ] Setup message handler
- [ ] Log setiap message yang diterima (belum save ke DB)
- [ ] Implement `/start` command
- [ ] Implement `/help` command

### Testing:
```go
// Start bot
bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
bot.Debug = true

// Test di Telegram:
// 1. Add bot ke group test
// 2. Send message: "Hello bot"
// 3. Send command: /start
// 4. Send command: /help
```

**Expected Output** (Console):
```
[INFO] Authorized on account: YourBotName
[DEBUG] Received message: ChatID=123456, UserID=789, Text="Hello bot"
[DEBUG] Received command: /start from user 789
[INFO] Bot is running... Press Ctrl+C to stop
```

**Status**: ⏳ Pending

---

## 📦 Phase 4: Message Storage Integration
**Goal**: Integrate Telegram bot dengan database untuk save messages

### Tasks:
- [ ] Connect Phase 3 (Bot) dengan Phase 2 (Database)
- [ ] Implement message filtering:
  - Minimum length: 10 characters
  - Skip bot messages
  - Skip commands
  - Skip media-only messages
- [ ] Save filtered messages to database
- [ ] Add logging untuk setiap message saved

### Testing:
```
// Test di Telegram Group:
1. Send: "hi" → Should be filtered (too short)
2. Send: "/test" → Should be filtered (command)
3. Send: "This is a longer message that should be saved" → Should be saved
4. Send: "Another important message here" → Should be saved
5. Check database: SELECT * FROM messages;
```

**Expected Output** (Console):
```
[DEBUG] Message received: "hi" - FILTERED (too short)
[DEBUG] Message received: "/test" - FILTERED (command)
[INFO] Message saved: ID=1, User=testuser, Length=45
[INFO] Message saved: ID=2, User=testuser, Length=32
```

**Expected Output** (Database):
```sql
sqlite> SELECT id, username, message_text FROM messages;
1|testuser|This is a longer message that should be saved
2|testuser|Another important message here
```

**Status**: ⏳ Pending

---

## 📦 Phase 5: Gemini API Client
**Goal**: Create working Gemini API client untuk summarization

### Tasks:
- [ ] Create Gemini HTTP client
- [ ] Implement `GenerateSummary()` function
- [ ] Add retry logic (3 attempts)
- [ ] Error handling dan logging
- [ ] Test dengan sample text

### Testing:
```go
// Test 1: Simple summarization
client := gemini.NewClient(config.GeminiAPIKey)
summary, err := client.GenerateSummary("This is a test message. Another message. Third message.")
fmt.Printf("Summary: %s\n", summary)

// Test 2: Longer text
longText := `
User1: Hey guys, what time is the meeting?
User2: I think it's at 2 PM
User1: Thanks! Don't forget to bring the documents
User3: Will do. See you there.
`
summary, err = client.GenerateSummary(longText)
fmt.Printf("Summary: %s\n", summary)
```

**Expected Output**:
```
[INFO] Calling Gemini API...
[DEBUG] Request: {prompt length: 45}
[DEBUG] Response: {candidates: 1, tokens: 25}
[INFO] Summary generated successfully
Summary: Users discussed meeting time (2 PM) and reminded each other about documents.
```

**Status**: ⏳ Pending

---

## 📦 Phase 6: Incremental Summarizer Logic
**Goal**: Implement incremental summarization (every 4 hours)

### Tasks:
- [ ] Create summarizer service
- [ ] Implement 4-hour window logic
- [ ] Format messages untuk Gemini:
  ```
  [10:00] username1: message text
  [10:05] username2: message text
  ```
- [ ] Create summary prompt template
- [ ] Save incremental summaries to database

### Testing:
```go
// Test dengan mock data
// Populate database dengan messages dari 4 jam terakhir
for i := 0; i < 20; i++ {
    SaveMessage(db, Message{
        ChatID: 123456,
        Username: fmt.Sprintf("user%d", i%3),
        MessageText: fmt.Sprintf("Test message number %d", i),
        Timestamp: time.Now().Add(-time.Hour * time.Duration(i/5)),
    })
}

// Run summarizer
summarizer := NewSummarizer(db, geminiClient)
summary, err := summarizer.CreateIncrementalSummary(123456, 4*time.Hour)
fmt.Printf("Summary: %s\n", summary)

// Check database
summaries, _ := GetSummaries(db, 123456)
fmt.Printf("Total summaries: %d\n", len(summaries))
```

**Expected Output**:
```
[INFO] Creating incremental summary for chat 123456
[DEBUG] Found 20 messages in last 4 hours
[DEBUG] Formatted text length: 450 characters
[INFO] Calling Gemini API for summarization...
[INFO] Summary created and saved to database
Summary: Users discussed various topics including...
Total summaries: 1
```

**Status**: ⏳ Pending

---

## 📦 Phase 7: Daily Summary Merger
**Goal**: Merge incremental summaries menjadi daily summary

### Tasks:
- [ ] Implement daily summary logic
- [ ] Get all incremental summaries dari 24 jam terakhir
- [ ] Merge dengan Gemini
- [ ] Better prompt untuk daily summary:
  ```
  Format:
  📌 TOPIK UTAMA:
  💬 DISKUSI PENTING:
  📋 ACTION ITEMS:
  ⚡ HIGHLIGHT:
  ```
- [ ] Save daily summary

### Testing:
```go
// Mock: Create 6 incremental summaries (4 jam x 6 = 24 jam)
for i := 0; i < 6; i++ {
    SaveSummary(db, Summary{
        ChatID: 123456,
        SummaryType: "incremental",
        SummaryText: fmt.Sprintf("Incremental summary %d: Users discussed topic %d", i, i),
        PeriodStart: time.Now().Add(-time.Hour * time.Duration(4*(6-i))),
        PeriodEnd: time.Now().Add(-time.Hour * time.Duration(4*(5-i))),
    })
}

// Create daily summary
dailySummary, err := summarizer.CreateDailySummary(123456)
fmt.Printf("Daily Summary:\n%s\n", dailySummary)
```

**Expected Output**:
```
[INFO] Creating daily summary for chat 123456
[DEBUG] Found 6 incremental summaries
[INFO] Merging summaries with Gemini...
[INFO] Daily summary created

Daily Summary:
📌 TOPIK UTAMA:
- Topic 1: Discussion about...
- Topic 2: Planning for...

💬 DISKUSI PENTING:
- User1 mentioned important point about...

📋 ACTION ITEMS:
- Follow up on task X

⚡ HIGHLIGHT:
- Interesting discussion about...
```

**Status**: ⏳ Pending

---

## 📦 Phase 8: Scheduler Implementation
**Goal**: Automatic scheduling untuk incremental dan daily summaries

### Tasks:
- [ ] Implement ticker untuk incremental summary (every 4 hours)
- [ ] Implement ticker untuk daily summary (every 24 hours at specific time)
- [ ] Add configuration untuk schedule time
- [ ] Post summary ke group atau admin
- [ ] Error handling dan recovery

### Testing:
```go
// Test dengan interval pendek untuk development
// Normal: 4 hours, Testing: 2 minutes
scheduler := NewScheduler(summarizer, bot)
scheduler.StartIncremental(2 * time.Minute) // Test mode
scheduler.StartDaily("23:59") // Daily at 23:59

// Wait and observe logs
// After 2 minutes:
// [INFO] Running incremental summary for all active chats...
// [INFO] Sending summary to chat 123456...
```

**Expected Output** (in Telegram Group):
```
🤖 Summary (Last 4 Hours)

Users discussed various topics:
- Topic A: 5 messages
- Topic B: 10 messages

Most active: user1 (8 messages)
```

**Status**: ⏳ Pending

---

## 📦 Phase 9: Command Interface
**Goal**: Add commands untuk manual control

### Tasks:
- [ ] `/summary` - Generate summary on demand
- [ ] `/summary_4h` - Last 4 hours summary
- [ ] `/summary_today` - Today's summary
- [ ] `/stats` - Statistics (message count, active users)
- [ ] `/help` - Show all commands
- [ ] Admin-only commands (optional)

### Testing:
```
Test di Telegram:
1. /summary → Should show daily summary
2. /summary_4h → Should show last 4 hours
3. /summary_today → Should show today's summary
4. /stats → Should show statistics
5. /help → Should show command list
```

**Expected Output** (Telegram):
```
/stats
📊 Statistics for this group:
- Total messages today: 150
- Most active user: user1 (45 messages)
- Last summary: 2 hours ago
- Messages since last summary: 30
```

**Status**: ⏳ Pending

---

## 📦 Phase 10: Polish & Production Ready
**Goal**: Final touches, error handling, dan production readiness

### Tasks:
- [ ] Add graceful shutdown
- [ ] Add database backup routine
- [ ] Add rate limiting untuk Gemini API
- [ ] Improve error messages
- [ ] Add health check endpoint (optional)
- [ ] Create systemd service file (Linux)
- [ ] Documentation dalam README.md
- [ ] Cleanup temporary files

### Testing:
```bash
# Test graceful shutdown
go run cmd/bot/main.go &
PID=$!
sleep 5
kill -SIGTERM $PID
# Should see: [INFO] Shutting down gracefully...

# Test restart after crash
# Kill database connection
# Bot should recover and reconnect
```

**Status**: ⏳ Pending

---

## 🎯 Success Criteria
- [ ] Bot can receive and store messages from Telegram group
- [ ] Incremental summaries generated every 4 hours
- [ ] Daily summaries generated at 23:59
- [ ] Summaries posted to Telegram group
- [ ] All commands working correctly
- [ ] Full debug logging implemented
- [ ] Database persists correctly
- [ ] Bot recovers from errors gracefully

---

## 📝 Development Notes
- Setiap phase harus PASS testing sebelum lanjut ke phase berikutnya
- Gunakan `tmp_rovodev_*` untuk temporary test files
- Full logging di setiap operation untuk debugging
- Test di private group dulu sebelum production

---

## 🚀 Quick Start Commands

```bash
# Initialize project
go mod init telegram-summarizer

# Run bot
go run cmd/bot/main.go

# Run with debug
go run cmd/bot/main.go --debug

# Build binary
go build -o bot cmd/bot/main.go

# Run binary
./bot
```

---

**Last Updated**: [Will be updated at each phase completion]
**Current Phase**: Phase 0 - Project Setup
