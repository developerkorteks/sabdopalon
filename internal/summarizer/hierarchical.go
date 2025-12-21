package summarizer

import (
	"fmt"
	"strings"
	"telegram-summarizer/internal/ai"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/logger"
	"time"
)

// HierarchicalSummarizer handles recursive summarization with chunking
type HierarchicalSummarizer struct {
	fallbackManager   *ai.FallbackManager
	promptManager     *PromptManager
	chunkManager      *ChunkManager
	formatter         *SummaryFormatter
	maxRecursionDepth int
	progressCallback  func(string) // Callback to send progress updates
	summaryCallback   func(string) // Callback to send partial summaries
}

// NewHierarchicalSummarizer creates a new hierarchical summarizer
func NewHierarchicalSummarizer(fallbackManager *ai.FallbackManager, progressCallback func(string), summaryCallback func(string)) *HierarchicalSummarizer {
	return &HierarchicalSummarizer{
		fallbackManager:   fallbackManager,
		promptManager:     NewPromptManager(),
		chunkManager:      NewChunkManager(),
		formatter:         NewSummaryFormatter(),
		maxRecursionDepth: 3, // Max 3 levels of recursion
		progressCallback:  progressCallback,
		summaryCallback:   summaryCallback,
	}
}

// SummarizeMessages generates a summary from messages with automatic chunking
func (hs *HierarchicalSummarizer) SummarizeMessages(messages []db.Message, groupName string, startTime, endTime time.Time) (string, error) {
	logger.Info("Starting hierarchical summarization for %d messages from %s", len(messages), groupName)
	
	// Check if we need to split messages
	if !hs.chunkManager.ShouldSplitMessages(messages) {
		// Small enough - direct summarization
		logger.Info("Messages small enough for direct summarization")
		hs.sendProgress("Generating summary...")
		return hs.summarizeChunkDirect(messages, groupName, startTime, endTime)
	}
	
	// Need streaming approach for large chats
	logger.Info("Messages too large - using streaming summarization")
	hs.sendProgress("Chat is large - using streaming multi-part summarization...")
	
	return hs.summarizeMessagesStreaming(messages, groupName, startTime, endTime)
}

// summarizeMessagesStreaming processes messages in batches and sends partial summaries
func (hs *HierarchicalSummarizer) summarizeMessagesStreaming(messages []db.Message, groupName string, startTime, endTime time.Time) (string, error) {
	logger.Info("Starting streaming summarization for %d messages", len(messages))
	
	// Split into chunks (30 messages per chunk)
	chunks := hs.chunkManager.SplitMessages(messages)
	totalChunks := len(chunks)
	
	logger.Info("Split into %d chunks", totalChunks)
	
	// Batch processing: process 3 chunks at a time, then merge and send
	const chunksPerBatch = 3
	totalBatches := (totalChunks + chunksPerBatch - 1) / chunksPerBatch
	
	logger.Info("Will process in %d batches (%d chunks per batch)", totalBatches, chunksPerBatch)
	
	for batchIdx := 0; batchIdx < totalBatches; batchIdx++ {
		// Calculate batch boundaries
		batchStart := batchIdx * chunksPerBatch
		batchEnd := batchStart + chunksPerBatch
		if batchEnd > totalChunks {
			batchEnd = totalChunks
		}
		
		batchChunks := chunks[batchStart:batchEnd]
		
		logger.Info("Processing batch %d/%d (%d chunks)", batchIdx+1, totalBatches, len(batchChunks))
		hs.sendProgress(fmt.Sprintf("📦 Processing batch %d/%d (%d chunks, ~%d messages)...", 
			batchIdx+1, totalBatches, len(batchChunks), len(batchChunks)*30))
		
		// Process each chunk in this batch
		var chunkSummaries []string
		for i, chunk := range batchChunks {
			globalChunkIdx := batchStart + i + 1
			hs.sendProgress(fmt.Sprintf("📝 Processing chunk %d/%d (batch %d/%d)...", 
				globalChunkIdx, totalChunks, batchIdx+1, totalBatches))
			
			// Get time range for this chunk
			chunkStart := chunk[0].Timestamp
			chunkEnd := chunk[len(chunk)-1].Timestamp
			
			// Summarize chunk with fallback chain
			summary, err := hs.summarizeChunkDirect(chunk, groupName, chunkStart, chunkEnd)
			if err != nil {
				logger.Error("Failed to summarize chunk %d: %v", globalChunkIdx, err)
				return "", fmt.Errorf("failed to summarize chunk %d/%d: %w", globalChunkIdx, totalChunks, err)
			}
			
			chunkSummaries = append(chunkSummaries, summary)
			logger.Info("✅ Chunk %d/%d completed (%d chars)", globalChunkIdx, totalChunks, len(summary))
		}
		
		// Merge summaries for this batch
		hs.sendProgress(fmt.Sprintf("🔄 Merging batch %d/%d (%d summaries)...", batchIdx+1, totalBatches, len(chunkSummaries)))
		
		// Calculate time range for this batch
		batchStartTime := batchChunks[0][0].Timestamp
		batchEndTime := batchChunks[len(batchChunks)-1][len(batchChunks[len(batchChunks)-1])-1].Timestamp
		
		// Merge summaries for this batch
		partialSummary, err := hs.mergeSummariesDirect(chunkSummaries, groupName, batchStartTime, batchEndTime)
		if err != nil {
			logger.Error("Failed to merge batch %d: %v", batchIdx+1, err)
			return "", fmt.Errorf("failed to merge batch %d/%d: %w", batchIdx+1, totalBatches, err)
		}
		
		logger.Info("✅ Batch %d/%d merged (%d chars)", batchIdx+1, totalBatches, len(partialSummary))
		
		// Send partial summary to user
		hs.sendPartialSummary(partialSummary, batchIdx+1, totalBatches, groupName, batchStartTime, batchEndTime, len(batchChunks)*30)
	}
	
	// Return completion message with elegant formatting
	completionMsg := hs.formatter.FormatCompletionMessage(totalBatches)
	return completionMsg, nil
}

// sendPartialSummary sends a partial summary to the user
func (hs *HierarchicalSummarizer) sendPartialSummary(summary string, part, total int, groupName string, startTime, endTime time.Time, messageCount int) {
	if hs.summaryCallback == nil {
		logger.Warn("Summary callback not set, skipping partial summary send")
		return
	}
	
	// Use elegant formatter
	formattedSummary := hs.formatter.FormatPartialSummary(summary, part, total, groupName, startTime, endTime, messageCount)
	
	// Send via callback
	hs.summaryCallback(formattedSummary)
	
	logger.Info("📤 Sent partial summary %d/%d to user", part, total)
}

// summarizeMessagesRecursive handles recursive message summarization
func (hs *HierarchicalSummarizer) summarizeMessagesRecursive(messages []db.Message, groupName string, startTime, endTime time.Time, depth int) (string, error) {
	// Check recursion depth
	if depth > hs.maxRecursionDepth {
		return "", fmt.Errorf("maximum recursion depth (%d) reached", hs.maxRecursionDepth)
	}
	
	logger.Info("Recursive summarization at depth %d with %d messages", depth, len(messages))
	
	// Split messages into chunks
	chunks := hs.chunkManager.SplitMessages(messages)
	logger.Info("Split into %d chunks", len(chunks))
	
	// Summarize each chunk
	var chunkSummaries []string
	for i, chunk := range chunks {
		hs.sendProgress(fmt.Sprintf("📝 Processing chunk %d/%d (%d messages)...", i+1, len(chunks), len(chunk)))
		
		logger.Info("Processing chunk %d/%d (%d messages)", i+1, len(chunks), len(chunk))
		
		// Get time range for this chunk
		chunkStart := chunk[0].Timestamp
		chunkEnd := chunk[len(chunk)-1].Timestamp
		
		// Summarize chunk with fallback chain
		summary, err := hs.summarizeChunkDirect(chunk, groupName, chunkStart, chunkEnd)
		if err != nil {
			logger.Error("Failed to summarize chunk %d: %v", i+1, err)
			return "", fmt.Errorf("failed to summarize chunk %d/%d: %w", i+1, len(chunks), err)
		}
		
		chunkSummaries = append(chunkSummaries, summary)
		logger.Info("✅ Chunk %d/%d completed", i+1, len(chunks))
	}
	
	// Now merge the summaries
	hs.sendProgress(fmt.Sprintf("🔄 Merging %d summaries...", len(chunkSummaries)))
	
	return hs.mergeSummariesRecursive(chunkSummaries, groupName, startTime, endTime, depth+1)
}

// mergeSummariesRecursive handles recursive summary merging
func (hs *HierarchicalSummarizer) mergeSummariesRecursive(summaries []string, groupName string, startTime, endTime time.Time, depth int) (string, error) {
	// Check recursion depth
	if depth > hs.maxRecursionDepth {
		return "", fmt.Errorf("maximum recursion depth (%d) reached during merge", hs.maxRecursionDepth)
	}
	
	logger.Info("Merging %d summaries at depth %d", len(summaries), depth)
	
	// Check if summaries can be merged directly
	if !hs.chunkManager.ShouldSplitSummaries(summaries) {
		// Small enough - direct merge
		logger.Info("Summaries small enough for direct merge")
		return hs.mergeSummariesDirect(summaries, groupName, startTime, endTime)
	}
	
	// Too large - need recursive merge
	logger.Info("Summaries too large - splitting for recursive merge")
	hs.sendProgress(fmt.Sprintf("⚙️  Summaries too large - doing multi-level merge..."))
	
	// Split summaries into groups
	groups := hs.chunkManager.SplitSummaries(summaries)
	logger.Info("Split summaries into %d groups", len(groups))
	
	// Merge each group
	var metaSummaries []string
	for i, group := range groups {
		hs.sendProgress(fmt.Sprintf("🔄 Merging group %d/%d (%d summaries)...", i+1, len(groups), len(group)))
		
		logger.Info("Merging group %d/%d (%d summaries)", i+1, len(groups), len(group))
		
		metaSummary, err := hs.mergeSummariesDirect(group, groupName, startTime, endTime)
		if err != nil {
			logger.Error("Failed to merge group %d: %v", i+1, err)
			return "", fmt.Errorf("failed to merge group %d/%d: %w", i+1, len(groups), err)
		}
		
		metaSummaries = append(metaSummaries, metaSummary)
		logger.Info("✅ Group %d/%d merged", i+1, len(groups))
	}
	
	// Recursively merge the meta-summaries
	return hs.mergeSummariesRecursive(metaSummaries, groupName, startTime, endTime, depth+1)
}

// summarizeChunkDirect summarizes a single chunk with fallback
func (hs *HierarchicalSummarizer) summarizeChunkDirect(messages []db.Message, groupName string, startTime, endTime time.Time) (string, error) {
	logger.Info("Direct summarization of %d messages", len(messages))
	
	// Format messages
	messagesText := hs.chunkManager.FormatMessagesForPrompt(messages)
	
	// Build prompt
	prompt := hs.promptManager.GetManual24HPrompt(messagesText, groupName, startTime, endTime)
	
	logger.Debug("Prompt size: %d chars", len(prompt))
	
	// Generate summary with fallback chain (tries all 18 providers)
	summary, err := hs.fallbackManager.GenerateSummary(prompt)
	if err != nil {
		return "", fmt.Errorf("fallback chain failed: %w", err)
	}
	
	logger.Info("✅ Summary generated: %d chars", len(summary))
	return summary, nil
}

// mergeSummariesDirect merges summaries directly with fallback
func (hs *HierarchicalSummarizer) mergeSummariesDirect(summaries []string, groupName string, startTime, endTime time.Time) (string, error) {
	logger.Info("Direct merge of %d summaries", len(summaries))
	
	// Combine summaries into one text
	var combined strings.Builder
	for i, summary := range summaries {
		combined.WriteString(fmt.Sprintf("\n## Bagian %d/%d:\n%s\n", i+1, len(summaries), summary))
	}
	
	// Build merge prompt
	prompt := hs.buildMergePrompt(combined.String(), groupName, startTime, endTime)
	
	logger.Debug("Merge prompt size: %d chars", len(prompt))
	
	// Generate merged summary with fallback chain
	finalSummary, err := hs.fallbackManager.GenerateSummary(prompt)
	if err != nil {
		return "", fmt.Errorf("merge fallback chain failed: %w", err)
	}
	
	logger.Info("✅ Merged summary generated: %d chars", len(finalSummary))
	return finalSummary, nil
}

// buildMergePrompt builds a compact prompt for merging multiple summaries
func (hs *HierarchicalSummarizer) buildMergePrompt(summariesText, groupName string, startTime, endTime time.Time) string {
	// More compact prompt to save space
	prompt := fmt.Sprintf(`Gabungkan ringkasan partial berikut menjadi SATU ringkasan lengkap dan koheren dalam BAHASA INDONESIA.

Grup: "%s" | Periode: %s - %s

Ringkasan Partial:
%s

Instruksi:
1. Gabungkan semua informasi menjadi satu ringkasan utuh
2. Hilangkan duplikasi - gabungkan topik yang sama
3. Pertahankan detail penting: produk, harga, FC, testimoni, kredibilitas
4. Format: Struktur ringkasan 24 jam standar dengan section lengkap
5. Koheren, rinci, dan mudah dibaca

Output ringkasan final:`, groupName, startTime.Format("2006-01-02 15:04"), endTime.Format("15:04"), summariesText)
	
	return prompt
}

// sendProgress sends progress update via callback
func (hs *HierarchicalSummarizer) sendProgress(message string) {
	if hs.progressCallback != nil {
		hs.progressCallback(message)
	}
	logger.Info("Progress: %s", message)
}

// escapeMarkdownSimple escapes basic markdown characters for Telegram
func escapeMarkdownSimple(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"`", "\\`",
	)
	return replacer.Replace(text)
}
