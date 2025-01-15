package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-bot/models"
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
	client         *mongo.Client
	chatCollection *mongo.Collection
	OpenAIAPIKey   string
	APIKey         string
)

func init() {
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
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := connectToMongoDB(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}

	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.Use(apiKeyMiddleware)
	router.POST("/stream", handleStream)
	router.POST("/chat", handleChat)
	router.GET("/status", handleStatus)
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
	if err := client.Ping(context.TODO(), nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}
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

func handleStatus(c *gin.Context) {
	log.Info().Msg("Status check received")
	c.JSON(http.StatusOK, gin.H{"status": "OK"})
}

func handleStream(c *gin.Context) {
	var chatRequest = ChatRequest{}

	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		log.Error().Err(err).Msg("Invalid chat request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	payload := buildChatGPTPayload(chatRequest.UserID, chatRequest.Message)
	payload.Stream = true

	req, err := buildChatGPTRequest(payload)
	if err != nil {
		log.Error().Err(err).Msg("Error creating the request for chatgpt")
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call ChatGPT API"})
		return
	}
	defer resp.Body.Close()

	// Set headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	// Stream the response from OpenAI to the client
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip keep-alive lines and end marker
		if line == "" || line == "data: [DONE]" {
			continue
		}

		streamBytes := bytes.TrimLeft(scanner.Bytes(), "data:")

		var resp ChatGPTStreamBody
		if err := json.Unmarshal(streamBytes, &resp); err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal stream data")
			continue
		}

		// Send the data to the client
		fmt.Fprintf(c.Writer, "{response: \"%s\", finished: false}\n", resp.Choices[0].Delta.Content)
		c.Writer.Flush()
	}

	fmt.Fprintf(c.Writer, "{response: \"\", finished: true}\n")
	c.Writer.Flush()

	if err := scanner.Err(); err != nil {
		log.Error().Err(err).Msg("Error reading stream")
	}
}

type ChatRequest struct {
	UserID  string `json:"user_id" binding:"required"`
	Message string `json:"message" binding:"required"`
}

type ChatGPTRequestPayload struct {
	Messages []map[string]string `json:"messages"`
	Model    string              `json:"model"`
	Stream   bool                `json:"stream"`
}

type ChatGPTResponseBody struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type ChatGPTStreamBody struct {
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func buildChatGPTPayload(userId string, request string) ChatGPTRequestPayload {
	chatHistory, err := getChatHistoryFromDB(userId)

	messages := []map[string]string{
		{"role": "system", "content": "You are a helpful assistant."},
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch chat history")
		messages = append(messages, map[string]string{"role": "user", "content": request})
		return ChatGPTRequestPayload{Messages: messages, Model: "gpt-4-turbo", Stream: false}
	}

	for _, chat := range chatHistory {
		messages = append(messages, map[string]string{"role": "user", "content": chat.Message})
	}
	messages = append(messages, map[string]string{"role": "user", "content": request})

	return ChatGPTRequestPayload{Messages: messages, Model: "gpt-4-turbo", Stream: false}
}

func handleChat(c *gin.Context) {
	var chatRequest = ChatRequest{}

	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		log.Error().Err(err).Msg("Invalid chat request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	payload := buildChatGPTPayload(chatRequest.UserID, chatRequest.Message)

	response, err := callChatGPTWithHistory(payload)

	if err != nil {
		log.Error().Err(err).Msg("Failed to call ChatGPT API")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ChatGPT API call failed"})
		return
	}

	err = saveChatToDB(chatRequest.UserID, chatRequest.Message, response)
	if err != nil {
		log.Error().Err(err).Msg("Failed to save chat to database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save chat history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func buildChatGPTRequest(payload ChatGPTRequestPayload) (*http.Request, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", ChatGPTEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func callChatGPTWithHistory(payload ChatGPTRequestPayload) (string, error) {
	req, err := buildChatGPTRequest(payload)

	if err != nil {
		log.Error().Err(err).Msg("Error creating the request for chatgpt")
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Request to ChatGPT API failed")
		return "", err
	}
	defer resp.Body.Close()

	var responseBody ChatGPTResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		log.Error().Err(err).Msg("Failed to decode ChatGPT API response")
		return "", err
	}

	log.Debug().Interface("response_body", responseBody).Msg("Raw ChatGPT API response")

	if len(responseBody.Choices) > 0 {
		return responseBody.Choices[0].Message.Content, nil
	}

	log.Warn().Msg("No valid response from ChatGPT API")
	return "", fmt.Errorf("no response from ChatGPT")
}

func saveChatToDB(userID, message, response string) error {
	chatMessage := models.ChatMessage{
		UserID:    userID,
		Message:   message,
		Response:  response,
		Timestamp: time.Now(),
	}

	_, err := chatCollection.InsertOne(context.TODO(), chatMessage)
	return err
}

func getChatHistoryFromDB(userID string) ([]models.ChatMessage, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := chatCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var chatHistory []models.ChatMessage
	if err := cursor.All(context.TODO(), &chatHistory); err != nil {
		return nil, err
	}

	for i, j := 0, len(chatHistory)-1; i < j; i, j = i+1, j-1 {
		chatHistory[i], chatHistory[j] = chatHistory[j], chatHistory[i]
	}
	return chatHistory, nil
}
