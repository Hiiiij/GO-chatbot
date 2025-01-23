package api

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.Engine) {
	// public routes
	router.GET("/status", handleStatus) // health check/  public route

	//add protected routes with middleware
	protected := router.Group("/")
	protected.Use(APIKeyMiddleware())        // apply API key validation
	protected.Use(RequestLoggerMiddleware()) // apply request logging
	{
		protected.POST("/chat", handleChat)     // get full chat response
		protected.POST("/stream", handleStream) // get streaming chat responses
	}
}
