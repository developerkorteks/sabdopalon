package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GPT5Client implements AIProvider for GPT-5 Smart API
type GPT5Client struct {
	baseURL    string
	httpClient *http.Client
}

// GPT5Response represents the API response
type GPT5Response struct {
	Status  bool     `json:"status"`
	Model   string   `json:"model"`
	Result  string   `json:"result"`
	Citations []interface{} `json:"citations"`
}

// NewGPT5Client creates a new GPT-5 client
func NewGPT5Client() *GPT5Client {
	return &GPT5Client{
		baseURL: "https://api.yupra.my.id/api/ai/gpt5",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateSummary generates a summary using GPT-5 API
func (g *GPT5Client) GenerateSummary(prompt string) (string, error) {
	// URL encode the prompt
	apiURL := fmt.Sprintf("%s?text=%s", g.baseURL, url.QueryEscape(prompt))
	
	// Make request
	resp, err := g.httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("gpt5 request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read gpt5 response: %w", err)
	}
	
	// Parse JSON
	var result GPT5Response
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse gpt5 response: %w", err)
	}
	
	if !result.Status {
		return "", fmt.Errorf("gpt5 returned status false")
	}
	
	return strings.TrimSpace(result.Result), nil
}

// GetName returns the provider name
func (g *GPT5Client) GetName() string {
	return "GPT-5 Smart"
}

// IsAvailable checks if GPT-5 API is available
func (g *GPT5Client) IsAvailable() bool {
	resp, err := g.httpClient.Get(g.baseURL + "?text=test")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
