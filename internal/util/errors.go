package util

import "github.com/gin-gonic/gin"

// sends a JSON-formatted error response
func RespondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"error": message})
}

// send a generic JSON response with a message
func RespondWithMessage(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"message": message})
}
