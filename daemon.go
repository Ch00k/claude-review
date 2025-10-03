package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
)

// getPIDFilePath returns the path to the PID file
func getPIDFilePath() (string, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "server.pid"), nil
}

// writePIDFile writes the current process PID to the PID file
func writePIDFile() error {
	pidFile, err := getPIDFilePath()
	if err != nil {
		return err
	}

	// Check if already running
	if isServerRunning() {
		return fmt.Errorf("server is already running (PID file exists: %s)", pidFile)
	}

	pid := os.Getpid()
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// removePIDFile removes the PID file
func removePIDFile() error {
	pidFile, err := getPIDFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// readPID reads the PID from the PID file
func readPID() (int, error) {
	pidFile, err := getPIDFilePath()
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// isServerRunning checks if the server is running based on PID file
func isServerRunning() bool {
	pid, err := readPID()
	if err != nil || pid == 0 {
		return false
	}

	if !isProcessRunning(pid) {
		// PID file exists but process is not running - clean up stale PID file
		_ = removePIDFile()
		return false
	}

	return true
}

// daemonize starts the process as a daemon using double-fork
func daemonize() error {
	// Check if server is already running
	if isServerRunning() {
		pidFile, _ := getPIDFilePath()
		return fmt.Errorf("server is already running (PID file exists: %s)", pidFile)
	}

	// Get the executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Prepare log file paths
	dataDir, err := getDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	logFile := filepath.Join(dataDir, "server.log")

	// Create log file
	outFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		_ = outFile.Close()
	}()

	// First fork
	cmd := exec.Command(executable, "server", "--daemon-child")
	cmd.Stdout = outFile
	cmd.Stderr = outFile
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Parent exits immediately
	pidFile, _ := getPIDFilePath()
	fmt.Printf("Server started as daemon\n")
	fmt.Printf("PID file: %s\n", pidFile)
	fmt.Printf("Log file: %s\n", logFile)
	return nil
}

// setupSignalHandlers sets up graceful shutdown on SIGTERM/SIGINT
func setupSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v, shutting down gracefully...", sig)

		// Cleanup PID file
		if err := removePIDFile(); err != nil {
			log.Printf("Failed to remove PID file: %v", err)
		}

		// Close database
		if db != nil {
			_ = db.Close()
		}

		os.Exit(0)
	}()
}

// stopDaemon stops the running daemon
func stopDaemon() error {
	pid, err := readPID()
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	if pid == 0 {
		return fmt.Errorf("server is not running (no PID file found)")
	}

	if !isProcessRunning(pid) {
		// Clean up stale PID file
		_ = removePIDFile()
		return fmt.Errorf("server is not running (stale PID file)")
	}

	// Send SIGTERM
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	fmt.Printf("Sent SIGTERM to server (PID: %d)\n", pid)
	return nil
}

// statusDaemon checks and prints the daemon status
// Returns nil (exit 0) if server is running, error (exit 1) if not running
func statusDaemon() error {
	pid, err := readPID()
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	if pid == 0 {
		fmt.Println("Server is not running")
		return fmt.Errorf("server not running")
	}

	if !isProcessRunning(pid) {
		// Clean up stale PID file
		_ = removePIDFile()
		fmt.Println("Server is not running (stale PID file removed)")
		return fmt.Errorf("server not running")
	}

	pidFile, _ := getPIDFilePath()
	dataDir, _ := getDataDir()
	logFile := filepath.Join(dataDir, "server.log")

	fmt.Printf("Server is running (PID: %d)\n", pid)
	fmt.Printf("PID file: %s\n", pidFile)
	fmt.Printf("Log file: %s\n", logFile)
	return nil
}
