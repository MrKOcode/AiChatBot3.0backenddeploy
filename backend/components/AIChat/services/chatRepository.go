package services
import (
	"fmt"
	"time"
)

// SaveChatRecord saves a chat record into the ChatHistory table
func SaveChatRecord(userMessage, botResponse string) (int64, error) {
	query := `
	INSERT INTO ChatHistory (userMessage, response, timestamp)
	VALUES (?, ?, ?);
	`
	//Get the current timestamp
	currentTime := time.Now().Format(time.RFC3339)

	// Execute the insert query
	result, err := DB.Exec(query, userMessage, botResponse, currentTime)
	if err != nil {
		return 0, fmt.Errorf("error saving chat record: %v", err)
	}

	// Retrieve the last inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error retrieving last insert ID: %v", err)
	}

	fmt.Printf("Chat record saved with ID: %d\n", id)
	return id, nil
}


