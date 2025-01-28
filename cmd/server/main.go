package main

import (
	"context"
	"go-bot/internal/api"
	"go-bot/internal/config"
	"go-bot/internal/db"
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
	return ginLambda.ProxyWithContext(ctx, req)
}

func init() {
	// Set logging level to debug and use console output
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.LoadConfig()

	if err := db.Connect(cfg.MongoURI); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to mongodb")
	}

	router := gin.Default()
	api.RegisterRoutes(router)

	// running in AWS Lambda
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		ginLambda = ginadapter.New(router)
	} else {
		// running locally
		log.Info().Msgf("server running on port %s", cfg.Port)
		router.Run(":" + cfg.Port)
	}
}

func main() {
	lambda.Start(Handler)
}
