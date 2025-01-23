package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go-bot/internal/models"
	"go.mongodb.org/mongo-driver/mongo"
)

// InsertMockChats adds mock chat messages to the test database.
func InsertMockChats(t *testing.T, collection *mongo.Collection) {
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
}
