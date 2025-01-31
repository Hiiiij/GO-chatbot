package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// register all API routes with the Gin router
func RegisterRoutes(router *gin.Engine) {

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-API-KEY"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/status", handleStatus) // health check/public route

	// add protected routes with middleware
	protected := router.Group("/")
	protected.Use(APIKeyMiddleware())        // apply API key validation
	protected.Use(RequestLoggerMiddleware()) // apply request logging
	{

		protected.POST("/chat", handleChat)     // get full chat response
		protected.POST("/stream", handleStream) // get streaming chat responses
	}
}
