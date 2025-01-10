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

func init() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal().Msg("Error loading .env file")
	}
}


func main() {
	// Set up zerolog to output pretty logs during development
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Initialize Gin router
	router := gin.Default()

	// Fix trusted proxies warning
	router.SetTrustedProxies(nil)

	// Middleware for API key authentication
	router.Use(apiKeyMiddleware)

	// Define endpoints
	router.POST("/chat", handleChat)
	router.GET("/status", handleStatus)

	// Start the server
	router.Run(":8080")
}


const (
	ChatGPTEndpoint = "https://api.openai.com/v1/chat/completions"
)

var (
	OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	APIKey       = os.Getenv("API_KEY")
)

func apiKeyMiddleware(c *gin.Context) {
	clientKey := c.GetHeader("X-API-KEY")
	log.Info().Msgf("Received API key: %s", clientKey)

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

	response, err := callChatGPT(chatRequest.Message)
	if err != nil {
		log.Error().Err(err).Msg("Failed to call ChatGPT API")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ChatGPT API call failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

func callChatGPT(userMessage string) (string, error) {
	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": userMessage},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", ChatGPTEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Msg("Request failed")
		return "", err
	}
	defer resp.Body.Close()

	var responseBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		log.Err(err).Msg("Decoding of response failed")
		return "", err
	}

	log.Debug().Msg(fmt.Sprintf("%v", responseBody))

	if choices, ok := responseBody["choices"].([]interface{}); ok && len(choices) > 0 {
		if message, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{}); ok {
			return message["content"].(string), nil
		}
	}

	return "", fmt.Errorf("no response from ChatGPT")
}
