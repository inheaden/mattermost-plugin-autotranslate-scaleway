package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const translationResponseSchemaName = "translation_result"

type scalewayChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type scalewayJSONSchema struct {
	Name   string                 `json:"name"`
	Schema map[string]interface{} `json:"schema"`
}

type scalewayResponseFormat struct {
	Type       string              `json:"type"`
	JSONSchema *scalewayJSONSchema `json:"json_schema,omitempty"`
}

type scalewayChatCompletionRequest struct {
	Model          string                 `json:"model"`
	Messages       []scalewayChatMessage  `json:"messages"`
	Temperature    float64                `json:"temperature,omitempty"`
	MaxTokens      int                    `json:"max_tokens,omitempty"`
	ResponseFormat scalewayResponseFormat `json:"response_format"`
}

type scalewayChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type scalewayTranslationPayload struct {
	TranslatedText         string `json:"translated_text"`
	DetectedSourceLanguage string `json:"detected_source_language,omitempty"`
}

func buildTranslationPrompt(text, sourceCode, sourceLanguage, targetLanguage string) string {
	if sourceCode == autoLanguage {
		return fmt.Sprintf("Translate the following text to %s. Detect the source language automatically and preserve formatting.\n\n%s", targetLanguage, text)
	}

	return fmt.Sprintf("Translate the following text from %s to %s. Preserve formatting.\n\n%s", sourceLanguage, targetLanguage, text)
}

func buildScalewayTranslationRequest(configuration *configuration, text, sourceCode, targetCode string) scalewayChatCompletionRequest {
	sourceLanguage := languageCodes[sourceCode]
	targetLanguage := languageCodes[targetCode]

	if sourceLanguage == "" {
		sourceLanguage = sourceCode
	}
	if targetLanguage == "" {
		targetLanguage = targetCode
	}

	request := scalewayChatCompletionRequest{
		Model: configuration.getScalewayModel(),
		Messages: []scalewayChatMessage{
			{
				Role:    "system",
				Content: configuration.getScalewaySystemPrompt(),
			},
			{
				Role:    "user",
				Content: buildTranslationPrompt(text, sourceCode, sourceLanguage, targetLanguage),
			},
		},
		Temperature: configuration.getScalewayTemperature(),
		ResponseFormat: scalewayResponseFormat{
			Type: "json_schema",
			JSONSchema: &scalewayJSONSchema{
				Name: translationResponseSchemaName,
				Schema: map[string]interface{}{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]interface{}{
						"translated_text": map[string]string{
							"type": "string",
						},
						"detected_source_language": map[string]string{
							"type": "string",
						},
					},
					"required": []string{"translated_text"},
				},
			},
		},
	}

	if configuration.ScalewayMaxTokens > 0 {
		request.MaxTokens = configuration.ScalewayMaxTokens
	}

	return request
}

func parseScalewayTranslationResponse(response scalewayChatCompletionResponse) (*scalewayTranslationPayload, error) {
	if len(response.Choices) == 0 {
		if response.Error != nil && response.Error.Message != "" {
			return nil, fmt.Errorf(response.Error.Message)
		}

		return nil, fmt.Errorf("Scaleway returned no choices")
	}

	content := response.Choices[0].Message.Content
	if content == "" {
		return nil, fmt.Errorf("Scaleway returned an empty message")
	}

	var payload scalewayTranslationPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, fmt.Errorf("failed to parse Scaleway translation payload: %w", err)
	}

	if strings.TrimSpace(payload.TranslatedText) == "" {
		return nil, fmt.Errorf("Scaleway returned an empty translated_text field")
	}

	return &payload, nil
}

func (p *Plugin) translateWithScaleway(text, sourceCode, targetCode string) (*scalewayTranslationPayload, error) {
	configuration := p.getConfiguration()
	requestPayload := buildScalewayTranslationRequest(configuration, text, sourceCode, targetCode)

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize Scaleway request: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, configuration.getScalewayBaseURL()+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create Scaleway request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+configuration.ScalewaySecretKey)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call Scaleway: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Scaleway response: %w", err)
	}

	var completionResponse scalewayChatCompletionResponse
	if err := json.Unmarshal(body, &completionResponse); err != nil {
		return nil, fmt.Errorf("failed to decode Scaleway response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		if completionResponse.Error != nil && completionResponse.Error.Message != "" {
			return nil, fmt.Errorf("Scaleway request failed: %s", completionResponse.Error.Message)
		}

		return nil, fmt.Errorf("Scaleway request failed with status %d", response.StatusCode)
	}

	return parseScalewayTranslationResponse(completionResponse)
}
