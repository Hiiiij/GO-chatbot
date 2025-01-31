package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`    // MongoDB ObjectID
	UserID    string             `bson:"user_id" json:"user_id"`     // The ID of the user sending the message
	Message   string             `bson:"message" json:"message"`     // The user's message
	Response  string             `bson:"response" json:"response"`   // The AI's response
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"` // Timestamp of the message
}

// chat req represents incoming chat request from the client
type ChatRequest struct {
	UserID  string `json:"user_id" example:"12345"`                     // The user ID making the request
	Message string `json:"message" binding:"required" example:"Hello!"` // The message from the user
}
