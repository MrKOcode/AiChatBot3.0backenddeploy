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
	"sort"

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

		moreMaterialPrompt := "This student is not ready. Provide additional educational materials for the user based on the chosen topic according to this conversation:\n" + convoText
		additionalMaterial, _ := services.GetChatGPTResponse(moreMaterialPrompt)

		saveMessageWithHistory(convoId, userId, "system", additionalMaterial)
		saveMessageWithHistory(convoId, userId, "system", "Let me know if you need to study more or perform a self-assessment.")

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
You are a teacher. Based on the conversation, generate 5 self-assessment questions for the user.
Format exactly like:
-Self assessment-
Question 1: ...
Question 2: ...
Question 3: ...
Question 4: ...
Question 5: ...
`

		assessment, _ := services.GetChatGPTResponse(questionPrompt + convoText)
		saveMessageWithHistory(convoId, userId, "system", "-ASSESSMENT_STARTED-")
		saveMessageWithHistory(convoId, userId, "system", assessment)
		saveMessageWithHistory(convoId, userId, "system", "Please answer these questions one by one.")
		c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": assessment, "role": "system"}})
		return false
	}

	return true // Not a readiness phase
}

// func AssessmentAnswering(c *gin.Context, convoId int64, userMessage string, userId string) bool {
// 	messages, _ := services.GetMessages(convoId, 50, 0)

// 	inAssessmentPhase := false
// 	answerCount := 0

// 	for i := len(messages) - 1; i >= 0; i-- {
// 		msg := messages[i]
// 		if msg.Role == "system" && strings.Contains(msg.Content, "Please answer these questions") {
// 			inAssessmentPhase = true
// 			break
// 		}
// 	}

// 	if inAssessmentPhase {
// 		for i := len(messages) - 1; i >= 0; i-- {
// 			if messages[i].Role == "system" && strings.Contains(messages[i].Content, "Please answer these questions") {
// 				break
// 			}
// 			if messages[i].Role == "user" {
// 				answerCount++
// 			}
// 		}

// 		if answerCount < 5 {
// 			saveMessageWithHistory(convoId, userId, "user", userMessage)
// 			left := 5 - answerCount - 1
// 			followup := fmt.Sprintf("Got your answer. Please answer the remaining %d question(s).", left)
// 			saveMessageWithHistory(convoId, userId, "system", followup)
// 			c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": followup, "role": "system"}})
// 			return true
// 		}

// 		if answerCount == 5 {
// 			saveMessageWithHistory(convoId, userId, "user", userMessage)

// 			// grading logic
// 			allMessages, _ := services.GetAllMessagesByUser(userId)
// 			var fullHistory string
// 			for _, m := range allMessages {
// 				fullHistory += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
// 			}

// 			gradingPrompt := `
// You are a teaching assistant chatbot. Provide feedback on the student's 5 answers.
// Title: -Self assessment result-
// Answer 1 feedback: ...
// Answer 2 feedback: ...
// Answer 3 feedback: ...
// Answer 4 feedback: ...
// Answer 5 feedback: ...
// Conclusion: ...`
// 			feedback, _ := services.GetChatGPTResponse(gradingPrompt + "\n" + fullHistory)
// 			saveMessageWithHistory(convoId, userId, "system", feedback)
// 			saveMessageWithHistory(convoId, userId, "system", "-ASSESSMENT_COMPLETED-")
// 			c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": feedback, "role": "chatbot"}})
// 			return true
// 		}
// 	}
// 	return false
// }

// func AssessmentAnswering(c *gin.Context, convoId int64, userMessage string, userId string) bool {
//     // Pull recent messages
//     messages, _ := services.GetMessages(convoId, 500, 0)

//     // Find the instruction marker that starts the answer window
//     startIdx := -1
//     for i := len(messages) - 1; i >= 0; i-- {
//         if messages[i].Role == "system" && strings.Contains(messages[i].Content, "Please answer these questions") {
//             startIdx = i
//             break
//         }
//     }
//     if startIdx == -1 {
//         // not in assessment phase
//         return false
//     }

//     // 1) Save THIS answer first so it is included in the count
//     saveMessageWithHistory(convoId, userId, "user", userMessage)

//     // 2) Re-read messages so we include the just-saved answer
//     messages, _ = services.GetMessages(convoId, 500, 0)

//     // 3) Count user answers AFTER the instruction marker
//     answerCount := 0
//     for i := len(messages) - 1; i > startIdx; i-- {
//         if messages[i].Role == "user" {
//             answerCount++
//         }
//     }

//     // 4) If fewer than 5, tell how many are left (but never say "0")
//     if answerCount < 5 {
//         left := 5 - answerCount
//         if left > 0 {
//             followup := fmt.Sprintf("Got your answer. Please answer the remaining %d question(s).", left)
//             saveMessageWithHistory(convoId, userId, "system", followup)
//             c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": followup, "role": "system"}})
//             return true
//         }
//     }

//     // 5) When we have 5 answers, gather exactly those 5 and grade
//     if answerCount >= 5 {
//         var userAnswers []string
//         for i := len(messages) - 1; i > startIdx; i-- {
//             if messages[i].Role == "user" {
//                 userAnswers = append(userAnswers, messages[i].Content)
//                 if len(userAnswers) == 5 {
//                     break
//                 }
//             }
//         }
//         // Reverse to original order: Answer 1..5
//         for i, j := 0, len(userAnswers)-1; i < j; i, j = i+1, j-1 {
//             userAnswers[i], userAnswers[j] = userAnswers[j], userAnswers[i]
//         }

//         gradingPrompt := `
// You are a teaching assistant chatbot. Provide feedback on the student's 5 answers.
// Title: -Self assessment result-
// Answer 1 feedback: ...
// Answer 2 feedback: ...
// Answer 3 feedback: ...
// Answer 4 feedback: ...
// Answer 5 feedback: ...
// Conclusion: ...`

//         feedback, _ := services.GetChatGPTResponse(gradingPrompt + "\n" + strings.Join(userAnswers, "\n"))
//         saveMessageWithHistory(convoId, userId, "system", feedback)
//         saveMessageWithHistory(convoId, userId, "system", "-ASSESSMENT_COMPLETED-")
//         c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": feedback, "role": "chatbot"}})
//         return true
//     }

//     return false
// }
func AssessmentAnswering(c *gin.Context, convoId int64, userMessage string, userId string) bool {
	// Read recent messages
	msgs, _ := services.GetMessages(convoId, 500, 0)
	if len(msgs) == 0 {
		return false
	}

	// Normalize to chronological order (oldest -> newest)
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	// Find the latest "Please answer..." marker and the assessment block text
	startIdx := -1
	assessmentText := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "system" && strings.Contains(msgs[i].Content, "-Self assessment-") && assessmentText == "" {
			assessmentText = msgs[i].Content
		}
		if msgs[i].Role == "system" && strings.Contains(msgs[i].Content, "Please answer these questions") {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		// Not in assessment phase
		return false
	}

	// Collect all user answers AFTER the marker
	answersAfterMarker := []string{}
	for i := startIdx + 1; i < len(msgs); i++ {
		if msgs[i].Role == "user" {
			answersAfterMarker = append(answersAfterMarker, msgs[i].Content)
		}
	}

	// --- Try verifier first (maps multi-line replies to Q1..Q5) ---
	var ordered []string
	if assessmentText != "" {
		userAnswersText := strings.Join(answersAfterMarker, "\n")
		if result, err := verifyAnswersWithOpenAI(assessmentText, userAnswersText); err == nil && result != nil {
			// If fewer than 5, tell how many are left (never say 0)
			if result.Count < 5 {
				left := 5 - result.Count
				if left > 0 {
					follow := fmt.Sprintf("Got your answer. Please answer the remaining %d question(s).", left)
					saveMessageWithHistory(convoId, userId, "system", follow)
					c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": follow, "role": "system"}})
					return true
				}
			}
			// Have all 5 -> build ordered list
			if len(result.Answers) == 5 {
				sort.Slice(result.Answers, func(i, j int) bool { return result.Answers[i].Q < result.Answers[j].Q })
				for _, a := range result.Answers {
					ordered = append(ordered, fmt.Sprintf("Answer %d: %s", a.Q, strings.TrimSpace(a.Text)))
				}
			}
		}
	}

	// --- Fallback if verifier failed or didn't return 5 answers ---
	if len(ordered) == 0 {
		count := len(answersAfterMarker)
		if count < 5 {
			left := 5 - count
			if left > 0 {
				follow := fmt.Sprintf("Got your answer. Please answer the remaining %d question(s).", left)
				saveMessageWithHistory(convoId, userId, "system", follow)
				c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": follow, "role": "system"}})
				return true
			}
		}
		// Build exactly the last 5 answers in Q1..Q5 order
		last5 := answersAfterMarker
		if len(last5) > 5 {
			last5 = last5[len(last5)-5:]
		}
		for i, t := range last5 {
			ordered = append(ordered, fmt.Sprintf("Answer %d: %s", i+1, t))
		}
	}

	// Grade
	gradingPrompt := `
You are a teaching assistant chatbot. Provide feedback on the student's 5 answers.
Title: -Self assessment result-
Answer 1 feedback: ...
Answer 2 feedback: ...
Answer 3 feedback: ...
Answer 4 feedback: ...
Answer 5 feedback: ...
Conclusion: ...`

	feedback, _ := services.GetChatGPTResponse(gradingPrompt + "\n" + strings.Join(ordered, "\n"))
	saveMessageWithHistory(convoId, userId, "system", feedback)
	saveMessageWithHistory(convoId, userId, "system", "-ASSESSMENT_COMPLETED-")
	c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": feedback, "role": "chatbot"}})
	return true
}


// 4. Fallback generic chatbot response
func FallbackResponse(c *gin.Context, convoId int64, userMessage string, userId string) {
	//1. Call ChatGPT API for a fallback response
	resp, _ := services.GetChatGPTResponse(userMessage)

	//2. Save the response as a chatbot reply
	saveMessageWithHistory(convoId, userId, "", resp)
	// c.JSON(http.StatusOK, gin.H{"response": gin.H{"content": resp, "role": "system"}})

	//3.Send follow-up message about assessment readiness
	followUp := "Let me know if you need to study more or perform a self-assessment."
	saveMessageWithHistory(convoId, userId, "system", followUp)
	//4.Return the response to the user
	c.JSON(http.StatusOK, gin.H{
		"response": gin.H{
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

	// Prevent new questions if assessment already started
msgs, _ := services.GetMessages(req.Message.ConversationID, 100, 0)
for _, m := range msgs {
    if m.Content == "-ASSESSMENT_STARTED-" {
        if AssessmentAnswering(c, req.Message.ConversationID, req.Message.Content, req.UserID) {
            return
        }
        return // skip readiness check once assessment has started
    }
    if m.Content == "-ASSESSMENT_COMPLETED-" {
        // After grading, fallback to normal chatbot mode
        FallbackResponse(c, req.Message.ConversationID, req.Message.Content, req.UserID)
        return
    }
}

	//Step 3: Check Readiness flow (yes/no)
	if !ReadinessCheck(c, req.Message.ConversationID, req.Message.Content, req.UserID) {
		return
	}

	//Step 4: check if user has answered the assessment questions
	if AssessmentAnswering(c, req.Message.ConversationID, req.Message.Content, req.UserID) {
		return
	}

	
	wg.Add(1)
	go func() {
		defer wg.Done()
		FallbackResponse(c, req.Message.ConversationID, req.Message.Content, req.UserID)
	}()

	wg.Wait()

}


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


// --- Verifier types
type verifyAns struct {
	Count   int `json:"count"`
	Answers []struct {
		Q     int    `json:"q"`
		Text  string `json:"text"`
	} `json:"answers"`
}

// --- Use OpenAI to verify how many questions (1..5) have answers, returning normalized answers.
func verifyAnswersWithOpenAI(assessmentText, userAnswersText string) (*verifyAns, error) {
	prompt := `
You are a strict grader. The assessment has exactly 5 questions (Q1..Q5) shown below.
The student responses (possibly multiple in one message) are also provided.
Extract answers for each question Q1..Q5 ONLY from the student's responses.

Return JSON ONLY in this exact schema (no extra commentary):

{
  "count": <number from 0 to 5>,
  "answers": [
    {"q": 1, "text": "<answer for Q1 or empty string if missing>"},
    {"q": 2, "text": "<answer for Q2 or empty string if missing>"},
    {"q": 3, "text": "<answer for Q3 or empty string if missing>"},
    {"q": 4, "text": "<answer for Q4 or empty string if missing>"},
    {"q": 5, "text": "<answer for Q5 or empty string if missing>"}
  ]
}

Rules:
- "count" is how many of Q1..Q5 have a NON-empty "text".
- Do not invent content. If an answer cannot be found, leave "text" empty.
- Output valid JSON only.

ASSESSMENT:
` + assessmentText + `

STUDENT RESPONSES:
` + userAnswersText

	raw, err := services.GetChatGPTResponse(prompt)
	if err != nil {
		return nil, err
	}

	// Some models wrap JSON in code fences; strip fences if present.
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		if i := strings.Index(raw, "{"); i >= 0 {
			raw = raw[i:]
		}
		if j := strings.LastIndex(raw, "}"); j >= 0 && j+1 <= len(raw) {
			raw = raw[:j+1]
		}
	}

	var out verifyAns
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("verifier JSON parse error: %v; raw: %s", err, raw)
	}
	return &out, nil
}
