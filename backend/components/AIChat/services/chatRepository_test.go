package services

import (
	"testing"
)

// These tests will:

// Create a new conversation

// Save a message

// Query messages

// Ensure basic DB logic is working
func TestSaveChatRecord(t *testing.T) {
	InitDB()
	userMsg := "I am testing you"
	botResp := "Fine"

	id, err := SaveChatRecord(userMsg, botResp)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected valid ID, got %v", id)
	}
}

func TestCreateConversation(t *testing.T) {
	InitDB()
	userId := "test-user"

	id, err := CreateConversation(userId)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected valid conversation ID, got %v", id)
	}
}

func TestSaveAndGetMessages(t *testing.T) {
	InitDB()
	userId := "test-user"
	convoID, _ := CreateConversation(userId)

	err := SaveMessage(convoID, "user", "Hello test!")
	if err != nil {
		t.Errorf("Failed to save message: %v", err)
	}

	msgs, err := GetMessages(convoID, 10, 0)
	if err != nil {
		t.Errorf("Failed to retrieve messages: %v", err)
	}

	if len(msgs) == 0 {
		t.Errorf("Expected messages, got 0")
	}
}
