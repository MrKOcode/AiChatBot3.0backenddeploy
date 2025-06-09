package services

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// User 用户模型
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"` // 不在JSON中显示密码
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

// InitDB 初始化数据库
func InitDB() {
	var err error
	
	// 使用SQLite作为本地数据库
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./auth.db"
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 测试数据库连接
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// 创建用户表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}

	log.Println("✅ Database initialized successfully")
}

// HashPassword 加密密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash 验证密码
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// UserExists 检查用户名是否存在
func UserExists(username string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM users WHERE username = ?"
	err := db.QueryRow(query, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// determineUserRole 根据用户名确定角色
func determineUserRole(username string) string {
	if strings.HasPrefix(strings.ToLower(username), "admin") {
		log.Printf("🔑 Admin user detected: %s", username)
		return "admin"
	}
	log.Printf("🎓 Student user detected: %s", username)
	return "student"
}

// CreateUser 创建新用户
func CreateUser(username, password string) (int, error) {
	// 加密密码
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return 0, err
	}

	// 插入用户
	role := determineUserRole(username)

	// 插入用户
	query := "INSERT INTO users (username, password, role) VALUES (?, ?, ?)"
	result, err := db.Exec(query, username, hashedPassword, determineUserRole(username))
	if err != nil {
		return 0, err
	}

	// 获取新用户ID
	userID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	log.Printf("✅ User created successfully: %s (ID: %d, Role: %s)", username, userID, role)
	return int(userID), nil
}

// ValidateUser 验证用户登录
func ValidateUser(username, password string) (*User, error) {
	var user User
	query := "SELECT id, username, password, role, created_at FROM users WHERE username = ?"
	
	err := db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	// 验证密码
	if !CheckPasswordHash(password, user.Password) {
		return nil, errors.New("invalid password")
	}

	log.Printf("✅ User validated successfully: %s (ID: %d, Role: %s)", username, user.ID, user.Role)
	return &user, nil
}

// GetUserByID 根据ID获取用户信息
func GetUserByID(userID string) (*User, error) {
	var user User
	query := "SELECT id, username, role, created_at FROM users WHERE id = ?"
	
	err := db.QueryRow(query, userID).Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// GetAllUsers 获取所有用户（管理功能）
func GetAllUsers() ([]User, error) {
	query := "SELECT id, username, role, created_at FROM users ORDER BY created_at DESC"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
} 
