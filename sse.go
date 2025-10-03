package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type SSEClient struct {
	ProjectDir string
	FilePath   string
	Channel    chan []byte
}

type SSEHub struct {
	clients map[*SSEClient]bool
	mu      sync.RWMutex
}

var sseHub = &SSEHub{
	clients: make(map[*SSEClient]bool),
}

func (h *SSEHub) addClient(client *SSEClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
}

func (h *SSEHub) removeClient(client *SSEClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
	close(client.Channel)
}

func (h *SSEHub) broadcast(projectDir, filePath, event string, data interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	jsonData, _ := json.Marshal(data)
	message := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, jsonData))

	for client := range h.clients {
		if client.ProjectDir == projectDir && client.FilePath == filePath {
			select {
			case client.Channel <- message:
			default:
				// Client can't keep up, skip
			}
		}
	}
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	projectDir := r.URL.Query().Get("project_directory")
	filePath := r.URL.Query().Get("file_path")

	if projectDir == "" || filePath == "" {
		http.Error(w, "Missing project_directory or file_path", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create client
	client := &SSEClient{
		ProjectDir: projectDir,
		FilePath:   filePath,
		Channel:    make(chan []byte, 10),
	}

	sseHub.addClient(client)
	defer sseHub.removeClient(client)

	// Send initial connection message
	if _, err := fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n"); err != nil {
		return
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream messages
	for {
		select {
		case msg := <-client.Channel:
			if _, err := w.Write(msg); err != nil {
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func handleBroadcast(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectDirectory string `json:"project_directory"`
		FilePath         string `json:"file_path"`
		Event            string `json:"event"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sseHub.broadcast(req.ProjectDirectory, req.FilePath, req.Event, map[string]string{
		"file_path": req.FilePath,
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "broadcast"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
