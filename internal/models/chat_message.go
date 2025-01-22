package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChatMessage stored in the MongoDB database
type ChatMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"` // MongoDB ObjectID
	UserID    string             `bson:"user_id"`       // The ID of the user sending the message
	Message   string             `bson:"message"`       // The user's message
	Response  string             `bson:"response"`      // The AI's response
	Timestamp time.Time          `bson:"timestamp"`     // Timestamp of the message
}

// ChatRequest is incoming chat request from the client
type ChatRequest struct {
	UserID  string `json:"user_id"`                    // The user ID making the request
	Message string `json:"message" binding:"required"` // The message from the user
}
