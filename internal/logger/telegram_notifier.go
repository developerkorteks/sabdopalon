package logger

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramNotifier sends logs to a Telegram bot
type TelegramNotifier struct {
	bot       *tgbotapi.BotAPI
	chatID    int64
	enabled   bool
	mu        sync.Mutex
	buffer    map[string][]string // Grouped by level
	lastSend  time.Time
	batchSize int
	flushTime time.Duration
}

var (
	notifier     *TelegramNotifier
	notifierOnce sync.Once
)

// InitTelegramNotifier initializes the Telegram notifier
func InitTelegramNotifier(botToken string, chatID int64) error {
	var err error
	notifierOnce.Do(func() {
		bot, botErr := tgbotapi.NewBotAPI(botToken)
		if botErr != nil {
			err = fmt.Errorf("failed to create bot for notifier: %w", botErr)
			return
		}

		notifier = &TelegramNotifier{
			bot:       bot,
			chatID:    chatID,
			enabled:   true,
			buffer:    make(map[string][]string), // Grouped logs
			lastSend:  time.Now(),
			batchSize: 15,        // Send every 15 logs
			flushTime: 10 * time.Second, // Or every 10 seconds
		}

		// Start background flusher
		go notifier.backgroundFlusher()

		Info("✅ Telegram notifier initialized (Chat ID: %d)", chatID)
	})

	return err
}

// GetTelegramNotifier returns the global notifier instance
func GetTelegramNotifier() *TelegramNotifier {
	return notifier
}

// SendLog sends a log message to Telegram
func (tn *TelegramNotifier) SendLog(level, message string) {
	if tn == nil || !tn.enabled {
		return
	}

	tn.mu.Lock()
	defer tn.mu.Unlock()

	// Format log entry (without emoji, we'll add later)
	timestamp := time.Now().Format("15:04:05")
	formattedLog := fmt.Sprintf("[%s] %s", timestamp, message)

	// Group by level
	if tn.buffer[level] == nil {
		tn.buffer[level] = make([]string, 0)
	}
	tn.buffer[level] = append(tn.buffer[level], formattedLog)

	// Count total logs
	totalLogs := 0
	for _, logs := range tn.buffer {
		totalLogs += len(logs)
	}

	// Check if we should flush
	if totalLogs >= tn.batchSize || time.Since(tn.lastSend) >= tn.flushTime {
		tn.flushLogs()
	}
}

// SendSummary sends a complete summary to Telegram (immediate, not buffered)
func (tn *TelegramNotifier) SendSummary(groupName, summary string) {
	if tn == nil || !tn.enabled {
		return
	}

	message := fmt.Sprintf("📊 *Summary Generated*\n\n*Group:* %s\n\n%s", 
		escapeMarkdown(groupName), summary)

	// Send immediately (don't buffer summaries)
	go tn.sendMessage(message)
}

// flushLogs sends buffered logs to Telegram
func (tn *TelegramNotifier) flushLogs() {
	if len(tn.buffer) == 0 {
		return
	}

	// Build formatted message with grouping
	var message strings.Builder
	
	// Header with timestamp range
	message.WriteString("```\n")
	message.WriteString("┌─────────────────────────────┐\n")
	message.WriteString("│     📊 SYSTEM LOGS          │\n")
	message.WriteString("└─────────────────────────────┘\n")
	message.WriteString("```\n\n")
	
	// Count logs per level
	counts := make(map[string]int)
	for level, logs := range tn.buffer {
		counts[level] = len(logs)
	}
	
	// Summary
	if len(counts) > 0 {
		message.WriteString("📈 *Summary:* ")
		parts := []string{}
		if counts["INFO"] > 0 {
			parts = append(parts, fmt.Sprintf("ℹ️ %d", counts["INFO"]))
		}
		if counts["DEBUG"] > 0 {
			parts = append(parts, fmt.Sprintf("🔍 %d", counts["DEBUG"]))
		}
		if counts["WARN"] > 0 {
			parts = append(parts, fmt.Sprintf("⚠️ %d", counts["WARN"]))
		}
		if counts["ERROR"] > 0 {
			parts = append(parts, fmt.Sprintf("❌ %d", counts["ERROR"]))
		}
		message.WriteString(strings.Join(parts, " • "))
		message.WriteString("\n\n")
	}
	
	// Priority order: ERROR, WARN, INFO, DEBUG
	levels := []string{"ERROR", "WARN", "INFO", "DEBUG"}
	
	for _, level := range levels {
		logs, exists := tn.buffer[level]
		if !exists || len(logs) == 0 {
			continue
		}
		
		// Section header
		emoji := tn.getEmojiForLevel(level)
		message.WriteString(fmt.Sprintf("%s *%s* (%d)\n", emoji, level, len(logs)))
		message.WriteString("```\n")
		
		// Show logs (max 5 per level to avoid spam)
		displayCount := len(logs)
		if displayCount > 5 {
			displayCount = 5
		}
		
		for i := 0; i < displayCount; i++ {
			message.WriteString(logs[i])
			message.WriteString("\n")
		}
		
		// If more than 5, show truncation message
		if len(logs) > 5 {
			message.WriteString(fmt.Sprintf("... +%d more\n", len(logs)-5))
		}
		
		message.WriteString("```\n")
	}
	
	// Clear buffer
	tn.buffer = make(map[string][]string)
	tn.lastSend = time.Now()

	// Send in background
	go tn.sendMessage(message.String())
}

// backgroundFlusher periodically flushes logs
func (tn *TelegramNotifier) backgroundFlusher() {
	ticker := time.NewTicker(tn.flushTime)
	defer ticker.Stop()

	for range ticker.C {
		tn.mu.Lock()
		if len(tn.buffer) > 0 {
			tn.flushLogs()
		}
		tn.mu.Unlock()
	}
}

// sendMessage sends a message to Telegram
func (tn *TelegramNotifier) sendMessage(text string) {
	// Split if message is too long (Telegram limit: 4096 chars)
	const maxLen = 4000
	
	if len(text) <= maxLen {
		tn.sendSingleMessage(text)
		return
	}

	// Split into multiple messages
	for len(text) > 0 {
		end := maxLen
		if end > len(text) {
			end = len(text)
		}

		// Try to split at newline
		if end < len(text) {
			lastNewline := strings.LastIndex(text[:end], "\n")
			if lastNewline > 0 {
				end = lastNewline
			}
		}

		chunk := text[:end]
		tn.sendSingleMessage(chunk)

		text = text[end:]
		if len(text) > 0 {
			time.Sleep(100 * time.Millisecond) // Avoid rate limiting
		}
	}
}

// sendSingleMessage sends a single message
func (tn *TelegramNotifier) sendSingleMessage(text string) {
	msg := tgbotapi.NewMessage(tn.chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	_, err := tn.bot.Send(msg)
	if err != nil {
		// If markdown parsing fails, try without markdown
		if strings.Contains(err.Error(), "can't parse entities") {
			msg.ParseMode = ""
			_, err2 := tn.bot.Send(msg)
			if err2 != nil {
				fmt.Printf("Failed to send log to Telegram (no markdown): %v\n", err2)
			}
		} else {
			// Don't use logger here to avoid infinite recursion
			fmt.Printf("Failed to send log to Telegram: %v\n", err)
		}
	}
}

// getEmojiForLevel returns emoji for log level
func (tn *TelegramNotifier) getEmojiForLevel(level string) string {
	switch level {
	case "INFO":
		return "ℹ️"
	case "DEBUG":
		return "🔍"
	case "WARN", "WARNING":
		return "⚠️"
	case "ERROR":
		return "❌"
	case "FATAL":
		return "🚨"
	default:
		return "📝"
	}
}

// Flush forces sending all buffered logs
func (tn *TelegramNotifier) Flush() {
	if tn == nil {
		return
	}

	tn.mu.Lock()
	defer tn.mu.Unlock()

	tn.flushLogs()
}

// Enable/disable notifier
func (tn *TelegramNotifier) Enable() {
	if tn != nil {
		tn.enabled = true
	}
}

func (tn *TelegramNotifier) Disable() {
	if tn != nil {
		tn.enabled = false
	}
}

// escapeMarkdown escapes markdown special characters
func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}
