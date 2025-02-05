package db

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"go-bot/internal/models"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client         *mongo.Client
	chatCollection *mongo.Collection
	clientMutex    sync.RWMutex
)

func Connect(mongoURI string) error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	// reuse existing connection if valid
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := client.Ping(ctx, nil); err == nil {
			return nil
		}
		_ = client.Disconnect(context.Background())
	}

	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetMaxPoolSize(5).
		SetServerSelectionTimeout(5 * time.Second).
		SetConnectTimeout(5 * time.Second).
		SetSocketTimeout(5 * time.Second).
		SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12})

	// try to connect with retries
	var err error
	for attempts := 1; attempts <= 3; attempts++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		log.Debug().Msgf("MongoDB connection attempt %d/3", attempts)
		client, err = mongo.Connect(ctx, clientOptions)

		if err == nil {
			// test connection
			if err = client.Ping(ctx, nil); err == nil {
				chatCollection = client.Database("go-chat-backend").Collection("chatSchema")
				log.Info().Msg("MongoDB connection successful")
				cancel()
				return nil
			}
		}

		cancel()
		if attempts < 3 {
			time.Sleep(time.Second * 2)
		}
	}

	log.Error().Err(err).Msg("All MongoDB connection attempts failed")
	return err
}

func SaveChat(userID, message, response string) error {
	clientMutex.RLock()
	defer clientMutex.RUnlock()

	if chatCollection == nil {
		return mongo.ErrClientDisconnected
	}

	chat := models.ChatMessage{
		UserID:    userID,
		Message:   message,
		Response:  response,
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Debug().
		Str("userID", userID).
		Str("message", message).
		Msg("Saving chat message")

	_, err := chatCollection.InsertOne(ctx, chat)
	if err != nil {
		log.Error().Err(err).Msg("Failed to save chat message")
	}
	return err
}

func GetChatHistory(userID string, limit int) ([]models.ChatMessage, error) {
	clientMutex.RLock()
	defer clientMutex.RUnlock()

	if chatCollection == nil {
		return nil, mongo.ErrClientDisconnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Debug().
		Str("userID", userID).
		Int("limit", limit).
		Msg("Retrieving chat history")

	filter := bson.M{"user_id": userID}
	findOptions := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(int64(limit))

	cursor, err := chatCollection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve chat history")
		return nil, err
	}
	defer cursor.Close(ctx)

	var chats []models.ChatMessage
	if err := cursor.All(ctx, &chats); err != nil {
		log.Error().Err(err).Msg("Failed to decode chat messages")
		return nil, err
	}

	// reverse order to send oldest messages first
	for i, j := 0, len(chats)-1; i < j; i, j = i+1, j-1 {
		chats[i], chats[j] = chats[j], chats[i]
	}

	log.Debug().
		Int("messageCount", len(chats)).
		Msg("Successfully retrieved chat history")

	return chats, nil
}

func IsConnected() bool {
	clientMutex.RLock()
	defer clientMutex.RUnlock()

	if client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return client.Ping(ctx, nil) == nil
}

func Disconnect() error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return client.Disconnect(ctx)
	}
	return nil
}
