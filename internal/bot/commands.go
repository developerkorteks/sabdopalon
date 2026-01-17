package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandHandler handles bot commands
type CommandHandler struct {
	bot      *Bot
	database *db.DB
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(bot *Bot, database *db.DB) *CommandHandler {
	return &CommandHandler{
		bot:      bot,
		database: database,
	}
}

// HandleListGroups handles /listgroups command with pagination
func (h *CommandHandler) HandleListGroups(message *tgbotapi.Message) {
	// Parse page number from command arguments (default to page 1)
	page := 1
	args := strings.Fields(message.CommandArguments())
	if len(args) > 0 {
		if p, err := strconv.Atoi(args[0]); err == nil && p > 0 {
			page = p
		}
	}
	
	text, keyboard, hasKeyboard := h.buildListGroupsResponse(page)
	
	if hasKeyboard {
		h.bot.sendMessageWithKeyboard(message.Chat.ID, text, keyboard)
	} else {
		h.bot.sendMessage(message.Chat.ID, text)
	}
}

// HandleListGroupsEdit handles /listgroups pagination by editing the existing message
func (h *CommandHandler) HandleListGroupsEdit(message *tgbotapi.Message, page int) {
	text, keyboard, hasKeyboard := h.buildListGroupsResponse(page)
	
	if hasKeyboard {
		h.bot.editMessageWithKeyboard(message.Chat.ID, message.MessageID, text, keyboard)
	} else {
		// Fallback to sending a new message if no keyboard
		h.bot.sendMessage(message.Chat.ID, text)
	}
}

// buildListGroupsResponse builds the response text and keyboard for /listgroups
func (h *CommandHandler) buildListGroupsResponse(page int) (string, tgbotapi.InlineKeyboardMarkup, bool) {
	logger.Info("Building /listgroups response for page %d", page)
	
	groups := h.database.GetTrackedGroups()
	
	if len(groups) == 0 {
		return "📋 No groups tracked yet.\n\nScraper needs to join groups first.", tgbotapi.InlineKeyboardMarkup{}, false
	}
	
	// Pagination settings
	const groupsPerPage = 20
	totalPages := (len(groups) + groupsPerPage - 1) / groupsPerPage
	
	// Validate page number
	if page > totalPages {
		page = totalPages
	}
	
	// Calculate slice boundaries
	startIdx := (page - 1) * groupsPerPage
	endIdx := startIdx + groupsPerPage
	if endIdx > len(groups) {
		endIdx = len(groups)
	}
	
	pageGroups := groups[startIdx:endIdx]
	
	var response strings.Builder
	response.WriteString(fmt.Sprintf("📋 *Your Tracked Groups* (Page %d/%d)\n\n", page, totalPages))
	
	// Count active groups on this page and overall
	activeCount := 0
	totalActiveCount := 0
	for _, group := range groups {
		if group.IsActive == 1 {
			totalActiveCount++
		}
	}
	
	for i, group := range pageGroups {
		globalIndex := startIdx + i + 1
		
		// Status emoji
		statusEmoji := "❌"
		statusText := "INACTIVE"
		if group.IsActive == 1 {
			statusEmoji = "✅"
			statusText = "ACTIVE"
			activeCount++
		}
		
		// Get message count
		msgCount := h.database.GetGroupMessageCount24h(group.ChatID)
		
		response.WriteString(fmt.Sprintf("%d. %s %s", globalIndex, statusEmoji, escapeMarkdown(group.GroupName)))
		if group.GroupUsername != "" {
			response.WriteString(fmt.Sprintf(" (@%s)", escapeMarkdown(group.GroupUsername)))
		}
		response.WriteString("\n")
		response.WriteString(fmt.Sprintf("   • Messages (24h): %d\n", msgCount))
		response.WriteString(fmt.Sprintf("   • Status: %s %s\n", statusText, 
			func() string {
				if group.IsActive == 1 {
					return "(will summarize)"
				}
				return "(won't summarize)"
			}()))
		response.WriteString(fmt.Sprintf("   • Chat ID: `%d`\n\n", group.ChatID))
	}
	
	response.WriteString(fmt.Sprintf("*Summary:* %d/%d groups active\n\n", totalActiveCount, len(groups)))
	
	response.WriteString("*Commands:*\n")
	response.WriteString("`/enable <chat_id>` - Enable summarization\n")
	response.WriteString("`/disable <chat_id>` - Disable summarization\n")
	response.WriteString("`/disableall` - Disable ALL groups\n")
	response.WriteString("`/groupstats` - Show detailed statistics")
	
	// Create inline keyboard for navigation
	if totalPages > 1 {
		var buttons [][]tgbotapi.InlineKeyboardButton
		var row []tgbotapi.InlineKeyboardButton
		
		// Previous button
		if page > 1 {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("⬅️ Previous", fmt.Sprintf("listgroups:%d", page-1)))
		}
		
		// Page indicator
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📄 %d/%d", page, totalPages), "noop"))
		
		// Next button
		if page < totalPages {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Next ➡️", fmt.Sprintf("listgroups:%d", page+1)))
		}
		
		buttons = append(buttons, row)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
		return response.String(), keyboard, true
	}
	
	return response.String(), tgbotapi.InlineKeyboardMarkup{}, false
}

// escapeMarkdown escapes special characters in text to prevent Markdown parsing errors
// Escapes ALL special characters used in Telegram MarkdownV2
func escapeMarkdown(text string) string {
	// Escape ALL special characters for Telegram MarkdownV2
	// Reference: https://core.telegram.org/bots/api#markdownv2-style
	replacer := strings.NewReplacer(
		"_", "\\_",   // Underscore (italic)
		"*", "\\*",   // Asterisk (bold)
		"[", "\\[",   // Square bracket (links)
		"]", "\\]",
		"(", "\\(",   // Parentheses (links)
		")", "\\)",
		"~", "\\~",   // Tilde (strikethrough)
		"`", "\\`",   // Backtick (code)
		">", "\\>",   // Greater than (quote)
		"#", "\\#",   // Hash (header)
		"+", "\\+",   // Plus (list)
		"-", "\\-",   // Minus (list)
		"=", "\\=",   // Equals
		"|", "\\|",   // Pipe (table)
		"{", "\\{",   // Curly braces
		"}", "\\}",
		".", "\\.",   // Period (list)
		"!", "\\!",   // Exclamation
		"@", "\\@",   // At symbol (mentions) ⭐ NEW
	)
	return replacer.Replace(text)
}

// HandleEnableGroup handles /enable command
func (h *CommandHandler) HandleEnableGroup(message *tgbotapi.Message, args []string) {
	logger.Info("Handling /enable command from user %d", message.From.ID)
	
	if len(args) < 1 {
		h.bot.sendMessage(message.Chat.ID, "❌ Usage: `/enable <chat_id>`\n\nExample: `/enable -1001234567890`\n\nUse /listgroups to see chat IDs.")
		return
	}
	
	chatID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		h.bot.sendMessage(message.Chat.ID, "❌ Invalid chat ID. Must be a number.\n\nExample: `/enable -1001234567890`")
		return
	}
	
	// Check if group exists
	groups := h.database.GetTrackedGroups()
	var targetGroup *db.TrackedGroup
	for _, g := range groups {
		if g.ChatID == chatID {
			targetGroup = &g
			break
		}
	}
	
	if targetGroup == nil {
		h.bot.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Group with chat ID `%d` not found.\n\nUse /listgroups to see available groups.", chatID))
		return
	}
	
	// Enable summarization
	if err := h.database.EnableGroupSummary(chatID); err != nil {
		logger.Error("Failed to enable group: %v", err)
		h.bot.sendMessage(message.Chat.ID, "❌ Failed to enable group. Check logs.")
		return
	}
	
	msgCount := h.database.GetGroupMessageCount24h(chatID)
	
	response := fmt.Sprintf("✅ *%s* is now ACTIVE\n\n", targetGroup.GroupName)
	if targetGroup.GroupUsername != "" {
		response += fmt.Sprintf("(@%s)\n\n", targetGroup.GroupUsername)
	}
	response += "This group will be included in:\n"
	response += "• 4-hour summaries\n"
	response += "• Daily summaries\n\n"
	response += fmt.Sprintf("*Messages (24h):* %d", msgCount)
	
	h.bot.sendMessage(message.Chat.ID, response)
}

// HandleDisableGroup handles /disable command
func (h *CommandHandler) HandleDisableGroup(message *tgbotapi.Message, args []string) {
	logger.Info("Handling /disable command from user %d", message.From.ID)
	
	if len(args) < 1 {
		h.bot.sendMessage(message.Chat.ID, "❌ Usage: `/disable <chat_id>`\n\nExample: `/disable -1001234567890`\n\nUse /listgroups to see chat IDs.")
		return
	}
	
	chatID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		h.bot.sendMessage(message.Chat.ID, "❌ Invalid chat ID. Must be a number.\n\nExample: `/disable -1001234567890`")
		return
	}
	
	// Check if group exists
	groups := h.database.GetTrackedGroups()
	var targetGroup *db.TrackedGroup
	for _, g := range groups {
		if g.ChatID == chatID {
			targetGroup = &g
			break
		}
	}
	
	if targetGroup == nil {
		h.bot.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Group with chat ID `%d` not found.\n\nUse /listgroups to see available groups.", chatID))
		return
	}
	
	// Disable summarization
	if err := h.database.DisableGroupSummary(chatID); err != nil {
		logger.Error("Failed to disable group: %v", err)
		h.bot.sendMessage(message.Chat.ID, "❌ Failed to disable group. Check logs.")
		return
	}
	
	response := fmt.Sprintf("❌ *%s* is now INACTIVE\n\n", targetGroup.GroupName)
	if targetGroup.GroupUsername != "" {
		response += fmt.Sprintf("(@%s)\n\n", targetGroup.GroupUsername)
	}
	response += "This group will NOT be summarized.\n"
	response += "Messages will still be saved for later."
	
	h.bot.sendMessage(message.Chat.ID, response)
}

// HandleDisableAllGroups handles /disableall command
func (h *CommandHandler) HandleDisableAllGroups(message *tgbotapi.Message) {
	logger.Info("Handling /disableall command from user %d", message.From.ID)
	
	// Get current active groups count
	activeGroups := h.database.GetActiveGroups()
	activeCount := len(activeGroups)
	
	if activeCount == 0 {
		h.bot.sendMessage(message.Chat.ID, "ℹ️ No active groups to disable.\n\nAll groups are already inactive.")
		return
	}
	
	// Disable all groups
	rowsAffected, err := h.database.DisableAllGroups()
	if err != nil {
		logger.Error("Failed to disable all groups: %v", err)
		h.bot.sendMessage(message.Chat.ID, "❌ Failed to disable all groups. Check logs.")
		return
	}
	
	response := fmt.Sprintf("✅ *Successfully disabled ALL groups*\n\n")
	response += fmt.Sprintf("• Groups affected: %d\n", rowsAffected)
	response += fmt.Sprintf("• Previously active: %d\n\n", activeCount)
	response += "🔕 *All auto-summaries are now STOPPED*\n\n"
	response += "No groups will be summarized until you enable them again.\n"
	response += "Messages will still be saved for later.\n\n"
	response += "To re-enable groups:\n"
	response += "`/enable <chat_id>` - Enable specific group\n"
	response += "`/listgroups` - View all groups"
	
	h.bot.sendMessage(message.Chat.ID, response)
}

// HandleSummary handles /summary command
func (h *CommandHandler) HandleSummary(message *tgbotapi.Message, args []string) {
	logger.Info("Handling /summary command from user %d", message.From.ID)
	
	if len(args) < 1 {
		h.bot.sendMessage(message.Chat.ID, "❌ Usage: `/summary <chat_id>`\n\nExample: `/summary 3103764752`\n\nUse /listgroups to see chat IDs of active groups.")
		return
	}
	
	chatID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		h.bot.sendMessage(message.Chat.ID, "❌ Invalid chat ID. Must be a number.\n\nExample: `/summary 3103764752`")
		return
	}
	
	// Check if group exists and is active
	group := h.database.GetTrackedGroup(chatID)
	if group == nil {
		h.bot.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Group with chat ID `%d` not found.\n\nUse /listgroups to see available groups.", chatID))
		return
	}
	
	if group.IsActive == 0 {
		h.bot.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Group *%s* is INACTIVE.\n\nPlease enable it first:\n`/enable %d`", escapeMarkdown(group.GroupName), chatID))
		return
	}
	
	// Send "generating" message
	h.bot.sendMessage(message.Chat.ID, fmt.Sprintf("⏳ Generating summary for *%s*...\n\nThis may take a few seconds.", escapeMarkdown(group.GroupName)))
	
	// Get messages from last 24 hours
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)
	
	messages, err := h.database.GetMessagesByTimeRange(chatID, startTime, endTime)
	if err != nil {
		logger.Error("Failed to get messages: %v", err)
		h.bot.sendMessage(message.Chat.ID, "❌ Failed to retrieve messages. Check logs.")
		return
	}
	
	if len(messages) == 0 {
		h.bot.sendMessage(message.Chat.ID, fmt.Sprintf("📭 No messages found in last 24 hours for *%s*.\n\nThe group might have been recently enabled.", escapeMarkdown(group.GroupName)))
		return
	}
	
	// Count unique users
	uniqueUsers := make(map[int64]string)
	for _, msg := range messages {
		uniqueUsers[msg.UserID] = msg.Username
	}
	
	// Progress callback to send updates to user (without Markdown to avoid parsing errors)
	progressCallback := func(progressMsg string) {
		msg := tgbotapi.NewMessage(message.Chat.ID, progressMsg)
		// Do NOT set ParseMode for progress messages
		h.bot.GetAPI().Send(msg)
	}
	
	// Summary callback to send partial summaries
	summaryCallback := func(partialSummary string) {
		// Send partial summary without extra headers (formatter already adds them)
		h.sendMessageWithoutHeader(message.Chat.ID, partialSummary)
		
		// Also send to monitoring bot
		logger.SendSummaryNotification(group.GroupName, partialSummary)
	}
	
	// Use hierarchical summarization (automatic chunking for large chats)
	logger.Info("Using streaming summarization for %d messages", len(messages))
	summary, err := h.bot.summarizer.GenerateSummaryHierarchical(messages, group.GroupName, startTime, endTime, progressCallback, summaryCallback)
	if err != nil {
		logger.Error("Failed to generate summary: %v", err)
		h.bot.sendMessage(message.Chat.ID, fmt.Sprintf("❌ Failed to generate summary: %v\n\nPlease check your configuration and try again.", err))
		return
	}
	
	// Send completion message (already formatted by formatter)
	// Use sendMessageWithoutHeader to avoid Markdown parsing errors
	h.sendMessageWithoutHeader(message.Chat.ID, summary)
	
	// Parse metadata from summary (use raw summary text without formatting)
	parser := h.bot.summarizer.GetMetadataParser()
	metadata := parser.Parse(summary)
	
	// Save summary to database with metadata
	dbSummary := &db.Summary{
		ChatID:            chatID,
		SummaryType:       "manual-24h",
		PeriodStart:       startTime,
		PeriodEnd:         endTime,
		SummaryText:       summary,
		MessageCount:      len(messages),
		Sentiment:         metadata.Sentiment,
		CredibilityScore:  metadata.CredibilityScore,
		ProductsMentioned: metadata.ProductsJSON,
		RedFlagsCount:     metadata.RedFlagsCount,
		ValidationStatus:  metadata.ValidationStatus,
	}
	
	if err := h.database.SaveSummary(dbSummary); err != nil {
		logger.Error("Failed to save summary: %v", err)
		// Don't fail, just log
	} else {
		// Save product mentions
		for i := range metadata.Products {
			metadata.Products[i].SummaryID = dbSummary.ID
			if err := h.database.SaveProductMention(&metadata.Products[i]); err != nil {
				logger.Error("Failed to save product mention: %v", err)
			}
		}
	}
	
	logger.Info("✅ Summary generated successfully for group %s (%d messages)", group.GroupName, len(messages))
}

// HandleGroupStats handles /groupstats command with pagination
func (h *CommandHandler) HandleGroupStats(message *tgbotapi.Message) {
	// Parse page number from command arguments (default to page 1)
	page := 1
	args := strings.Fields(message.CommandArguments())
	if len(args) > 0 {
		if p, err := strconv.Atoi(args[0]); err == nil && p > 0 {
			page = p
		}
	}
	
	text, keyboard, hasKeyboard := h.buildGroupStatsResponse(page)
	
	if hasKeyboard {
		h.bot.sendMessageWithKeyboard(message.Chat.ID, text, keyboard)
	} else {
		h.bot.sendMessage(message.Chat.ID, text)
	}
}

// HandleGroupStatsEdit handles /groupstats pagination by editing the existing message
func (h *CommandHandler) HandleGroupStatsEdit(message *tgbotapi.Message, page int) {
	text, keyboard, hasKeyboard := h.buildGroupStatsResponse(page)
	
	if hasKeyboard {
		h.bot.editMessageWithKeyboard(message.Chat.ID, message.MessageID, text, keyboard)
	} else {
		// Fallback to sending a new message if no keyboard
		h.bot.sendMessage(message.Chat.ID, text)
	}
}

// buildGroupStatsResponse builds the response text and keyboard for /groupstats
func (h *CommandHandler) buildGroupStatsResponse(page int) (string, tgbotapi.InlineKeyboardMarkup, bool) {
	logger.Info("Building /groupstats response for page %d", page)
	
	groups := h.database.GetTrackedGroups()
	
	if len(groups) == 0 {
		return "📊 No groups tracked yet.", tgbotapi.InlineKeyboardMarkup{}, false
	}
	
	// Calculate overall stats first
	activeCount := 0
	totalMessages := 0
	maxMessages := 0
	var mostActiveGroup string
	
	for _, group := range groups {
		if group.IsActive == 1 {
			activeCount++
		}
		
		msgCount := h.database.GetGroupMessageCount24h(group.ChatID)
		totalMessages += msgCount
		
		if msgCount > maxMessages {
			maxMessages = msgCount
			mostActiveGroup = group.GroupName
		}
	}
	
	// Pagination settings
	const groupsPerPage = 30
	totalPages := (len(groups) + groupsPerPage - 1) / groupsPerPage
	
	// Validate page number
	if page > totalPages {
		page = totalPages
	}
	
	// Calculate slice boundaries
	startIdx := (page - 1) * groupsPerPage
	endIdx := startIdx + groupsPerPage
	if endIdx > len(groups) {
		endIdx = len(groups)
	}
	
	pageGroups := groups[startIdx:endIdx]
	
	var response strings.Builder
	response.WriteString(fmt.Sprintf("📊 *Group Statistics* (Page %d/%d)\n\n", page, totalPages))
	
	response.WriteString(fmt.Sprintf("*Active Groups:* %d/%d\n", activeCount, len(groups)))
	response.WriteString(fmt.Sprintf("*Total Messages (24h):* %d\n", totalMessages))
	if mostActiveGroup != "" {
		response.WriteString(fmt.Sprintf("*Most Active:* %s (%d msgs)\n\n", escapeMarkdown(mostActiveGroup), maxMessages))
	}
	
	response.WriteString("*Breakdown:*\n")
	for _, group := range pageGroups {
		statusEmoji := "❌"
		if group.IsActive == 1 {
			statusEmoji = "✅"
		}
		
		msgCount := h.database.GetGroupMessageCount24h(group.ChatID)
		response.WriteString(fmt.Sprintf("%s %s: %d msgs", statusEmoji, escapeMarkdown(group.GroupName), msgCount))
		if group.IsActive == 0 {
			response.WriteString(" (inactive)")
		}
		response.WriteString("\n")
	}
	
	response.WriteString("\n*Next Summary:* Manual trigger only (Phase 8)")
	response.WriteString("\nUse /listgroups to manage groups")
	
	// Create inline keyboard for navigation
	if totalPages > 1 {
		var buttons [][]tgbotapi.InlineKeyboardButton
		var row []tgbotapi.InlineKeyboardButton
		
		// Previous button
		if page > 1 {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("⬅️ Previous", fmt.Sprintf("groupstats:%d", page-1)))
		}
		
		// Page indicator
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📄 %d/%d", page, totalPages), "noop"))
		
		// Next button
		if page < totalPages {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Next ➡️", fmt.Sprintf("groupstats:%d", page+1)))
		}
		
		buttons = append(buttons, row)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
		return response.String(), keyboard, true
	}
	
	return response.String(), tgbotapi.InlineKeyboardMarkup{}, false
}

// sendMessageWithAutoSplit sends a message, automatically splitting if too long
func (h *CommandHandler) sendMessageWithAutoSplit(chatID int64, text string) {
	const maxLength = 4000 // Leave some margin under 4096
	
	if len(text) <= maxLength {
		// Send as single message
		msg := tgbotapi.NewMessage(chatID, text)
		if _, err := h.bot.GetAPI().Send(msg); err != nil {
			logger.Error("Failed to send message: %v", err)
			h.bot.sendMessage(chatID, "❌ Failed to send summary.")
		}
		return
	}
	
	// Split into multiple messages
	chunks := splitMessageAtSectionBreaks(text, maxLength)
	
	logger.Info("📄 Message too long (%d chars), splitting into %d parts", len(text), len(chunks))
	
	for i, chunk := range chunks {
		// Add part indicator
		partHeader := fmt.Sprintf("📄 Part %d/%d\n\n", i+1, len(chunks))
		messageText := partHeader + chunk
		
		msg := tgbotapi.NewMessage(chatID, messageText)
		if _, err := h.bot.GetAPI().Send(msg); err != nil {
			logger.Error("Failed to send part %d/%d: %v", i+1, len(chunks), err)
			continue
		}
		
		logger.Info("✅ Sent part %d/%d", i+1, len(chunks))
		
		// Small delay between messages
		time.Sleep(500 * time.Millisecond)
	}
}

// sendMessageWithoutHeader sends a message without adding "Part X/Y" header
// Used for formatted summaries that already have their own headers
// Note: ParseMode is disabled to prevent Markdown parsing errors from AI-generated content
func (h *CommandHandler) sendMessageWithoutHeader(chatID int64, text string) {
	const maxLength = 4000 // Leave some margin under 4096
	
	if len(text) <= maxLength {
		// Send as single message WITHOUT ParseMode to avoid Markdown errors
		msg := tgbotapi.NewMessage(chatID, text)
		// Do NOT set ParseMode - summaries may contain malformed Markdown from AI
		if _, err := h.bot.GetAPI().Send(msg); err != nil {
			logger.Error("Failed to send message: %v", err)
			// Try to send error message without Markdown
			errorMsg := tgbotapi.NewMessage(chatID, "❌ Failed to send summary. The content may contain formatting issues.")
			h.bot.GetAPI().Send(errorMsg)
		}
		return
	}
	
	// Split into multiple messages WITHOUT adding part headers
	chunks := splitMessageAtSectionBreaks(text, maxLength)
	
	logger.Info("📄 Message too long (%d chars), splitting into %d parts (no headers)", len(text), len(chunks))
	
	for i, chunk := range chunks {
		msg := tgbotapi.NewMessage(chatID, chunk)
		// Do NOT set ParseMode - summaries may contain malformed Markdown from AI
		if _, err := h.bot.GetAPI().Send(msg); err != nil {
			logger.Error("Failed to send part %d/%d: %v", i+1, len(chunks), err)
			continue
		}
		
		logger.Info("✅ Sent part %d/%d", i+1, len(chunks))
		
		// Small delay between messages
		time.Sleep(500 * time.Millisecond)
	}
}

// splitMessageAtSectionBreaks splits message at section headers to avoid breaking content
func splitMessageAtSectionBreaks(text string, maxLength int) []string {
	if len(text) <= maxLength {
		return []string{text}
	}
	
	var chunks []string
	lines := strings.Split(text, "\n")
	
	var currentChunk strings.Builder
	
	for _, line := range lines {
		lineWithNewline := line + "\n"
		
		// Check if adding this line would exceed limit
		if currentChunk.Len()+len(lineWithNewline) > maxLength {
			// Save current chunk if it has content
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
			}
			
			// If a single line is too long, split it forcefully
			if len(lineWithNewline) > maxLength {
				for len(lineWithNewline) > maxLength {
					chunks = append(chunks, lineWithNewline[:maxLength])
					lineWithNewline = lineWithNewline[maxLength:]
				}
				if len(lineWithNewline) > 0 {
					currentChunk.WriteString(lineWithNewline)
				}
				continue
			}
		}
		
		currentChunk.WriteString(lineWithNewline)
	}
	
	// Add the last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}
	
	return chunks
}
