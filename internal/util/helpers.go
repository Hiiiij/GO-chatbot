package util

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// checks if a string is empty or whitespace
func IsStringEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// create a new UUID
func GenerateUserID() string {
	return uuid.New().String()
}

// log either req or res based on the flag
func LogRequestOrResponse(c *gin.Context, duration time.Duration, isResponse bool) {
	logger := log.Info().
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Str("client_ip", c.ClientIP())

	if isResponse {
		logger = logger.
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Int("response_size", c.Writer.Size())
	}

	logger.Msg("http lifecycle event")
}

func SendOpenAIRequest(payload interface{}, stream bool) (*http.Response, error) {
	// convert payload to a map to ensure "stream" is added
	if stream {
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			payloadMap["stream"] = true
		} else {
			// Convert struct to map for adding "stream"
			payloadBytes, _ := json.Marshal(payload)
			var payloadMap map[string]interface{}
			_ = json.Unmarshal(payloadBytes, &payloadMap)
			payloadMap["stream"] = true
			payload = payloadMap
		}
	}

	// convert the payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal payload")
		return nil, err
	}

	// create the HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create OpenAI request")
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	log.Debug().Str("request_payload", string(jsonData)).Msg("Sending OpenAI request")

	// send request using an HTTP client
	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}
