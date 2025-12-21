package scheduler

import (
	"fmt"
	"strings"
	"time"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/logger"
	"telegram-summarizer/internal/summarizer"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Scheduler handles scheduled summary generation
type Scheduler struct {
	database     *db.DB
	summarizer   *summarizer.Summarizer
	bot          *tgbotapi.BotAPI
	targetChatID int64  // Chat ID to send summaries to
	stopCh       chan struct{}
	ticker1h     *time.Ticker
	tickerDaily  *time.Ticker
}

// NewScheduler creates a new scheduler
func NewScheduler(database *db.DB, summarizer *summarizer.Summarizer, bot *tgbotapi.BotAPI, targetChatID int64) *Scheduler {
	return &Scheduler{
		database:    database,
		summarizer:  summarizer,
		bot:         bot,
		targetChatID: targetChatID,
		stopCh:      make(chan struct{}),
	}
}

// Start starts the scheduler with both 1h and daily summaries
func (s *Scheduler) Start(dailySummaryTime string) {
	logger.Info("📅 Starting schedulers...")
	logger.Info("  ⏰ 1-hour summaries: Every hour (00:00, 01:00, 02:00, ... 23:00)")
	logger.Info("  🌅 Daily summary: %s", dailySummaryTime)
	
	// Start 1-hour scheduler
	go s.run1HourScheduler()
	
	// Start daily scheduler
	go s.runDailyScheduler(dailySummaryTime)
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.ticker1h != nil {
		s.ticker1h.Stop()
	}
	if s.tickerDaily != nil {
		s.tickerDaily.Stop()
	}
	close(s.stopCh)
	logger.Info("🛑 Scheduler stopped")
}

// run1HourScheduler runs 1-hour summary generation
func (s *Scheduler) run1HourScheduler() {
	logger.Info("⏰ Starting 1-hour summary scheduler")
	
	// Align to next hour mark (00:00, 01:00, 02:00, etc)
	waitDuration := s.alignToNextHour()
	logger.Info("⏰ Next 1h summary in: %s", formatDuration(waitDuration))
	
	// Wait until first aligned time
	time.Sleep(waitDuration)
	
	// Generate first summary immediately
	s.generate1HourSummaries()
	
	// Then run every 1 hour
	s.ticker1h = time.NewTicker(1 * time.Hour)
	defer s.ticker1h.Stop()
	
	for {
		select {
		case <-s.ticker1h.C:
			s.generate1HourSummaries()
		case <-s.stopCh:
			logger.Info("⏰ 1-hour scheduler stopped")
			return
		}
	}
}

// alignToNextHour calculates time until next hour mark
func (s *Scheduler) alignToNextHour() time.Duration {
	now := time.Now()
	
	// Next hour mark (round up to next hour)
	nextHour := now.Add(1 * time.Hour).Truncate(1 * time.Hour)
	
	return time.Until(nextHour)
}

// generate1HourSummaries generates summaries for all active groups
func (s *Scheduler) generate1HourSummaries() {
	logger.Info("🕐 Generating 1-hour summaries...")
	
	// Get all active groups
	groups := s.database.GetActiveGroups()
	
	if len(groups) == 0 {
		logger.Info("No active groups for 1h summary")
		return
	}
	
	logger.Info("Processing %d active groups", len(groups))
	
	// Time range: last 1 hour
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)
	
	successCount := 0
	totalMessages := 0
	
	for _, group := range groups {
		logger.Info("📝 1h summary for: %s (ID: %d)", group.GroupName, group.ChatID)
		
		// Get messages from last 1 hour
		messages, err := s.database.GetMessagesByTimeRange(group.ChatID, startTime, endTime)
		if err != nil {
			logger.Error("Failed to get messages: %v", err)
			continue
		}
		
		if len(messages) < 3 {
			logger.Info("⏭️  Skipping %s: only %d messages (need at least 3)", group.GroupName, len(messages))
			continue
		}
		
		// Use hierarchical streaming summarization (same as manual summary)
		// This prevents "prompt too large" errors for active groups
		progressCallback := func(progressMsg string) {
			logger.Debug("1h summary progress: %s", progressMsg)
		}
		
		summaryCallback := func(partialSummary string) {
			logger.Debug("1h summary partial generated (%d chars)", len(partialSummary))
		}
		
		// Generate summary with hierarchical chunking (handles any size)
		summaryText, err := s.summarizer.GenerateSummaryHierarchical(messages, group.GroupName, startTime, endTime, progressCallback, summaryCallback)
		if err != nil {
			logger.Error("Failed to generate summary: %v", err)
			continue
		}
		
		// Parse metadata
		parser := s.summarizer.GetMetadataParser()
		metadata := parser.Parse(summaryText)
		
		// Save summary with metadata
		summary := &db.Summary{
			ChatID:            group.ChatID,
			SummaryType:       "1h",
			PeriodStart:       startTime,
			PeriodEnd:         endTime,
			SummaryText:       summaryText,
			MessageCount:      len(messages),
			Sentiment:         metadata.Sentiment,
			CredibilityScore:  metadata.CredibilityScore,
			ProductsMentioned: metadata.ProductsJSON,
			RedFlagsCount:     metadata.RedFlagsCount,
			ValidationStatus:  metadata.ValidationStatus,
		}
		
		if err := s.database.SaveSummary(summary); err != nil {
			logger.Error("Failed to save summary: %v", err)
			continue
		}
		
		// Save product mentions
		for i := range metadata.Products {
			metadata.Products[i].SummaryID = summary.ID
			if err := s.database.SaveProductMention(&metadata.Products[i]); err != nil {
				logger.Error("Failed to save product mention: %v", err)
			}
		}
		
		logger.Info("✅ 1h summary saved for %s (%d messages, %d products)", 
			group.GroupName, len(messages), len(metadata.Products))
		
		successCount++
		totalMessages += len(messages)
	}
	
	logger.Info("✅ 1-hour summaries complete: %d/%d groups, %d messages processed", 
		successCount, len(groups), totalMessages)
}

// runDailyScheduler runs the daily summary job
func (s *Scheduler) runDailyScheduler(targetTime string) {
	for {
		// Parse target time (format: "23:00")
		now := time.Now()
		targetHour, targetMin := parseTime(targetTime)
		
		// Calculate next run time
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), targetHour, targetMin, 0, 0, now.Location())
		if nextRun.Before(now) {
			// If target time already passed today, schedule for tomorrow
			nextRun = nextRun.Add(24 * time.Hour)
		}
		
		waitDuration := time.Until(nextRun)
		logger.Info("⏰ Next daily summary scheduled at: %s (in %s)", nextRun.Format("2006-01-02 15:04:05"), formatDuration(waitDuration))
		
		// Wait until next run or stop signal
		select {
		case <-time.After(waitDuration):
			// Run daily summary
			s.runDailySummaryForAllGroups()
		case <-s.stopCh:
			return
		}
	}
}

// runDailySummaryForAllGroups generates summary for all active groups
func (s *Scheduler) runDailySummaryForAllGroups() {
	logger.Info("🌅 Starting daily summary generation for all active groups...")
	
	// Get all active groups
	groups := s.database.GetTrackedGroups()
	activeGroups := make([]db.TrackedGroup, 0)
	
	for _, group := range groups {
		if group.IsActive == 1 {
			activeGroups = append(activeGroups, group)
		}
	}
	
	if len(activeGroups) == 0 {
		logger.Info("ℹ️  No active groups to summarize")
		return
	}
	
	logger.Info("📋 Found %d active group(s) to summarize", len(activeGroups))
	
	successCount := 0
	failCount := 0
	
	// Generate summary for each active group
	for _, group := range activeGroups {
		logger.Info("📝 Processing group: %s (ID: %d)", group.GroupName, group.ChatID)
		
		if err := s.generateAndSendDailySummary(group); err != nil {
			logger.Error("❌ Failed to generate summary for %s: %v", group.GroupName, err)
			failCount++
		} else {
			successCount++
		}
		
		// Small delay between groups to avoid rate limiting
		time.Sleep(2 * time.Second)
	}
	
	logger.Info("✅ Daily summary complete: %d succeeded, %d failed", successCount, failCount)
	
	// Send completion report to target chat
	report := fmt.Sprintf("📊 Daily Summary Report\n\n"+
		"✅ Successfully summarized: %d groups\n"+
		"❌ Failed: %d groups\n"+
		"📅 Date: %s",
		successCount, failCount, time.Now().Format("2006-01-02"))
	
	msg := tgbotapi.NewMessage(s.targetChatID, report)
	s.bot.Send(msg)
}

// generateAndSendDailySummary generates and sends summary for a single group
// This now aggregates from 1h summaries instead of direct messages
func (s *Scheduler) generateAndSendDailySummary(group db.TrackedGroup) error {
	// Get time range for today (start of day to now)
	endTime := time.Now()
	startTime := time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 0, 0, 0, 0, endTime.Location())
	
	// Get all 1h summaries from today
	summaries := s.database.GetSummariesByTimeRange(group.ChatID, "1h", startTime, endTime)
	
	if len(summaries) == 0 {
		logger.Info("ℹ️  No 1h summaries for %s today, skipping", group.GroupName)
		return nil
	}
	
	logger.Info("Found %d one-hour summaries for %s", len(summaries), group.GroupName)
	
	// Combine all 1h summaries into one text
	var combinedText strings.Builder
	totalMessages := 0
	
	for i, summary := range summaries {
		combinedText.WriteString(fmt.Sprintf("=== Periode %d: %s - %s ===\n\n",
			i+1,
			summary.PeriodStart.Format("15:04"),
			summary.PeriodEnd.Format("15:04")))
		combinedText.WriteString(summary.SummaryText)
		combinedText.WriteString("\n\n")
		totalMessages += summary.MessageCount
	}
	
	// For daily summary, we aggregate 1h summaries which are already summarized
	// Create pseudo-messages from summaries for hierarchical processing
	// This ensures consistency with manual summary format
	
	logger.Info("Generating daily summary from %d hourly summaries", len(summaries))
	
	// Build a comprehensive text from all 1h summaries
	var aggregatedText strings.Builder
	aggregatedText.WriteString(fmt.Sprintf("Berikut adalah ringkasan per jam untuk grup %s:\n\n", group.GroupName))
	
	for _, summary := range summaries {
		aggregatedText.WriteString(fmt.Sprintf("## Periode %s - %s (%d pesan)\n",
			summary.PeriodStart.Format("15:04"),
			summary.PeriodEnd.Format("15:04"),
			summary.MessageCount))
		aggregatedText.WriteString(summary.SummaryText)
		aggregatedText.WriteString("\n\n---\n\n")
	}
	
	// Use hierarchical summarization on aggregated summaries
	// Convert aggregated text to "messages" format for consistency
	pseudoMessages := []db.Message{
		{
			ChatID:        group.ChatID,
			Username:      "System",
			MessageText:   aggregatedText.String(),
			MessageLength: len(aggregatedText.String()),
			Timestamp:     endTime,
		},
	}
	
	progressCallback := func(progressMsg string) {
		logger.Debug("Daily summary progress: %s", progressMsg)
	}
	
	summaryCallback := func(partialSummary string) {
		logger.Debug("Daily summary partial generated (%d chars)", len(partialSummary))
	}
	
	// Generate daily summary with hierarchical chunking
	dailySummaryText, err := s.summarizer.GenerateSummaryHierarchical(pseudoMessages, group.GroupName, startTime, endTime, progressCallback, summaryCallback)
	if err != nil {
		return fmt.Errorf("failed to generate daily summary: %w", err)
	}
	
	// Parse metadata
	parser := s.summarizer.GetMetadataParser()
	metadata := parser.Parse(dailySummaryText)
	
	// Save daily summary with metadata
	dbSummary := &db.Summary{
		ChatID:            group.ChatID,
		SummaryType:       "daily",
		PeriodStart:       startTime,
		PeriodEnd:         endTime,
		SummaryText:       dailySummaryText,
		MessageCount:      totalMessages,
		Sentiment:         metadata.Sentiment,
		CredibilityScore:  metadata.CredibilityScore,
		ProductsMentioned: metadata.ProductsJSON,
		RedFlagsCount:     metadata.RedFlagsCount,
		ValidationStatus:  metadata.ValidationStatus,
	}
	
	if err := s.database.SaveSummary(dbSummary); err != nil {
		logger.Error("Failed to save daily summary: %v", err)
		// Don't fail, just log
	} else {
		// Save product mentions
		for i := range metadata.Products {
			metadata.Products[i].SummaryID = dbSummary.ID
			if err := s.database.SaveProductMention(&metadata.Products[i]); err != nil {
				logger.Error("Failed to save product mention: %v", err)
			}
		}
	}
	
	// Format response
	var response strings.Builder
	response.WriteString(fmt.Sprintf("📝 Daily Summary for %s\n\n", group.GroupName))
	response.WriteString(fmt.Sprintf("📅 Date: %s\n", endTime.Format("2006-01-02")))
	response.WriteString(fmt.Sprintf("💬 Total Messages: %d\n", totalMessages))
	response.WriteString(fmt.Sprintf("📊 Based on %d one-hour summaries\n\n", len(summaries)))
	response.WriteString("━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	response.WriteString(dailySummaryText)
	response.WriteString("\n\n━━━━━━━━━━━━━━━━━━━━━━━\n")
	response.WriteString("Generated by AI ✨")
	
	// Auto-split if message is too long
	s.sendMessageWithAutoSplit(s.targetChatID, response.String())
	
	logger.Info("✅ Daily summary sent for %s", group.GroupName)
	
	// Delete old messages (older than 24h) after successful daily summary
	cleanupTime := endTime.Add(-24 * time.Hour)
	deletedCount, err := s.database.DeleteMessagesOlderThan(group.ChatID, cleanupTime)
	if err != nil {
		logger.Error("⚠️  Failed to cleanup old messages for %s: %v", group.GroupName, err)
	} else if deletedCount > 0 {
		logger.Info("🗑️  Cleaned up %d old messages for %s", deletedCount, group.GroupName)
	}
	
	return nil
}

// parseTime parses time string in format "HH:MM"
func parseTime(timeStr string) (hour, min int) {
	fmt.Sscanf(timeStr, "%d:%d", &hour, &min)
	return
}

// formatDuration formats duration in human-readable format
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// sendMessageWithAutoSplit sends a message, automatically splitting if too long
func (s *Scheduler) sendMessageWithAutoSplit(chatID int64, text string) {
	const maxLength = 4000 // Leave margin under 4096
	
	if len(text) <= maxLength {
		// Send as single message
		msg := tgbotapi.NewMessage(chatID, text)
		if _, err := s.bot.Send(msg); err != nil {
			logger.Error("Failed to send message: %v", err)
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
		if _, err := s.bot.Send(msg); err != nil {
			logger.Error("Failed to send part %d/%d: %v", i+1, len(chunks), err)
			continue
		}
		
		logger.Info("✅ Sent part %d/%d", i+1, len(chunks))
		
		// Small delay between messages
		time.Sleep(500 * time.Millisecond)
	}
}

// splitMessageAtSectionBreaks splits message at section headers
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
			// Save current chunk
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
			}
			
			// If single line too long, split forcefully
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
	
	// Add last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}
	
	return chunks
}
