package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"telegram-summarizer/internal/bot"
	"telegram-summarizer/internal/client"
	"telegram-summarizer/internal/config"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/gemini"
	"telegram-summarizer/internal/logger"
	"telegram-summarizer/internal/scheduler"
	"telegram-summarizer/internal/summarizer"
)

var (
	version     = "1.0.0"
	showVersion = flag.Bool("version", false, "Show version information")
	mode        = flag.String("mode", "all", "Run mode: 'bot', 'scraper', or 'all' (default: all)")
	phone       = flag.String("phone", "", "Phone number for scraper (with country code)")
)

func main() {
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("Telegram Summarizer (Unified) v%s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.DebugMode)
	
	// Initialize Telegram notifier for remote logging
	monitorBotToken := os.Getenv("MONITOR_BOT_TOKEN")
	monitorChatID := os.Getenv("MONITOR_CHAT_ID")
	
	// Use default monitoring bot if not specified
	if monitorBotToken == "" {
		monitorBotToken = "8458117186:AAGywdxpEdRqgM2_8rgUi1Ch8TPqdFOszNY"
	}
	if monitorChatID == "" {
		monitorChatID = "6491485169" // Your Telegram user ID
	}
	
	// Convert chat ID to int64
	var chatID int64
	if _, err := fmt.Sscanf(monitorChatID, "%d", &chatID); err == nil {
		if err := logger.InitTelegramNotifier(monitorBotToken, chatID); err != nil {
			logger.Warn("Failed to initialize Telegram notifier: %v", err)
		}
	}

	logger.Info("═══════════════════════════════════════════════════════")
	logger.Info("🤖 TELEGRAM SUMMARIZER - UNIFIED")
	logger.Info("Version: %s", version)
	logger.Info("Mode: %s", *mode)
	logger.Info("═══════════════════════════════════════════════════════")

	// Validate mode
	if *mode != "bot" && *mode != "scraper" && *mode != "all" {
		logger.Error("Invalid mode: %s. Use 'bot', 'scraper', or 'all'", *mode)
		os.Exit(1)
	}

	// Initialize database (shared by both bot and scraper)
	logger.Info("\n📦 Initializing database...")
	database, err := db.InitDB(cfg.DatabasePath)
	if err != nil {
		logger.Error("Failed to initialize database: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("\n\n🛑 Shutting down gracefully...")
		logger.FlushTelegramLogs()
		cancel()
	}()

	// Start services based on mode
	switch *mode {
	case "bot":
		runBot(cfg, database, ctx)
	case "scraper":
		runScraper(cfg, database, ctx)
	case "all":
		// Run both bot and scraper in parallel
		go runScraper(cfg, database, ctx)
		runBot(cfg, database, ctx)
	}
}

func runBot(cfg *config.Config, database *db.DB, ctx context.Context) {
	logger.Info("\n🤖 Starting BOT service...")
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

	// Create Telegram bot
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
	targetChatID := int64(6491485169)

	// Create and start scheduler
	logger.Info("\n📅 Initializing daily summary scheduler...")
	logger.Info("   Target Chat ID: %d (hardcoded)", targetChatID)
	summaryScheduler := scheduler.NewScheduler(database, summarizerService, telegramBot.GetAPI(), targetChatID)
	summaryScheduler.Start(cfg.DailySummaryTime)
	logger.Info("✅ Scheduler ready (Daily summary at %s)", cfg.DailySummaryTime)

	// Setup graceful shutdown for bot
	go func() {
		<-ctx.Done()
		logger.Info("\n🛑 Stopping bot service...")
		if summaryScheduler != nil {
			summaryScheduler.Stop()
		}
		telegramBot.Stop()
		logger.Info("✅ Bot service stopped")
	}()

	logger.Info("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("✅ ✅ ✅ Bot is fully operational!")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("\n📱 Bot Features:")
	logger.Info("  • Automatically saves all group messages")
	logger.Info("  • Filters out short messages and spam")
	logger.Info("  • Commands: /start, /help, /listgroups")
	logger.Info("  • Group management: /enable, /disable, /groupstats")
	logger.Info("  • AI Summarization: 18 providers with fallback")
	logger.Info("  • Auto-summary: Hourly + Daily (23:59)")
	logger.Info("\n🔧 Available Commands:")
	logger.Info("  /start - Bot introduction")
	logger.Info("  /help - Show help")
	logger.Info("  /listgroups - List all tracked groups")
	logger.Info("  /enable <chat_id> - Enable summarization for a group")
	logger.Info("  /disable <chat_id> - Disable summarization")
	logger.Info("  /groupstats - Show group statistics")
	logger.Info("  /summary <chat_id> - Generate 24h summary for a group")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Start bot (blocks until stopped)
	if err := telegramBot.Start(); err != nil {
		if ctx.Err() == context.Canceled {
			logger.Info("Bot stopped by context cancellation")
		} else {
			logger.Error("Bot error: %v", err)
			os.Exit(1)
		}
	}
}

func runScraper(cfg *config.Config, database *db.DB, ctx context.Context) {
	logger.Info("\n📱 Starting SCRAPER service...")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Phone number
	phoneNumber := *phone
	if phoneNumber == "" {
		phoneNumber = os.Getenv("PHONE_NUMBER")
	}
	if phoneNumber == "" {
		fmt.Print("\n📱 Enter your phone number (with country code, e.g. +628123456789): ")
		fmt.Scanln(&phoneNumber)
	}

	logger.Info("Phone: %s", phoneNumber)

	// Create client
	logger.Info("\n📱 Initializing Telegram Client...")

	// API CREDENTIALS
	apiID := 22527852
	apiHash := "4f595e6aac7dfe58a2cf6051360c3f14"

	telegramClient := client.NewClient(client.Config{
		AppID:      apiID,
		AppHash:    apiHash,
		Phone:      phoneNumber,
		SessionDir: ".",
		Database:   database,
	})

	logger.Info("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("✅ Scraper is ready to start!")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("\n📝 Features:")
	logger.Info("  • Auto-save messages from all joined groups")
	logger.Info("  • Smart filtering (min 10 characters)")
	logger.Info("  • Track group activity")
	logger.Info("  • Shared database with Bot service")
	logger.Info("\n⚠️  First run: You'll need to enter verification code")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Start client
	logger.Info("🚀 Starting client...\n")

	if err := telegramClient.Start(ctx); err != nil {
		if err == context.Canceled {
			logger.Info("\n✅ Scraper stopped successfully")
		} else {
			logger.Error("Scraper error: %v", err)
			// Don't exit if running in 'all' mode
			if *mode != "all" {
				os.Exit(1)
			}
		}
	}

	logger.Info("\n✅ Scraper service stopped")
}
