package service

import (
	"bufio"
	"encoding/json"
	"go-bot/internal/db"
	"go-bot/internal/models"
	"go-bot/internal/util"
	"strings"

	"github.com/rs/zerolog/log"
)

type ChatGPTStreamBody struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// handle streaming requests from OpenAI API
func ProcessStream(request models.ChatRequest) (<-chan string, error) {
	if request.UserID == "" {
		request.UserID = util.GenerateUserID()
		log.Debug().Msgf("Generated UserID: %s", request.UserID)
	}

	chatHistory, err := db.GetChatHistory(request.UserID, 3)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch chat history")
		return nil, err
	}

	payload := BuildChatGPTPayload(request.Message, chatHistory, "You are a helpful assistant.")
	payload.Stream = true
	log.Debug().Interface("payload", payload).Msg("Payload for streaming request")

	resp, err := util.SendOpenAIRequest(payload, true)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send streaming request to OpenAI")
		return nil, err
	}

	streamChannel := make(chan string)
	go func() {
		defer func() {
			resp.Body.Close()
			close(streamChannel)
		}()

		scanner := bufio.NewScanner(resp.Body)
		var aggregatedResponse string

		for scanner.Scan() {
			line := scanner.Text()

			if len(line) == 0 {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			// remove "data: " prefix and trim spaces
			data := strings.TrimSpace(line[6:])

			// check for stream end
			if data == "[DONE]" {
				log.Debug().Msg("Stream completed")
				break
			}

			var streamBody ChatGPTStreamBody
			if err := json.Unmarshal([]byte(data), &streamBody); err != nil {
				log.Error().Err(err).Str("data", data).Msg("Failed to decode stream data")
				continue
			}

			// process content chunks
			for _, choice := range streamBody.Choices {
				content := choice.Delta.Content
				if content != "" {
					aggregatedResponse += content
					streamChannel <- content
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Error().Err(err).Msg("Error reading streamed data")
		}

		log.Debug().Str("aggregated_response", aggregatedResponse).Msg("Final aggregated response")

		if saveErr := db.SaveChat(request.UserID, request.Message, aggregatedResponse); saveErr != nil {
			log.Error().Err(saveErr).Msg("Failed to save chat to database")
		}
	}()

	return streamChannel, nil
}

// handle non-streaming chat requests
func ProcessChat(request models.ChatRequest) (string, error) {
	if request.UserID == "" {
		request.UserID = util.GenerateUserID()
		log.Debug().Msgf("Generated UserID: %s", request.UserID)
	}

	chatHistory, err := db.GetChatHistory(request.UserID, 3)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch chat history")
		return "", err
	}

	payload := BuildChatGPTPayload(request.Message, chatHistory, "You are a helpful assistant.")
	response, err := CallOpenAI(payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get response from OpenAI")
		return "", err
	}

	if saveErr := db.SaveChat(request.UserID, request.Message, response); saveErr != nil {
		log.Error().Err(saveErr).Msg("Failed to save chat to database")
	}

	return response, nil
}
