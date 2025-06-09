package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	StartServer()
}

func StartServer() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.Use(corsMiddleware())

	// AI Chat Service
	router.Any("/api/AIchat", reverseProxy("http://localhost:5001"))
	router.Any("/api/AIchat/*path", reverseProxy("http://localhost:5001"))
	
	// Authentication Service
	router.Any("/api/auth", reverseProxy("http://localhost:5002"))
	router.Any("/api/auth/*path", reverseProxy("http://localhost:5002"))
	
	// Legacy login route
	router.Any("/api/login/*path", reverseProxy("http://localhost:5002"))
	
	// Chat History Service
	router.Any("/api/chat-history", reverseProxy("http://localhost:5004"))
	router.Any("/api/chat-history/*path", reverseProxy("http://localhost:5004"))

	// Admin Service
	router.Any("/api/admin/*path", reverseProxy("http://localhost:5003"))

	log.Printf("API Gateway is running on port %s", port)
	log.Printf("🔀 Route mappings:")
	log.Printf("   /api/AIchat/* -> http://localhost:5001")
	log.Printf("   /api/auth/* -> http://localhost:5002")
	log.Printf("   /api/login/* -> http://localhost:5002")
	log.Printf("   /api/admin/* -> http://localhost:5003")
	
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start API Gateway: %s", err)
	}
}

func reverseProxy(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetURL, err := url.Parse(target)
		if err != nil {
			log.Printf("Invalid target URL: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid backend service URL"})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		allowedOrigins := []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"http://127.0.0.1:5173",
		}
		
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}
		
		if isAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
