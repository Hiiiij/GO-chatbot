package db

import (
	"context"
	"time"

	"go-bot/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var chatCollection *mongo.Collection

// global MongoDB client
var client *mongo.Client

// Connect initializes the MongoDB connection and sets up the collection
func Connect(mongoURI string) error {
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}
	if err := client.Ping(context.TODO(), nil); err != nil {
		return err
	}
	chatCollection = client.Database("go-chat-backend").Collection("chatSchema")
	return nil
}

// Disconnect closes the MongoDB connection
func Disconnect() error {
	if client != nil {
		return client.Disconnect(context.TODO())
	}
	return nil
}

// SaveChat inserts a new chat message into the MongoDB collection.
func SaveChat(userID, message, response string) error {
	chat := models.ChatMessage{
		UserID:    userID,
		Message:   message,
		Response:  response,
		Timestamp: time.Now(),
	}
	_, err := chatCollection.InsertOne(context.TODO(), chat)
	return err
}

// GetChatHistory retrieves all chat messages for a given user ID from MongoDB.
func GetChatHistory(userID string) ([]models.ChatMessage, error) {
	filter := bson.M{"user_id": userID}
	cursor, err := chatCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var chats []models.ChatMessage
	if err := cursor.All(context.TODO(), &chats); err != nil {
		return nil, err
	}
	return chats, nil
}
