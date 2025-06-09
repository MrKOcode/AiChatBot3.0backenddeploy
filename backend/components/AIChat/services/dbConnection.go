package services

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
)

// Database file path
var dbPath = "./chatbot.db" // Update path as needed

// DB is a global database connection pool
var DB *sql.DB

// Initialize the database connection and create tables
func InitDB() {
	// Open database connection
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error opening database: %v\n", err)
	}

	// Test the database connection
	if err = DB.Ping(); err != nil {
		log.Fatalf("Error testing database connection: %v\n", err)
	}
	fmt.Println("Connected to the SQLite database")

	// Create tables
	createTables()
}

// CreateTables creates necessary tables if they don't exist
func createTables() {
	createChatHistoryTable := `
	CREATE TABLE IF NOT EXISTS ChatHistory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		userMessage TEXT NOT NULL,
		response TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createUsersTable := `
	CREATE TABLE IF NOT EXISTS Users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT CHECK(role IN ('teacher', 'student')) NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	createChaptersTable := `
	CREATE TABLE IF NOT EXISTS Chapters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT,
		teacher_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (teacher_id) REFERENCES Users(id)
	);`

	createMaterialsTable := `
	CREATE TABLE IF NOT EXISTS Materials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chapter_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (chapter_id) REFERENCES Chapters(id)
	);`

	createExercisesTable := `
	CREATE TABLE IF NOT EXISTS Exercises (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chapter_id INTEGER NOT NULL,
		question TEXT NOT NULL,
		answer TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (chapter_id) REFERENCES Chapters(id)
	);`

	createStudentProgressTable := `
	CREATE TABLE IF NOT EXISTS StudentProgress (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		student_id INTEGER NOT NULL,
		chapter_id INTEGER NOT NULL,
		status TEXT CHECK(status IN ('not started', 'in progress', 'completed')) DEFAULT 'not started',
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (student_id) REFERENCES Users(id),
		FOREIGN KEY (chapter_id) REFERENCES Chapters(id)
	);`

	createConversationsTable := `
    CREATE TABLE IF NOT EXISTS Conversations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id TEXT NOT NULL,
	title TEXT DEFAULT 'Untitled Conversation',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	createMessagesTable := `
	CREATE TABLE IF NOT EXISTS Messages(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id INTEGER NOT NULL,
	role TEXT CHECK (role IN ('user','chatbot','system'))NOT NULL,
	content TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(conversation_id) REFERENCES Conversations(id)

	);`

	// Execute the table creation queries
	for _, query := range []string{
		createChatHistoryTable,
		createUsersTable,
		createChaptersTable,
		createMaterialsTable,
		createExercisesTable,
		createStudentProgressTable,
		createConversationsTable,
		createMessagesTable,
	} {
		if _, err := DB.Exec(query); err != nil {
			log.Fatalf("Error creating table: %v\n", err)
		}
	}
	fmt.Println("Database tables created or already exist")
}

// CloseDB closes the database connection
func CloseDB() {
	if err := DB.Close(); err != nil {
		log.Fatalf("Error closing database: %v\n", err)
	}
	fmt.Println("Database connection closed")
}
