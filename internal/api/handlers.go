package api

import (
	"net/http"

	"go-bot/internal/models"
	"go-bot/internal/service"
	"go-bot/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// handleChat processes regular chat requests.
func handleChat(c *gin.Context) {
	var chatRequest models.ChatRequest
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		util.RespondWithError(c, http.StatusBadRequest, "Invalid chat request")
		return
	}

	response, err := service.ProcessChat(chatRequest)
	if err != nil {
		util.RespondWithError(c, http.StatusInternalServerError, "Failed to process chat")
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

// handleStream processes streaming chat requests.
func handleStream(c *gin.Context) {
	var chatRequest models.ChatRequest
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		util.RespondWithError(c, http.StatusBadRequest, "Invalid stream request")
		return
	}

	streamChannel, err := service.ProcessStream(chatRequest)
	if err != nil {
		util.RespondWithError(c, http.StatusInternalServerError, "Streaming failed")
		return
	}

	// Stream the response
	for msg := range streamChannel {
		c.SSEvent("message", msg)
	}

	c.Status(http.StatusOK)
}

func handleStatus(c *gin.Context) {
	log.Info().Msg("Status check received")
	c.JSON(http.StatusOK, gin.H{"status": "Service is running"})
}
