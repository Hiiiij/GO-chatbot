package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func RegisterRoutes(router *gin.Engine) {
	// setup CORS - keeping your original configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"}, // Keep allowing all for development
		AllowMethods:  []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-API-KEY"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}))

	// health check
	router.GET("/status", handleStatus)

	// service-level protected routes with middleware
	protected := router.Group("/")
	protected.Use(APIKeyMiddleware(), RequestLoggerMiddleware())
	{
		protected.POST("/chat", handleChat)
		protected.POST("/stream", handleStream)
	}

	log.Debug().Msg("Routes registered successfully")
}
