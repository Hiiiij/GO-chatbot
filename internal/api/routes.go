package api

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.Engine) {
	// Public routes
	router.GET("/status", handleStatus) // Health check route is public

	// Protected routes/ middleware applied
	protected := router.Group("/")
	protected.Use(APIKeyMiddleware())        // Apply API key validation
	protected.Use(RequestLoggerMiddleware()) // Apply request logging
	{
		protected.POST("/chat", handleChat)     // Chat requests
		protected.POST("/stream", handleStream) // Streaming chat responses
	}
}
