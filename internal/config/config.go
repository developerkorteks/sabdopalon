package config

import (
	"os"
)

// Config holds all configuration for the bot
type Config struct {
	TelegramToken    string
	GeminiAPIKey     string
	GeminiModel      string
	DatabasePath     string
	DebugMode        bool
	SummaryInterval  int // in hours
	DailySummaryTime string
}

// Load loads configuration from environment variables with fallbacks
func Load() *Config {
	cfg := &Config{
		// Telegram Bot Token
		TelegramToken: getEnv("TELEGRAM_BOT_TOKEN", "8470513084:AAH_Vz4SiMtO2pY3RNJ9ZZ7sUNGsd4O9cgc"),
		
		// Gemini API Configuration
		GeminiAPIKey: getEnv("GEMINI_API_KEY", "AIzaSyAbyIAJ9Jv8M_LVhBnd6J0FNvioAxGJA3w"),
		GeminiModel:  getEnv("GEMINI_MODEL", "gemini-2.0-flash-exp"),
		
		// Database
		DatabasePath: getEnv("DATABASE_PATH", "telegram_bot.db"),
		
		// Debug Mode
		DebugMode: getEnv("DEBUG_MODE", "true") == "true",
		
		// Summary Configuration
		SummaryInterval:  4,      // Every 4 hours
		DailySummaryTime: "23:59", // Daily summary time
	}
	
	return cfg
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.TelegramToken == "" {
		return ErrMissingTelegramToken
	}
	if c.GeminiAPIKey == "" {
		return ErrMissingGeminiKey
	}
	return nil
}

// Configuration errors
var (
	ErrMissingTelegramToken = &ConfigError{"telegram bot token is required"}
	ErrMissingGeminiKey     = &ConfigError{"gemini API key is required"}
)

// ConfigError represents a configuration error
type ConfigError struct {
	message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.message
}
