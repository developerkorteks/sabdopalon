# Changelog

All notable changes to Telegram Chat Summarizer Bot.

## [1.0.0] - 2024-12-06

### 🎉 Major Release - Production Ready

#### Added
- ✅ **Unified Binary** - Merged bot and scraper into single executable
  - Single binary deployment (21 MB)
  - Flexible modes: bot, scraper, or both
  - 32% size reduction from separate binaries
  
- ✅ **18 AI Providers** with automatic fallback
  - Google Gemini (Official - Primary)
  - Yupra.my.id (4 providers)
  - Deline.web.id (3 providers)
  - ElrayyXml.web.id (10 providers)
  - 88.9% success rate (16/18 working)
  - 99.99999% uptime potential
  
- ✅ **Auto-Summary System**
  - Hourly summaries (24x per day)
  - Daily summary at 23:59 WIB
  - Automatic message cleanup (>24h)
  - Hardcoded target chat ID
  
- ✅ **Enhanced Message Processing**
  - Smart filtering (min 10 chars, anti-spam)
  - Markdown escaping (19 special characters)
  - Username escaping
  - Real-time collection from 125+ groups
  
- ✅ **Metadata Extraction**
  - Sentiment analysis
  - Credibility scoring (1-5)
  - Product mention detection
  - Red flags detection

#### Fixed
- 🐛 Markdown parsing errors (parentheses, @, underscores)
- 🐛 Double escaping issues
- 🐛 Username special character handling

#### Changed
- 📦 Project structure reorganized
  - Documentation moved to `docs/`
  - Old docs moved to `archive/`
  - Binaries moved to `bin/`
  - Added `.gitignore`
  
- 🔧 Hardcoded configuration
  - Target chat ID: 6491485169
  - Scheduler always enabled
  - No environment variable needed

#### Performance
- ⚡ Average AI response: 3.4s
- ⚡ Fastest provider: 0.77s (ChatGPT)
- ⚡ Binary size: 31 MB → 21 MB (32% reduction)

### Technical Details

**AI Providers Added:**
- ElrayyXml: Venice AI, PowerBrain AI, Lumin AI, ChatGPT, Perplexity AI, Felo AI, Gemini, Copilot, Alisia AI, BibleGPT (10 providers)
- Deline: Copilot Think, Copilot, OpenAI (3 providers)

**Database:**
- 132 groups tracked
- 4 active groups
- Auto-cleanup after 24h
- SQLite optimization

**Deployment:**
- Single command startup
- Graceful shutdown
- Systemd compatible
- Docker ready

---

## [0.6.0] - 2024-12-04

### Added
- Initial bot implementation
- Basic scraper functionality
- Manual summary command
- Group management commands

### Features
- 5 AI providers (Gemini + Yupra)
- Message collection and filtering
- Database integration
- Basic scheduler

---

**Legend:**
- ✅ Added - New features
- 🐛 Fixed - Bug fixes
- 📦 Changed - Changes in existing functionality
- ⚡ Performance - Performance improvements
- 🔧 Configuration - Configuration changes
