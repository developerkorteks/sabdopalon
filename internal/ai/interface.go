package ai

// AIProvider is an interface for AI clients that can generate summaries
type AIProvider interface {
	// GenerateSummary generates a summary from the given prompt
	GenerateSummary(prompt string) (string, error)
	
	// GetName returns the name of the provider
	GetName() string
	
	// IsAvailable checks if the provider is available (optional health check)
	IsAvailable() bool
}
