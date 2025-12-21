package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"telegram-summarizer/internal/bot"
	"telegram-summarizer/internal/config"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/gemini"
	"telegram-summarizer/internal/logger"
	"telegram-summarizer/internal/scheduler"
	"telegram-summarizer/internal/summarizer"
)

var (
	version     = "0.6.0"
	showVersion = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("Telegram Summarizer Bot v%s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.DebugMode)

	logger.Info("🤖 Starting Telegram Summarizer Bot v%s", version)
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("Configuration error: %v", err)
		os.Exit(1)
	}
	
	logger.Info("✅ Configuration loaded")
	logger.Debug("  Telegram Token: %s...", cfg.TelegramToken[:20])
	logger.Debug("  Gemini API Key: %s...", cfg.GeminiAPIKey[:20])
	logger.Debug("  Gemini Model: %s", cfg.GeminiModel)
	logger.Debug("  Database Path: %s", cfg.DatabasePath)
	logger.Info("  Summary Interval: %d hours", cfg.SummaryInterval)
	logger.Info("  Daily Summary Time: %s", cfg.DailySummaryTime)

	// Initialize database
	logger.Info("\n📦 Initializing database...")
	database, err := db.InitDB(cfg.DatabasePath)
	if err != nil {
		logger.Error("Failed to initialize database: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create Gemini client
	logger.Info("\n🧠 Initializing Gemini AI client...")
	geminiClient := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	logger.Info("✅ Gemini client ready")

	// Create summarizer
	logger.Info("\n📝 Initializing summarizer service...")
	summarizerService := summarizer.NewSummarizer(database, geminiClient)
	logger.Info("✅ Summarizer service ready")

	// Create message handler
	logger.Info("\n💬 Initializing message handler...")
	messageHandler := bot.NewMessageHandler(database)
	logger.Info("✅ Message handler ready")

	// Create Telegram bot (temporarily without command handler)
	logger.Info("\n🤖 Connecting to Telegram...")
	telegramBot, err := bot.NewBot(cfg.TelegramToken, cfg.DebugMode, messageHandler, nil)
	if err != nil {
		logger.Error("Failed to create bot: %v", err)
		os.Exit(1)
	}
	
	// Create command handler
	logger.Info("\n🔧 Initializing command handler...")
	commandHandler := bot.NewCommandHandler(telegramBot, database)
	telegramBot.SetCommandHandler(commandHandler)
	telegramBot.SetSummarizer(summarizerService)
	logger.Info("✅ Command handler ready")

	// Hardcoded target chat ID for auto-summaries
	targetChatID := int64(6491485169) // Hardcoded target chat ID
	
	// Create and start scheduler
	var summaryScheduler *scheduler.Scheduler
	logger.Info("\n📅 Initializing daily summary scheduler...")
	logger.Info("   Target Chat ID: %d (hardcoded)", targetChatID)
	summaryScheduler = scheduler.NewScheduler(database, summarizerService, telegramBot.GetAPI(), targetChatID)
	summaryScheduler.Start(cfg.DailySummaryTime)
	logger.Info("✅ Scheduler ready (Daily summary at %s)", cfg.DailySummaryTime)

	// Setup graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("\n\n🛑 Shutting down gracefully...")
		if summaryScheduler != nil {
			summaryScheduler.Stop()
		}
		telegramBot.Stop()
		database.Close()
		logger.Info("✅ Cleanup complete. Goodbye!")
		os.Exit(0)
	}()

	logger.Info("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("✅ ✅ ✅ Bot is fully operational!")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("\n📱 Bot Features:")
	logger.Info("  • Automatically saves all group messages")
	logger.Info("  • Filters out short messages and spam")
	logger.Info("  • Commands: /start, /help, /listgroups")
	logger.Info("  • Group management: /enable, /disable, /groupstats")
	logger.Info("\n🔧 Available Commands:")
	logger.Info("  /start - Bot introduction")
	logger.Info("  /help - Show help")
	logger.Info("  /listgroups - List all tracked groups")
	logger.Info("  /enable <chat_id> - Enable summarization for a group")
	logger.Info("  /disable <chat_id> - Disable summarization")
	logger.Info("  /groupstats - Show group statistics")
	logger.Info("  /summary <chat_id> - Generate 24h summary for a group")
	logger.Info("\n⚠️  Note: Make sure scraper is running to collect messages!")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Start bot (blocks until stopped)
	if err := telegramBot.Start(); err != nil {
		logger.Error("Bot error: %v", err)
		os.Exit(1)
	}
}
