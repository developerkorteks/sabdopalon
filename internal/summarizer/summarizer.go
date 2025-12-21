package summarizer

import (
	"fmt"
	"strings"
	"telegram-summarizer/internal/ai"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/gemini"
	"telegram-summarizer/internal/logger"
	"time"
)

// Summarizer handles chat summarization
type Summarizer struct {
	database           *db.DB
	geminiClient       *gemini.Client
	aiProvider         ai.AIProvider // Fallback-enabled AI provider
	promptManager      *PromptManager
	metadataParser     *MetadataParser
	fallbackManager    *ai.FallbackManager // Direct access to fallback manager
	chunkManager       *ChunkManager       // Chunk manager for message splitting
}

// NewSummarizer creates a new summarizer instance with fallback AI providers
func NewSummarizer(database *db.DB, geminiClient *gemini.Client) *Summarizer {
	// Setup extended fallback chain with multiple API sources
	providers := []ai.AIProvider{
		// PRIMARY: Official Google Gemini API
		geminiClient,                     // Primary: Gemini (Official)
		
		// TIER 1: Yupra.my.id API providers
		ai.NewCopilotClient(true),        // Fallback 1: Copilot Think Deeper (Yupra)
		ai.NewGPT5Client(),                // Fallback 2: GPT-5 Smart (Yupra)
		ai.NewCopilotClient(false),       // Fallback 3: Copilot Default (Yupra)
		ai.NewYPAIClient(),                // Fallback 4: YP AI (Yupra)
		
		// TIER 2: Deline API providers - High Quality
		ai.NewDelineCopilotThinkClient(),  // Fallback 5: Copilot Think (Deline)
		ai.NewDelineCopilotClient(),       // Fallback 6: Copilot (Deline)
		ai.NewDelineOpenAIClient(),        // Fallback 7: OpenAI (Deline)
		
		// TIER 3: ElrayyXml API providers - High Quality Models
		ai.NewVeniceAIClient(),            // Fallback 8: Venice AI (ElrayyXml)
		ai.NewPowerBrainAIClient(),        // Fallback 9: PowerBrain AI (ElrayyXml)
		ai.NewLuminAIClient(),             // Fallback 10: Lumin AI (ElrayyXml)
		ai.NewElrayyChatGPTClient(),       // Fallback 11: ChatGPT (ElrayyXml)
		
		// TIER 4: ElrayyXml API providers - Additional Models
		ai.NewPerplexityAIClient(),        // Fallback 12: Perplexity AI (ElrayyXml)
		ai.NewFeloAIClient(),              // Fallback 13: Felo AI (ElrayyXml)
		ai.NewElrayyGeminiClient(),        // Fallback 14: Gemini (ElrayyXml)
		ai.NewElrayyCopilotClient(),       // Fallback 15: Copilot (ElrayyXml)
		
		// TIER 5: ElrayyXml API providers - Special Purpose
		ai.NewAlisiaClient(),              // Fallback 16: Alisia AI (ElrayyXml)
		ai.NewBibleGPTClient(),            // Fallback 17: BibleGPT (ElrayyXml)
	}
	
	fallbackManager := ai.NewFallbackManager(providers)
	
	logger.Info("🔄 AI Provider chain configured with %d providers:", len(providers))
	logger.Info("   Primary: Gemini (Official)")
	logger.Info("   Tier 1: Yupra.my.id (4 providers)")
	logger.Info("   Tier 2: Deline.web.id (3 providers)")
	logger.Info("   Tier 3-5: ElrayyXml (10 providers)")
	logger.Info("   Total: 18 AI providers with automatic fallback!")
	logger.Info("   Hierarchical chunking: Enabled for large chats")
	
	return &Summarizer{
		database:        database,
		geminiClient:    geminiClient,
		aiProvider:      fallbackManager,
		promptManager:   NewPromptManager(),
		metadataParser:  NewMetadataParser(),
		fallbackManager: fallbackManager,
		chunkManager:    NewChunkManager(),
	}
}

// GenerateSummary generates a summary with fallback support
func (s *Summarizer) GenerateSummary(messageText string, summaryType string) (string, error) {
	logger.Info("Generating %s summary with fallback AI providers", summaryType)
	
	// Use fallback manager (tries all providers in order)
	summary, err := s.aiProvider.GenerateSummary(messageText)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary (all providers failed): %w", err)
	}
	
	logger.Info("✅ Summary generated successfully")
	return summary, nil
}

// GenerateSummaryHierarchical generates a summary using hierarchical chunking
// This method automatically splits large message sets into manageable chunks
func (s *Summarizer) GenerateSummaryHierarchical(messages []db.Message, groupName string, startTime, endTime time.Time, progressCallback func(string), summaryCallback func(string)) (string, error) {
	logger.Info("Starting hierarchical summary generation for %d messages", len(messages))
	
	// Create hierarchical summarizer with both callbacks
	hierarchical := NewHierarchicalSummarizer(s.fallbackManager, progressCallback, summaryCallback)
	
	// Generate summary with automatic chunking (may send multiple partial summaries)
	summary, err := hierarchical.SummarizeMessages(messages, groupName, startTime, endTime)
	if err != nil {
		return "", fmt.Errorf("hierarchical summarization failed: %w", err)
	}
	
	logger.Info("✅ Hierarchical summary completed: %d chars", len(summary))
	return summary, nil
}

// CreateIncrementalSummary creates a summary for the specified time window
func (s *Summarizer) CreateIncrementalSummary(chatID int64, duration time.Duration) (string, error) {
	logger.Info("Creating incremental summary for chat %d (duration: %v)", chatID, duration)
	
	// Calculate time range
	endTime := time.Now()
	startTime := endTime.Add(-duration)
	
	logger.Debug("Time range: %s to %s", 
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"))
	
	// Get messages from database
	messages, err := s.database.GetMessagesByTimeRange(chatID, startTime, endTime)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}
	
	logger.Info("Found %d messages in time range", len(messages))
	
	// Check if there are enough messages to summarize
	if len(messages) == 0 {
		logger.Warn("No messages to summarize")
		return "", fmt.Errorf("no messages in time range")
	}
	
	if len(messages) < 5 {
		logger.Warn("Too few messages to summarize (%d)", len(messages))
		return "", fmt.Errorf("insufficient messages for summary (minimum 5)")
	}
	
	// Format messages for Gemini
	formattedMessages := s.formatMessagesForSummary(messages)
	logger.Debug("Formatted messages length: %d characters", len(formattedMessages))
	
	// Generate summary using Gemini
	summary, err := s.geminiClient.GenerateChatSummary(formattedMessages, "incremental")
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}
	
	// Save summary to database
	summaryRecord := &db.Summary{
		ChatID:       chatID,
		SummaryType:  "incremental",
		PeriodStart:  startTime,
		PeriodEnd:    endTime,
		SummaryText:  summary,
		MessageCount: len(messages),
	}
	
	if err := s.database.SaveSummary(summaryRecord); err != nil {
		logger.Error("Failed to save summary to database: %v", err)
		// Don't fail, summary was generated successfully
	}
	
	logger.Info("✅ Incremental summary created successfully (ID: %d)", summaryRecord.ID)
	return summary, nil
}

// CreateDailySummary creates a daily summary from incremental summaries
func (s *Summarizer) CreateDailySummary(chatID int64) (string, error) {
	logger.Info("Creating daily summary for chat %d", chatID)
	
	// Calculate time range (last 24 hours)
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)
	
	logger.Debug("Daily summary range: %s to %s",
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"))
	
	// Get all incremental summaries from last 24 hours
	summaries, err := s.database.GetSummaries(chatID, "incremental", 10)
	if err != nil {
		return "", fmt.Errorf("failed to get summaries: %w", err)
	}
	
	// Filter summaries within 24 hours
	var recentSummaries []db.Summary
	for _, summary := range summaries {
		if summary.PeriodStart.After(startTime) {
			recentSummaries = append(recentSummaries, summary)
		}
	}
	
	logger.Info("Found %d incremental summaries in last 24 hours", len(recentSummaries))
	
	if len(recentSummaries) == 0 {
		logger.Warn("No incremental summaries to merge")
		return "", fmt.Errorf("no incremental summaries found")
	}
	
	// Format summaries for merging
	formattedSummaries := s.formatSummariesForDaily(recentSummaries)
	logger.Debug("Formatted summaries length: %d characters", len(formattedSummaries))
	
	// Generate daily summary using Gemini
	dailySummary, err := s.geminiClient.GenerateChatSummary(formattedSummaries, "daily")
	if err != nil {
		return "", fmt.Errorf("failed to generate daily summary: %w", err)
	}
	
	// Calculate total message count
	totalMessages := 0
	for _, s := range recentSummaries {
		totalMessages += s.MessageCount
	}
	
	// Save daily summary to database
	summaryRecord := &db.Summary{
		ChatID:       chatID,
		SummaryType:  "daily",
		PeriodStart:  startTime,
		PeriodEnd:    endTime,
		SummaryText:  dailySummary,
		MessageCount: totalMessages,
	}
	
	if err := s.database.SaveSummary(summaryRecord); err != nil {
		logger.Error("Failed to save daily summary to database: %v", err)
		// Don't fail, summary was generated successfully
	}
	
	logger.Info("✅ Daily summary created successfully (ID: %d, Total messages: %d)", 
		summaryRecord.ID, totalMessages)
	return dailySummary, nil
}

// formatMessagesForSummary formats messages in a readable format for Gemini
func (s *Summarizer) formatMessagesForSummary(messages []db.Message) string {
	var builder strings.Builder
	
	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("15:04")
		builder.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.Username, msg.MessageText))
	}
	
	return builder.String()
}

// formatSummariesForDaily formats incremental summaries for daily summary
func (s *Summarizer) formatSummariesForDaily(summaries []db.Summary) string {
	var builder strings.Builder
	
	for i, summary := range summaries {
		timeRange := fmt.Sprintf("%s - %s", 
			summary.PeriodStart.Format("15:04"),
			summary.PeriodEnd.Format("15:04"))
		builder.WriteString(fmt.Sprintf("Summary %d (%s): %s\n\n", 
			i+1, timeRange, summary.SummaryText))
	}
	
	return builder.String()
}

// GetPromptManager returns the prompt manager instance
func (s *Summarizer) GetPromptManager() *PromptManager {
	return s.promptManager
}

// GetMetadataParser returns the metadata parser instance
func (s *Summarizer) GetMetadataParser() *MetadataParser {
	return s.metadataParser
}

// GetChatStats returns statistics for a chat
func (s *Summarizer) GetChatStats(chatID int64, duration time.Duration) (*ChatStats, error) {
	logger.Debug("Getting chat stats for chat %d (duration: %v)", chatID, duration)
	
	endTime := time.Now()
	startTime := endTime.Add(-duration)
	
	messages, err := s.database.GetMessagesByTimeRange(chatID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	
	// Calculate statistics
	stats := &ChatStats{
		TotalMessages: len(messages),
		TimeRange:     duration,
		UserStats:     make(map[string]int),
	}
	
	for _, msg := range messages {
		stats.UserStats[msg.Username]++
	}
	
	// Find most active user
	maxCount := 0
	for username, count := range stats.UserStats {
		if count > maxCount {
			maxCount = count
			stats.MostActiveUser = username
		}
	}
	
	return stats, nil
}

// ChatStats contains statistics about a chat
type ChatStats struct {
	TotalMessages  int
	MostActiveUser string
	UserStats      map[string]int
	TimeRange      time.Duration
}
