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

func IsStringEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func GenerateUserID() string {
	return uuid.New().String()
}

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
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal payload")
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create OpenAI request")
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	if stream {
		req.Header.Set("Accept", "text/event-stream")
		client := &http.Client{} // No timeout for streaming
		return client.Do(req)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}
