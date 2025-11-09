package main_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_SSE_Connection(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Connect to SSE endpoint
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=test.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	resp, err := http.Get(sseURL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))

	// Read initial connection message
	scanner := bufio.NewScanner(resp.Body)
	scanner.Scan()
	line := scanner.Text()
	assert.Contains(t, line, "event: connected")

	scanner.Scan()
	line = scanner.Text()
	assert.Contains(t, line, "data:")
	assert.Contains(t, line, "ok")
}

func TestE2E_SSE_FileUpdate(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Connect to SSE endpoint
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=test.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(sseURL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Skip connection message
	scanner := bufio.NewScanner(resp.Body)
	for i := 0; i < 3; i++ {
		scanner.Scan()
	}

	// Modify file to trigger file watcher
	go func() {
		time.Sleep(500 * time.Millisecond)
		mdPath := filepath.Join(env.ProjectDir, "test.md")
		content, _ := os.ReadFile(mdPath)
		// Append to file to trigger write event
		_ = os.WriteFile(mdPath, append(content, []byte("\n\n## New Section\n")...), 0644)
	}()

	// Wait for file_updated event
	eventReceived := false
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) && scanner.Scan() {
		line := scanner.Text()
		t.Logf("SSE line: %s", line)

		if strings.Contains(line, "event: file_updated") {
			eventReceived = true
			break
		}
	}

	assert.True(t, eventReceived, "Should receive file_updated event")
}

func TestE2E_SSE_CommentsResolved(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a comment first
	comment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "To be resolved",
	}
	resp := env.postJSON(t, "/api/comments", comment)
	_ = resp.Body.Close()

	// Connect to SSE endpoint
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=test.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 10 * time.Second}
	sseResp, err := client.Get(sseURL)
	require.NoError(t, err)
	defer func() { _ = sseResp.Body.Close() }()

	// Skip connection message
	scanner := bufio.NewScanner(sseResp.Body)
	for i := 0; i < 3; i++ {
		scanner.Scan()
	}

	// Resolve comments in background
	go func() {
		time.Sleep(500 * time.Millisecond)
		_, _ = env.runCLI(t, "resolve", "--file", "test.md", "--project", env.ProjectDir)
	}()

	// Wait for comments_resolved event
	eventReceived := false
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) && scanner.Scan() {
		line := scanner.Text()
		t.Logf("SSE line: %s", line)

		if strings.Contains(line, "event: comments_resolved") {
			eventReceived = true
			break
		}
	}

	assert.True(t, eventReceived, "Should receive comments_resolved event")
}

func TestE2E_SSE_Broadcast(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Connect to SSE endpoint
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=test.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 10 * time.Second}
	sseResp, err := client.Get(sseURL)
	require.NoError(t, err)
	defer func() { _ = sseResp.Body.Close() }()

	// Skip connection message
	scanner := bufio.NewScanner(sseResp.Body)
	for i := 0; i < 3; i++ {
		scanner.Scan()
	}

	// Send broadcast via API
	go func() {
		time.Sleep(500 * time.Millisecond)
		broadcastData := map[string]interface{}{
			"project_directory": env.ProjectDir,
			"file_path":         "test.md",
			"event":             "comments_resolved",
		}
		resp := env.postJSON(t, "/api/events", broadcastData)
		_ = resp.Body.Close()
	}()

	// Wait for broadcast event
	eventReceived := false
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) && scanner.Scan() {
		line := scanner.Text()
		t.Logf("SSE line: %s", line)

		if strings.Contains(line, "event: comments_resolved") {
			eventReceived = true
			break
		}
	}

	assert.True(t, eventReceived, "Should receive broadcast event")
}

func TestE2E_SSE_MissingParams(t *testing.T) {
	env := setupE2E(t)

	testCases := []struct {
		name  string
		url   string
		wants int
	}{
		{
			name:  "missing project_directory",
			url:   env.BaseURL + "/api/events?file_path=test.md",
			wants: http.StatusBadRequest,
		},
		{
			name:  "missing file_path",
			url:   env.BaseURL + "/api/events?project_directory=" + url.QueryEscape(env.ProjectDir),
			wants: http.StatusBadRequest,
		},
		{
			name:  "missing both",
			url:   env.BaseURL + "/api/events",
			wants: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tc.wants, resp.StatusCode)
		})
	}
}

func TestE2E_SSE_MultipleClients(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Connect multiple clients to the same file
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=test.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 10 * time.Second}

	// Client 1
	resp1, err := client.Get(sseURL)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()

	// Client 2
	resp2, err := client.Get(sseURL)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	// Both should receive connection message
	scanner1 := bufio.NewScanner(resp1.Body)
	scanner2 := bufio.NewScanner(resp2.Body)

	// Skip connection messages
	for i := 0; i < 3; i++ {
		scanner1.Scan()
		scanner2.Scan()
	}

	// Broadcast event
	go func() {
		time.Sleep(500 * time.Millisecond)
		broadcastData := map[string]interface{}{
			"project_directory": env.ProjectDir,
			"file_path":         "test.md",
			"event":             "comments_resolved",
		}
		resp := env.postJSON(t, "/api/events", broadcastData)
		_ = resp.Body.Close()
	}()

	// Both clients should receive the event
	received1 := false
	received2 := false

	deadline := time.Now().Add(5 * time.Second)

	done := make(chan bool, 2)

	go func() {
		for time.Now().Before(deadline) && scanner1.Scan() {
			line := scanner1.Text()
			if strings.Contains(line, "event: comments_resolved") {
				received1 = true
				done <- true
				return
			}
		}
		done <- false
	}()

	go func() {
		for time.Now().Before(deadline) && scanner2.Scan() {
			line := scanner2.Text()
			if strings.Contains(line, "event: comments_resolved") {
				received2 = true
				done <- true
				return
			}
		}
		done <- false
	}()

	// Wait for both
	<-done
	<-done

	assert.True(t, received1, "Client 1 should receive event")
	assert.True(t, received2, "Client 2 should receive event")
}

func TestE2E_SSE_ClientFiltering(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Connect to test.md
	sseURL1 := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=test.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	// Connect to simple.md
	sseURL2 := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=simple.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 10 * time.Second}

	resp1, err := client.Get(sseURL1)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()

	resp2, err := client.Get(sseURL2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	scanner1 := bufio.NewScanner(resp1.Body)
	scanner2 := bufio.NewScanner(resp2.Body)

	// Skip connection messages
	for i := 0; i < 3; i++ {
		scanner1.Scan()
		scanner2.Scan()
	}

	// Broadcast event only to test.md
	go func() {
		time.Sleep(500 * time.Millisecond)
		broadcastData := map[string]interface{}{
			"project_directory": env.ProjectDir,
			"file_path":         "test.md", // Only test.md
			"event":             "comments_resolved",
		}
		resp := env.postJSON(t, "/api/events", broadcastData)
		_ = resp.Body.Close()
	}()

	received1 := false
	received2 := false

	deadline := time.Now().Add(3 * time.Second)

	done := make(chan bool, 2)

	go func() {
		for time.Now().Before(deadline) && scanner1.Scan() {
			line := scanner1.Text()
			if strings.Contains(line, "event: comments_resolved") {
				received1 = true
				done <- true
				return
			}
		}
		done <- false
	}()

	go func() {
		for time.Now().Before(deadline) && scanner2.Scan() {
			line := scanner2.Text()
			if strings.Contains(line, "event: comments_resolved") {
				received2 = true
				done <- true
				return
			}
		}
		done <- false
	}()

	// Wait for both
	<-done
	<-done

	assert.True(t, received1, "Client watching test.md should receive event")
	assert.False(t, received2, "Client watching simple.md should NOT receive event")
}

func TestE2E_Broadcast_API(t *testing.T) {
	env := setupE2E(t)

	broadcastData := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"event":             "test_event",
	}

	resp := env.postJSON(t, "/api/events", broadcastData)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "broadcast", result["status"])
}
