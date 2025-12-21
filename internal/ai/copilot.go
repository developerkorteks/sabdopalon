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

// CopilotClient implements AIProvider for Microsoft Copilot API
type CopilotClient struct {
	baseURL    string
	thinkDeep  bool
	httpClient *http.Client
}

// CopilotResponse represents the API response
type CopilotResponse struct {
	Status  bool     `json:"status"`
	Model   string   `json:"model"`
	Result  string   `json:"result"`
	Citations []interface{} `json:"citations"`
}

// NewCopilotClient creates a new Copilot client
func NewCopilotClient(thinkDeep bool) *CopilotClient {
	return &CopilotClient{
		baseURL:   "https://api.yupra.my.id/api/ai",
		thinkDeep: thinkDeep,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateSummary generates a summary using Copilot API
func (c *CopilotClient) GenerateSummary(prompt string) (string, error) {
	endpoint := "/copilot"
	if c.thinkDeep {
		endpoint = "/copilot-think"
	}
	
	// URL encode the prompt
	apiURL := fmt.Sprintf("%s%s?text=%s", c.baseURL, endpoint, url.QueryEscape(prompt))
	
	// Make request
	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("copilot request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read copilot response: %w", err)
	}
	
	// Parse JSON
	var result CopilotResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse copilot response: %w", err)
	}
	
	if !result.Status {
		return "", fmt.Errorf("copilot returned status false")
	}
	
	return strings.TrimSpace(result.Result), nil
}

// GetName returns the provider name
func (c *CopilotClient) GetName() string {
	if c.thinkDeep {
		return "Copilot Think Deeper"
	}
	return "Copilot Default"
}

// IsAvailable checks if Copilot API is available
func (c *CopilotClient) IsAvailable() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/copilot?text=test")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
