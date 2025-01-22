package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"go-bot/internal/models"

	"github.com/rs/zerolog/log"
)

// structure of the request payload for the OpenAI API
type ChatGPTRequestPayload struct {
	Messages []map[string]string `json:"messages"`
	Model    string              `json:"model"`
	Stream   bool                `json:"stream,omitempty"`
}

// structure of response body from the OpenAI API
type ChatGPTResponseBody struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// construct the payload for the OpenAI API
func BuildChatGPTPayload(userMessage string, history []models.ChatMessage) ChatGPTRequestPayload {
	messages := []map[string]string{
		{"role": "system", "content": "You are a helpful assistant."},
	}

	// add chat history to the messages
	for _, chat := range history {
		messages = append(messages, map[string]string{"role": "user", "content": chat.Message})
	}
	messages = append(messages, map[string]string{"role": "user", "content": userMessage})

	return ChatGPTRequestPayload{
		Messages: messages,
		Model:    "gpt-4-turbo",
	}
}

// send request to OpenAI's API and returns the response.=
func CallOpenAI(payload ChatGPTRequestPayload) (string, error) {
	// Marshal the payload into JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// create a new HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	// add headers for authentication and content type.
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	// send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to call OpenAI API")
		return "", err
	}
	defer resp.Body.Close()

	// parse the response
	if resp.StatusCode != http.StatusOK {
		log.Error().Int("status_code", resp.StatusCode).Msg("Unexpected OpenAI API response status")
		return "", errors.New("failed to get a valid response from OpenAI API")
	}

	var responseBody ChatGPTResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", err
	}

	// return the content of the first response choice
	if len(responseBody.Choices) > 0 {
		return responseBody.Choices[0].Message.Content, nil
	}

	return "", errors.New("no response content from OpenAI API")
}
