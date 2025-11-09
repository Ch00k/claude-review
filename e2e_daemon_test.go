package main_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_Daemon_StartAndStop(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		_ = env.ServerCmd.Process.Kill()
		_ = env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Ensure daemon is stopped on test completion (even on failure)
	t.Cleanup(func() {
		_, _ = env.runCLI(t, "server", "--stop")
		time.Sleep(500 * time.Millisecond)
	})

	// Start daemon
	output, err := env.runCLI(t, "server", "--daemon")
	require.NoError(t, err, "Failed to start daemon")
	assert.Contains(t, output, "Server started as daemon")
	assert.Contains(t, output, "PID file:")

	// Wait for daemon to be ready
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second))

	// Verify PID file exists
	pidFile := filepath.Join(env.DataDir, "server.pid")
	pidData, err := os.ReadFile(pidFile)
	require.NoError(t, err, "PID file should exist")
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	require.NoError(t, err, "PID should be valid integer")
	assert.Greater(t, pid, 0, "PID should be positive")

	// Verify process is running
	process, err := os.FindProcess(pid)
	require.NoError(t, err)
	err = process.Signal(syscall.Signal(0))
	assert.NoError(t, err, "Process should be running")

	// Stop daemon
	output, err = env.runCLI(t, "server", "--stop")
	require.NoError(t, err, "Failed to stop daemon")
	assert.Contains(t, output, "Sent SIGTERM to server")
	assert.Contains(t, output, strconv.Itoa(pid))

	// Wait for daemon to stop
	time.Sleep(500 * time.Millisecond)

	// Verify PID file is removed
	_, err = os.ReadFile(pidFile)
	assert.True(t, os.IsNotExist(err), "PID file should be removed after stop")

	// Verify process is not running
	err = process.Signal(syscall.Signal(0))
	assert.Error(t, err, "Process should not be running")
}

func TestE2E_Daemon_Status(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		_ = env.ServerCmd.Process.Kill()
		_ = env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Ensure daemon is stopped on test completion
	t.Cleanup(func() {
		_, _ = env.runCLI(t, "server", "--stop")
		time.Sleep(500 * time.Millisecond)
	})

	// Check status when not running
	output, err := env.runCLI(t, "server", "--status")
	assert.Error(t, err, "Status should return error when server not running")
	assert.Contains(t, output, "Server is not running")

	// Start daemon
	_, err = env.runCLI(t, "server", "--daemon")
	require.NoError(t, err)
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second))

	// Check status when running
	output, err = env.runCLI(t, "server", "--status")
	require.NoError(t, err, "Status should succeed when server running")
	assert.Contains(t, output, "Server is running")
	assert.Contains(t, output, "PID:")
	assert.Contains(t, output, "Port:")
	assert.Contains(t, output, "PID file:")
	assert.Contains(t, output, "Log file:")

	// Stop daemon for cleanup
	_, _ = env.runCLI(t, "server", "--stop")
	time.Sleep(500 * time.Millisecond)
}

func TestE2E_Daemon_MultipleStartAttempts(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		_ = env.ServerCmd.Process.Kill()
		_ = env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Ensure daemon is stopped on test completion
	t.Cleanup(func() {
		_, _ = env.runCLI(t, "server", "--stop")
		time.Sleep(500 * time.Millisecond)
	})

	// Start daemon
	_, err := env.runCLI(t, "server", "--daemon")
	require.NoError(t, err)
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second))

	// Try to start again - should fail
	output, err := env.runCLI(t, "server", "--daemon")
	assert.Error(t, err, "Second start should fail")
	assert.Contains(t, output, "server is already running")

	// Cleanup
	_, _ = env.runCLI(t, "server", "--stop")
	time.Sleep(500 * time.Millisecond)
}

func TestE2E_Daemon_StalePIDFile(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		_ = env.ServerCmd.Process.Kill()
		_ = env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Ensure daemon is stopped on test completion
	t.Cleanup(func() {
		_, _ = env.runCLI(t, "server", "--stop")
		time.Sleep(500 * time.Millisecond)
	})

	// Create a stale PID file with a non-existent PID
	pidFile := filepath.Join(env.DataDir, "server.pid")
	stalePID := 99999
	err := os.WriteFile(pidFile, []byte(strconv.Itoa(stalePID)), 0644)
	require.NoError(t, err)

	// Check status - should detect stale PID file and clean it up
	output, err := env.runCLI(t, "server", "--status")
	assert.Error(t, err)
	assert.Contains(t, output, "Server is not running")

	// Verify PID file was cleaned up
	_, err = os.ReadFile(pidFile)
	assert.True(t, os.IsNotExist(err), "Stale PID file should be removed")

	// Should be able to start server now
	output, err = env.runCLI(t, "server", "--daemon")
	require.NoError(t, err, "Should be able to start after stale PID cleanup")
	assert.Contains(t, output, "Server started as daemon")

	// Cleanup
	_, _ = env.runCLI(t, "server", "--stop")
	time.Sleep(500 * time.Millisecond)
}

func TestE2E_Daemon_StopWhenNotRunning(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		_ = env.ServerCmd.Process.Kill()
		_ = env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Try to stop when no server is running
	output, err := env.runCLI(t, "server", "--stop")
	assert.Error(t, err, "Stop should fail when server not running")
	assert.Contains(t, output, "server is not running")
}

func TestE2E_Daemon_LogFile(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		_ = env.ServerCmd.Process.Kill()
		_ = env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Ensure daemon is stopped on test completion
	t.Cleanup(func() {
		_, _ = env.runCLI(t, "server", "--stop")
		time.Sleep(500 * time.Millisecond)
	})

	// Start daemon
	_, err := env.runCLI(t, "server", "--daemon")
	require.NoError(t, err)
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second))

	// Check that log file exists and has content
	logFile := filepath.Join(env.DataDir, "server.log")
	_, err = os.Stat(logFile)
	require.NoError(t, err, "Log file should exist")

	// Read log file
	logContent, err := os.ReadFile(logFile)
	require.NoError(t, err)
	logStr := string(logContent)

	// Should contain server startup messages
	assert.NotEmpty(t, logStr, "Log file should not be empty")

	// Cleanup
	_, _ = env.runCLI(t, "server", "--stop")
	time.Sleep(500 * time.Millisecond)
}

func TestE2E_Daemon_GracefulShutdown(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		_ = env.ServerCmd.Process.Kill()
		_ = env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Ensure daemon is stopped on test completion
	t.Cleanup(func() {
		_, _ = env.runCLI(t, "server", "--stop")
		time.Sleep(500 * time.Millisecond)
	})

	// Register project and create comment
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Start daemon
	_, err = env.runCLI(t, "server", "--daemon")
	require.NoError(t, err)
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second))

	// Create a comment via API
	comment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "Test comment",
	}
	resp := env.postJSON(t, "/api/comments", comment)
	_ = resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)

	// Stop daemon gracefully
	output, err := env.runCLI(t, "server", "--stop")
	require.NoError(t, err)
	assert.Contains(t, output, "Sent SIGTERM")

	// Wait for shutdown
	time.Sleep(500 * time.Millisecond)

	// Restart daemon
	_, err = env.runCLI(t, "server", "--daemon")
	require.NoError(t, err)
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second))

	// Verify comment still exists (database was properly closed)
	output, err = env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Test comment")

	// Cleanup
	_, _ = env.runCLI(t, "server", "--stop")
	time.Sleep(500 * time.Millisecond)
}

func TestE2E_Daemon_InvalidPIDFile(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		_ = env.ServerCmd.Process.Kill()
		_ = env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Create a PID file with invalid content
	pidFile := filepath.Join(env.DataDir, "server.pid")
	err := os.WriteFile(pidFile, []byte("not-a-number"), 0644)
	require.NoError(t, err)

	// Status should handle invalid PID gracefully
	output, err := env.runCLI(t, "server", "--status")
	assert.Error(t, err)
	// Should either report error or clean up and say not running
	assert.True(t,
		strings.Contains(output, "Server is not running") ||
			strings.Contains(output, "invalid PID"),
		"Should handle invalid PID file")
}

func TestE2E_Daemon_ProcessIsolation(t *testing.T) {
	env := setupE2E(t)

	// Kill the foreground server started by setupE2E
	if env.ServerCmd.Process != nil {
		_ = env.ServerCmd.Process.Kill()
		_ = env.ServerCmd.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	// Ensure daemon is stopped on test completion
	t.Cleanup(func() {
		_, _ = env.runCLI(t, "server", "--stop")
		time.Sleep(500 * time.Millisecond)
	})

	// Start daemon
	_, err := env.runCLI(t, "server", "--daemon")
	require.NoError(t, err)
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second))

	// Read PID
	pidFile := filepath.Join(env.DataDir, "server.pid")
	pidData, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	daemonPID, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	require.NoError(t, err)

	// Verify daemon process is different from our test process
	testPID := os.Getpid()
	assert.NotEqual(t, testPID, daemonPID, "Daemon should run in separate process")

	// Verify daemon is still running after we continue
	time.Sleep(100 * time.Millisecond)
	process, err := os.FindProcess(daemonPID)
	require.NoError(t, err)
	err = process.Signal(syscall.Signal(0))
	assert.NoError(t, err, "Daemon should still be running")

	// Cleanup
	_, _ = env.runCLI(t, "server", "--stop")
	time.Sleep(500 * time.Millisecond)
}
