package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	ID               int        `json:"id"`
	ProjectDirectory string     `json:"project_directory"`
	FilePath         string     `json:"file_path"`
	LineStart        *int       `json:"line_start,omitempty"`
	LineEnd          *int       `json:"line_end,omitempty"`
	SelectedText     string     `json:"selected_text"`
	CommentText      string     `json:"comment_text"`
	RenderedHTML     string     `json:"rendered_html,omitempty"` // Populated on-the-fly for web UI (not stored in DB)
	CreatedAt        time.Time  `json:"created_at"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	RootID           *int       `json:"root_id,omitempty"`
	Author           string     `json:"author"`
	ResolvedBy       *string    `json:"resolved_by,omitempty"`
}

var db *sql.DB

// getDataDir returns the data directory for claude-review and ensures it exists
func getDataDir() (string, error) {
	var dataDir string

	// Check for CR_DATA_DIR environment variable first
	if envDataDir := os.Getenv("CR_DATA_DIR"); envDataDir != "" {
		dataDir = envDataDir
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".local", "share", "claude-review")
	}

	// Ensure the directory exists
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return dataDir, nil
}

func initDB() error {
	// Get data directory (ensures it exists)
	dbDir, err := getDataDir()
	if err != nil {
		return err
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
		line_start INTEGER,
		line_end INTEGER,
		selected_text TEXT,
		comment_text TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMP,
		root_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
		author TEXT CHECK(author IN ('user', 'agent')),
		resolved_by TEXT,
		FOREIGN KEY (project_directory) REFERENCES projects(directory)
	);

	CREATE INDEX IF NOT EXISTS idx_comments_lookup ON comments(project_directory, file_path, resolved_at, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_comments_thread ON comments(root_id, created_at);
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
	// Generate timestamp in Go
	c.CreatedAt = time.Now()

	query := `
		INSERT INTO comments (project_directory, file_path, line_start, line_end, selected_text, comment_text, root_id, author, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	logQuery(
		query,
		c.ProjectDirectory,
		c.FilePath,
		c.LineStart,
		c.LineEnd,
		c.SelectedText,
		c.CommentText,
		c.RootID,
		c.Author,
		c.CreatedAt,
	)
	result, err := db.Exec(
		query,
		c.ProjectDirectory,
		c.FilePath,
		c.LineStart,
		c.LineEnd,
		c.SelectedText,
		c.CommentText,
		c.RootID,
		c.Author,
		c.CreatedAt,
	)
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
	var query string
	if resolved {
		query = `
			SELECT id, project_directory, file_path, line_start, line_end, selected_text, comment_text, created_at, resolved_at, root_id, author, resolved_by
			FROM comments
			WHERE project_directory = ? AND file_path = ? AND resolved_at IS NOT NULL
			ORDER BY COALESCE(root_id, id) ASC, created_at ASC`
	} else {
		query = `
			SELECT id, project_directory, file_path, line_start, line_end, selected_text, comment_text, created_at, resolved_at, root_id, author, resolved_by
			FROM comments
			WHERE project_directory = ? AND file_path = ? AND resolved_at IS NULL
			ORDER BY COALESCE(root_id, id) ASC, created_at ASC`
	}
	logQuery(query, projectDir, filePath)
	rows, err := db.Query(query, projectDir, filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ProjectDirectory, &c.FilePath, &c.LineStart, &c.LineEnd, &c.SelectedText, &c.CommentText, &c.CreatedAt, &c.ResolvedAt, &c.RootID, &c.Author, &c.ResolvedBy); err != nil {
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
		SET resolved_at = CURRENT_TIMESTAMP, resolved_by = 'user'
		WHERE project_directory = ? AND file_path = ? AND resolved_at IS NULL`
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

func getCommentByID(commentID int) (*Comment, error) {
	query := `
		SELECT id, project_directory, file_path, line_start, line_end, selected_text, comment_text, created_at, resolved_at, root_id, author, resolved_by
		FROM comments
		WHERE id = ?`
	logQuery(query, commentID)

	var c Comment
	err := db.QueryRow(query, commentID).Scan(
		&c.ID, &c.ProjectDirectory, &c.FilePath, &c.LineStart, &c.LineEnd,
		&c.SelectedText, &c.CommentText, &c.CreatedAt,
		&c.ResolvedAt, &c.RootID, &c.Author, &c.ResolvedBy,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func resolveThread(rootCommentID int, resolvedBy string) (int, error) {
	query := `
		UPDATE comments
		SET resolved_at = CURRENT_TIMESTAMP, resolved_by = ?
		WHERE (id = ? OR root_id = ?) AND resolved_at IS NULL`
	logQuery(query, resolvedBy, rootCommentID, rootCommentID)
	result, err := db.Exec(query, resolvedBy, rootCommentID, rootCommentID)
	if err != nil {
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

func hasReplies(commentID int) (bool, error) {
	query := `
		SELECT COUNT(*) FROM comments WHERE root_id = ?`
	logQuery(query, commentID)

	var count int
	err := db.QueryRow(query, commentID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// renderCommentsAsHTML renders the comment_text field of each comment as HTML
// and stores it in the RenderedHTML field for web UI display
func renderCommentsAsHTML(comments []Comment) error {
	for i := range comments {
		rendered, err := RenderMarkdown([]byte(comments[i].CommentText))
		if err != nil {
			return fmt.Errorf("failed to render comment markdown: %w", err)
		}
		// Trim whitespace to avoid issues in inline JavaScript
		comments[i].RenderedHTML = strings.TrimSpace(string(rendered))
	}
	return nil
}
