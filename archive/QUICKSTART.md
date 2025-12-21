# ⚡ Quick Start Guide

## 🎯 Hybrid Solution: Scraper + Bot

Anda sekarang punya **2 komponen yang bekerja sama**:

1. **Python Scraper** - Join & scrape groups (pakai Client API)
2. **Go Bot** - Generate & post summaries (pakai Bot API)

---

## 🚀 Setup dalam 5 Menit

### 1. Install Dependencies

```bash
# Python dependencies
pip install telethon python-dotenv

# Go dependencies (sudah ada)
go mod tidy
```

### 2. Start Python Scraper

```bash
cd scraper
python main.py
```

**First time:** Masukkan phone number Anda (dengan kode negara):
```
+628123456789
```

Kemudian masukkan verification code dari Telegram app.

### 3. Join Groups

```bash
> join https://t.me/your_target_group
> join @another_group
```

### 4. Start Listening

```bash
> run
```

**Biarkan running!** Scraper akan otomatis save semua messages.

### 5. (Optional) Start Go Bot

Di terminal baru:
```bash
go run cmd/bot/main.go
```

---

## 📋 Yang Sudah Dikonfigurasi

✅ **API Credentials Anda:**
- API ID: `22527852`
- API Hash: `4f595e6aac7dfe58a2cf6051360c3f14`
- App Name: `cendrawasih`
- Bot Token: `8255703783:AAG4Vq8itkxsoUw4Nx03wb0H8DAIeVzFSy0`
- Gemini API: `AIzaSyAJY8DSWZlpeUidWC_T7z6zR7MLXC1DTDE`

✅ **Database:**
- Shared SQLite: `telegram_bot.db`
- Tables: messages, summaries, tracked_groups

✅ **Logging:**
- Full debug mode enabled
- Console output dengan timestamps

---

## 💡 Cara Kerja

```
1. Python Scraper → Join group via link
2. Python Scraper → Listen & save messages ke database
3. Go Bot → Read messages dari database
4. Go Bot → Generate summary dengan Gemini AI
5. Go Bot → Post summary ke group (coming in Phase 7-8)
```

---

## 🎯 Use Cases

### Use Case 1: Monitor Public Groups

```bash
cd scraper
python main.py

> join https://t.me/tech_news
> join https://t.me/crypto_signals
> join https://t.me/python_devs
> run
```

**Result**: Scraper akan collect semua messages dari 3 groups tersebut.

---

### Use Case 2: Generate Summary

Setelah collect messages beberapa jam:

```bash
# Terminal 1: Scraper tetap running
cd scraper && python main.py
> run

# Terminal 2: Generate summary
go run cmd/bot/main.go
# Bot akan read dari database dan generate summaries
```

---

### Use Case 3: Check Data

```bash
# Berapa messages terkumpul?
sqlite3 telegram_bot.db "SELECT COUNT(*) FROM messages;"

# Messages per group
sqlite3 telegram_bot.db "
SELECT chat_id, COUNT(*) 
FROM messages 
GROUP BY chat_id;
"

# Recent messages
sqlite3 telegram_bot.db "
SELECT username, message_text 
FROM messages 
ORDER BY timestamp DESC 
LIMIT 10;
"
```

---

## ⚠️ Penting!

### 1. **Session File**

File `cendrawasih_session.session` berisi login Anda.

**JANGAN SHARE FILE INI!** Sudah ditambahkan ke `.gitignore`.

### 2. **Rate Limits**

Jangan join terlalu banyak groups sekaligus:
- Max: 5-10 groups per jam
- Jika dapat `FloodWaitError`, tunggu sesuai yang diminta

### 3. **Privacy**

- Gunakan untuk public groups saja
- Jangan scrape private/sensitive data tanpa consent
- Be respectful!

### 4. **Phone Number**

Gunakan phone number sendiri (dengan country code):
- ✅ `+628123456789`
- ❌ `08123456789`

---

## 🔍 Troubleshooting

### "Phone number required"

**Solution:**
```bash
export PHONE_NUMBER="+628123456789"
python main.py
```

### "Flood wait error"

**Solution:** Tunggu waktu yang diminta (e.g., 1 jam), jangan spam join.

### "Channel is private"

**Solution:** Group private, butuh invite dari admin. Tidak bisa join via link.

### Messages tidak tersimpan

**Check:**
- Scraper running dengan `> run`?
- Group actually active?
- Message >= 10 characters?

---

## 📊 Success Indicators

### Scraper Working ✅

```
✅ Logged in as: Your Name (@username)
✅ Message handler registered
💬 [Group Name] user: message text...
💾 Message saved: ID=123
```

### Database Has Data ✅

```bash
$ sqlite3 telegram_bot.db "SELECT COUNT(*) FROM messages;"
150
```

### Bot Working ✅

```
✅ Bot is fully operational!
✅ Database initialized
✅ Gemini client ready
```

---

## 📚 Documentation

- **HYBRID_SETUP.md** - Complete setup guide
- **scraper/README.md** - Scraper details
- **README.md** - Full project overview
- **IMPLEMENTATION_PLAN.md** - Development roadmap

---

## 🎉 Next Steps

1. ✅ **Sekarang**: Collect messages dengan scraper
2. ⏳ **Phase 7-8**: Auto-scheduling summaries
3. ⏳ **Phase 9**: Commands (/summary, /stats)
4. ⏳ **Phase 10**: Production deployment

---

## 💬 Example Session

```bash
$ cd scraper
$ python main.py

[INFO] 2024-12-03 04:00:00 - 🤖 TELEGRAM SCRAPER - Starting
[INFO] 2024-12-03 04:00:01 - ✅ Logged in as: John Doe
[INFO] 2024-12-03 04:00:02 - ✅ Client is ready!

> join https://t.me/python_group
[INFO] 2024-12-03 04:00:05 - ✅ Successfully joined: Python Group

> run
[INFO] 2024-12-03 04:00:10 - 🏃 Running client...

[INFO] 2024-12-03 04:01:00 - 💬 [Python Group] alice: Hello everyone!
[INFO] 2024-12-03 04:01:05 - 💾 Message saved: ID=1
[INFO] 2024-12-03 04:01:20 - 💬 [Python Group] bob: How's Python 3.12?
[INFO] 2024-12-03 04:01:25 - 💾 Message saved: ID=2
```

---

**Version**: 1.0.0 (Hybrid)  
**Status**: Ready to use!  
**Last Updated**: 2024-12-03
