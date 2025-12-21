# 📊 Telegram Monitoring System

## Overview

Sistem monitoring otomatis yang mengirim semua logs dan summary ke bot Telegram monitoring secara real-time.

## Features

✅ **Real-time Logs** - Semua logs (INFO, DEBUG, WARN, ERROR) dikirim ke Telegram
✅ **Batch Processing** - Logs di-batch untuk efisiensi (20 logs atau 5 detik)
✅ **Summary Notifications** - Setiap summary yang dibuat dikirim ke monitoring bot
✅ **Auto-flush** - Logs di-flush otomatis saat shutdown
✅ **Rate Limit Protection** - Built-in protection untuk avoid Telegram rate limits
✅ **Message Splitting** - Long messages di-split otomatis (max 4000 chars)

---

## Configuration

### Environment Variables (Optional)

```bash
# Custom monitoring bot (optional, ada default)
export MONITOR_BOT_TOKEN="your_monitoring_bot_token"
export MONITOR_CHAT_ID="your_telegram_user_id"
```

### Default Configuration

Jika tidak di-set, akan menggunakan default:
- **Bot Token**: `8458117186:AAGywdxpEdRqgM2_8rgUi1Ch8TPqdFOszNY`
- **Chat ID**: `6491485169` (Your Telegram ID)

---

## How It Works

### 1. Log Flow

```
Application Log
  ↓
logger.Info("message")
  ↓
Console Output (stdout)
  ↓
Telegram Notifier Buffer
  ↓ (Batch: 20 logs or 5 seconds)
Telegram Bot API
  ↓
Monitoring Bot sends to your Telegram
```

### 2. Summary Flow

```
User: /summary 3285318090
  ↓
Generate Summary (with progress updates)
  ↓
Send to user
  ↓
ALSO send to Monitoring Bot
  ↓
You receive notification with full summary
```

---

## Log Format

### Console Logs (Standard)
```
[INFO ] 2025-12-07 01:17:27 - 🤖 TELEGRAM SUMMARIZER - UNIFIED
[DEBUG] 2025-12-07 01:17:27 - ✅ Tables created successfully
[WARN ] 2025-12-07 01:17:28 - Rate limit approaching
[ERROR] 2025-12-07 01:17:29 - Failed to connect to API
```

### Telegram Logs (With Emoji)
```
ℹ️ [01:17:27] 🤖 TELEGRAM SUMMARIZER - UNIFIED
🔍 [01:17:27] ✅ Tables created successfully
⚠️ [01:17:28] Rate limit approaching
❌ [01:17:29] Failed to connect to API
```

### Log Levels & Emojis

| Level | Emoji | Description |
|-------|-------|-------------|
| INFO  | ℹ️    | General information |
| DEBUG | 🔍    | Debug details |
| WARN  | ⚠️    | Warnings |
| ERROR | ❌    | Errors |
| FATAL | 🚨    | Critical errors |

---

## Summary Notifications

When a summary is generated, you'll receive:

```
📊 Summary Generated

Group: (λ)³

📝 Summary for (λ)³

📅 Period: 2025-12-06 15:52 - 16:52
💬 Messages: 98
👥 Active Users: 25

━━━━━━━━━━━━━━━━━━━━━━━

## 📅 RINGKASAN 24 JAM
...
(Full summary content)
```

---

## Batching & Performance

### Batch Settings

- **Batch Size**: 20 logs per message
- **Flush Time**: 5 seconds
- **Max Message Length**: 4000 characters (Telegram limit: 4096)

### Example Batching

```
Log 1: ℹ️ [01:17:27] Starting bot...
Log 2: ℹ️ [01:17:27] Database initialized
Log 3: 🔍 [01:17:27] Connected to Telegram
...
Log 20: ℹ️ [01:17:32] Bot ready

↓ (After 20 logs OR 5 seconds)

Sent as ONE message to Telegram
```

---

## API Calls & Rate Limits

### Normal Operation

- **Logs**: ~1 message per 5 seconds = ~720 messages/hour
- **Summaries**: 1 message per summary generated
- **Telegram Limit**: 30 messages/second (we're well below this)

### During Heavy Activity

- Logs are buffered
- Auto-batching prevents rate limit
- Built-in 100ms delay between split messages

---

## Usage Examples

### Monitoring Different Services

#### Monitor Bot Only
```bash
./bin/telegram-summarizer --mode bot
```
You'll receive:
- Bot startup logs
- Command execution logs
- Summary generation logs
- Error logs

#### Monitor Scraper Only
```bash
./bin/telegram-summarizer --mode scraper --phone +628123456789
```
You'll receive:
- Scraper startup logs
- Message scraping logs
- Group join/leave logs
- Connection status

#### Monitor Both (Default)
```bash
./bin/telegram-summarizer --phone +628123456789
```
You'll receive:
- All logs from both services
- Complete system monitoring

---

## Filtering Logs (Optional)

### Disable DEBUG logs in Telegram

Modify `internal/logger/telegram_notifier.go`:

```go
// In SendLog method, add filter
func (tn *TelegramNotifier) SendLog(level, message string) {
    // Skip DEBUG logs
    if level == "DEBUG" {
        return
    }
    
    // ... rest of code
}
```

### Only Send Errors & Warnings

```go
// Only send WARN, ERROR, FATAL
func (tn *TelegramNotifier) SendLog(level, message string) {
    if level != "WARN" && level != "ERROR" && level != "FATAL" {
        return
    }
    
    // ... rest of code
}
```

---

## Troubleshooting

### No Logs Received

**Check:**
1. Bot token is correct
2. Chat ID is correct (your Telegram user ID)
3. You've started the monitoring bot (`/start`)
4. Bot has permission to send messages

**Test:**
```bash
# Send test message
curl -X POST "https://api.telegram.org/bot8458117186:AAGywdxpEdRqgM2_8rgUi1Ch8TPqdFOszNY/sendMessage" \
  -d "chat_id=6491485169" \
  -d "text=Test message"
```

### Logs Delayed

**Normal Behavior:**
- Logs are batched (5 seconds delay is normal)
- Force flush on shutdown

**Manual Flush:**
```go
logger.FlushTelegramLogs()
```

### Rate Limit Errors

**Symptoms:**
```
Failed to send log to Telegram: Too Many Requests: retry after X
```

**Solution:**
- Increase `flushTime` (e.g., 10 seconds)
- Increase `batchSize` (e.g., 50 logs)
- Filter DEBUG logs

---

## Advanced Configuration

### Custom Batch Settings

Edit `internal/logger/telegram_notifier.go`:

```go
notifier = &TelegramNotifier{
    bot:       bot,
    chatID:    chatID,
    enabled:   true,
    buffer:    make([]string, 0, 100),  // Buffer size
    lastSend:  time.Now(),
    batchSize: 50,                      // Send every 50 logs
    flushTime: 10 * time.Second,        // Or every 10 seconds
}
```

### Multiple Monitoring Bots

You can send logs to multiple bots:

1. Modify `InitTelegramNotifier` to accept multiple chat IDs
2. Loop through chat IDs in `sendMessage`

---

## Benefits

✅ **Remote Monitoring** - Monitor dari Telegram, tidak perlu SSH
✅ **Real-time Alerts** - Terima error notifications instantly
✅ **Summary Archive** - All summaries tersimpan di chat history
✅ **Mobile Friendly** - Monitor dari HP
✅ **No External Services** - Pure Telegram, tidak perlu Grafana/Prometheus
✅ **Zero Configuration** - Works out of the box with defaults

---

## Security Notes

⚠️ **Bot Token Exposure**
- Bot token hardcoded di code (untuk kemudahan)
- Jika public repo, use environment variables instead

⚠️ **Log Content**
- Logs might contain sensitive info (API keys truncated in DEBUG)
- Be careful with DEBUG mode in production

⚠️ **Chat ID Privacy**
- Your Telegram user ID is visible in code
- Not a security risk, but be aware

---

## Summary

Monitoring system sudah fully integrated! Setiap log yang muncul di terminal akan otomatis dikirim ke monitoring bot Telegram Anda. 

**No configuration needed** - Langsung jalan dengan default settings!

**To test:**
```bash
./bin/telegram-summarizer --phone +628123456789
```

Check your Telegram - you should start receiving logs! 🚀
