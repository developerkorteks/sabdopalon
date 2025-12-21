package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"telegram-summarizer/internal/logger"
	"time"
)

const (
	baseURL     = "https://generativelanguage.googleapis.com/v1beta/models"
	maxRetries  = 3
	retryDelay  = 2 * time.Second
)

// Client represents a Gemini API client
type Client struct {
	apiKey      string
	model       string
	httpClient  *http.Client
}

// Request structures for Gemini API
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// Response structures from Gemini API
type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata usageMetadata     `json:"usageMetadata"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type usageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// NewClient creates a new Gemini API client
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateSummary generates a summary using Gemini API
func (c *Client) GenerateSummary(prompt string) (string, error) {
	var lastErr error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Debug("Gemini API call attempt %d/%d", attempt, maxRetries)
		
		summary, err := c.callAPI(prompt)
		if err == nil {
			return summary, nil
		}
		
		lastErr = err
		logger.Warn("Attempt %d failed: %v", attempt, err)
		
		if attempt < maxRetries {
			logger.Debug("Retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
		}
	}
	
	return "", fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// callAPI makes the actual API call to Gemini
func (c *Client) callAPI(prompt string) (string, error) {
	logger.Info("Calling Gemini API...")
	logger.Debug("Prompt length: %d characters", len(prompt))
	
	// Build request
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Build URL
	url := fmt.Sprintf("%s/%s:generateContent", baseURL, c.model)
	
	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", c.apiKey)
	
	// Send request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	duration := time.Since(startTime)
	logger.Debug("API response received in %v", duration)
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		logger.Error("API error (status %d): %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("API error: status %d", resp.StatusCode)
	}
	
	// Parse response
	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Extract summary
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}
	
	summary := geminiResp.Candidates[0].Content.Parts[0].Text
	
	logger.Info("✅ Summary generated successfully")
	logger.Debug("Summary length: %d characters", len(summary))
	logger.Debug("Tokens used: Prompt=%d, Response=%d, Total=%d",
		geminiResp.UsageMetadata.PromptTokenCount,
		geminiResp.UsageMetadata.CandidatesTokenCount,
		geminiResp.UsageMetadata.TotalTokenCount,
	)
	
	return summary, nil
}

// GenerateChatSummary generates a summary specifically for chat messages
func (c *Client) GenerateChatSummary(messages string, summaryType string) (string, error) {
	var prompt string
	
	if summaryType == "incremental" {
		prompt = c.buildIncrementalPrompt(messages)
	} else {
		prompt = c.buildDailyPrompt(messages)
	}
	
	return c.GenerateSummary(prompt)
}

// buildIncrementalPrompt builds a prompt for incremental (4-hour) summaries
func (c *Client) buildIncrementalPrompt(messages string) string {
	prompt := fmt.Sprintf(`Summarize the following group chat messages. Focus on key topics and important discussions.

Chat messages:
%s

Provide a concise summary highlighting:
- Main topics discussed
- Important points or decisions
- Notable participants

Keep it brief and informative (2-3 paragraphs maximum).`, messages)
	
	return prompt
}

// buildDailyPrompt builds a prompt for daily summaries
func (c *Client) buildDailyPrompt(summaries string) string {
	prompt := fmt.Sprintf(`Create a comprehensive daily summary from these incremental summaries of a group chat.

Incremental summaries:
%s

Format the summary as follows:

📌 TOPIK UTAMA:
- [List main topics discussed]

💬 DISKUSI PENTING:
- [Key discussions with context]

📋 ACTION ITEMS:
- [Any tasks or decisions mentioned]

⚡ HIGHLIGHT:
- [Notable or interesting moments]

Be comprehensive but concise. Focus on valuable information.`, summaries)
	
	return prompt
}

// GetName returns the provider name (implements ai.AIProvider)
func (c *Client) GetName() string {
	return fmt.Sprintf("Gemini %s", c.model)
}

// IsAvailable checks if Gemini API is available (implements ai.AIProvider)
func (c *Client) IsAvailable() bool {
	// Simple check - try a minimal request
	return c.apiKey != ""
}
