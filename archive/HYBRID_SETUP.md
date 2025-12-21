# 🔧 Hybrid Setup Guide

## Architecture: Python Scraper + Go Bot

This project uses a **hybrid approach**:

1. **Python Scraper (Telegram Client)** - Scrapes messages from groups
2. **Go Bot (Telegram Bot)** - Generates and posts summaries

---

## 📦 Complete Setup

### Step 1: Install Python Dependencies

```bash
pip install telethon python-dotenv
```

### Step 2: Install Go Dependencies

```bash
go mod tidy
```

### Step 3: First Run - Authenticate Scraper

```bash
cd scraper
python main.py
```

**You'll be asked:**
1. Phone number: `+628123456789`
2. Verification code (check Telegram app)
3. 2FA password (if enabled)

This creates a session file for future use.

### Step 4: Join Groups

```bash
> join https://t.me/your_target_group
> join @another_group
> list  # See all joined groups
> run  # Start listening
```

**Keep this running in Terminal 1**

### Step 5: Start Go Bot (New Terminal)

```bash
go run cmd/bot/main.go
```

**Keep this running in Terminal 2**

---

## 🚀 Daily Workflow

### Option A: Both Running Together

**Terminal 1 (Scraper):**
```bash
cd scraper && python main.py
> run
```

**Terminal 2 (Go Bot):**
```bash
go run cmd/bot/main.go
```

**Result:**
- Scraper: Collects all messages from groups
- Go Bot: Generates summaries (manual for now, auto in Phase 7-8)

---

### Option B: Scraper Only (Collect Data)

```bash
cd scraper && python main.py
> join https://t.me/group1
> join https://t.me/group2
> run
```

Let it collect messages for hours/days.

Later, run Go bot to generate summaries from collected data.

---

## 📊 Check Data Collection

```bash
# Check how many messages collected
sqlite3 telegram_bot.db "SELECT COUNT(*) FROM messages;"

# Check messages by group
sqlite3 telegram_bot.db "
SELECT chat_id, COUNT(*) as count 
FROM messages 
GROUP BY chat_id;
"

# Check tracked groups
sqlite3 telegram_bot.db "SELECT * FROM tracked_groups;"

# Recent messages
sqlite3 telegram_bot.db "
SELECT username, message_text, timestamp 
FROM messages 
ORDER BY timestamp DESC 
LIMIT 10;
"
```

---

## 🎯 What Each Component Does

### Python Scraper (`scraper/main.py`)

**Purpose**: Collect messages from Telegram groups

**Can Do:**
- ✅ Join public groups via link
- ✅ Listen to all messages in joined groups
- ✅ Save messages to SQLite
- ✅ Track multiple groups
- ✅ Filter short messages

**Cannot Do:**
- ❌ Post messages (read-only)
- ❌ Join private groups without invite
- ❌ Generate summaries (that's Go bot's job)

---

### Go Bot (`cmd/bot/main.go`)

**Purpose**: Generate and post summaries

**Can Do:**
- ✅ Read messages from SQLite
- ✅ Generate summaries with Gemini AI
- ✅ Post summaries to groups
- ✅ Respond to commands (/start, /help)
- ✅ Track statistics

**Cannot Do:**
- ❌ Join groups by itself (needs admin to add)
- ❌ Collect messages from groups it's not in

---

## 🔄 Integration Flow

```
┌─────────────────────────────────────────┐
│  1. Python Scraper                      │
│     - Join groups via link              │
│     - Listen to messages                │
│     - Save to SQLite                    │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│  2. SQLite Database (Shared)            │
│     - telegram_bot.db                   │
│     - Tables: messages, summaries       │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│  3. Go Bot                              │
│     - Read from database                │
│     - Generate summary with Gemini      │
│     - Save summary to database          │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│  4. Post Summary                        │
│     - Bot posts to Telegram group       │
│     - Or DM to you                      │
└─────────────────────────────────────────┘
```

---

## 🛠️ Useful Commands

### Scraper Commands (in Python scraper)

```bash
join <link>         # Join a group
list                # List all joined groups
run                 # Start listening
quit                # Exit
```

### Bot Commands (in Telegram)

```
/start              # Bot introduction
/help               # Show help
```

**Coming in Phase 7-9:**
```
/summary            # Generate summary now
/summary_4h         # Last 4 hours
/summary_today      # Today's summary
/stats              # Group statistics
```

---

## ⚠️ Important Notes

### 1. Database is Shared

Both Python scraper and Go bot use the **same SQLite database**: `telegram_bot.db`

**Don't run database migrations separately!**

### 2. Session File Security

`cendrawasih_session.session` contains your login credentials.

**Keep it safe! Add to .gitignore:**
```bash
echo "*.session*" >> .gitignore
```

### 3. Two Different Accounts

- **Scraper**: Uses YOUR phone number (user account)
- **Bot**: Uses bot token (@tesstsummm_bot)

They work together but are separate entities.

### 4. Groups Bot Must Be In

For bot to **post summaries**, it must be in the group:
- Ask admin to add @tesstsummm_bot
- Or use scraper for read-only monitoring

---

## 📝 Example Complete Flow

### Day 1: Setup & Data Collection

```bash
# 1. Start scraper
cd scraper
python main.py

# 2. Join groups
> join https://t.me/tech_news
> join https://t.me/crypto_group
> join https://t.me/python_devs

# 3. Start listening
> run

# Let it run for 24 hours...
```

### Day 2: Generate Summary

```bash
# In new terminal
go run cmd/bot/main.go

# Bot will:
# - Read collected messages
# - Generate summaries
# - (Phase 7-9: Auto-post to groups)
```

---

## 🎉 Success Indicators

### Scraper Working:

```
✅ Logged in as: Your Name
✅ Message handler registered
💬 [Group Name] user: message text...
💾 Message saved: ID=123
```

### Bot Working:

```
✅ Bot is fully operational!
✅ Configuration loaded
✅ Database initialized
✅ Gemini client ready
```

### Database Has Data:

```bash
$ sqlite3 telegram_bot.db "SELECT COUNT(*) FROM messages;"
150  # ✅ Messages collected!
```

---

## 🐛 Troubleshooting

### Scraper won't start

**Check:**
- Python 3.8+ installed?
- Dependencies installed? (`pip install -r requirements.txt`)
- Phone number correct format? (`+62xxx`)

### Bot won't start

**Check:**
- Go 1.21+ installed?
- Dependencies installed? (`go mod tidy`)
- Database exists? (`ls telegram_bot.db`)

### No messages being saved

**Check:**
- Scraper running with `> run`?
- Groups actually active?
- Check logs for errors

### "Conflict: terminated by other getUpdates"

**Solution:** Only one bot instance can run at a time. Stop all instances and restart.

---

## 📚 Next Steps

Once you have data collected:

1. **Phase 7-8**: Implement auto-scheduling
2. **Phase 9**: Add summary commands
3. **Phase 10**: Production deployment

See `IMPLEMENTATION_PLAN.md` for details!

---

**Questions?** Check individual READMEs:
- `README.md` - Main project overview
- `scraper/README.md` - Scraper details
- `IMPLEMENTATION_PLAN.md` - Development roadmap
