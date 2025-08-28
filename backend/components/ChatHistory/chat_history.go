// ✅ chat_history.go - Lambda-Compatible Version (Phase 1)

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"

	"AiChatBotBackend/components/ChatHistory/services"
)

func main() {
	_ = godotenv.Load(".env")
	services.InitChatHistoryDB()
	lambda.Start(handler)
}

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "POST":
		if req.Path == "/api/chat-history/save" {
			return lambdaSaveChatMessage(req)
		}
	case "GET":
		if len(req.PathParameters["userId"]) > 0 {
			return lambdaGetUserChatHistory(req)
		}
		if req.Path == "/api/chat-history/admin/users" {
			return lambdaGetAllUsersChatSummary(req)
		}
	}
	return errorResponse(404, "Route not found"), nil
}

func lambdaSaveChatMessage(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var body struct {
		UserID         int    `json:"userId"`
		Username       string `json:"username"`
		ConversationID string `json:"conversationId"`
		MessageType    string `json:"messageType"`
		Content        string `json:"content"`
	}
	_ = json.Unmarshal([]byte(req.Body), &body)

	if body.MessageType != "user" && body.MessageType != "ai" {
		return errorResponse(400, "Invalid message type"), nil
	}

	err := services.SaveChatMessage(body.UserID, body.Username, body.ConversationID, body.MessageType, body.Content)
	if err != nil {
		log.Printf("Error saving chat message: %v", err)
		return errorResponse(500, "Failed to save chat message"), nil
	}

	return jsonResponse(201, map[string]string{"message": "Chat message saved successfully"}), nil
}

func lambdaGetUserChatHistory(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userIDStr := req.PathParameters["userId"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return errorResponse(400, "Invalid user ID"), nil
	}

	page, _ := strconv.Atoi(req.QueryStringParameters["page"])
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(req.QueryStringParameters["pageSize"])
	if pageSize <= 0 {
		pageSize = 20
	}

	query := services.ChatHistoryQuery{
		UserID:    userID,
		Keyword:   req.QueryStringParameters["keyword"],
		StartDate: req.QueryStringParameters["startDate"],
		EndDate:   req.QueryStringParameters["endDate"],
		Page:      page,
		PageSize:  pageSize,
	}

	result, err := services.GetUserChatHistory(query)
	if err != nil {
		log.Printf("Error getting user chat history: %v", err)
		return errorResponse(500, "Failed to get chat history"), nil
	}

	return jsonResponse(200, result), nil
}

func lambdaGetAllUsersChatSummary(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	summaries, err := services.GetAllUsersChatSummary()
	if err != nil {
		log.Printf("Error getting users chat summary: %v", err)
		return errorResponse(500, "Failed to get users chat summary"), nil
	}
	return jsonResponse(200, map[string]interface{}{"users": summaries}), nil
}

// ========== Helpers ==========

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
