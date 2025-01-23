package test

import (
	"context"
	"testing"
	"time"

	"go-bot/internal/db"
	"go-bot/internal/models"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDB() *mongo.Client {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017") // Change to a test database
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		panic(err)
	}
	return client
}

func teardownTestDB(client *mongo.Client) {
	client.Database("go-chat-backend").Drop(context.TODO())
	client.Disconnect(context.TODO())
}

func TestSaveChat(t *testing.T) {
	client := setupTestDB()
	defer teardownTestDB(client)

	db.Connect("mongodb://localhost:27017") // Use the same URI as your test DB
	chat := models.ChatMessage{
		UserID:    "test_user",
		Message:   "Hello!",
		Response:  "Hi there!",
		Timestamp: time.Now(),
	}

	err := db.SaveChat(chat.UserID, chat.Message, chat.Response)
	assert.NoError(t, err)

	// verify that the message was saved
	var result models.ChatMessage
	collection := client.Database("go-chat-backend").Collection("chatSchema")
	err = collection.FindOne(context.TODO(), bson.M{"user_id": "test_user"}).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, chat.Message, result.Message)
	assert.WithinDuration(t, chat.Timestamp, result.Timestamp, time.Second)
}

func TestGetChatHistory(t *testing.T) {
	client := setupTestDB()
	defer teardownTestDB(client)

	db.Connect("mongodb://localhost:27017")
	collection := client.Database("go-chat-backend").Collection("chatSchema")

	// insert mock data
	mockChats := []interface{}{
		models.ChatMessage{
			UserID:    "test_user",
			Message:   "Message 1",
			Response:  "Response 1",
			Timestamp: time.Now(),
		},
		models.ChatMessage{
			UserID:    "test_user",
			Message:   "Message 2",
			Response:  "Response 2",
			Timestamp: time.Now(),
		},
	}
	_, err := collection.InsertMany(context.TODO(), mockChats)
	assert.NoError(t, err)

	// fetch chat history
	history, err := db.GetChatHistory("test_user")
	assert.NoError(t, err)
	assert.Len(t, history, 2)
}
