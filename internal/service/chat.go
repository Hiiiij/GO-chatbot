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

func ProcessStream(request models.ChatRequest) (<-chan string, error) {
	if request.UserID == "" {
		request.UserID = util.GenerateUserID()
		log.Debug().Msgf("Generated UserID: %s", request.UserID)
	}

	chatHistory, err := db.GetChatHistory(request.UserID)
	if err != nil {
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
		defer resp.Body.Close()
		defer close(streamChannel)

		scanner := bufio.NewScanner(resp.Body)
		var aggregatedResponse string

		for scanner.Scan() {
			line := scanner.Text()
			log.Debug().Str("raw_line", line).Msg("Raw streamed data from OpenAI")

			// skip lines that don't start with "data:"
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
				log.Error().Err(err).Msg("Failed to decode stream data")
				continue
			}

			// each choice in the response
			for _, choice := range streamBody.Choices {
				content := choice.Delta.Content
				if content != "" {
					log.Debug().Str("chunk_content", content).Msg("Received chunk content")
					aggregatedResponse += content
					streamChannel <- content
				}
			}
		}

		log.Debug().Str("aggregated_response", aggregatedResponse).Msg("Final aggregated response")
	}()

	return streamChannel, nil
}

func ProcessChat(request models.ChatRequest) (string, error) {
	if request.UserID == "" {
		request.UserID = util.GenerateUserID()
		log.Debug().Msgf("Generated UserID: %s", request.UserID)
	}

	chatHistory, err := db.GetChatHistory(request.UserID)
	if err != nil {
		return "", err
	}

	payload := BuildChatGPTPayload(request.Message, chatHistory, "You are a helpful assistant.")
	log.Debug().Interface("payload", payload).Msg("Built payload for chat request")

	resp, err := util.SendOpenAIRequest(payload, false)
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
		log.Error().Err(err).Msg("Failed to decode OpenAI API response")
		return "", err
	}

	if len(responseBody.Choices) > 0 {
		db.SaveChat(request.UserID, request.Message, responseBody.Choices[0].Message.Content)
		return responseBody.Choices[0].Message.Content, nil
	}

	return "", nil
}
