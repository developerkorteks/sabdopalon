package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"telegram-summarizer/internal/client"
	"telegram-summarizer/internal/config"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/logger"
)

var (
	version     = "1.0.0-go"
	showVersion = flag.Bool("version", false, "Show version information")
	phone       = flag.String("phone", "", "Phone number (with country code)")
)

func main() {
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("Telegram Scraper (Pure Go) v%s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.DebugMode)

	logger.Info("═══════════════════════════════════════════════════════")
	logger.Info("🤖 TELEGRAM SCRAPER (Pure Golang)")
	logger.Info("Version: %s", version)
	logger.Info("═══════════════════════════════════════════════════════")

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

	// Initialize database
	logger.Info("\n📦 Initializing database...")
	database, err := db.InitDB(cfg.DatabasePath)
	if err != nil {
		logger.Error("Failed to initialize database: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("\n\n🛑 Shutting down gracefully...")
		cancel()
	}()
	
	// Add connection timeout monitor
	connectionTimeout := 60 * time.Second
	connectedCh := make(chan bool, 1)
	
	go func() {
		select {
		case <-time.After(connectionTimeout):
			if len(connectedCh) == 0 {
				logger.Error("\n❌ CONNECTION TIMEOUT after %v", connectionTimeout)
				logger.Error("The client couldn't connect to Telegram servers.")
				logger.Error("")
				logger.Error("Possible issues:")
				logger.Error("  1. Network/firewall blocking Telegram")
				logger.Error("  2. VPS region restrictions")
				logger.Error("  3. Session file corruption")
				logger.Error("")
				logger.Error("Solutions:")
				logger.Error("  • Remove session: rm session.json")
				logger.Error("  • Check network: curl https://api.telegram.org")
				logger.Error("  • Try different DC in code (change DC: 2 to DC: 4)")
				logger.Error("  • Use proxy/VPN")
				logger.Error("  • Read: cat TROUBLESHOOTING_SCRAPER.md")
				logger.Error("")
				cancel()
			}
		case <-connectedCh:
			// Connection successful, stop monitoring
			return
		}
	}()

	// Create client
	logger.Info("\n📱 Initializing Telegram Client...")
	
	// YOUR API CREDENTIALS
	apiID := 22527852
	apiHash := "4f595e6aac7dfe58a2cf6051360c3f14"
	
	telegramClient := client.NewClient(client.Config{
		AppID:      apiID,
		AppHash:    apiHash,
		Phone:      phoneNumber,
		SessionDir: ".",
		Database:   database,
	})

	logger.Info("\n═══════════════════════════════════════════════════════")
	logger.Info("✅ Scraper is ready to start!")
	logger.Info("═══════════════════════════════════════════════════════")
	logger.Info("\n📝 Features:")
	logger.Info("  • Auto-save messages from all joined groups")
	logger.Info("  • Smart filtering (min 10 characters)")
	logger.Info("  • Track group activity")
	logger.Info("  • Shared database with Go bot")
	logger.Info("\n🔧 To join groups, use Go bot commands:")
	logger.Info("  /listgroups - List all groups")
	logger.Info("  /enable <chat_id> - Enable summarization")
	logger.Info("  /disable <chat_id> - Disable summarization")
	logger.Info("\n⚠️  First run: You'll need to enter verification code")
	logger.Info("═══════════════════════════════════════════════════════\n")

	// Start client
	logger.Info("🚀 Starting client...\n")
	
	// Run client in goroutine to detect successful connection
	errCh := make(chan error, 1)
	go func() {
		errCh <- telegramClient.Start(ctx)
	}()
	
	// Wait for either error or timeout
	err = <-errCh
	
	// Signal successful connection (if no error before timeout)
	select {
	case connectedCh <- true:
	default:
	}
	
	if err != nil {
		if err == context.Canceled {
			logger.Info("\n✅ Client stopped successfully")
		} else {
			logger.Error("Client error: %v", err)
			os.Exit(1)
		}
	}

	logger.Info("\n👋 Goodbye!")
}
