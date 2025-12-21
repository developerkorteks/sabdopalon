package summarizer

import (
	"fmt"
	"strings"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/logger"
)

// ChunkManager handles intelligent message splitting
type ChunkManager struct {
	MaxMessagesPerChunk int // Maximum messages in one chunk
	MaxCharsPerPrompt   int // Maximum characters for prompt (before URL encoding)
}

// NewChunkManager creates a new chunk manager with safe defaults
func NewChunkManager() *ChunkManager {
	return &ChunkManager{
		MaxMessagesPerChunk: 30,   // 30 messages = ~6,000 chars (safe for GET requests)
		MaxCharsPerPrompt:   8000, // 8K chars before encoding = ~10.4K after encoding (more aggressive)
	}
}

// SplitMessages splits messages into manageable chunks
func (cm *ChunkManager) SplitMessages(messages []db.Message) [][]db.Message {
	if len(messages) == 0 {
		return nil
	}
	
	logger.Info("Splitting %d messages into chunks (max %d per chunk)", len(messages), cm.MaxMessagesPerChunk)
	
	var chunks [][]db.Message
	
	for i := 0; i < len(messages); i += cm.MaxMessagesPerChunk {
		end := i + cm.MaxMessagesPerChunk
		if end > len(messages) {
			end = len(messages)
		}
		
		chunk := messages[i:end]
		chunks = append(chunks, chunk)
		
		logger.Debug("Chunk %d: %d messages (indices %d-%d)", len(chunks), len(chunk), i, end-1)
	}
	
	logger.Info("Split into %d chunks", len(chunks))
	return chunks
}

// SplitSummaries splits summaries into groups for recursive merge
func (cm *ChunkManager) SplitSummaries(summaries []string) [][]string {
	if len(summaries) == 0 {
		return nil
	}
	
	// If only 1 summary, no need to split
	if len(summaries) == 1 {
		logger.Debug("Only 1 summary, no split needed")
		return [][]string{summaries}
	}
	
	// Target: merge 2-3 summaries at a time for reliability
	const templateOverhead = 2000 // Merge template is ~2000 chars
	const minGroupSize = 2        // MUST be at least 2 summaries per group
	const maxGroupSize = 3        // Prefer 2-3 summaries per group
	
	// Calculate actual summary sizes
	totalSize := 0
	for _, s := range summaries {
		totalSize += len(s)
	}
	
	logger.Info("Splitting %d summaries (total: %d chars) into groups", len(summaries), totalSize)
	
	var groups [][]string
	var currentGroup []string
	var currentGroupSize int
	
	maxCharsPerGroup := cm.MaxCharsPerPrompt - templateOverhead
	
	for i, summary := range summaries {
		summarySize := len(summary)
		
		// Check if we should start a new group
		shouldStartNewGroup := false
		
		if len(currentGroup) > 0 {
			// Start new group if:
			// 1. Would exceed char limit
			// 2. Already have maxGroupSize summaries
			if currentGroupSize+summarySize > maxCharsPerGroup || len(currentGroup) >= maxGroupSize {
				shouldStartNewGroup = true
			}
		}
		
		if shouldStartNewGroup {
			// Save current group
			groups = append(groups, currentGroup)
			logger.Debug("Group %d: %d summaries, %d chars", len(groups), len(currentGroup), currentGroupSize)
			
			// Start new group
			currentGroup = []string{summary}
			currentGroupSize = summarySize
		} else {
			// Add to current group
			currentGroup = append(currentGroup, summary)
			currentGroupSize += summarySize
		}
		
		// Last iteration - save remaining group
		if i == len(summaries)-1 && len(currentGroup) > 0 {
			groups = append(groups, currentGroup)
			logger.Debug("Group %d: %d summaries, %d chars", len(groups), len(currentGroup), currentGroupSize)
		}
	}
	
	// CRITICAL: If we ended up with groups of 1 summary each, we need to merge them differently
	// This happens when each summary is too large
	allSingleSummaries := true
	for _, group := range groups {
		if len(group) > 1 {
			allSingleSummaries = false
			break
		}
	}
	
	if allSingleSummaries && len(groups) > 1 {
		// Force pairing: combine every 2 single summaries into 1 group
		logger.Warn("All groups have only 1 summary - forcing pairs")
		var pairedGroups [][]string
		for i := 0; i < len(groups); i += 2 {
			if i+1 < len(groups) {
				// Pair two summaries
				paired := []string{groups[i][0], groups[i+1][0]}
				pairedGroups = append(pairedGroups, paired)
				logger.Debug("Paired group %d: 2 summaries", len(pairedGroups))
			} else {
				// Odd one out - add to previous group if exists
				if len(pairedGroups) > 0 {
					pairedGroups[len(pairedGroups)-1] = append(pairedGroups[len(pairedGroups)-1], groups[i][0])
					logger.Debug("Added remaining summary to last group")
				} else {
					// Only 1 summary total - return as is
					pairedGroups = append(pairedGroups, groups[i])
				}
			}
		}
		groups = pairedGroups
	}
	
	logger.Info("Split into %d groups (min %d summaries per group)", len(groups), minGroupSize)
	return groups
}

// FormatMessagesForPrompt formats messages into text for prompts
func (cm *ChunkManager) FormatMessagesForPrompt(messages []db.Message) string {
	var builder strings.Builder
	
	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("15:04")
		builder.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.Username, msg.MessageText))
	}
	
	return builder.String()
}

// EstimatePromptSize estimates the size of a prompt with given messages
func (cm *ChunkManager) EstimatePromptSize(messages []db.Message) int {
	const templateSize = 3500 // Prompt template is ~3500 chars
	
	messagesText := cm.FormatMessagesForPrompt(messages)
	totalSize := templateSize + len(messagesText)
	
	logger.Debug("Estimated prompt size: %d chars (template: %d, messages: %d)", 
		totalSize, templateSize, len(messagesText))
	
	return totalSize
}

// EstimateMergeSize estimates the size of merged summaries
func (cm *ChunkManager) EstimateMergeSize(summaries []string) int {
	const templateSize = 2000 // Merge template is smaller ~2000 chars
	
	totalSummariesSize := 0
	for _, s := range summaries {
		totalSummariesSize += len(s)
	}
	
	totalSize := templateSize + totalSummariesSize
	
	logger.Debug("Estimated merge size: %d chars (template: %d, summaries: %d)", 
		totalSize, templateSize, totalSummariesSize)
	
	return totalSize
}

// ShouldSplitMessages checks if messages need to be split
func (cm *ChunkManager) ShouldSplitMessages(messages []db.Message) bool {
	// Split if either:
	// 1. Too many messages
	// 2. Estimated prompt size too large
	
	if len(messages) > cm.MaxMessagesPerChunk {
		logger.Debug("Should split: message count %d > threshold %d", len(messages), cm.MaxMessagesPerChunk)
		return true
	}
	
	estimatedSize := cm.EstimatePromptSize(messages)
	if estimatedSize > cm.MaxCharsPerPrompt {
		logger.Debug("Should split: estimated size %d > threshold %d", estimatedSize, cm.MaxCharsPerPrompt)
		return true
	}
	
	logger.Debug("No split needed: %d messages, %d chars", len(messages), estimatedSize)
	return false
}

// ShouldSplitSummaries checks if summaries need recursive split
func (cm *ChunkManager) ShouldSplitSummaries(summaries []string) bool {
	estimatedSize := cm.EstimateMergeSize(summaries)
	
	if estimatedSize > cm.MaxCharsPerPrompt {
		logger.Debug("Should split summaries: estimated size %d > threshold %d", estimatedSize, cm.MaxCharsPerPrompt)
		return true
	}
	
	logger.Debug("Summaries can be merged directly: %d chars", estimatedSize)
	return false
}
