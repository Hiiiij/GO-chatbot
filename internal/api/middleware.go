package api

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// APIKeyMiddleware validates the API key from the request header
func APIKeyMiddleware() gin.HandlerFunc {
	apiKey := os.Getenv("API_KEY") // Load API key from environment variables

	if apiKey == "" {
		log.Fatal().Msg("API_KEY not set in environment variables. Shutting down.")
		os.Exit(1)
	}

	return func(c *gin.Context) {
		clientKey := c.GetHeader("X-API-KEY")

		// Validate the API key
		if clientKey != apiKey {
			log.Warn().
				Str("client_ip", c.ClientIP()).
				Str("received_key", clientKey).
				Msg("Unauthorized access attempt")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Log successful validation
		log.Debug().
			Str("client_ip", c.ClientIP()).
			Msg("API key validated successfully")

		c.Next()
	}
}

// RequestLoggerMiddleware logs details of incoming requests
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Log request details before processing
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("client_ip", c.ClientIP()).
			Msg("Incoming request")

		// Process the request
		c.Next()

		// Log response details after processing
		duration := time.Since(startTime)
		log.Info().
			Int("status", c.Writer.Status()).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Dur("duration", duration).
			Int("response_size", c.Writer.Size()).
			Msg("Request processed")
	}
}
