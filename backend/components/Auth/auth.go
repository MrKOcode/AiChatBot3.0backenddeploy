package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"AiChatBotBackend/components/Auth/services"
)

func determineRole(username string) string {
	if strings.HasPrefix(strings.ToLower(username), "admin") {
		return "admin"
	}
	return "student"
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file:", err)
	}

	services.InitDB()

	router := gin.Default()
	// router.Use(corsMiddleware()) // 移除CORS，由API Gateway处理

	router.POST("/api/auth/register", registerUser)
	router.POST("/api/auth/login", loginUser)
	router.GET("/api/auth/user/:id", getUserInfo)

	port := os.Getenv("AUTH_PORT")
	if port == "" {
		log.Println("AUTH_PORT is not set, using default port 5002")
		port = "5002"
	}

	log.Printf("Auth Service is running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start auth microservice: %s", err)
	}
}

func registerUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unvalid request format"})
		return
	}

	if len(req.Username) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username must be at least 3 characters long"})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters long"})
		return
	}

	exists, err := services.UserExists(req.Username)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
		return
	}

	userID, err := services.CreateUser(req.Username, req.Password)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create user failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "User registered successfully",
		"userId":   userID,
		"role":     determineRole(req.Username),
		"username": req.Username,
	})
}

func loginUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unvalid request format"})
		return
	}

	user, err := services.ValidateUser(req.Username, req.Password)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "invalid password" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Username or password is incorrect"})
			return
		}
		log.Printf("Error validating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Login successful",
		"userId":   user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

func getUserInfo(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	user, err := services.GetUserByID(userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		log.Printf("Error getting user info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":   user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}
