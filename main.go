package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const ChatGPTEndpoint = "https://api.openai.com/v1/chat/completions"

var (
	client        *mongo.Client
	chatCollection *mongo.Collection
	OpenAIAPIKey  string
	APIKey        string
)

func init() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal().Msg("Error loading .env file")
	}

	OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	APIKey = os.Getenv("API_KEY")

	if OpenAIAPIKey == "" || APIKey == "" {
		log.Fatal().Msg("Missing required environment variables: OPENAI_API_KEY or API_KEY")
	}
}

func main() {
	// Set up logging
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Connect to MongoDB
	if err := connectToMongoDB(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}

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

func connectToMongoDB() error {
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGO_URI"))
	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Check the connection
	if err := client.Ping(context.TODO(), nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Access the database and collection
	chatCollection = client.Database("go-chat-backed").Collection("chatSchema")
	log.Info().Msg("Connected to MongoDB")
	return nil
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
		UserID  string `json:"user_id" binding:"required"`
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

	// Save chat message and response to MongoDB
	err = saveChatToDB(chatRequest.UserID, chatRequest.Message, response)
	if err != nil {
		log.Error().Err(err).Msg("Failed to save chat to database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save chat history"})
		return
	}

	// Retrieve chat history for the user
	chatHistory, err := getChatHistoryFromDB(chatRequest.UserID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve chat history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve chat history"})
		return
	}

	// Respond with the ChatGPT response and chat history
	c.JSON(http.StatusOK, gin.H{
		"response":    response,
		"chatHistory": chatHistory,
	})
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

func saveChatToDB(userID, message, response string) error {
	doc := bson.D{
		{Key: "user_id", Value: userID},
		{Key: "message", Value: message},
		{Key: "response", Value: response},
		{Key: "timestamp", Value: time.Now()},
	}
	_, err := chatCollection.InsertOne(context.TODO(), doc)
	return err
}


func getChatHistoryFromDB(userID string) ([]bson.M, error) {
	filter := bson.M{"user_id": userID}
	cursor, err := chatCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var chatHistory []bson.M
	if err := cursor.All(context.TODO(), &chatHistory); err != nil {
		return nil, err
	}
	return chatHistory, nil
}
