package main

import (
	"go-bot/internal/api"
	"go-bot/internal/config"
	"go-bot/internal/db"
	"go-bot/internal/models"
	"go-bot/internal/service"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// set logging level to debug and use console output
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.LoadConfig()

	if err := db.Connect(cfg.MongoURI); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to mongodb")
	}

	router := gin.Default()
	api.RegisterRoutes(router)

	request := models.ChatRequest{
		UserID:  "test_user",
		Message: "What is the capital of France?",
	}
	_, err := service.ProcessStream(request)
	if err != nil {
		log.Error().Err(err).Msg("failed to process stream")
	}

	// start http server on configured port
	log.Info().Msgf("server running on port %s", cfg.Port)
	router.Run(":" + cfg.Port)
}
