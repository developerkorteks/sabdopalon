package main

import (
	"fmt"
	"log"
	"time"
	"telegram-summarizer/internal/ai"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/gemini"
	"telegram-summarizer/internal/logger"
	"telegram-summarizer/internal/summarizer"
)

func main() {
	fmt.Println("================================================")
	fmt.Println("🧪 TESTING 1-HOUR SUMMARY")
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
		fmt.Println("   Please activate groups first: /enable <chat_id>")
		return
	}
	
	fmt.Printf("📋 Found %d active groups\n", len(groups))
	fmt.Println()
	
	// Time range: last 1 hour
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)
	
	fmt.Printf("⏰ Time range: %s - %s\n", startTime.Format("15:04"), endTime.Format("15:04"))
	fmt.Println()
	
	successCount := 0
	totalMessages := 0
	
	for i, group := range groups {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Processing group %d/%d: %s\n", i+1, len(groups), group.GroupName)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		
		// Get messages
		messages, err := database.GetMessagesByTimeRange(group.ChatID, startTime, endTime)
		if err != nil {
			fmt.Printf("❌ Failed to get messages: %v\n", err)
			continue
		}
		
		fmt.Printf("📨 Messages found: %d\n", len(messages))
		
		if len(messages) < 3 {
			fmt.Printf("⏭️  Skipping (< 3 messages)\n")
			continue
		}
		
		totalMessages += len(messages)
		
		// Progress callback
		progressCallback := func(msg string) {
			fmt.Printf("   📝 %s\n", msg)
		}
		
		summaryCallback := func(partial string) {
			fmt.Printf("   📊 Partial summary: %d chars\n", len(partial))
		}
		
		// Generate summary
		fmt.Println("🤖 Generating summary...")
		summary, err := summarizerService.GenerateSummaryHierarchical(
			messages, 
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
		
		fmt.Printf("✅ Summary generated: %d chars\n", len(summary))
		
		// Save to database
		dbSummary := &db.Summary{
			ChatID:       group.ChatID,
			SummaryType:  "1h",
			PeriodStart:  startTime,
			PeriodEnd:    endTime,
			SummaryText:  summary,
			MessageCount: len(messages),
		}
		
		if err := database.SaveSummary(dbSummary); err != nil {
			fmt.Printf("❌ Failed to save: %v\n", err)
			continue
		}
		
		fmt.Printf("💾 Saved to database (ID: %d)\n", dbSummary.ID)
		successCount++
		fmt.Println()
	}
	
	fmt.Println("================================================")
	fmt.Println("📊 TEST SUMMARY")
	fmt.Println("================================================")
	fmt.Printf("✅ Success: %d/%d groups\n", successCount, len(groups))
	fmt.Printf("📨 Total messages processed: %d\n", totalMessages)
	fmt.Println()
	
	// Show saved summaries
	fmt.Println("📝 Saved 1h summaries:")
	fmt.Println()
	fmt.Println("To view:")
	fmt.Printf("  sqlite3 %s \"SELECT chat_id, datetime(period_start), message_count, LENGTH(summary_text) FROM summaries WHERE summary_type='1h' ORDER BY created_at DESC LIMIT 5\"\n", database.GetPath())
	fmt.Println()
	
	fmt.Println("✅ Test complete!")
}
