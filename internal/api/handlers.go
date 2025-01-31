package api

import (
	"io"
	"net/http"

	"go-bot/internal/models"
	"go-bot/internal/service"
	"go-bot/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func handleChat(c *gin.Context) {
	var chatRequest models.ChatRequest
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		util.RespondWithError(c, http.StatusBadRequest, "invalid chat request")
		return
	}

	// use centralized OpenAI request logic
	response, err := service.ProcessChat(chatRequest)
	if err != nil {
		util.RespondWithError(c, http.StatusInternalServerError, "failed to process chat")
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func handleStream(c *gin.Context) {
	var chatRequest models.ChatRequest
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		log.Error().Err(err).Msg("Invalid stream request")
		util.RespondWithError(c, http.StatusBadRequest, "Invalid stream request")
		return
	}

	//  process streaming
	streamChannel, err := service.ProcessStream(chatRequest)
	if err != nil {
		log.Error().Err(err).Msg("Failed to process streaming request")
		util.RespondWithError(c, http.StatusInternalServerError, "Streaming failed")
		return
	}

	// response back
	c.Stream(func(w io.Writer) bool {
		for msg := range streamChannel {
			log.Debug().Str("streamed_message", msg).Msg("Streaming message to client")
			c.SSEvent("message", msg)
		}

		// signal the end of the stream
		c.SSEvent("done", "Stream completed")
		return false
	})
}

// return the service status
func handleStatus(c *gin.Context) {
	log.Info().Msg("status check received")
	c.JSON(http.StatusOK, gin.H{"status": "service is running"})
}
