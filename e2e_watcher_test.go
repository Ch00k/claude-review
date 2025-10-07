// +build e2e

package main_test

import (
	"bufio"
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

func TestE2E_FileWatcher_NonExistentFile(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Try to connect SSE to a non-existent file
	// SSE connection should fail or handle gracefully
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=nonexistent.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(sseURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Connection should succeed (SSE endpoint accepts request)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Read initial connection message
	scanner := bufio.NewScanner(resp.Body)
	scanner.Scan()
	line := scanner.Text()
	assert.Contains(t, line, "event: connected")

	// File watcher should handle the error internally
	// Server should not crash and connection should stay alive
	time.Sleep(500 * time.Millisecond)

	// Verify server is still responsive
	healthResp, err := http.Get(env.BaseURL + "/")
	require.NoError(t, err)
	healthResp.Body.Close()
	assert.Equal(t, http.StatusOK, healthResp.StatusCode)
}

func TestE2E_FileWatcher_NoReadPermission(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a file with no read permissions
	restrictedFile := filepath.Join(env.ProjectDir, "restricted.md")
	err = os.WriteFile(restrictedFile, []byte("# Restricted"), 0200) // Write-only
	require.NoError(t, err)

	// Ensure cleanup
	t.Cleanup(func() {
		os.Chmod(restrictedFile, 0644)
		os.Remove(restrictedFile)
	})

	// Try to connect SSE to restricted file
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=restricted.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(sseURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// SSE connection should handle gracefully
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify server is still responsive
	healthResp, err := http.Get(env.BaseURL + "/")
	require.NoError(t, err)
	healthResp.Body.Close()
	assert.Equal(t, http.StatusOK, healthResp.StatusCode)
}

func TestE2E_FileWatcher_MultipleFiles(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create additional test files
	file1 := filepath.Join(env.ProjectDir, "watch1.md")
	file2 := filepath.Join(env.ProjectDir, "watch2.md")
	file3 := filepath.Join(env.ProjectDir, "watch3.md")

	require.NoError(t, os.WriteFile(file1, []byte("# File 1"), 0644))
	require.NoError(t, os.WriteFile(file2, []byte("# File 2"), 0644))
	require.NoError(t, os.WriteFile(file3, []byte("# File 3"), 0644))

	// Connect SSE to all three files simultaneously
	client := &http.Client{Timeout: 10 * time.Second}

	responses := make([]*http.Response, 3)
	files := []string{"watch1.md", "watch2.md", "watch3.md"}

	for i, file := range files {
		sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=%s",
			env.BaseURL, url.QueryEscape(env.ProjectDir), file)
		resp, err := client.Get(sseURL)
		require.NoError(t, err)
		responses[i] = resp
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Give watchers time to initialize
	time.Sleep(500 * time.Millisecond)

	// Modify file 2 - only its watcher should trigger
	err = os.WriteFile(file2, []byte("# File 2 Updated"), 0644)
	require.NoError(t, err)

	// Read from responses[1] (watch2.md connection)
	scanner := bufio.NewScanner(responses[1].Body)

	// Skip connection message
	for i := 0; i < 3; i++ {
		scanner.Scan()
	}

	// Wait for file_updated event
	eventReceived := false
	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) && scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "event: file_updated") {
			eventReceived = true
			break
		}
	}

	assert.True(t, eventReceived, "Should receive file_updated for watch2.md")

	// Verify server is still responsive after watching multiple files
	healthResp, err := http.Get(env.BaseURL + "/")
	require.NoError(t, err)
	healthResp.Body.Close()
	assert.Equal(t, http.StatusOK, healthResp.StatusCode)
}

func TestE2E_FileWatcher_FileDeletion(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a file
	testFile := filepath.Join(env.ProjectDir, "delete-me.md")
	err = os.WriteFile(testFile, []byte("# To Be Deleted"), 0644)
	require.NoError(t, err)

	// Connect SSE to the file
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=delete-me.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(sseURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Wait for watcher to initialize
	time.Sleep(500 * time.Millisecond)

	// Delete the file while it's being watched
	err = os.Remove(testFile)
	require.NoError(t, err)

	// Watcher should handle deletion gracefully
	time.Sleep(500 * time.Millisecond)

	// Verify server is still responsive
	healthResp, err := http.Get(env.BaseURL + "/")
	require.NoError(t, err)
	healthResp.Body.Close()
	assert.Equal(t, http.StatusOK, healthResp.StatusCode)
}

func TestE2E_FileWatcher_RapidChanges(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(env.ProjectDir, "rapid.md")
	err = os.WriteFile(testFile, []byte("# Initial"), 0644)
	require.NoError(t, err)

	// Connect SSE
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=rapid.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(sseURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	// Skip connection message
	for i := 0; i < 3; i++ {
		scanner.Scan()
	}

	// Make rapid changes
	go func() {
		time.Sleep(500 * time.Millisecond)
		for i := 0; i < 10; i++ {
			content := fmt.Sprintf("# Update %d", i)
			os.WriteFile(testFile, []byte(content), 0644)
			time.Sleep(50 * time.Millisecond) // Rapid changes
		}
	}()

	// Count file_updated events received
	eventCount := 0
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) && scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "event: file_updated") {
			eventCount++
		}
		// Stop after receiving some events
		if eventCount >= 3 {
			break
		}
	}

	// Should receive at least some events (file watcher may coalesce rapid changes)
	assert.Greater(t, eventCount, 0, "Should receive at least one file_updated event")
	t.Logf("Received %d file_updated events from 10 rapid changes", eventCount)

	// Verify server is still responsive after rapid changes
	healthResp, err := http.Get(env.BaseURL + "/")
	require.NoError(t, err)
	healthResp.Body.Close()
	assert.Equal(t, http.StatusOK, healthResp.StatusCode)
}

func TestE2E_FileWatcher_Cleanup(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		env.ServerCmd.Process.Kill()
		env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Register project
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Start daemon
	_, err = env.runCLI(t, "server", "--daemon")
	require.NoError(t, err)
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second))

	// Create a file and connect SSE to start watching
	testFile := filepath.Join(env.ProjectDir, "cleanup-test.md")
	err = os.WriteFile(testFile, []byte("# Cleanup Test"), 0644)
	require.NoError(t, err)

	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=cleanup-test.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(sseURL)
	require.NoError(t, err)
	resp.Body.Close() // Close connection immediately

	// Give watcher time to register
	time.Sleep(300 * time.Millisecond)

	// Stop daemon - should clean up file watcher
	output, err := env.runCLI(t, "server", "--stop")
	require.NoError(t, err)
	assert.Contains(t, output, "Sent SIGTERM")

	// Wait for shutdown
	time.Sleep(500 * time.Millisecond)

	// Restart and verify no issues from previous watchers
	_, err = env.runCLI(t, "server", "--daemon")
	require.NoError(t, err)
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second))

	// Should be able to watch the same file again without issues
	resp, err = client.Get(sseURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Cleanup
	env.runCLI(t, "server", "--stop")
	time.Sleep(500 * time.Millisecond)
}

func TestE2E_FileWatcher_SameFileMultipleClients(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Multiple clients watch the same file
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=test.md",
		env.BaseURL, url.QueryEscape(env.ProjectDir))

	client := &http.Client{Timeout: 10 * time.Second}

	// Connect 3 clients to the same file
	responses := make([]*http.Response, 3)
	scanners := make([]*bufio.Scanner, 3)

	for i := 0; i < 3; i++ {
		resp, err := client.Get(sseURL)
		require.NoError(t, err)
		responses[i] = resp
		defer resp.Body.Close()

		scanners[i] = bufio.NewScanner(resp.Body)
		// Skip connection messages
		for j := 0; j < 3; j++ {
			scanners[i].Scan()
		}
	}

	// Modify the file
	testFile := filepath.Join(env.ProjectDir, "test.md")
	go func() {
		time.Sleep(500 * time.Millisecond)
		content, _ := os.ReadFile(testFile)
		os.WriteFile(testFile, append(content, []byte("\n\n## Multiple Watchers\n")...), 0644)
	}()

	// All clients should receive the event
	received := make([]bool, 3)
	deadline := time.Now().Add(3 * time.Second)

	done := make(chan int, 3)

	for i := 0; i < 3; i++ {
		go func(idx int) {
			for time.Now().Before(deadline) && scanners[idx].Scan() {
				line := scanners[idx].Text()
				if strings.Contains(line, "event: file_updated") {
					received[idx] = true
					done <- idx
					return
				}
			}
			done <- -1
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// All clients should have received the event
	for i := 0; i < 3; i++ {
		assert.True(t, received[i], "Client %d should receive file_updated event", i)
	}
}

func TestE2E_FileWatcher_DirectoryDeletion(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a subdirectory with a file
	subDir := filepath.Join(env.ProjectDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	subFile := filepath.Join(subDir, "nested.md")
	err = os.WriteFile(subFile, []byte("# Nested File"), 0644)
	require.NoError(t, err)

	// Connect SSE to the nested file
	sseURL := fmt.Sprintf("%s/api/events?project_directory=%s&file_path=%s",
		env.BaseURL, url.QueryEscape(env.ProjectDir), url.QueryEscape("subdir/nested.md"))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(sseURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Wait for watcher to initialize
	time.Sleep(500 * time.Millisecond)

	// Delete the entire subdirectory
	err = os.RemoveAll(subDir)
	require.NoError(t, err)

	// Watcher should handle directory deletion gracefully
	time.Sleep(500 * time.Millisecond)

	// Verify server is still responsive
	healthResp, err := http.Get(env.BaseURL + "/")
	require.NoError(t, err)
	healthResp.Body.Close()
	assert.Equal(t, http.StatusOK, healthResp.StatusCode)
}
