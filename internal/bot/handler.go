package bot

import (
	"strings"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/logger"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MessageHandler handles message processing and storage
type MessageHandler struct {
	database *db.DB
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(database *db.DB) *MessageHandler {
	return &MessageHandler{
		database: database,
	}
}

// ProcessMessage processes and potentially saves a message
func (h *MessageHandler) ProcessMessage(message *tgbotapi.Message) error {
	// Skip if no text
	if message.Text == "" {
		logger.Debug("Skipping message: no text content")
		return nil
	}
	
	// Skip commands
	if message.IsCommand() {
		logger.Debug("Skipping message: is a command")
		return nil
	}
	
	// Skip bot messages
	if message.From.IsBot {
		logger.Debug("Skipping message: from bot")
		return nil
	}
	
	// Filter: minimum length
	if len(message.Text) < 10 {
		logger.Debug("Skipping message: too short (%d chars)", len(message.Text))
		return nil
	}
	
	// Filter: only emoji or special characters
	if isOnlyEmoji(message.Text) {
		logger.Debug("Skipping message: only emoji/special chars")
		return nil
	}
	
	// Create message object
	msg := &db.Message{
		ChatID:        message.Chat.ID,
		UserID:        int64(message.From.ID),
		Username:      message.From.UserName,
		MessageText:   message.Text,
		MessageLength: len(message.Text),
		Timestamp:     time.Unix(int64(message.Date), 0),
	}
	
	// Handle empty username
	if msg.Username == "" {
		msg.Username = message.From.FirstName
		if msg.Username == "" {
			msg.Username = "Unknown"
		}
	}
	
	// Auto-track group if not already tracked
	chatID := message.Chat.ID
	chatTitle := message.Chat.Title
	chatUsername := message.Chat.UserName
	
	// Add to tracked groups (will be ignored if already exists)
	h.database.AddTrackedGroup(chatID, chatTitle, chatUsername)
	
	// Save to database
	if err := h.database.SaveMessage(msg); err != nil {
		logger.Error("Failed to save message: %v", err)
		return err
	}
	
	logger.Info("💾 Saved message: [%s] %s: %q (ID=%d)", 
		message.Chat.Title,
		msg.Username,
		truncateText(msg.MessageText, 50),
		msg.ID,
	)
	
	return nil
}

// isOnlyEmoji checks if text contains only emoji and special characters
func isOnlyEmoji(text string) bool {
	text = strings.TrimSpace(text)
	
	// Check if string has any alphanumeric character
	hasAlphaNum := false
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || 
		   (r >= 'A' && r <= 'Z') || 
		   (r >= '0' && r <= '9') {
			hasAlphaNum = true
			break
		}
	}
	
	return !hasAlphaNum
}

// truncateText truncates text to specified length
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
