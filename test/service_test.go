package test

import (
	"errors"
	"testing"

	"go-bot/internal/db"
	"go-bot/internal/models"

	"github.com/stretchr/testify/assert"
)

// simulates external dependencies/ OpenAI API calls
type MockService struct{}

func (m *MockService) ProcessChat(request models.ChatRequest) (string, error) {
	if request.Message == "" {
		return "", errors.New("message cannot be empty")
	}
	return "Paris", nil
}

func (m *MockService) ProcessStream(request models.ChatRequest) (chan string, error) {
	if request.Message == "" {
		return nil, errors.New("message cannot be empty")
	}
	stream := make(chan string, 3)
	go func() {
		defer close(stream)
		stream <- "Once upon a time,"
		stream <- "in a faraway land,"
		stream <- "there lived a king."
	}()
	return stream, nil
}

func TestProcessChat(t *testing.T) {
	// setup mock DB with test URI
	db.Connect("mongodb://localhost:27017/test-db")
	defer db.Disconnect()

	// use mock service
	mockService := &MockService{}

	// test valid request
	request := models.ChatRequest{
		UserID:  "test_user",
		Message: "What is the capital of France?",
	}

	response, err := mockService.ProcessChat(request)
	assert.NoError(t, err)
	assert.Equal(t, "Paris", response) // Validate mocked response

	// test empty message
	request.Message = ""
	response, err = mockService.ProcessChat(request)
	assert.Error(t, err)
	assert.Equal(t, "message cannot be empty", err.Error())
	assert.Empty(t, response)
}

func TestProcessStream(t *testing.T) {
	// setup mock DB with test URI
	db.Connect("mongodb://localhost:27017/test-db")
	defer db.Disconnect()

	// use mock service
	mockService := &MockService{}

	// test valid request
	request := models.ChatRequest{
		UserID:  "test_user",
		Message: "Tell me a story.",
	}

	streamChannel, err := mockService.ProcessStream(request)
	assert.NoError(t, err)

	var messages []string
	for msg := range streamChannel {
		messages = append(messages, msg)
	}
	assert.Equal(t, []string{
		"Once upon a time,",
		"in a faraway land,",
		"there lived a king.",
	}, messages)

	// test empty message
	request.Message = ""
	streamChannel, err = mockService.ProcessStream(request)
	assert.Error(t, err)
	assert.Nil(t, streamChannel)
	assert.Equal(t, "message cannot be empty", err.Error())
}
