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

// DelineClient implements AIProvider for Deline API
type DelineClient struct {
	baseURL    string
	endpoint   string
	modelName  string
	httpClient *http.Client
}

// DelineResponse represents the standard API response
type DelineResponse struct {
	Status  bool   `json:"status"`
	Creator string `json:"creator"`
	Result  string `json:"result"`
}

// DelineCopilotThinkResponse represents Copilot Think's special response format
type DelineCopilotThinkResponse struct {
	Status  bool   `json:"status"`
	Creator string `json:"creator"`
	Result  struct {
		Text      string `json:"text"`
		Citations []struct {
			Title string `json:"title"`
			Icon  string `json:"icon"`
			URL   string `json:"url"`
		} `json:"citations"`
	} `json:"result"`
}

// NewDelineClient creates a new Deline API client
func NewDelineClient(endpoint, modelName string) *DelineClient {
	return &DelineClient{
		baseURL:   "https://api.deline.web.id",
		endpoint:  endpoint,
		modelName: modelName,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateSummary generates a summary using Deline API
func (d *DelineClient) GenerateSummary(prompt string) (string, error) {
	var apiURL string
	
	// OpenAI endpoint requires additional 'prompt' parameter
	if d.endpoint == "/ai/openai" {
		systemPrompt := "You are a helpful AI assistant that creates concise summaries of conversations."
		apiURL = fmt.Sprintf("%s%s?text=%s&prompt=%s", 
			d.baseURL, d.endpoint, 
			url.QueryEscape(prompt),
			url.QueryEscape(systemPrompt))
	} else {
		// Standard endpoints only need 'text' parameter
		apiURL = fmt.Sprintf("%s%s?text=%s", d.baseURL, d.endpoint, url.QueryEscape(prompt))
	}
	
	// Make request
	resp, err := d.httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("deline request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read deline response: %w", err)
	}
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: status %d", resp.StatusCode)
	}
	
	// Handle Copilot Think's special format
	if d.endpoint == "/ai/copilot-think" {
		return d.parseCopilotThinkResponse(body)
	}
	
	// Parse standard JSON
	var result DelineResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse deline response: %w", err)
	}
	
	if !result.Status {
		return "", fmt.Errorf("deline returned status false")
	}
	
	return strings.TrimSpace(result.Result), nil
}

// parseCopilotThinkResponse parses Copilot Think's special response format
func (d *DelineClient) parseCopilotThinkResponse(body []byte) (string, error) {
	var result DelineCopilotThinkResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse copilot-think response: %w", err)
	}
	
	if !result.Status {
		return "", fmt.Errorf("copilot-think returned status false")
	}
	
	return strings.TrimSpace(result.Result.Text), nil
}

// GetName returns the provider name
func (d *DelineClient) GetName() string {
	return d.modelName
}

// IsAvailable checks if Deline API is available
func (d *DelineClient) IsAvailable() bool {
	resp, err := d.httpClient.Get(d.baseURL + d.endpoint + "?text=test")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// Factory functions for each model

// NewDelineCopilotClient creates a Copilot client via Deline
func NewDelineCopilotClient() *DelineClient {
	return NewDelineClient("/ai/copilot", "Copilot (Deline)")
}

// NewDelineCopilotThinkClient creates a Copilot Think client via Deline
func NewDelineCopilotThinkClient() *DelineClient {
	return NewDelineClient("/ai/copilot-think", "Copilot Think (Deline)")
}

// NewDelineOpenAIClient creates an OpenAI client via Deline
func NewDelineOpenAIClient() *DelineClient {
	return NewDelineClient("/ai/openai", "OpenAI (Deline)")
}
