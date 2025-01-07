package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Configuration constants
const (
	OpenAIAPIKey    = "your_openai_api_key"
	ChatGPTEndpoint = "https://api.openai.com/v1/chat/completions"
	WhisperEndpoint = "https://api.openai.com/v1/audio/transcriptions"
	S3BucketName    = "your_s3_bucket_name"
	APIKey          = "your_secret_api_key"
)

// Logger setup
var logger = logrus.New()

func main() {
	// Initialize Gin router
	router := gin.Default()

	// Middleware for API key authentication
	router.Use(apiKeyMiddleware)

	// Define endpoints
	router.POST("/chat", handleChat)
	router.GET("/status", handleStatus)

	// Start the server
	router.Run(":8080")
}

// handleChat processes text-based queries
func handleChat(c *gin.Context) {
	var chatRequest struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		logger.Error("Invalid chat request: ", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Call ChatGPT API
	response, err := callChatGPT(chatRequest.Message)
	if err != nil {
		logger.Error("Failed to call ChatGPT API: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ChatGPT API call failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

// handleStatus provides a health check endpoint
func handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

// callChatGPT makes requests to the OpenAI ChatGPT API
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var responseBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", err
	}

	if choices, ok := responseBody["choices"].([]interface{}); ok && len(choices) > 0 {
		if message, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{}); ok {
			return message["content"].(string), nil
		}
	}

	return "", fmt.Errorf("no response from ChatGPT")
}


// Middleware for API key authentication
func apiKeyMiddleware(c *gin.Context) {
	clientKey := c.GetHeader("X-API-KEY")
	if clientKey != APIKey {
		logger.Warn("Unauthorized access attempt")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	c.Next()
}
