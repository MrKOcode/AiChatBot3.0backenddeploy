package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

)

// It represents a user/AI message stored in the database
type ChatMessage struct {
	ID             int64  `json:"messageId,omitempty"`
	ConversationID int64  `json:"conversationId"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	CreatedAt      string `json:"createdAt,omitempty"`
}

type Conversation struct {
	ID        int64
	UserID    string
	Title     string
	CreatedAt string
}

// For New Conversation
func CreateConversation(userId string) (int64, error) {
	query := `INSERT INTO Conversations (user_id) VALUES (?)`
	result, err := DB.Exec(query, userId)
	if err != nil {
		return 0, fmt.Errorf("failed to create conversation: %v", err)
	}
	return result.LastInsertId()
}

// Get all conversations
func GetConversations(userId string) ([]Conversation, error) {
	query := `SELECT id, user_id, title, created_at FROM Conversations WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := DB.Query(query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt); err != nil {
			return nil, err
		}
		conversations = append(conversations, c)
	}
	return conversations, nil
}

// Save message to DB
func SaveMessage(convoId int64, role, content string) error {
	query := `INSERT INTO Messages (conversation_id, role, content, created_at) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, convoId, role, content, time.Now().Format(time.RFC3339))
	return err
}

// Get messages in conversation
func GetMessages(convoId int64, limit, offset int) ([]ChatMessage, error) {
	query := `SELECT id, conversation_id, role, content, created_at FROM Messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := DB.Query(query, convoId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

// Delete a conversation
func DeleteConversation(convoId int64) error {
	_, err := DB.Exec(`DELETE FROM Messages WHERE conversation_id = ?`, convoId)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`DELETE FROM Conversations WHERE id = ?`, convoId)
	return err
}

func GetAllMessagesByUser(userId string) ([]ChatMessage, error) {
	query := `
		SELECT M.id, M.conversation_id, M.role, M.content, M.created_at
		FROM Messages M
		JOIN Conversations C ON M.conversation_id = C.id
		WHERE C.user_id = ? AND M.created_at >= datetime('now','-1 day')
		ORDER BY M.created_at ASC`
	rows, err := DB.Query(query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func ClassifyMessageTopic(message string) (bool, error) {
	prompt := fmt.Sprintf(
		`Consider if this message relates to STEM education (Math, Physics, Chemistry, 
        Computer Science, Engineering) OR is part of educational workflows (self-assessment, 
        course navigation, learning materials). Respond ONLY 'yes' or 'no'. Message: "%s"`,
		message,
	)

	response, err := GetChatGPTResponse(prompt)
	if err != nil {
		return false, err
	}
	return strings.ToLower(strings.TrimSpace(response)) == "yes", nil
}

func IsEducationalFlow(conversationID int64) (bool, error) {
	query := `
    SELECT m1.content, m2.content
    FROM Messages m1
    LEFT JOIN Messages m2 
        ON m2.conversation_id = m1.conversation_id 
        AND m2.id < m1.id 
        AND m2.role = 'system'
    WHERE m1.conversation_id = ? 
    ORDER BY m1.created_at DESC 
    LIMIT 1
    `

	var currentMsg, prevSystemMsg string
	err := DB.QueryRow(query, conversationID).Scan(&currentMsg, &prevSystemMsg)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	// Check for educational context indicators
	isEducational := strings.Contains(prevSystemMsg, "self-assessment") ||
		strings.Contains(prevSystemMsg, "learning materials") ||
		strings.Contains(prevSystemMsg, "educational content")

	return isEducational, nil
}
