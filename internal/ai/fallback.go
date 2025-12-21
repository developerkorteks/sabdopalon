package ai

import (
	"fmt"
	"telegram-summarizer/internal/logger"
)

// FallbackManager manages multiple AI providers with fallback logic
type FallbackManager struct {
	providers []AIProvider
}

// NewFallbackManager creates a new fallback manager with multiple providers
func NewFallbackManager(providers []AIProvider) *FallbackManager {
	return &FallbackManager{
		providers: providers,
	}
}

// GenerateSummary tries to generate summary with fallback logic
func (f *FallbackManager) GenerateSummary(prompt string) (string, error) {
	if len(f.providers) == 0 {
		return "", fmt.Errorf("no AI providers configured")
	}
	
	var lastError error
	
	for i, provider := range f.providers {
		providerName := provider.GetName()
		logger.Info("Trying provider %d/%d: %s", i+1, len(f.providers), providerName)
		
		summary, err := provider.GenerateSummary(prompt)
		if err == nil {
			logger.Info("✅ Success with %s", providerName)
			return summary, nil
		}
		
		logger.Warn("⚠️  %s failed: %v", providerName, err)
		lastError = err
		
		// Continue to next provider
	}
	
	// All providers failed
	return "", fmt.Errorf("all %d providers failed, last error: %w", len(f.providers), lastError)
}

// GetName returns the fallback manager name
func (f *FallbackManager) GetName() string {
	names := ""
	for i, p := range f.providers {
		if i > 0 {
			names += " → "
		}
		names += p.GetName()
	}
	return fmt.Sprintf("Fallback Chain: %s", names)
}

// IsAvailable checks if at least one provider is available
func (f *FallbackManager) IsAvailable() bool {
	for _, provider := range f.providers {
		if provider.IsAvailable() {
			return true
		}
	}
	return false
}

// GetProviderCount returns the number of configured providers
func (f *FallbackManager) GetProviderCount() int {
	return len(f.providers)
}
