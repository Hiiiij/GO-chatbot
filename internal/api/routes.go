package api

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.SetTrustedProxies([]string{"127.0.0.1"}) // trust the local proxy
	// setup CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-API-KEY"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}))

	// health check
	router.GET("/status", handleStatus)

	// protected routes with middleware
	protected := router.Group("/")
	protected.Use(APIKeyMiddleware(), RequestLoggerMiddleware())
	{
		protected.POST("/chat", handleChat)
		protected.POST("/stream", handleStream)
	}

	// register proxy routes separately
	RegisterProxyRoutes(router)
}

// proxy routes for AWS API Gateway
func RegisterProxyRoutes(router *gin.Engine) {
	proxy := router.Group("/proxy") // avoids conflict with existing routes
	proxy.Any("/*proxyPath", proxyHandler)
}

func proxyHandler(c *gin.Context) {
	proxyPath := c.Param("proxyPath")
	apiGatewayURL := getAPIGatewayURL() + proxyPath

	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, apiGatewayURL, c.Request.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	req.Header = c.Request.Header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		c.Writer.Header()[k] = v
	}
	c.Writer.WriteHeader(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// returns API Gateway URL
func getAPIGatewayURL() string {
	return "https://md2vufnx60.execute-api.eu-central-1.amazonaws.com/default/goBot"
}
