package main

import (
	"go-bot/internal/api"
	"go-bot/internal/config"
	"go-bot/internal/db"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}) // Fix requires os.Stderr

	// load configuration
	cfg := config.LoadConfig() // Fix: Remove the second return value `err`

	// connect to MongoDB
	if err := db.Connect(cfg.MongoURI); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}

	// setup Gin router
	router := gin.Default()
	api.RegisterRoutes(router)

	log.Info().Msgf("Server running on port %s", cfg.Port) // use `cfg.Port` dynamically
	router.Run(":" + cfg.Port)                             // use configured port
}
