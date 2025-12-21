package bot

import (
	"fmt"
	"strconv"
	"strings"
	"telegram-summarizer/internal/logger"
	"telegram-summarizer/internal/summarizer"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot represents the Telegram bot
type Bot struct {
	api            *tgbotapi.BotAPI
	stopCh         chan struct{}
	messageHandler *MessageHandler
	commandHandler *CommandHandler
	summarizer     *summarizer.Summarizer
}

// NewBot creates a new Telegram bot instance
func NewBot(token string, debug bool, messageHandler *MessageHandler, commandHandler *CommandHandler) (*Bot, error) {
	logger.Info("Initializing Telegram bot...")
	
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}
	
	api.Debug = debug
	
	logger.Info("✅ Authorized on account: @%s", api.Self.UserName)
	logger.Debug("Bot ID: %d", api.Self.ID)
	
	return &Bot{
		api:            api,
		stopCh:         make(chan struct{}),
		messageHandler: messageHandler,
		commandHandler: commandHandler,
	}, nil
}

// Start starts the bot and begins listening for messages
func (b *Bot) Start() error {
	logger.Info("🚀 Starting bot message listener...")
	
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	
	updates := b.api.GetUpdatesChan(u)
	
	logger.Info("✅ Bot is running... Listening for messages")
	logger.Info("Press Ctrl+C to stop")
	
	for {
		select {
		case update := <-updates:
			b.handleUpdate(update)
		case <-b.stopCh:
			logger.Info("Bot stopped")
			return nil
		}
	}
}

// handleUpdate processes incoming updates from Telegram
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	// Handle messages
	if update.Message != nil {
		b.handleMessage(update.Message)
		return
	}
	
	// Handle callback queries (button clicks)
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
		return
	}
	
	logger.Debug("Received non-message update: %+v", update.UpdateID)
}

// handleMessage processes incoming messages
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	logger.Debug("Received message: ChatID=%d, UserID=%d, User=%s, Text=%q",
		message.Chat.ID,
		message.From.ID,
		message.From.UserName,
		message.Text,
	)
	
	// Handle commands
	if message.IsCommand() {
		b.handleCommand(message)
		return
	}
	
	// Process and save regular messages
	if message.Text != "" && b.messageHandler != nil {
		if err := b.messageHandler.ProcessMessage(message); err != nil {
			logger.Error("Error processing message: %v", err)
		}
	}
}

// handleCommand processes bot commands
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	command := message.Command()
	args := strings.Fields(message.CommandArguments())
	
	logger.Info("🔧 Command received: /%s from user %s (UserID=%d)",
		command,
		message.From.UserName,
		message.From.ID,
	)
	
	var responseText string
	
	switch command {
	case "start":
		responseText = b.handleStartCommand(message)
	case "help":
		responseText = b.handleHelpCommand(message)
	case "listgroups":
		if b.commandHandler != nil {
			b.commandHandler.HandleListGroups(message)
			return
		}
	case "enable":
		if b.commandHandler != nil {
			b.commandHandler.HandleEnableGroup(message, args)
			return
		}
	case "disable":
		if b.commandHandler != nil {
			b.commandHandler.HandleDisableGroup(message, args)
			return
		}
	case "disableall":
		if b.commandHandler != nil {
			b.commandHandler.HandleDisableAllGroups(message)
			return
		}
	case "groupstats":
		if b.commandHandler != nil {
			b.commandHandler.HandleGroupStats(message)
			return
		}
	case "summary":
		if b.commandHandler != nil {
			b.commandHandler.HandleSummary(message, args)
			return
		}
	default:
		logger.Debug("Unknown command: /%s", command)
		return
	}
	
	// Send response
	if responseText != "" {
		b.sendMessage(message.Chat.ID, responseText)
	}
}

// handleStartCommand handles /start command
func (b *Bot) handleStartCommand(message *tgbotapi.Message) string {
	logger.Info("Handling /start command")
	
	response := `🤖 *Telegram Chat Summarizer Bot*

Hello! I'm a bot that summarizes group chat messages.

*What I do:*
• Record all messages in this group
• Generate summaries every 4 hours
• Create daily summaries at 23:59
• Provide statistics and insights

*Commands:*
/help - Show available commands
/summary - Get current summary (coming soon)
/stats - Show statistics (coming soon)

Add me to your group and I'll start working!`

	return response
}

// handleHelpCommand handles /help command
func (b *Bot) handleHelpCommand(message *tgbotapi.Message) string {
	logger.Info("Handling /help command")
	
	response := `📚 *Available Commands:*

*Group Management:*
/listgroups - View all tracked groups
/enable <chat_id> - Enable auto-summary for a group
/disable <chat_id> - Disable auto-summary for a group
/disableall - Disable ALL groups at once
/groupstats - Show detailed group statistics

*Summary Commands:*
/summary <chat_id> - Generate on-demand summary
/summary <chat_id> 4h - Last 4 hours summary
/summary <chat_id> daily - Today's summary

*General:*
/start - Introduction and welcome message
/help - Show this help message

*How it works:*
1. Add bot to your group
2. Bot records all messages
3. Enable group with /enable <chat_id>
4. Automatic summaries every 4 hours
5. Daily summary at 23:59

*Privacy:*
All messages are stored locally in SQLite database.

Questions? Contact bot admin.`

	return response
}

// sendMessage sends a message to a chat
func (b *Bot) sendMessage(chatID int64, text string) error {
	logger.Debug("Sending message to ChatID=%d", chatID)
	
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	
	_, err := b.api.Send(msg)
	if err != nil {
		logger.Error("Failed to send message: %v", err)
		return err
	}
	
	logger.Info("✅ Message sent to ChatID=%d", chatID)
	return nil
}

// sendMessageWithKeyboard sends a message with an inline keyboard
func (b *Bot) sendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	logger.Debug("Sending message with keyboard to ChatID=%d", chatID)
	
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	
	_, err := b.api.Send(msg)
	if err != nil {
		logger.Error("Failed to send message with keyboard: %v", err)
		return err
	}
	
	logger.Info("✅ Message with keyboard sent to ChatID=%d", chatID)
	return nil
}

// editMessageWithKeyboard edits an existing message with an inline keyboard
func (b *Bot) editMessageWithKeyboard(chatID int64, messageID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	logger.Debug("Editing message MessageID=%d in ChatID=%d", messageID, chatID)
	
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	edit.ReplyMarkup = &keyboard
	
	_, err := b.api.Send(edit)
	if err != nil {
		// Ignore "message is not modified" errors - this happens when clicking the same page
		if strings.Contains(err.Error(), "message is not modified") {
			logger.Debug("Message content unchanged, skipping edit")
			return nil
		}
		logger.Error("Failed to edit message: %v", err)
		return err
	}
	
	logger.Info("✅ Message edited MessageID=%d in ChatID=%d", messageID, chatID)
	return nil
}

// handleCallbackQuery handles button clicks
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	logger.Info("Handling callback query: %s from user %s", query.Data, query.From.UserName)
	
	// Parse the callback data (format: "command:arg")
	parts := strings.Split(query.Data, ":")
	if len(parts) < 1 {
		// Answer the callback to remove the loading state
		callback := tgbotapi.NewCallback(query.ID, "")
		b.api.Request(callback)
		return
	}
	
	command := parts[0]
	
	// Ignore "noop" callbacks (page indicator button)
	if command == "noop" {
		// Answer the callback to remove the loading state
		callback := tgbotapi.NewCallback(query.ID, "")
		b.api.Request(callback)
		return
	}
	
	// Extract current page from the inline keyboard buttons
	var currentPage string
	if query.Message.ReplyMarkup != nil {
		for _, row := range query.Message.ReplyMarkup.InlineKeyboard {
			for _, button := range row {
				// Find the page indicator button (format: "📄 X/Y")
				if strings.HasPrefix(button.Text, "📄 ") && button.CallbackData != nil && *button.CallbackData == "noop" {
					// Extract current page from button text
					parts := strings.Fields(button.Text)
					if len(parts) >= 2 {
						pageInfo := strings.Split(parts[1], "/")
						if len(pageInfo) >= 1 {
							currentPage = pageInfo[0]
						}
					}
					break
				}
			}
		}
	}
	
	// Create a pseudo-message for the command handler
	message := &tgbotapi.Message{
		Chat: query.Message.Chat,
		From: query.From,
		MessageID: query.Message.MessageID,
	}
	
	switch command {
	case "listgroups":
		if b.commandHandler != nil && len(parts) > 1 {
			requestedPage := parts[1]
			// Check if already on this page
			if currentPage == requestedPage {
				logger.Debug("Already on page %s, skipping edit", requestedPage)
				// Answer the callback to remove the loading state
				callback := tgbotapi.NewCallback(query.ID, "")
				b.api.Request(callback)
				return
			}
			// Parse page number
			page, err := strconv.Atoi(requestedPage)
			if err != nil || page < 1 {
				page = 1
			}
			// Answer the callback to remove the loading state
			callback := tgbotapi.NewCallback(query.ID, "")
			b.api.Request(callback)
			b.commandHandler.HandleListGroupsEdit(message, page)
		}
	case "groupstats":
		if b.commandHandler != nil && len(parts) > 1 {
			requestedPage := parts[1]
			// Check if already on this page
			if currentPage == requestedPage {
				logger.Debug("Already on page %s, skipping edit", requestedPage)
				// Answer the callback to remove the loading state
				callback := tgbotapi.NewCallback(query.ID, "")
				b.api.Request(callback)
				return
			}
			// Parse page number
			page, err := strconv.Atoi(requestedPage)
			if err != nil || page < 1 {
				page = 1
			}
			// Answer the callback to remove the loading state
			callback := tgbotapi.NewCallback(query.ID, "")
			b.api.Request(callback)
			b.commandHandler.HandleGroupStatsEdit(message, page)
		}
	default:
		logger.Debug("Unknown callback command: %s", command)
		// Answer the callback to remove the loading state
		callback := tgbotapi.NewCallback(query.ID, "")
		b.api.Request(callback)
	}
}

// Stop stops the bot gracefully
func (b *Bot) Stop() {
	logger.Info("Stopping bot...")
	close(b.stopCh)
	b.api.StopReceivingUpdates()
}

// GetAPI returns the underlying bot API (for advanced usage)
func (b *Bot) GetAPI() *tgbotapi.BotAPI {
	return b.api
}

// SetCommandHandler sets the command handler (for initialization after bot creation)
func (b *Bot) SetCommandHandler(handler *CommandHandler) {
	b.commandHandler = handler
}

// SetSummarizer sets the summarizer (for initialization after bot creation)
func (b *Bot) SetSummarizer(s *summarizer.Summarizer) {
	b.summarizer = s
}
