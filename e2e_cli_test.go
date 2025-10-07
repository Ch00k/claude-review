// +build e2e

package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_CLI_Register tests the register command variants
func TestE2E_CLI_Register(t *testing.T) {
	env := setupE2E(t)

	t.Run("register with explicit project flag", func(t *testing.T) {
		output, err := env.runCLI(t, "register", "--project", env.ProjectDir)
		require.NoError(t, err)
		assert.Contains(t, output, "Registered project")
		assert.Contains(t, output, env.ProjectDir)
	})

	t.Run("register without project flag uses current directory", func(t *testing.T) {
		// Change to project directory
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		os.Chdir(env.ProjectDir)

		output, err := env.runCLI(t, "register")
		require.NoError(t, err)
		assert.Contains(t, output, "Registered project")
		assert.Contains(t, output, env.ProjectDir)
	})

	t.Run("register same project twice is idempotent", func(t *testing.T) {
		// First registration
		output1, err1 := env.runCLI(t, "register", "--project", env.ProjectDir)
		require.NoError(t, err1)
		assert.Contains(t, output1, "Registered project")

		// Second registration should also succeed (idempotent)
		output2, err2 := env.runCLI(t, "register", "--project", env.ProjectDir)
		require.NoError(t, err2)
		assert.Contains(t, output2, "Registered project")
	})

	t.Run("register with dot as project directory", func(t *testing.T) {
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		os.Chdir(env.ProjectDir)

		output, err := env.runCLI(t, "register", "--project", ".")
		require.NoError(t, err)
		assert.Contains(t, output, "Registered project")
		assert.Contains(t, output, env.ProjectDir)
	})
}

// TestE2E_CLI_Address tests the address command edge cases
// Note: Full workflow testing is covered in TestE2E_CommentWorkflow
func TestE2E_CLI_Address(t *testing.T) {
	t.Run("address without file flag shows error", func(t *testing.T) {
		env := setupE2E(t)
		_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
		require.NoError(t, err)

		output, err := env.runCLI(t, "address", "--project", env.ProjectDir)
		require.Error(t, err)
		assert.Contains(t, output, "--file flag is required")
	})

	t.Run("address with no comments", func(t *testing.T) {
		env := setupE2E(t)
		_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
		require.NoError(t, err)

		output, err := env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
		require.NoError(t, err)
		assert.Contains(t, output, "No unresolved comments")
	})

	t.Run("address strips @ prefix from file path", func(t *testing.T) {
		env := setupE2E(t)
		_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
		require.NoError(t, err)

		// Use @ prefix in CLI (should work identically to without prefix)
		output, err := env.runCLI(t, "address", "--file", "@test.md", "--project", env.ProjectDir)
		require.NoError(t, err)
		assert.Contains(t, output, "No unresolved comments")
	})

	t.Run("address without project flag uses current directory", func(t *testing.T) {
		env := setupE2E(t)
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		os.Chdir(env.ProjectDir)

		_, err := env.runCLI(t, "register")
		require.NoError(t, err)

		output, err := env.runCLI(t, "address", "--file", "test.md")
		require.NoError(t, err)
		assert.Contains(t, output, "No unresolved comments")
	})
}

// TestE2E_CLI_Resolve tests the resolve command edge cases
// Note: Full workflow testing is covered in TestE2E_CommentWorkflow
func TestE2E_CLI_Resolve(t *testing.T) {
	t.Run("resolve without file flag shows error", func(t *testing.T) {
		env := setupE2E(t)
		_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
		require.NoError(t, err)

		output, err := env.runCLI(t, "resolve", "--project", env.ProjectDir)
		require.Error(t, err)
		assert.Contains(t, output, "--file flag is required")
	})

	t.Run("resolve with no comments", func(t *testing.T) {
		env := setupE2E(t)
		_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
		require.NoError(t, err)

		output, err := env.runCLI(t, "resolve", "--file", "test.md", "--project", env.ProjectDir)
		require.NoError(t, err)
		assert.Contains(t, output, "No unresolved comments")
	})

	t.Run("resolve strips @ prefix from file path", func(t *testing.T) {
		env := setupE2E(t)
		_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
		require.NoError(t, err)

		// Resolve with @ prefix (should work identically to without prefix)
		output, err := env.runCLI(t, "resolve", "--file", "@test.md", "--project", env.ProjectDir)
		require.NoError(t, err)
		assert.Contains(t, output, "No unresolved comments")
	})

	t.Run("resolve without project flag uses current directory", func(t *testing.T) {
		env := setupE2E(t)
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		os.Chdir(env.ProjectDir)

		_, err := env.runCLI(t, "register")
		require.NoError(t, err)

		output, err := env.runCLI(t, "resolve", "--file", "test.md")
		require.NoError(t, err)
		assert.Contains(t, output, "No unresolved comments")
	})
}

// TestE2E_CLI_Review tests the review command
func TestE2E_CLI_Review(t *testing.T) {
	// Create isolated environment without starting server
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	projectDir := filepath.Join(tempDir, "project")
	binaryPath := filepath.Join(tempDir, "claude-review")

	require.NoError(t, os.MkdirAll(dataDir, 0755))
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	createTestMarkdownFiles(t, projectDir)

	// Build binary
	buildCmd := exec.Command("go", "build", "-cover", "-o", binaryPath, ".")
	require.NoError(t, buildCmd.Run())

	env := &TestEnv{
		TempDir:    tempDir,
		DataDir:    dataDir,
		ProjectDir: projectDir,
		Port:       "14780", // Different port to avoid conflict
		BinaryPath: binaryPath,
	}

	// Ensure daemon is stopped on test completion
	t.Cleanup(func() {
		env.runCLI(t, "server", "--stop")
		time.Sleep(500 * time.Millisecond)
	})

	t.Run("review without file flag shows error", func(t *testing.T) {
		output, err := env.runCLI(t, "review", "--project", env.ProjectDir)
		require.Error(t, err)
		assert.Contains(t, output, "--file flag is required")
	})

	t.Run("review outputs URL and starts daemon", func(t *testing.T) {
		output, err := env.runCLI(t, "review", "--file", "test.md", "--project", env.ProjectDir)
		require.NoError(t, err)
		assert.Contains(t, output, "Open this URL")
		assert.Contains(t, output, fmt.Sprintf("http://localhost:%s/projects", env.Port))
		assert.Contains(t, output, "test.md")
	})

	t.Run("review strips @ prefix from file path", func(t *testing.T) {
		output, err := env.runCLI(t, "review", "--file", "@simple.md", "--project", env.ProjectDir)
		require.NoError(t, err)
		assert.Contains(t, output, "simple.md")
		assert.NotContains(t, output, "@simple.md")
	})
}

// TestE2E_CLI_Version tests the version command
func TestE2E_CLI_Version(t *testing.T) {
	env := setupE2E(t)

	output, err := env.runCLI(t, "version")
	require.NoError(t, err)
	// Should output version string (currently "dev" or semver)
	assert.NotEmpty(t, strings.TrimSpace(output))
}

// TestE2E_CLI_UnknownCommand tests error handling for unknown commands
func TestE2E_CLI_UnknownCommand(t *testing.T) {
	env := setupE2E(t)

	output, err := env.runCLI(t, "nonexistent-command")
	require.Error(t, err)
	assert.Contains(t, output, "Unknown command")
}

// TestE2E_CLI_NoCommand tests help output when no command is provided
func TestE2E_CLI_NoCommand(t *testing.T) {
	env := setupE2E(t)

	// Run with no arguments
	cmd := exec.Command(env.BinaryPath)
	cmd.Env = append(os.Environ(),
		"CR_DATA_DIR="+env.DataDir,
		"CR_LISTEN_PORT="+env.Port,
		"GOCOVERDIR=tmp/coverage",
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err) // Should exit with error
	outputStr := string(output)

	assert.Contains(t, outputStr, "Usage: claude-review <command>")
	assert.Contains(t, outputStr, "Commands:")
	assert.Contains(t, outputStr, "server")
	assert.Contains(t, outputStr, "register")
	assert.Contains(t, outputStr, "address")
	assert.Contains(t, outputStr, "resolve")
	assert.Contains(t, outputStr, "review")
}

// TestE2E_CLI_Install tests the install command for slash commands
func TestE2E_CLI_Install(t *testing.T) {
	// Create isolated environment with custom home directory
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	// Build binary
	binaryPath := filepath.Join(tempDir, "claude-review")
	buildCmd := exec.Command("go", "build", "-cover", "-o", binaryPath, ".")
	require.NoError(t, buildCmd.Run())

	t.Run("install creates commands directory and files", func(t *testing.T) {
		// Run install command with custom HOME
		cmd := exec.Command(binaryPath, "install")
		cmd.Env = append(os.Environ(),
			"HOME="+homeDir,
			"GOCOVERDIR=tmp/coverage",
		)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		outputStr := string(output)

		// Check output messages
		assert.Contains(t, outputStr, "Successfully installed")
		assert.Contains(t, outputStr, "slash command")
		assert.Contains(t, outputStr, filepath.Join(homeDir, ".claude", "commands"))

		// Verify commands directory was created
		commandsDir := filepath.Join(homeDir, ".claude", "commands")
		stat, err := os.Stat(commandsDir)
		require.NoError(t, err)
		assert.True(t, stat.IsDir())

		// Verify slash command files were installed
		entries, err := os.ReadDir(commandsDir)
		require.NoError(t, err)
		assert.Greater(t, len(entries), 0, "Should install at least one slash command")

		// Check for expected slash commands
		var foundCommands []string
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".md") {
				foundCommands = append(foundCommands, entry.Name())
			}
		}

		// Verify expected commands exist
		assert.Contains(t, foundCommands, "cr-review.md")
		assert.Contains(t, foundCommands, "cr-address.md")

		// Verify output lists the installed commands
		assert.Contains(t, outputStr, "/cr-review")
		assert.Contains(t, outputStr, "/cr-address")
	})

	t.Run("install is idempotent and overwrites existing files", func(t *testing.T) {
		commandsDir := filepath.Join(homeDir, ".claude", "commands")

		// Get initial file count
		initialEntries, err := os.ReadDir(commandsDir)
		require.NoError(t, err)
		initialCount := len(initialEntries)

		// Modify one of the files
		testFile := filepath.Join(commandsDir, "cr-review.md")
		require.NoError(t, os.WriteFile(testFile, []byte("MODIFIED CONTENT"), 0644))

		// Run install again
		cmd := exec.Command(binaryPath, "install")
		cmd.Env = append(os.Environ(),
			"HOME="+homeDir,
			"GOCOVERDIR=tmp/coverage",
		)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Verify file was overwritten
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.NotContains(t, string(content), "MODIFIED CONTENT")

		// Verify same number of files (no duplicates)
		finalEntries, err := os.ReadDir(commandsDir)
		require.NoError(t, err)
		assert.Equal(t, initialCount, len(finalEntries), "Should not create duplicate files")

		// Verify output still shows success
		outputStr := string(output)
		assert.Contains(t, outputStr, "Successfully installed")
	})

	t.Run("install creates missing parent directories", func(t *testing.T) {
		// Create fresh home directory without .claude
		freshHome := filepath.Join(tempDir, "fresh-home")
		require.NoError(t, os.MkdirAll(freshHome, 0755))

		cmd := exec.Command(binaryPath, "install")
		cmd.Env = append(os.Environ(),
			"HOME="+freshHome,
			"GOCOVERDIR=tmp/coverage",
		)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Verify .claude/commands was created
		commandsDir := filepath.Join(freshHome, ".claude", "commands")
		stat, err := os.Stat(commandsDir)
		require.NoError(t, err)
		assert.True(t, stat.IsDir())

		// Verify files were installed
		outputStr := string(output)
		assert.Contains(t, outputStr, "Successfully installed")
	})

	t.Run("install verifies file contents are correct", func(t *testing.T) {
		commandsDir := filepath.Join(homeDir, ".claude", "commands")

		// Read installed cr-address.md
		addressFile := filepath.Join(commandsDir, "cr-address.md")
		content, err := os.ReadFile(addressFile)
		require.NoError(t, err)
		contentStr := string(content)

		// Verify it contains expected content (should reference claude-review command)
		assert.Contains(t, contentStr, "claude-review")
		assert.Contains(t, contentStr, "address")

		// Read installed cr-review.md
		reviewFile := filepath.Join(commandsDir, "cr-review.md")
		content, err = os.ReadFile(reviewFile)
		require.NoError(t, err)
		contentStr = string(content)

		// Verify it contains expected content
		assert.Contains(t, contentStr, "claude-review")
		assert.Contains(t, contentStr, "review")
	})

	t.Run("install file permissions are correct", func(t *testing.T) {
		commandsDir := filepath.Join(homeDir, ".claude", "commands")

		// Check directory permissions
		dirStat, err := os.Stat(commandsDir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), dirStat.Mode().Perm())

		// Check file permissions
		entries, err := os.ReadDir(commandsDir)
		require.NoError(t, err)

		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".md") {
				filePath := filepath.Join(commandsDir, entry.Name())
				fileStat, err := os.Stat(filePath)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0644), fileStat.Mode().Perm(),
					"File %s should have 0644 permissions", entry.Name())
			}
		}
	})
}
