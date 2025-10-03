package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

// notifyServerCommentsResolved sends a broadcast event to the server
// to notify connected clients that comments have been resolved
func notifyServerCommentsResolved(projectDir, filePath string) {
	payload := map[string]string{
		"project_directory": projectDir,
		"file_path":         filePath,
		"event":             "comments_resolved",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal broadcast payload: %v", err)
		return
	}

	resp, err := http.Post("http://localhost:4779/api/events", "application/json", bytes.NewReader(data))
	if err != nil {
		// Server might not be running - just log and continue
		log.Printf("Note: Could not notify server (server might not be running): %v", err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Server returned non-OK status: %d", resp.StatusCode)
	}
}
