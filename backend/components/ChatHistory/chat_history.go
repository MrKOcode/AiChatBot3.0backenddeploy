package main

import (
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
			if req.Path == "/api/chat/history/save" {
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


func saveChatMessage(c *gin.Context) {
	var req struct {
		UserID         int    `json:"userId" binding:"required"`
		Username       string `json:"username" binding:"required"`
		ConversationID string `json:"conversationId" binding:"required"`
		MessageType    string `json:"messageType" binding:"required"`
		Content        string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if req.MessageType != "user" && req.MessageType != "ai" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message type"})
		return
	}

	err := services.SaveChatMessage(req.UserID, req.Username, req.ConversationID, req.MessageType, req.Content)
	if err != nil {
		log.Printf("Error saving chat message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save chat message"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Chat message saved successfully"})
}
func getUserChatHistory(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	keyword := c.Query("keyword")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}

	query := services.ChatHistoryQuery{
		UserID:    userID,
		Keyword:   keyword,
		StartDate: startDate,
		EndDate:   endDate,
		Page:      page,
		PageSize:  pageSize,
	}

	result, err := services.GetUserChatHistory(query)
	if err != nil {
		log.Printf("Error getting user chat history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat history"})
		return
	}

	c.JSON(http.StatusOK, result)
}
func getAllUsersChatSummary(c *gin.Context) {
	summaries, err := services.GetAllUsersChatSummary()
	if err != nil {
		log.Printf("Error getting users chat summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users chat summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": summaries})
}
