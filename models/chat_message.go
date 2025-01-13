package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    string             `bson:"user_id"`
	Message   string             `bson:"message"`
	Response  string             `bson:"response"`
	Timestamp time.Time          `bson:"timestamp"`
}
