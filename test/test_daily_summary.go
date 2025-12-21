package main

import (
	"fmt"
	"log"
	"strings"
	"time"
	"telegram-summarizer/internal/ai"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/gemini"
	"telegram-summarizer/internal/logger"
	"telegram-summarizer/internal/summarizer"
)

func main() {
	fmt.Println("================================================")
	fmt.Println("🧪 TESTING DAILY SUMMARY")
	fmt.Println("================================================")
	
	// Initialize logger
	logger.Init(true)
	
	// Initialize database
	database, err := db.NewDB("telegram_bot.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	
	// Initialize Gemini client
	geminiAPIKey := "AIzaSyAbyIAJ9Jv8M_LVhBnd6J0FNvioAxGJA3w"
	geminiClient := gemini.NewClient(geminiAPIKey, "gemini-2.0-flash-exp")
	
	// Initialize AI providers
	providers := []ai.AIProvider{
		geminiClient,
		ai.NewCopilotProvider("think"),
		ai.NewGPT5Provider(),
		ai.NewCopilotProvider("default"),
		ai.NewYPAIProvider(),
		ai.NewDelineProvider("copilot-think"),
		ai.NewDelineProvider("copilot"),
		ai.NewDelineProvider("openai"),
		ai.NewElrayyXmlProvider("venice"),
		ai.NewElrayyXmlProvider("powerbrain"),
		ai.NewElrayyXmlProvider("lumin"),
		ai.NewElrayyXmlProvider("chatgpt"),
		ai.NewElrayyXmlProvider("perplexity"),
		ai.NewElrayyXmlProvider("felo"),
		ai.NewElrayyXmlProvider("gemini"),
		ai.NewElrayyXmlProvider("copilot"),
		ai.NewElrayyXmlProvider("alisia"),
		ai.NewElrayyXmlProvider("biblegpt"),
	}
	
	fallbackManager := ai.NewFallbackManager(providers)
	
	// Initialize summarizer
	summarizerService := summarizer.NewSummarizer(database, geminiClient, fallbackManager)
	
	fmt.Println()
	fmt.Println("✅ Services initialized")
	fmt.Println()
	
	// Get active groups
	groups := database.GetActiveGroups()
	
	if len(groups) == 0 {
		fmt.Println("❌ No active groups found!")
		return
	}
	
	fmt.Printf("📋 Found %d active groups\n", len(groups))
	fmt.Println()
	
	// Time range: today (00:00 - 23:59)
	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	
	fmt.Printf("⏰ Time range: %s - %s\n", startTime.Format("2006-01-02 15:04"), endTime.Format("15:04"))
	fmt.Println()
	
	successCount := 0
	
	for i, group := range groups {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Processing group %d/%d: %s\n", i+1, len(groups), group.GroupName)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		
		// Get 1h summaries from today
		summaries, err := database.GetSummariesByTypeAndDate(group.ChatID, "1h", startTime)
		if err != nil {
			fmt.Printf("❌ Failed to get 1h summaries: %v\n", err)
			continue
		}
		
		fmt.Printf("📝 1h summaries found: %d\n", len(summaries))
		
		if len(summaries) == 0 {
			fmt.Printf("⏭️  Skipping (no 1h summaries)\n")
			continue
		}
		
		// Aggregate summaries
		var aggregatedText strings.Builder
		aggregatedText.WriteString(fmt.Sprintf("Berikut adalah ringkasan per jam untuk grup %s:\n\n", group.GroupName))
		
		totalMsgs := 0
		for _, summary := range summaries {
			totalMsgs += summary.MessageCount
			aggregatedText.WriteString(fmt.Sprintf("## Periode %s - %s (%d pesan)\n",
				summary.PeriodStart.Format("15:04"),
				summary.PeriodEnd.Format("15:04"),
				summary.MessageCount))
			aggregatedText.WriteString(summary.SummaryText)
			aggregatedText.WriteString("\n\n---\n\n")
		}
		
		fmt.Printf("📊 Aggregated: %d chars from %d messages\n", len(aggregatedText.String()), totalMsgs)
		
		// Create pseudo-messages
		pseudoMessages := []db.Message{
			{
				ChatID:        group.ChatID,
				Username:      "System",
				MessageText:   aggregatedText.String(),
				MessageLength: len(aggregatedText.String()),
				Timestamp:     endTime,
			},
		}
		
		// Progress callback
		progressCallback := func(msg string) {
			fmt.Printf("   📝 %s\n", msg)
		}
		
		summaryCallback := func(partial string) {
			fmt.Printf("   📊 Partial summary: %d chars\n", len(partial))
		}
		
		// Generate daily summary
		fmt.Println("🤖 Generating daily summary...")
		dailySummary, err := summarizerService.GenerateSummaryHierarchical(
			pseudoMessages,
			group.GroupName,
			startTime,
			endTime,
			progressCallback,
			summaryCallback,
		)
		
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			continue
		}
		
		fmt.Printf("✅ Daily summary generated: %d chars\n", len(dailySummary))
		
		// Save to database
		dbSummary := &db.Summary{
			ChatID:       group.ChatID,
			SummaryType:  "daily-test",
			PeriodStart:  startTime,
			PeriodEnd:    endTime,
			SummaryText:  dailySummary,
			MessageCount: totalMsgs,
		}
		
		if err := database.SaveSummary(dbSummary); err != nil {
			fmt.Printf("❌ Failed to save: %v\n", err)
			continue
		}
		
		fmt.Printf("💾 Saved to database (ID: %d)\n", dbSummary.ID)
		
		// Display summary preview
		fmt.Println()
		fmt.Println("📄 Summary Preview (first 500 chars):")
		fmt.Println("─────────────────────────────────────────────")
		preview := dailySummary
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Println(preview)
		fmt.Println("─────────────────────────────────────────────")
		
		successCount++
		fmt.Println()
	}
	
	fmt.Println("================================================")
	fmt.Println("📊 TEST SUMMARY")
	fmt.Println("================================================")
	fmt.Printf("✅ Success: %d/%d groups\n", successCount, len(groups))
	fmt.Println()
	fmt.Println("✅ Test complete!")
}
