package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const ChatGPTEndpoint = "https://api.openai.com/v1/chat/completions"

var (
	OpenAIAPIKey string
	APIKey       string
)

func init() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal().Msg("Error loading .env file")
	}

	// Get API keys from environment variables
	OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	APIKey = os.Getenv("API_KEY")

	// Check if required environment variables are set
	if OpenAIAPIKey == "" {
		log.Fatal().Msg("Missing OPENAI_API_KEY environment variable")
	}
	if APIKey == "" {
		log.Fatal().Msg("Missing API_KEY environment variable")
	}

	log.Info().Msg("Environment variables loaded successfully")
}

func main() {
	// Set up ZeroLog for structured logging
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Initialize Gin router
	router := gin.Default()
	router.SetTrustedProxies(nil)

	// Middleware for API key authentication
	router.Use(apiKeyMiddleware)

	// Define endpoints
	router.POST("/chat", handleChat)
	router.GET("/status", handleStatus)

	// Start the server
	log.Info().Msg("Server running on port 8080")
	router.Run(":8080")
}

func apiKeyMiddleware(c *gin.Context) {
	clientKey := c.GetHeader("X-API-KEY")
	log.Info().Str("received_key", clientKey).Msg("API key received")

	if clientKey != APIKey {
		log.Warn().Msg("Unauthorized access attempt")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	c.Next()
}

func handleChat(c *gin.Context) {
	var chatRequest struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		log.Error().Err(err).Msg("Invalid chat request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Call ChatGPT API
	response, err := callChatGPT(chatRequest.Message)
	if err != nil {
		log.Error().Err(err).Msg("Failed to call ChatGPT API")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ChatGPT API call failed"})
		return
	}

	// Respond with the ChatGPT response
	c.JSON(http.StatusOK, gin.H{"response": response})
}

func handleStatus(c *gin.Context) {
	log.Info().Msg("Status check received")
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

func callChatGPT(userMessage string) (string, error) {
	requestBody := map[string]interface{}{
		"model": "gpt-4-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": userMessage},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal ChatGPT request body")
		return "", err
	}

	req, err := http.NewRequest("POST", ChatGPTEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create HTTP request for ChatGPT API")
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Request to ChatGPT API failed")
		return "", err
	}
	defer resp.Body.Close()

	var responseBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		log.Error().Err(err).Msg("Failed to decode ChatGPT API response")
		return "", err
	}

	log.Debug().Interface("response_body", responseBody).Msg("Raw ChatGPT API response")

	if choices, ok := responseBody["choices"].([]interface{}); ok && len(choices) > 0 {
		if message, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{}); ok {
			return message["content"].(string), nil
		}
	}

	log.Warn().Msg("No valid response from ChatGPT API")
	return "", fmt.Errorf("no response from ChatGPT")
}
