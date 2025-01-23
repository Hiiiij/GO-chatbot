package test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-bot/internal/api"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandleChat(t *testing.T) {
	// mock request
	req := httptest.NewRequest("POST", "/chat", bytes.NewBuffer([]byte(`{"user_id": "123", "message": "Hello!"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "a7d93eded416cfc6631847132dab0ef8226d2854c865284da5c0d107e3af96b2")

	// mock response recorder
	w := httptest.NewRecorder()

	// initialize router
	router := gin.Default()
	api.RegisterRoutes(router)

	// perform the request
	router.ServeHTTP(w, req)

	// assert the status code
	assert.Equal(t, http.StatusOK, w.Code) // set status 200
}
