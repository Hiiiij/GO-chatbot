package db

import (
	"context"
	"time"

	"go-bot/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var chatCollection *mongo.Collection // chat messages collection
var client *mongo.Client             // mongodb client

// connect to mongodb and set up collection
func Connect(mongoURI string) error {
	clientOptions := options.Client().ApplyURI(mongoURI)
	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}

	if err := client.Ping(context.TODO(), nil); err != nil {
		return err
	}

	chatCollection = client.Database("go-chat-backend").Collection("chatSchema")
	return nil
}

// disconnect from mongodb
func Disconnect() error {
	if client != nil {
		return client.Disconnect(context.TODO())
	}
	return nil
}

// save chat message to mongodb
func SaveChat(userID, message, response string) error {
	if chatCollection == nil {
		return mongo.ErrClientDisconnected
	}

	chat := models.ChatMessage{
		UserID:    userID,
		Message:   message,
		Response:  response,
		Timestamp: time.Now(),
	}

	_, err := chatCollection.InsertOne(context.TODO(), chat)
	return err
}

// get chat history by user id
func GetChatHistory(userID string) ([]models.ChatMessage, error) {
	if chatCollection == nil {
		return nil, mongo.ErrClientDisconnected
	}

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
