package api

import (
	"net/http"
	"os"
	"time"

	"go-bot/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// validate the API key from the request header
func APIKeyMiddleware() gin.HandlerFunc {
	apiKey := os.Getenv("API_KEY") // load API key from environment variables

	if apiKey == "" {
		log.Fatal().Msg("API_KEY not set in environment variables. Shutting down.")
		os.Exit(1)
	}

	return func(c *gin.Context) {
		clientKey := c.GetHeader("X-API-KEY")

		// validate the API key
		if clientKey != apiKey {
			log.Warn().
				Str("client_ip", c.ClientIP()).
				Str("received_key", clientKey).
				Msg("unauthorized access attempt")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		log.Debug().
			Str("client_ip", c.ClientIP()).
			Msg("api key validated successfully")

		c.Next()
	}
}

// logs details of incoming requests and responses
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// log req details
		util.LogRequestOrResponse(c, 0, false)

		// process the request
		c.Next()

		// log res details
		duration := time.Since(startTime)
		util.LogRequestOrResponse(c, duration, true)
	}
}
