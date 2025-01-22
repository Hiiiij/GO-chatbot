package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// Config represents the application configuration loaded from environment variables.
type Config struct {
	OpenAIAPIKey string
	MongoURI     string
	APIKey       string // Internal API key for authentication
	Port         string
}

// LoadConfig loads and validates environment variables into the Config struct.
func LoadConfig() *Config {
	// Load .env file (if it exists)
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found. Using environment variables directly.")
	}

	// Create a Config instance with loaded values
	config := &Config{
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
		MongoURI:     getEnv("MONGO_URI", ""),
		APIKey:       getEnv("API_KEY", ""),
		Port:         getEnv("PORT", "8080"), // Default port to 8080 if not provided
	}

	// Validate required configuration
	if config.OpenAIAPIKey == "" {
		log.Fatal().Msg("OPENAI_API_KEY is required but not set")
	}
	if config.MongoURI == "" {
		log.Fatal().Msg("MONGO_URI is required but not set")
	}
	if config.APIKey == "" {
		log.Fatal().Msg("API_KEY is required but not set")
	}

	return config
}

// getEnv retrieves the value of an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
