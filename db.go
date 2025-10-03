package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var debugSQL = os.Getenv("DEBUG_SQL") == "1"

func logQuery(query string, args ...interface{}) {
	if debugSQL {
		log.Printf("[SQL] %s | args: %v", query, args)
	}
}

type Project struct {
	Directory string    `json:"directory"`
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	ID               int       `json:"id"`
	ProjectDirectory string    `json:"project_directory"`
	FilePath         string    `json:"file_path"`
	LineStart        int       `json:"line_start"`
	LineEnd          int       `json:"line_end"`
	SelectedText     string    `json:"selected_text"`
	CommentText      string    `json:"comment_text"`
	Resolved         bool      `json:"resolved"`
	CreatedAt        time.Time `json:"created_at"`
}

var db *sql.DB

// getDataDir returns the data directory for claude-review
func getDataDir() (string, error) {
	// Check for CR_DATA_DIR environment variable first
	if dataDir := os.Getenv("CR_DATA_DIR"); dataDir != "" {
		return dataDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".local", "share", "claude-review"), nil
}

func initDB() error {
	// Create claude-review directory
	dbDir, err := getDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	dbPath := filepath.Join(dbDir, "comments.db")
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		directory TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_directory TEXT NOT NULL,
		file_path TEXT NOT NULL,
		line_start INTEGER NOT NULL,
		line_end INTEGER NOT NULL,
		selected_text TEXT NOT NULL,
		comment_text TEXT NOT NULL,
		resolved BOOLEAN DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_directory) REFERENCES projects(directory)
	);
	`

	logQuery(schema)
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

func createProject(directory string) (*Project, error) {
	// Idempotent insert
	query := "INSERT OR IGNORE INTO projects (directory) VALUES (?)"
	logQuery(query, directory)
	_, err := db.Exec(query, directory)
	if err != nil {
		return nil, err
	}

	var project Project
	query = "SELECT directory, created_at FROM projects WHERE directory = ?"
	logQuery(query, directory)
	err = db.QueryRow(query, directory).
		Scan(&project.Directory, &project.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func getAllProjects() ([]Project, error) {
	query := "SELECT directory, created_at FROM projects ORDER BY created_at DESC"
	logQuery(query)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.Directory, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

func createComment(c *Comment) error {
	query := `
		INSERT INTO comments (project_directory, file_path, line_start, line_end, selected_text, comment_text)
		VALUES (?, ?, ?, ?, ?, ?)`
	logQuery(query, c.ProjectDirectory, c.FilePath, c.LineStart, c.LineEnd, c.SelectedText, c.CommentText)
	result, err := db.Exec(query, c.ProjectDirectory, c.FilePath, c.LineStart, c.LineEnd, c.SelectedText, c.CommentText)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = int(id)

	return nil
}

func getComments(projectDir, filePath string, resolved bool) ([]Comment, error) {
	query := `
		SELECT id, project_directory, file_path, line_start, line_end, selected_text, comment_text, resolved, created_at
		FROM comments
		WHERE project_directory = ? AND file_path = ? AND resolved = ?
		ORDER BY line_start ASC`
	logQuery(query, projectDir, filePath, resolved)
	rows, err := db.Query(query, projectDir, filePath, resolved)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ProjectDirectory, &c.FilePath, &c.LineStart, &c.LineEnd, &c.SelectedText, &c.CommentText, &c.Resolved, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	return comments, nil
}

func updateComment(commentID, commentText string) error {
	query := `
		UPDATE comments
		SET comment_text = ?
		WHERE id = ?`
	logQuery(query, commentText, commentID)
	_, err := db.Exec(query, commentText, commentID)
	return err
}

func deleteComment(commentID string) error {
	query := `
		DELETE FROM comments
		WHERE id = ?`
	logQuery(query, commentID)
	_, err := db.Exec(query, commentID)
	return err
}

func resolveComments(projectDir, filePath string) (int, error) {
	query := `
		UPDATE comments
		SET resolved = 1
		WHERE project_directory = ? AND file_path = ? AND resolved = 0`
	logQuery(query, projectDir, filePath)
	result, err := db.Exec(query, projectDir, filePath)
	if err != nil {
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(count), nil
}
