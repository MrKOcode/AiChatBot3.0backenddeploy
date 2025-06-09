package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// Structs to represent request and response payloads
type ChatGPTRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message here represents a chat message formatted for openai api
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatGPTResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

// GetChatGPTResponse interacts with the OpenAI API and retrieves the response
func GetChatGPTResponse(message string) (string, error) {
	// Load API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	log.Printf("🔑 Checking API Key: %s", func() string {
		if apiKey == "" {
			return "NOT SET"
		}
		if len(apiKey) > 20 {
			return apiKey[:20] + "..."
		}
		return apiKey + "..."
	}())
	
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set in environment variables")
	}

	// OpenAI API URL
	apiURL := "https://api.openai.com/v1/chat/completions"

	// Construct the request payload
	requestPayload := ChatGPTRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "user", Content: message},
		},
	}

	// Serialize the request payload to JSON
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		log.Printf("❌ Failed to marshal request payload: %v", err)
		return "", fmt.Errorf("failed to marshal request payload: %v", err)
	}

	log.Printf("📤 Sending request to OpenAI: %s", string(requestBody))

	// Create the HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("❌ Failed to create HTTP request: %v", err)
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("🌐 Making HTTP request to: %s", apiURL)

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Failed to make HTTP request: %v", err)
		return "", fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("📋 Response Status Code: %d", resp.StatusCode)

	// Read the response body
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response body: %v", err)
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("📥 Raw response body: %s", string(responseBody))

	// Check for API errors
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ OpenAI API returned %d: %s", resp.StatusCode, string(responseBody))
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(responseBody))
	}

	// Parse the response JSON
	var chatResponse ChatGPTResponse
	err = json.Unmarshal(responseBody, &chatResponse)
	if err != nil {
		log.Printf("❌ Failed to unmarshal response: %v", err)
		log.Printf("❌ Response body was: %s", string(responseBody))
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	log.Printf("🔍 Parsed response: %+v", chatResponse)
	log.Printf("🔍 Number of choices: %d", len(chatResponse.Choices))

	// Return the content of the first choice
	if len(chatResponse.Choices) > 0 {
		content := chatResponse.Choices[0].Message.Content
		log.Printf("✅ Extracted content: '%s'", content)
		return content, nil
	}

	log.Printf("❌ No choices found in response")
	return "", fmt.Errorf("no response received from ChatGPT API")
}
