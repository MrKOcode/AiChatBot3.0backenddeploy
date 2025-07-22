package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"AiChatBotBackend/components/AIChat/services"
)

func main() {
	// Load environment variables **only once**
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file:", err)
	}

	services.InitDB()

	router := gin.Default()

	// Register AI Chat routes
	//router.POST("/api/AIchat/conversations/:conversationId/messages", processChat)
	router.GET("/api/AIchat/history/", fetchChatHistory)

	//Register AI CHat "conversations"
	router.GET("/api/AIchat/conversations", fetchConversations)
	router.GET("/api/AIchat/conversations/:id/content/:limit/:offset", fetchConversationContent)
	router.POST("/api/AIchat/conversations", createConversation)
	router.POST("/api/AIchat/conversations/:id/messages", sendMessage)
	router.DELETE("/api/AIchat/conversations/:id", deleteConversation)

	// Get the port from environment variables
	port := os.Getenv("AI_CHAT_PORT")
	if port == "" {
		log.Println("AI_CHAT_PORT is not set, using default port 5001")
		port = "5001" // Default port
	}

	// Start the microservice
	log.Printf("AI Chat Service is running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start AI chat microservice: %s", err)
	}
}

// Process user input and get response **review**
// func processChat(c *gin.Context) {
// 	var req struct {
// 		Message string `json:"message" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
// 		return
// 	}

// 	// Call ChatGPT API to get a response
// 	response, err := services.GetChatGPTResponse(req.Message)
// 	if err != nil {
// 		log.Printf("Error fetching ChatGPT response: %v", err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch response from ChatGPT"})
// 		return
// 	}

// 	// Save chat history using SaveChatRecord function
// 	chatID, err := services.SaveChatRecord(req.Message, response)
// 	if err != nil {
// 		log.Printf("Error saving chat record: %v", err)
// 	}

// 	c.JSON(http.StatusOK, gin.H{"response": response, "chat_id": chatID})
// }

func processChat(c *gin.Context) {
	conversationId := c.Param("conversationId")
	if conversationId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	response, err := services.GetChatGPTResponse(req.Message)
	if err != nil {
		log.Printf("Error fetching ChatGPT response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch response from ChatGPT"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

// Fetch the last 5 chat records
func fetchChatHistory(c *gin.Context) {
	query := `
		SELECT userMessage, response, timestamp 
		FROM ChatHistory 
		ORDER BY timestamp DESC 
		LIMIT 5
	`

	rows, err := services.DB.Query(query)
	if err != nil {
		log.Printf("Error retrieving chat history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve chat history"})
		return
	}
	defer rows.Close()

	var history []struct {
		UserMessage string `json:"userMessage"`
		Response    string `json:"response"`
		Timestamp   string `json:"timestamp"`
	}

	for rows.Next() {
		var entry struct {
			UserMessage string `json:"userMessage"`
			Response    string `json:"response"`
			Timestamp   string `json:"timestamp"`
		}
		if err := rows.Scan(&entry.UserMessage, &entry.Response, &entry.Timestamp); err != nil {
			log.Printf("Error scanning chat history row: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process chat history"})
			return
		}
		history = append(history, entry)
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

func createConversation(c *gin.Context) {
	var req struct {
		UserID string `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing userId"})
		return
	}

	// 1. Create conversation
	id, err := services.CreateConversation(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	//2. Greet the user
	greeting := "This is your personal AiChatBot, what can I help you study today?"
	saveMessageWithHistory(id, req.UserID, "chatbot", greeting)

	c.JSON(http.StatusOK, gin.H{
		"conversationId": id,
		"conversation":   gin.H{"title": "New Academic Chat"},
	})

}

func fetchConversations(c *gin.Context) {
	userId := c.Query("userId")
	if userId == "" {
		userId = "1" // default
	}
	convos, err := services.GetConversations(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if convos == nil {
		convos = []services.Conversation{} // ensure it's an empty array, not null
	}
	c.JSON(http.StatusOK, gin.H{"content": gin.H{"data": convos}})
}

func OffTopic(c *gin.Context, convoId int64, userId string) {
	redirectMsg := "This topic is not allowed. Please choose a different topic for your study session."
	saveMessageWithHistory(convoId, userId, "system", redirectMsg)
	c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": redirectMsg, "role": "system"}})
}

func ReadinessCheck(c *gin.Context, convoId int64, userMessage string, userId string) bool {
	intentPrompt := fmt.Sprintf(`User said: "%s" Does this indicate readiness for a self-assessment? Reply only 'yes' or 'no'.`, userMessage)
	intentResult, err := services.GetChatGPTResponse(intentPrompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return false
	}

	intent := strings.ToLower(strings.TrimSpace(intentResult))

	if intent == "no" {
		// Student not ready → Provide more material
		messages, _ := services.GetMessages(convoId, 100, 0)
		var convoText string
		for i := len(messages) - 1; i >= 0; i-- {
			convoText += fmt.Sprintf("%s: %s\n", messages[i].Role, messages[i].Content)
		}

		moreMaterialPrompt := "This student is not ready. Provide additional educational material about inertia based on this conversation:\n" + convoText
		additionalMaterial, _ := services.GetChatGPTResponse(moreMaterialPrompt)

		saveMessageWithHistory(convoId, userId, "system", additionalMaterial)
		saveMessageWithHistory(convoId, userId, "system", "Are you ready to do a self-assessment?")
		c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": additionalMaterial, "role": "system"}})
		return false
	}

	if intent == "yes" {
		// Student ready → Generate questions
		messages, _ := services.GetMessages(convoId, 100, 0)
		var convoText string
		for i := len(messages) - 1; i >= 0; i-- {
			convoText += fmt.Sprintf("%s: %s\n", messages[i].Role, messages[i].Content)
		}

		questionPrompt := `
You are a physics tutor. Based on the conversation about Newton's First Law and inertia, generate 5 self-assessment questions.
Format exactly like:
-Self assessment-
Question 1: ...
Question 2: ...
Question 3: ...
Question 4: ...
Question 5: ...
`

		assessment, _ := services.GetChatGPTResponse(questionPrompt + convoText)
		saveMessageWithHistory(convoId, userId, "system", assessment)
		saveMessageWithHistory(convoId, userId, "system", "Please answer these questions one by one.")
		c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": assessment, "role": "system"}})
		return false
	}

	return true // Not a readiness phase
}

func AssessmentAnswering(c *gin.Context, convoId int64, userMessage string, userId string) bool {
	messages, _ := services.GetMessages(convoId, 50, 0)

	inAssessmentPhase := false
	answerCount := 0

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == "system" && strings.Contains(msg.Content, "Please answer these questions") {
			inAssessmentPhase = true
			break
		}
	}

	if inAssessmentPhase {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "system" && strings.Contains(messages[i].Content, "Please answer these questions") {
				break
			}
			if messages[i].Role == "user" {
				answerCount++
			}
		}

		if answerCount < 5 {
			saveMessageWithHistory(convoId, userId, "user", userMessage)
			left := 5 - answerCount - 1
			followup := fmt.Sprintf("Got your answer. Please answer the remaining %d question(s).", left)
			saveMessageWithHistory(convoId, userId, "system", followup)
			c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": followup, "role": "system"}})
			return true
		}

		if answerCount == 5 {
			saveMessageWithHistory(convoId, userId, "user", userMessage)

			// grading logic
			allMessages, _ := services.GetAllMessagesByUser(userId)
			var fullHistory string
			for _, m := range allMessages {
				fullHistory += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
			}

			gradingPrompt := `
You are a teaching assistant chatbot. Provide feedback on the student's 5 answers.
Title: -Self assessment result-
Answer 1 feedback: ...
Answer 2 feedback: ...
Answer 3 feedback: ...
Answer 4 feedback: ...
Answer 5 feedback: ...
Conclusion: ...
Here is the full conversation:
`
			fullPrompt := gradingPrompt + fullHistory
			feedback, _ := services.GetChatGPTResponse(fullPrompt)

			saveMessageWithHistory(convoId, userId, "system", feedback)
			c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": feedback, "role": "chatbot"}})
			return true
		}
	}
	return false
}

// 4. Fallback generic chatbot response
func FallbackResponse(c *gin.Context, convoId int64, userMessage string, userId string) {
	//1. Call ChatGPT API for a fallback response
	resp, _ := services.GetChatGPTResponse(userMessage)

	//2. Save the response as a chatbot reply
	saveMessageWithHistory(convoId, userId, "", resp)
	c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": resp, "role": "system"}})

	//3.Send follow-up message about assessment readiness
	followUp := "Advise me if you finish reading and ready to do a self-assessment to test your understanding of the knowledge."
	saveMessageWithHistory(convoId, userId, "system", followUp)
//4.Return the response to the user
	c.JSON(http.StatusOK, gin.H{"response": gin.H{
		"content": resp,
		"role":    "chatbot",
	}})
}

func sendMessage(c *gin.Context) {

	var req struct {
		UserID  string               `json:"userId"`
		Message services.ChatMessage `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message format"})
		return
	}

	var wg sync.WaitGroup
	var topicErr, saveUserErr error
	isOnTopic := true

	//step 1:
	// Check if topics are alloweed
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		isOnTopic, err = services.ClassifyMessageTopic(req.Message.Content)
		if err != nil {
			topicErr = err
			isOnTopic = false
		}
	}()
	// Before Goroutine
	// isOnTopic, err := services.ClassifyMessageTopic(req.Message.Content)
	// if err != nil || !isOnTopic {
	// 	OffTopic(c, req.Message.ConversationID)
	// 	return
	// }

	//Step 2, Save user message
	wg.Add(1)
	go func() {
		defer wg.Done()
		saveUserErr = services.SaveMessage(req.Message.ConversationID, "user", req.Message.Content)
		// 同时保存用户消息到ChatHistory
		if saveUserErr == nil {
			if userIdInt, err := strconv.Atoi(req.UserID); err == nil {
				saveToChatHistory(userIdInt, strconv.FormatInt(req.Message.ConversationID, 10), "user", req.Message.Content)
			}
		}
	}()
	wg.Wait()

	if topicErr != nil || !isOnTopic {
		OffTopic(c, req.Message.ConversationID, req.UserID)
		return
	}
	if saveUserErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": saveUserErr.Error()})
		return
	}

	//before Goroutine
	// if err := services.SaveMessage(req.Message.ConversationID, "user", req.Message.Content); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }

	//Step 3: Check Readiness flow (yes/no)
	if !ReadinessCheck(c, req.Message.ConversationID, req.Message.Content, req.UserID) {
		return
	}

	//Step 4: check if user has answered the assessment questions
	if AssessmentAnswering(c, req.Message.ConversationID, req.Message.Content, req.UserID) {
		return
	}

	// If this is the 5th answer → proceed to grading (Step 4)
	// **proved working**if answerCount == 5 {
	// 	gradingPrompt := `You are a teaching chatbot. Provide feedback on each of the 5 answers submitted by the student. Format your feedback like this:

	// Title: -Self assessment result-

	// Question 1 feedback: ...
	// Question 2 feedback: ...
	// Question 3 feedback: ...
	// Question 4 feedback: ...
	// Question 5 feedback: ...

	// Conclusion: Summarize the student's understanding and performance based on the full conversation. Provide a clear recommendation on whether the student has mastered the concept of inertia based on their answers and the prior conversation.

	// Here is the full conversation and answers:\n\n`

	// 	var convo string
	// 	for i := len(messages) - 1; i >= 0; i-- {
	// 		convo += fmt.Sprintf("%s: %s\n", messages[i].Role, messages[i].Content)
	// 	}
	// 	fullPrompt := gradingPrompt + "\n\n" + convo

	// 	feedback, err := services.GetChatGPTResponse(fullPrompt)
	// 	if err != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get feedback"})
	// 		return
	// 	}

	// 	_ = services.SaveMessage(req.Message.ConversationID, "system", feedback)
	// 	c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": feedback, "role": "system"}})
	// 	return
	// }

	//Step 5:Fallback normal charbot response
	wg.Add(1)
	go func() {
		defer wg.Done()
		FallbackResponse(c, req.Message.ConversationID, req.Message.Content, req.UserID)
	}()

	wg.Wait()

}

// **Call ChatGPT API**Deeleted for self-assessment implementation
// resp, err := services.GetChatGPTResponse(req.Message.Content)
// if err != nil {
// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 	return
// }

// // Save chatbot reply
// if err := services.SaveMessage(req.Message.ConversationID, "chatbot", resp); err != nil {
// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 	return
// }

// c.JSON(http.StatusOK, gin.H{
// 	"response": gin.H{"content": resp, "role": "chatbot"},
// })

func fetchConversationContent(c *gin.Context) {
	convoID := c.Param("id")
	limit := 50
	offset := 0
	messages, err := services.GetMessages(atoi(convoID), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": gin.H{"content": messages}})
}

func deleteConversation(c *gin.Context) {
	convoID := c.Param("id")
	if err := services.DeleteConversation(atoi(convoID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversationId": convoID})
}

// // Helper function to convert string to int64
func atoi(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// Helper function to save message and chat history
func saveMessageWithHistory(convoId int64, userId string, role string, content string) {

	_ = services.SaveMessage(convoId, role, content)

	if userIdInt, err := strconv.Atoi(userId); err == nil {

		messageType := "ai"
		if role == "user" {
			messageType = "user"
		}
		saveToChatHistory(userIdInt, strconv.FormatInt(convoId, 10), messageType, content)
	}
}

func saveToChatHistory(userId int, conversationId string, messageType string, content string) {
	// Fetch username from Auth service

	username := getUsernameFromAuth(userId)

	requestData := map[string]interface{}{
		"userId":         userId,
		"username":       username,
		"conversationId": conversationId,
		"messageType":    messageType,
		"content":        content,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		log.Printf("Error marshaling chat history data: %v", err)
		return
	}

	resp, err := http.Post("http://localhost:5004/api/chat-history/save", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("Error saving to chat history: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Printf("Failed to save chat history, status code: %d", resp.StatusCode)
	} else {
		log.Printf("✅ Chat history saved: userId=%d, username=%s, %s -> %s", userId, username, messageType, func() string {
			if len(content) > 50 {
				return content
			}
			return content
		}())
	}
}

// Helper function to get username from Auth service
func getUsernameFromAuth(userId int) string {
	resp, err := http.Get(fmt.Sprintf("http://localhost:5002/api/auth/user/%d", userId))
	if err != nil {
		log.Printf("Error fetching username from auth service: %v", err)
		return fmt.Sprintf("user_%d", userId)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Auth service returned status %d for user %d", resp.StatusCode, userId)
		return fmt.Sprintf("user_%d", userId)
	}

	var authResponse struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		log.Printf("Error decoding auth response: %v", err)
		return fmt.Sprintf("user_%d", userId)
	}

	return authResponse.Username
}
