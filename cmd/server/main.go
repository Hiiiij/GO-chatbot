package main

import (
	"context"
	"go-bot/internal/api"
	"go-bot/internal/config"
	"go-bot/internal/db"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

var ginLambda *ginadapter.GinLambda

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("panic occurred: %v", r)
		}
	}()

	return ginLambda.ProxyWithContext(ctx, req)
}

func init() {
	// set debug and use console output
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.LoadConfig()

	validateEnvVars([]string{"MONGO_URI", "OPENAI_API_KEY"})

	// initialize db connection with error handling
	if err := db.Connect(cfg.MongoURI); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}

	// initialze Gin router
	router := gin.Default()
	api.RegisterRoutes(router)

	router.Use(errorHandlingMiddleware())

	// set up Gin-Lambda adapter for Lambda
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		ginLambda = ginadapter.New(router)
	} else {
		// run locally
		log.Info().Msgf("server running on port %s", cfg.Port)
		if err := router.Run(":" + cfg.Port); err != nil {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}
}

func main() {
	lambda.Start(Handler)
}

// validate environment variables
func validateEnvVars(requiredVars []string) {
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			log.Fatal().Msgf("environment variable %s is missing", v)
		}
	}
}

func errorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// Check for errors after the request is processed
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Error().Err(e).Msg("request error")
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
	}
}
