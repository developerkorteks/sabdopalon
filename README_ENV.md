# 🔧 Environment Configuration Guide

## 📋 Quick Start

### 1. **Setup Environment Variables**

```bash
# Copy example file
cp .env.example .env

# Edit with your values
nano .env
```

### 2. **Run the Bot**

#### **Option A: Using run.sh (Recommended)**
```bash
# Make script executable
chmod +x run.sh

# Run bot (loads .env automatically)
./run.sh
```

#### **Option B: Using Make**
```bash
# Export env vars manually
export $(cat .env | grep -v '^#' | xargs)

# Run bot
make run
```

#### **Option C: Manual Export**
```bash
# Export each variable
export TELEGRAM_BOT_TOKEN="your_token"
export PHONE_NUMBER="+6287742028130"
export GEMINI_API_KEY="your_key"

# Run binary
./bin/telegram-summarizer
```

---

## 🔑 Configuration Variables

### **Required:**
- `TELEGRAM_BOT_TOKEN` - Bot token from @BotFather
- `PHONE_NUMBER` - Your phone number (+CountryCode format)
- `GEMINI_API_KEY` - API key from Google AI Studio

### **Optional:**
- `GEMINI_MODEL` - Model name (default: gemini-2.0-flash-exp)
- `DATABASE_PATH` - Database file (default: telegram_bot.db)
- `DEBUG_MODE` - Enable debug logs (default: false)
- `DAILY_SUMMARY_TIME` - Daily summary time (default: 23:59)
- `TARGET_CHAT_ID` - Chat ID for daily summaries
- `MONITOR_BOT_TOKEN` - Monitoring bot token
- `MONITOR_CHAT_ID` - Chat ID for logs

---

## ⚠️ Important Notes

### **1. .env File Loading**

This project does NOT use automatic .env loading packages. You must either:

**A. Use `run.sh` script (Automatic)**
```bash
./run.sh
```

**B. Export manually (Every session)**
```bash
export $(cat .env | grep -v '^#' | xargs)
./bin/telegram-summarizer
```

**C. Use hardcoded fallbacks in code**
- Default values are set in `internal/config/config.go`
- Will be used if environment variables are not set

---

### **2. API Key Issues**

If you see "API key expired" error:

1. **Check if .env is loaded:**
   ```bash
   echo $GEMINI_API_KEY
   ```
   
2. **If empty, load it:**
   ```bash
   export $(cat .env | grep -v '^#' | xargs)
   echo $GEMINI_API_KEY  # Should show your key now
   ```

3. **Verify key in config:**
   ```bash
   # Check hardcoded fallback in config.go
   grep "GEMINI_API_KEY" internal/config/config.go
   ```

---

### **3. Run Modes**

#### **Mode: all (Default)**
```bash
./run.sh --mode all
# Runs both bot and scraper
```

#### **Mode: bot**
```bash
./run.sh --mode bot
# Runs only bot (no scraper, no phone needed)
```

#### **Mode: scraper**
```bash
./run.sh --mode scraper
# Runs only scraper (needs phone number)
```

---

## 🔍 Troubleshooting

### **Problem: API Key Expired**
```bash
# 1. Check .env file
cat .env | grep GEMINI_API_KEY

# 2. Create new key at:
#    https://aistudio.google.com/app/apikey

# 3. Update .env
nano .env

# 4. Reload env vars
export $(cat .env | grep -v '^#' | xargs)

# 5. Restart bot
./run.sh
```

### **Problem: Phone Number Not Found**
```bash
# 1. Check .env
cat .env | grep PHONE_NUMBER

# 2. Or pass via command line
./run.sh --phone +6287742028130
```

### **Problem: Bot Token Invalid**
```bash
# 1. Get new token from @BotFather
# 2. Update .env
nano .env

# 3. Reload and restart
export $(cat .env | grep -v '^#' | xargs)
./run.sh
```

---

## 📝 Examples

### **Production Run:**
```bash
# 1. Ensure .env is configured
cat .env

# 2. Run with automatic loading
./run.sh

# 3. Monitor logs
tail -f logs/bot.log
```

### **Development Run:**
```bash
# 1. Enable debug mode in .env
echo "DEBUG_MODE=true" >> .env

# 2. Run
./run.sh

# 3. Watch detailed logs
```

### **Bot Only (No Scraper):**
```bash
./run.sh --mode bot
# Good for testing commands without scraping
```

---

## ✅ Checklist

- [ ] Created `.env` file from `.env.example`
- [ ] Added Telegram bot token
- [ ] Added phone number
- [ ] Added Gemini API key
- [ ] Made `run.sh` executable (`chmod +x run.sh`)
- [ ] Tested loading: `export $(cat .env | grep -v '^#' | xargs)`
- [ ] Verified: `echo $GEMINI_API_KEY`
- [ ] Run bot: `./run.sh`
- [ ] Check no "API key expired" error

---

## 🚀 Ready to Go!

Once `.env` is configured and loaded:

```bash
./run.sh
```

Bot will start with all your configuration! 🎉
