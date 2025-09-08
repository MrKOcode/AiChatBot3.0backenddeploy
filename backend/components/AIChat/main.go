// ✅ AIchat.go - Lambda-Ready Version (Phase 1)

package main

import (
	
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"

	"github.com/MrKOcode/AiChatBot3.0backenddeploy/backend/components/AIChat/services"


)

func main() {
	_ = godotenv.Load(".env")
	if err := services.InitDAL(); err != nil {
	return errorResponse(500, "DAL init failed: "+err.Error()), nil
}
	lambda.Start(handler)
}

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "GET":
		if req.Path == "/api/AIchat/conversations" {
			return lambdaFetchConversations(req)
		}
		if req.Path == "/api/AIchat/history/" {
			return lambdaFetchChatHistory(req)
		}
	case "POST":
		if req.Path == "/api/AIchat/conversations" {
			return lambdaCreateConversation(req)
		}
		if strings.Contains(req.Path, "/messages") {
			return lambdaSendMessage(req)
		}
	case "DELETE":
		if strings.Contains(req.Path, "/conversations/") {
			return lambdaDeleteConversation(req)
		}
	}
	return errorResponse(404, "Route not found"), nil
}

// ========== Handlers ==========

func lambdaCreateConversation(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var body struct {
		UserID string `json:"userId"`
	}
	_ = json.Unmarshal([]byte(req.Body), &body)

	id, err := services.CreateConversation(body.UserID)
	if err != nil {
		return errorResponse(500, err.Error()), nil
	}
	greeting := "This is your personal AiChatBot, what can I help you study today?"
	saveMessageWithHistory(id, body.UserID, "chatbot", greeting)

	return jsonResponse(200, map[string]interface{}{
		"conversationId": id,
		"conversation": map[string]string{"title": "New Academic Chat"},
	}), nil
}

func lambdaFetchConversations(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userId := req.QueryStringParameters["userId"]
	if userId == "" {
		userId = "1" // default fallback
	}
	convos, err := services.GetConversations(userId)
	if err != nil {
		return errorResponse(500, err.Error()), nil
	}
	return jsonResponse(200, map[string]interface{}{"content": map[string]interface{}{"data": convos}}), nil
}

func lambdaSendMessage(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var body struct {
		UserID  string               `json:"userId"`
		Message services.ChatMessage `json:"message"`
	}
	_ = json.Unmarshal([]byte(req.Body), &body)

	err := services.SaveMessage(body.Message.ConversationID, "user", body.Message.Content)
	if err != nil {
		return errorResponse(500, "Failed to save message"), nil
	}
	saveMessageWithHistory(body.Message.ConversationID, body.UserID, "user", body.Message.Content)

	// Basic fallback response (simulating GPT)
	resp := fmt.Sprintf("You said: %s", body.Message.Content)
	saveMessageWithHistory(body.Message.ConversationID, body.UserID, "chatbot", resp)

	return jsonResponse(200, map[string]string{"response": resp}), nil
}

func lambdaFetchChatHistory(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	rows, err := services.DB.Query("SELECT userMessage, response, timestamp FROM ChatHistory ORDER BY timestamp DESC LIMIT 5")
	if err != nil {
		return errorResponse(500, "Failed to fetch chat history"), nil
	}
	defer rows.Close()

	var history []map[string]string
	for rows.Next() {
		var u, r, t string
		_ = rows.Scan(&u, &r, &t)
		history = append(history, map[string]string{"userMessage": u, "response": r, "timestamp": t})
	}
	return jsonResponse(200, map[string]interface{}{"history": history}), nil
}

func lambdaDeleteConversation(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := strings.TrimPrefix(req.Path, "/api/AIchat/conversations/")
	cid, _ := strconv.ParseInt(id, 10, 64)
	if err := services.DeleteConversation(cid); err != nil {
		return errorResponse(500, err.Error()), nil
	}
	return jsonResponse(200, map[string]string{"conversationId": id}), nil
}

// ========== Helpers ==========

func saveMessageWithHistory(convoId int64, userId, role, content string) {
	_ = services.SaveMessage(convoId, role, content)
}

func jsonResponse(status int, data interface{}) events.APIGatewayProxyResponse {
	body, _ := json.Marshal(data)
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}
}

func errorResponse(status int, msg string) events.APIGatewayProxyResponse {
	return jsonResponse(status, map[string]string{"error": msg})
}
