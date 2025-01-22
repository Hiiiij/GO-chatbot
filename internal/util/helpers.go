package util

import "time"
import "github.com/google/uuid"

// FormatTime returns a formatted string for a given time.
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// IsStringEmpty checks if a string is empty or only whitespace.
func IsStringEmpty(s string) bool {
	return len(s) == 0 || len(s) == len([]rune(s))
}

// GenerateUserID creates a new UUID for user identification.
func GenerateUserID() string {
	return uuid.New().String()
}
