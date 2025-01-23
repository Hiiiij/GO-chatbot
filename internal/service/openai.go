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

// structure of the request payload
type ChatGPTRequestPayload struct {
	Messages []map[string]string `json:"messages"`         // including user and system roles
	Model    string              `json:"model"`            // OpenAI model to use
	Stream   bool                `json:"stream,omitempty"` // flag for streaming responses
}

// structure of response body from the OpenAI API
type ChatGPTResponseBody struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`    // role of the message (e.g., assistant)
			Content string `json:"content"` // content of the response
		} `json:"message"`
	} `json:"choices"`
}

// construct payload for OpenAI API
func BuildChatGPTPayload(userMessage string, history []models.ChatMessage, systemMessage string) ChatGPTRequestPayload {
	log.Debug().Str("userMessage", userMessage).Msg("Building payload")
	log.Debug().Int("history_length", len(history)).Msg("Chat history length")
	log.Debug().Str("systemMessage", systemMessage).Msg("System message used")

	messages := []map[string]string{
		{"role": "system", "content": systemMessage},
	}

	// add chat history to the messages
	for _, chat := range history {
		messages = append(messages, map[string]string{"role": "user", "content": chat.Message})
	}
	messages = append(messages, map[string]string{"role": "user", "content": userMessage})

	payload := ChatGPTRequestPayload{
		Messages: messages,
		Model:    "gpt-4-turbo",
	}

	log.Debug().Interface("payload", payload).Msg("Constructed OpenAI payload")
	return payload
}

// send request to OpenAI's API and return the response
func CallOpenAI(payload ChatGPTRequestPayload) (string, error) {
	// convert the payload into JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal payload")
		return "", err
	}
	log.Debug().Str("json_payload", string(jsonData)).Msg("Marshalled payload for OpenAI")

	// create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create OpenAI API request")
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", "application/json")
	log.Debug().Msg("Sending request to OpenAI API")

	// send the request using an HTTP client
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to call OpenAI API")
		return "", err
	}
	defer resp.Body.Close()

	log.Debug().Int("status_code", resp.StatusCode).Msg("Received response from OpenAI API")

	// decode the response body
	var responseBody ChatGPTResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		log.Error().Err(err).Msg("Failed to decode OpenAI API response")
		return "", err
	}

	if len(responseBody.Choices) > 0 {
		log.Debug().Str("response_content", responseBody.Choices[0].Message.Content).Msg("OpenAI response content")
		return responseBody.Choices[0].Message.Content, nil
	}

	log.Error().Msg("No response content from OpenAI API")
	return "", errors.New("no response content from OpenAI API")
}
