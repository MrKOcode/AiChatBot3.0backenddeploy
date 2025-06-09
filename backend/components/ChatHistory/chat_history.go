package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"AiChatBotBackend/components/ChatHistory/services"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file:", err)
	}

	services.InitChatHistoryDB()

	router := gin.Default()

	// 聊天记录相关路由
	router.POST("/api/chat-history/save", saveChatMessage)
	router.GET("/api/chat-history/user/:userId", getUserChatHistory)
	router.GET("/api/chat-history/admin/users", getAllUsersChatSummary)

	port := os.Getenv("CHAT_HISTORY_PORT")
	if port == "" {
		log.Println("CHAT_HISTORY_PORT is not set, using default port 5004")
		port = "5004"
	}

	log.Printf("Chat History Service is running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start chat history microservice: %s", err)
	}
}

// saveChatMessage 保存聊天消息
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

// getUserChatHistory 获取用户聊天历史
func getUserChatHistory(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// 从查询参数获取分页和搜索条件
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

// getAllUsersChatSummary 获取所有用户聊天摘要（管理员功能）
func getAllUsersChatSummary(c *gin.Context) {
	summaries, err := services.GetAllUsersChatSummary()
	if err != nil {
		log.Printf("Error getting users chat summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users chat summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": summaries})
}
