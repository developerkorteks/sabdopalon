package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// YPAIClient implements AIProvider for YP AI API
type YPAIClient struct {
	baseURL    string
	httpClient *http.Client
}

// YPAIResponse represents the API response
type YPAIResponse struct {
	Status bool   `json:"status"`
	Result string `json:"result"`
}

// NewYPAIClient creates a new YPAI client
func NewYPAIClient() *YPAIClient {
	return &YPAIClient{
		baseURL: "https://api.yupra.my.id/api/ai/ypai",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateSummary generates a summary using YPAI API
func (y *YPAIClient) GenerateSummary(prompt string) (string, error) {
	// URL encode the prompt
	apiURL := fmt.Sprintf("%s?text=%s", y.baseURL, url.QueryEscape(prompt))
	
	// Make request
	resp, err := y.httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("ypai request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read ypai response: %w", err)
	}
	
	// Parse JSON
	var result YPAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse ypai response: %w", err)
	}
	
	if !result.Status {
		return "", fmt.Errorf("ypai returned status false")
	}
	
	// YPAI includes <think>...</think> tags, remove them
	cleaned := removeThinkTags(result.Result)
	
	return strings.TrimSpace(cleaned), nil
}

// removeThinkTags removes <think>...</think> tags from YPAI response
func removeThinkTags(text string) string {
	// Remove <think>...</think> blocks
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	return re.ReplaceAllString(text, "")
}

// GetName returns the provider name
func (y *YPAIClient) GetName() string {
	return "YP AI"
}

// IsAvailable checks if YPAI API is available
func (y *YPAIClient) IsAvailable() bool {
	resp, err := y.httpClient.Get(y.baseURL + "?text=test")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
