package api

import (
	"io"
	"net/http"
	"strings"

	"go-bot/internal/models"
	"go-bot/internal/service"
	"go-bot/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func handleChat(c *gin.Context) {
	var chatRequest models.ChatRequest
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		log.Error().Err(err).Msg("Invalid chat request payload")
		util.RespondWithError(c, http.StatusBadRequest, "Invalid chat request payload")
		return
	}

	// validate the input
	if strings.TrimSpace(chatRequest.Message) == "" {
		log.Error().Msg("Chat request message is empty")
		util.RespondWithError(c, http.StatusBadRequest, "Message cannot be empty")
		return
	}

	// centralized OpenAI request logic
	response, err := service.ProcessChat(chatRequest)
	if err != nil {
		log.Error().Err(err).Msg("Failed to process chat request")
		util.RespondWithError(c, http.StatusInternalServerError, "Failed to process chat request")
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

func handleStream(c *gin.Context) {
	var chatRequest models.ChatRequest
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		log.Error().Err(err).Msg("Invalid stream request payload")
		util.RespondWithError(c, http.StatusBadRequest, "Invalid stream request payload")
		return
	}

	// validate the input
	if strings.TrimSpace(chatRequest.Message) == "" {
		log.Error().Msg("Stream request message is empty")
		util.RespondWithError(c, http.StatusBadRequest, "Message cannot be empty")
		return
	}

	streamChannel, err := service.ProcessStream(chatRequest)
	if err != nil {
		log.Error().Err(err).Msg("Failed to process streaming request")
		util.RespondWithError(c, http.StatusInternalServerError, "Streaming failed")
		return
	}

	// check if streamChannel exists
	if streamChannel == nil {
		log.Error().Msg("Stream channel is nil")
		util.RespondWithError(c, http.StatusInternalServerError, "Streaming initialization failed")
		return
	}

	// stream response
	c.Stream(func(w io.Writer) bool {
		for msg := range streamChannel {
			cleanMsg := strings.TrimSpace(msg)

			// ignore "[DONE]" token
			if cleanMsg == "[DONE]" {
				log.Debug().Msg("Received [DONE] token, closing stream")
				return false
			}

			log.Debug().Str("streamed_message", cleanMsg).Msg("Streaming message to client")
			c.SSEvent("message", cleanMsg)
		}

		c.SSEvent("done", "Stream completed")
		return false
	})
}

func handleStatus(c *gin.Context) {
	log.Info().Msg("Status check received")
	c.JSON(http.StatusOK, gin.H{
		"status":      "service is running",
		"status_code": http.StatusOK,
	})
}
