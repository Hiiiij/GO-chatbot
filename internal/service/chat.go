package service

import (
	"bufio"
	"encoding/json"
	"go-bot/internal/db"
	"go-bot/internal/models"
	"go-bot/internal/util"

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
		// generate a new userid if not provided
		request.UserID = util.GenerateUserID()
		log.Debug().Msgf("Generated UserID: %s", request.UserID)
	}

	// fetch the last 3 messages for context
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
			// close response body and channel to avoid leaks
			resp.Body.Close()
			close(streamChannel)
		}()

		scanner := bufio.NewScanner(resp.Body)
		var aggregatedResponse string

		for scanner.Scan() {
			line := scanner.Text()

			// skip lines not starting with "data:"
			if len(line) < 6 || line[:5] != "data:" {
				continue
			}

			jsonData := line[5:]
			if jsonData == "[DONE]" {
				log.Debug().Msg("Stream completed")
				break
			}

			var streamBody ChatGPTStreamBody
			if err := json.Unmarshal([]byte(jsonData), &streamBody); err != nil {
				log.Error().Err(err).Str("json_data", jsonData).Msg("Failed to decode stream data")
				continue
			}

			// send each chunk to the channel
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
		// generate a new userid if not provided
		request.UserID = util.GenerateUserID()
		log.Debug().Msgf("Generated UserID: %s", request.UserID)
	}

	// fetch the last 3 messages for context
	chatHistory, err := db.GetChatHistory(request.UserID, 3)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch chat history")
		return "", err
	}

	payload := BuildChatGPTPayload(request.Message, chatHistory, "You are a helpful assistant.")
	log.Debug().Interface("payload", payload).Msg("Built payload for chat request")

	resp, err := util.SendOpenAIRequest(payload, false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send chat request to OpenAI")
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
		log.Error().Err(err).Msg("Failed to decode OpenAI API response")
		return "", err
	}

	if len(responseBody.Choices) > 0 {
		response := responseBody.Choices[0].Message.Content
		if saveErr := db.SaveChat(request.UserID, request.Message, response); saveErr != nil {
			log.Error().Err(saveErr).Msg("Failed to save chat to database")
		}
		return response, nil
	}

	log.Warn().Msg("no choices found in OpenAI API response")
	return "", nil
}
