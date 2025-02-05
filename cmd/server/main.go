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
	log.Debug().Msg("Handler called")
	return ginLambda.ProxyWithContext(ctx, req)
}

func init() {
	log.Debug().Msg("=== Starting initialization ===")

	// set debug and use console output
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Debug().Msg("Logger initialized")

	log.Debug().Msg("Loading config...")
	cfg := config.LoadConfig()
	log.Debug().Str("mongo_uri", cfg.MongoURI).Msg("Config loaded")

	log.Debug().Msg("Validating environment variables...")
	validateEnvVars([]string{"MONGO_URI", "OPENAI_API_KEY"})
	log.Debug().Msg("Environment variables validated")

	// initialize db connection with error handling
	log.Debug().Msg("Attempting MongoDB connection...")
	if err := db.Connect(cfg.MongoURI); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}
	log.Debug().Msg("MongoDB connected successfully")

	// initialize Gin router
	log.Debug().Msg("Initializing Gin router...")
	router := gin.Default()
	api.RegisterRoutes(router)
	router.Use(errorHandlingMiddleware())
	log.Debug().Msg("Router initialized")

	// set up Gin-Lambda adapter for Lambda
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		log.Debug().Msg("Setting up Lambda adapter...")
		ginLambda = ginadapter.New(router)
		log.Debug().Msg("Lambda adapter initialized")
	} else {
		// run locally
		log.Info().Msgf("Server running locally on port %s", cfg.Port)
		if err := router.Run(":" + cfg.Port); err != nil {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}
	log.Debug().Msg("=== Initialization completed ===")
}

func main() {
	log.Debug().Msg("Starting Lambda handler")
	lambda.Start(Handler)
}

// validate environment variables
func validateEnvVars(requiredVars []string) {
	for _, v := range requiredVars {
		log.Debug().Str("var", v).Msg("Checking environment variable")
		if os.Getenv(v) == "" {
			log.Fatal().Msgf("environment variable %s is missing", v)
		}
	}
}

func errorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// check for errors after the request is processed
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Error().Err(e).Msg("request error")
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
	}
}
