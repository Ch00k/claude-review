package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: claude-review <command>")
		fmt.Println("\nCommands:")
		fmt.Println("  server    Start the web server")
		fmt.Println("  address   Show unresolved comments for a file")
		fmt.Println("  resolve   Mark all comments for a file as resolved")
		fmt.Println("  list      List all comments for a file (resolved and unresolved)")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "server":
		runServer()
	case "address":
		runAddress()
	case "resolve":
		runResolve()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func runServer() {
	// Initialize database
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize templates
	if err := initTemplates(); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// HTML Routes
	r.Get("/", handleHome)
	r.Get("/projects/*", handleProjectFiles)

	// API Routes
	r.Post("/api/projects", handleCreateProject)
	r.Post("/api/comments", handleCreateComment)
	r.Get("/api/comments", handleGetComments)
	r.Patch("/api/comments/{id}", handleUpdateComment)
	r.Delete("/api/comments/{id}", handleDeleteComment)
	r.Post("/api/comments/resolve", handleResolveComments)
	r.Get("/api/events", handleSSE)
	r.Post("/api/events", handleBroadcast)
	r.Get("/health", handleHealth)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("frontend/static"))))

	// Start server
	port := "4779"
	fmt.Printf("Starting server on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runAddress() {
	// Parse flags
	reviewCmd := flag.NewFlagSet("address", flag.ExitOnError)
	projectDir := reviewCmd.String("project", "", "Project directory")
	filePath := reviewCmd.String("file", "", "File path relative to project directory")

	if err := reviewCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	// Resolve project directory (default to current directory)
	if *projectDir == "" || *projectDir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}
		*projectDir = cwd
	}
	if *filePath == "" {
		fmt.Println("Error: --file flag is required")
		os.Exit(1)
	}

	// Remove @ prefix if present
	*filePath = strings.TrimPrefix(*filePath, "@")

	// Initialize database
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Debug: show what we're searching for
	log.Printf("Searching for comments: project_directory=%q, file_path=%q", *projectDir, *filePath)

	// Get unresolved comments
	comments, err := getComments(*projectDir, *filePath, false)
	if err != nil {
		log.Fatalf("Failed to get comments: %v", err)
	}
	log.Printf("Found %d unresolved comments", len(comments))

	// Format and output comments
	if len(comments) == 0 {
		fmt.Printf("No unresolved comments for %s\n", *filePath)
		return
	}

	fmt.Printf("Found %d unresolved comment(s) for %s:\n\n", len(comments), *filePath)

	for i, comment := range comments {
		fmt.Printf("## Comment %d (lines %d-%d)\n", i+1, comment.LineStart, comment.LineEnd)
		fmt.Printf("**Context:**\n")

		// Format selected text as blockquote
		selectedLines := strings.Split(comment.SelectedText, "\n")
		for _, line := range selectedLines {
			fmt.Printf("> %s\n", line)
		}

		fmt.Printf("\n**Feedback:**\n")
		fmt.Printf("%s\n", comment.CommentText)
		fmt.Printf("\n---\n\n")
	}

	fmt.Println("Please address each comment by updating the file accordingly.")
}

func runResolve() {
	// Parse flags
	resolveCmd := flag.NewFlagSet("resolve", flag.ExitOnError)
	projectDir := resolveCmd.String("project", "", "Project directory")
	filePath := resolveCmd.String("file", "", "File path relative to project directory")

	if err := resolveCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	// Resolve project directory (default to current directory)
	if *projectDir == "" || *projectDir == "." {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}
		*projectDir = cwd
	}
	if *filePath == "" {
		fmt.Println("Error: --file flag is required")
		os.Exit(1)
	}

	// Remove @ prefix if present
	*filePath = strings.TrimPrefix(*filePath, "@")

	// Initialize database
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Debug: show what we're searching for
	log.Printf("Searching for comments: project_directory=%q, file_path=%q", *projectDir, *filePath)

	// First check if there are any unresolved comments
	comments, err := getComments(*projectDir, *filePath, false)
	if err != nil {
		log.Fatalf("Failed to get comments: %v", err)
	}
	log.Printf("Found %d unresolved comments", len(comments))

	// Resolve comments
	count, err := resolveComments(*projectDir, *filePath)
	if err != nil {
		log.Fatalf("Failed to resolve comments: %v", err)
	}

	if count == 0 {
		fmt.Printf("No unresolved comments found for %s\n", *filePath)
	} else {
		fmt.Printf("Resolved %d comment(s) for %s\n", count, *filePath)
	}
}
