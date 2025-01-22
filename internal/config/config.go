package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	OpenAIAPIKey string
	MongoURI     string
	APIKey       string
	Port         string
}

// load configuration from environment variables
func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("no .env file found, using environment variables")
	}
	/// populate Config with .env values
	config := &Config{
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
		MongoURI:     getEnv("MONGO_URI", ""),
		APIKey:       getEnv("API_KEY", ""),
		Port:         getEnv("PORT", "8080"),
	}

	if config.OpenAIAPIKey == "" {
		log.Fatal().Msg("environment variable OPENAI_API_KEY is missing")
	}
	if config.MongoURI == "" {
		log.Fatal().Msg("environment variable MONGO_URI is missing")
	}
	if config.APIKey == "" {
		log.Fatal().Msg("environment variable API_KEY is missing")
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
