package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"go-bot/internal/db"
	"go-bot/internal/models"
	"go-bot/internal/util"

	"github.com/rs/zerolog/log"
)

// represents a single message chunk from the OpenAI streaming response
type ChatGPTStreamBody struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// handles streaming requests by interacting with OpenAI's API.
func ProcessStream(request models.ChatRequest) (<-chan string, error) {
	// Auto-generate UserID if not provided
	if request.UserID == "" {
		request.UserID = util.GenerateUserID()
		log.Debug().Msgf("Generated UserID: %s", request.UserID)
	}

	// fetch chat history
	chatHistory, err := db.GetChatHistory(request.UserID)
	if err != nil {
		return nil, err
	}

	// build OpenAI payload
	payload := BuildChatGPTPayload(request.Message, chatHistory)
	payload.Stream = true

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// send request to OpenAI
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// stream response back to client
	streamChannel := make(chan string)
	go func() {
		defer resp.Body.Close()
		defer close(streamChannel)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// skip invalid lines
			if len(line) < 6 || line[:5] != "data:" {
				continue
			}

			// remove "data:" prefix and decode JSON
			jsonData := line[5:]
			if jsonData == "[DONE]" {
				break
			}

			var streamBody ChatGPTStreamBody
			if err := json.Unmarshal([]byte(jsonData), &streamBody); err != nil {
				log.Error().Err(err).Msg("Failed to decode stream data")
				continue
			}

			// send chunk content to client
			if len(streamBody.Choices) > 0 {
				content := streamBody.Choices[0].Delta.Content
				streamChannel <- content
			}
		}
	}()

	return streamChannel, nil
}

// handle non-streaming chat requests
func ProcessChat(request models.ChatRequest) (string, error) {
	// auto-generate UserID if not provided
	if request.UserID == "" {
		request.UserID = util.GenerateUserID()
		log.Debug().Msgf("Generated UserID: %s", request.UserID)
	}

	// fetch from db
	chatHistory, err := db.GetChatHistory(request.UserID)
	if err != nil {
		return "", err
	}

	// build payload and send request to OpenAI API
	payload := BuildChatGPTPayload(request.Message, chatHistory)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var responseBody struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", err
	}

	if len(responseBody.Choices) > 0 {
		db.SaveChat(request.UserID, request.Message, responseBody.Choices[0].Message.Content)
		return responseBody.Choices[0].Message.Content, nil
	}

	return "", nil
}
