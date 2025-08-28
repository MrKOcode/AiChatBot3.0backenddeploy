// ✅ auth.go - Lambda-Compatible Version (Phase 1)

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"

	"AiChatBotBackend/components/Auth/services"
)

func main() {
	_ = godotenv.Load(".env")
	services.InitDB()
	lambda.Start(handler)
}

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "POST":
		if req.Path == "/api/auth/register" {
			return lambdaRegisterUser(req)
		}
		if req.Path == "/api/auth/login" {
			return lambdaLoginUser(req)
		}
	case "GET":
		if strings.HasPrefix(req.Path, "/api/auth/user/") {
			return lambdaGetUserInfo(req)
		}
	}
	return errorResponse(404, "Route not found"), nil
}

func lambdaRegisterUser(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	_ = json.Unmarshal([]byte(req.Body), &body)

	if len(body.Username) < 3 || len(body.Password) < 6 {
		return errorResponse(400, "Username or password too short"), nil
	}

	exists, err := services.UserExists(body.Username)
	if err != nil {
		return errorResponse(500, "Database error"), nil
	}
	if exists {
		return errorResponse(409, "Username already taken"), nil
	}

	userID, err := services.CreateUser(body.Username, body.Password)
	if err != nil {
		return errorResponse(500, "Create user failed"), nil
	}

	role := determineRole(body.Username)
	return jsonResponse(201, map[string]interface{}{
		"message":  "User registered successfully",
		"userId":   userID,
		"role":     role,
		"username": body.Username,
	}), nil
}

func lambdaLoginUser(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	_ = json.Unmarshal([]byte(req.Body), &body)

	user, err := services.ValidateUser(body.Username, body.Password)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "invalid password" {
			return errorResponse(401, "Username or password is incorrect"), nil
		}
		return errorResponse(500, "Database error"), nil
	}

	return jsonResponse(200, map[string]interface{}{
		"message":  "Login successful",
		"userId":   user.ID,
		"username": user.Username,
		"role":     user.Role,
	}), nil
}

func lambdaGetUserInfo(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := strings.TrimPrefix(req.Path, "/api/auth/user/")
	user, err := services.GetUserByID(id)
	if err != nil {
		if err.Error() == "user not found" {
			return errorResponse(404, "User not found"), nil
		}
		return errorResponse(500, "Database error"), nil
	}

	return jsonResponse(200, map[string]interface{}{
		"userId":   user.ID,
		"username": user.Username,
		"role":     user.Role,
	}), nil
}

func determineRole(username string) string {
	if strings.HasPrefix(strings.ToLower(username), "admin") {
		return "admin"
	}
	return "student"
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
