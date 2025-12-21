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

// ElrayyXmlClient implements AIProvider for ElrayyXml API
type ElrayyXmlClient struct {
	baseURL    string
	endpoint   string
	modelName  string
	httpClient *http.Client
}

// ElrayyXmlResponse represents the standard API response
type ElrayyXmlResponse struct {
	Status bool   `json:"status"`
	Author string `json:"author"`
	Result string `json:"result"`
}

// ElrayyXmlAlisiaResponse represents Alisia's special response format
type ElrayyXmlAlisiaResponse struct {
	Status bool   `json:"status"`
	Author string `json:"author"`
	Result struct {
		Status int `json:"status"`
		Data   struct {
			RefinedResults string        `json:"refined_results"`
			Results        []interface{} `json:"results"`
		} `json:"data"`
		Resources interface{} `json:"resources"`
		Message   string      `json:"message"`
	} `json:"result"`
}

// NewElrayyXmlClient creates a new ElrayyXml API client
func NewElrayyXmlClient(endpoint, modelName string) *ElrayyXmlClient {
	return &ElrayyXmlClient{
		baseURL:   "https://api.elrayyxml.web.id",
		endpoint:  endpoint,
		modelName: modelName,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateSummary generates a summary using ElrayyXml API
func (e *ElrayyXmlClient) GenerateSummary(prompt string) (string, error) {
	// URL encode the prompt
	apiURL := fmt.Sprintf("%s%s?text=%s", e.baseURL, e.endpoint, url.QueryEscape(prompt))
	
	// Make request
	resp, err := e.httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("elrayyxml request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read elrayyxml response: %w", err)
	}
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: status %d", resp.StatusCode)
	}
	
	// Handle Alisia's special format
	if e.endpoint == "/api/ai/alisia" {
		return e.parseAlisiaResponse(body)
	}
	
	// Parse standard JSON
	var result ElrayyXmlResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse elrayyxml response: %w", err)
	}
	
	if !result.Status {
		return "", fmt.Errorf("elrayyxml returned status false")
	}
	
	return strings.TrimSpace(result.Result), nil
}

// parseAlisiaResponse parses Alisia's special response format
func (e *ElrayyXmlClient) parseAlisiaResponse(body []byte) (string, error) {
	var result ElrayyXmlAlisiaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse alisia response: %w", err)
	}
	
	if !result.Status {
		return "", fmt.Errorf("alisia returned status false")
	}
	
	if result.Result.Status != 200 {
		return "", fmt.Errorf("alisia returned status %d", result.Result.Status)
	}
	
	return strings.TrimSpace(result.Result.Data.RefinedResults), nil
}

// GetName returns the provider name
func (e *ElrayyXmlClient) GetName() string {
	return e.modelName
}

// IsAvailable checks if ElrayyXml API is available
func (e *ElrayyXmlClient) IsAvailable() bool {
	resp, err := e.httpClient.Get(e.baseURL + e.endpoint + "?text=test")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// Factory functions for each model

// NewVeniceAIClient creates a Venice AI client
func NewVeniceAIClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/veniceai", "Venice AI (ElrayyXml)")
}

// NewPowerBrainAIClient creates a PowerBrain AI client
func NewPowerBrainAIClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/powerbrainai", "PowerBrain AI (ElrayyXml)")
}

// NewPerplexityAIClient creates a Perplexity AI client
func NewPerplexityAIClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/perplexityai", "Perplexity AI (ElrayyXml)")
}

// NewLuminAIClient creates a Lumin AI client
func NewLuminAIClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/luminai", "Lumin AI (ElrayyXml)")
}

// NewElrayyGeminiClient creates a Gemini client via ElrayyXml
func NewElrayyGeminiClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/gemini", "Gemini (ElrayyXml)")
}

// NewFeloAIClient creates a Felo AI client
func NewFeloAIClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/feloai", "Felo AI (ElrayyXml)")
}

// NewElrayyCopilotClient creates a Copilot client via ElrayyXml
func NewElrayyCopilotClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/copilot", "Copilot (ElrayyXml)")
}

// NewElrayyChatGPTClient creates a ChatGPT client via ElrayyXml
func NewElrayyChatGPTClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/chatgpt", "ChatGPT (ElrayyXml)")
}

// NewBibleGPTClient creates a BibleGPT client
func NewBibleGPTClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/biblegpt", "BibleGPT (ElrayyXml)")
}

// NewAlisiaClient creates an Alisia AI client
func NewAlisiaClient() *ElrayyXmlClient {
	return NewElrayyXmlClient("/api/ai/alisia", "Alisia AI (ElrayyXml)")
}
