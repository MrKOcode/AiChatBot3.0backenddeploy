package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ChatMessage struct {
	ID            int       `json:"id"`
	UserID        int       `json:"userId"`
	Username      string    `json:"username"`
	ConversationID string   `json:"conversationId"`
	MessageType   string    `json:"messageType"`
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"createdAt"`
}

type ChatHistoryQuery struct {
	UserID    int    `json:"userId"`
	Keyword   string `json:"keyword"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Page      int    `json:"page"`
	PageSize  int    `json:"pageSize"`
}

type ChatHistoryResponse struct {
	Messages    []ChatMessage `json:"messages"`
	Total       int          `json:"total"`
	Page        int          `json:"page"`
	PageSize    int          `json:"pageSize"`
	TotalPages  int          `json:"totalPages"`
}

type UserChatSummary struct {
	UserID       int       `json:"userId"`
	Username     string    `json:"username"`
	MessageCount int       `json:"messageCount"`
	LastChatTime time.Time `json:"lastChatTime"`
}

var chatDB *sql.DB

func InitChatHistoryDB() {
	var err error
	
	dbPath := os.Getenv("CHAT_HISTORY_DB_PATH")
	if dbPath == "" {
		dbPath = "./chat_history.db"
	}

	chatDB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to chat history database: %v", err)
	}

	if err = chatDB.Ping(); err != nil {
		log.Fatalf("Failed to ping chat history database: %v", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS chat_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		conversation_id TEXT NOT NULL,
		message_type TEXT NOT NULL CHECK (message_type IN ('user', 'ai')),
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_chat_user_id ON chat_messages(user_id);
	CREATE INDEX IF NOT EXISTS idx_chat_conversation_id ON chat_messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_chat_created_at ON chat_messages(created_at);
	`

	_, err = chatDB.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create chat_messages table: %v", err)
	}

	log.Println("✅ Chat History Database initialized successfully")
}

func SaveChatMessage(userID int, username, conversationID, messageType, content string) error {
	query := `INSERT INTO chat_messages (user_id, username, conversation_id, message_type, content) VALUES (?, ?, ?, ?, ?)`
	_, err := chatDB.Exec(query, userID, username, conversationID, messageType, content)
	if err != nil {
		log.Printf("Error saving chat message: %v", err)
		return err
	}
	log.Printf("💬 Chat message saved: User %s, Type %s", username, messageType)
	return nil
}

func GetUserChatHistory(query ChatHistoryQuery) (*ChatHistoryResponse, error) {
	whereClause := "WHERE user_id = ?"
	args := []interface{}{query.UserID}
	
	if query.Keyword != "" {
		whereClause += " AND content LIKE ?"
		args = append(args, "%"+query.Keyword+"%")
	}
	
	if query.StartDate != "" {
		whereClause += " AND created_at >= ?"
		args = append(args, query.StartDate)
	}
	
	if query.EndDate != "" {
		whereClause += " AND created_at <= ?"
		args = append(args, query.EndDate)
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM chat_messages " + whereClause
	err := chatDB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	offset := (query.Page - 1) * query.PageSize
	
	messagesQuery := fmt.Sprintf(`SELECT id, user_id, username, conversation_id, message_type, content, created_at FROM chat_messages %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, whereClause)
	args = append(args, query.PageSize, offset)
	rows, err := chatDB.Query(messagesQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.Username, &msg.ConversationID, &msg.MessageType, &msg.Content, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize

	return &ChatHistoryResponse{
		Messages:   messages,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}, nil
}

func GetAllUsersChatSummary() ([]UserChatSummary, error) {
	query := `SELECT user_id, username, COUNT(*) as message_count, MAX(created_at) as last_chat_time FROM chat_messages GROUP BY user_id, username ORDER BY last_chat_time DESC`
	rows, err := chatDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []UserChatSummary
	for rows.Next() {
		var summary UserChatSummary
		var lastChatTimeStr string
		err := rows.Scan(&summary.UserID, &summary.Username, &summary.MessageCount, &lastChatTimeStr)
		if err != nil {
			return nil, err
		}
		
		// 解析时间字符串为 time.Time
		if lastChatTimeStr != "" {
			parsedTime, err := time.Parse("2006-01-02 15:04:05", lastChatTimeStr)
			if err != nil {
				log.Printf("Error parsing time %s: %v", lastChatTimeStr, err)
				summary.LastChatTime = time.Time{} // 设置为零值
			} else {
				summary.LastChatTime = parsedTime
			}
		} else {
			summary.LastChatTime = time.Time{} // 设置为零值
		}
		
		summaries = append(summaries, summary)
	}

	return summaries, nil
}
