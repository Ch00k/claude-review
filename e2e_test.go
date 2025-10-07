// +build e2e

package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestEnv struct {
	ServerCmd  *exec.Cmd
	TempDir    string
	DataDir    string
	ProjectDir string
	Port       string
	BaseURL    string
	BinaryPath string
	LogFile    string
}

func setupE2E(t *testing.T) *TestEnv {
	t.Helper()

	// Create temp directories
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	projectDir := filepath.Join(tempDir, "project")
	binaryPath := filepath.Join(tempDir, "claude-review")

	require.NoError(t, os.MkdirAll(dataDir, 0755))
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Create test markdown files
	createTestMarkdownFiles(t, projectDir)

	// Build the binary
	t.Logf("Building binary to %s", binaryPath)
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", buildOutput)
	}
	require.NoError(t, err, "Failed to build binary")

	// Start server
	port := "14779"
	logFile := filepath.Join(tempDir, "server.log")
	serverCmd := exec.Command(binaryPath, "server")
	serverCmd.Env = append(os.Environ(),
		"CR_DATA_DIR="+dataDir,
		"CR_LISTEN_PORT="+port,
	)

	// Capture server logs
	logWriter, err := os.Create(logFile)
	require.NoError(t, err)
	serverCmd.Stdout = logWriter
	serverCmd.Stderr = logWriter

	require.NoError(t, serverCmd.Start(), "Failed to start server")

	env := &TestEnv{
		ServerCmd:  serverCmd,
		TempDir:    tempDir,
		DataDir:    dataDir,
		ProjectDir: projectDir,
		Port:       port,
		BaseURL:    "http://localhost:" + port,
		BinaryPath: binaryPath,
		LogFile:    logFile,
	}

	// Wait for server to be ready
	require.NoError(t, waitForServer(env.BaseURL, 10*time.Second), "Server did not start")
	t.Logf("Server started at %s", env.BaseURL)

	t.Cleanup(func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
			serverCmd.Wait()
		}
		logWriter.Close()

		// Print logs on failure
		if t.Failed() {
			logs, _ := os.ReadFile(logFile)
			t.Logf("Server logs:\n%s", logs)
		}
	})

	return env
}

func createTestMarkdownFiles(t *testing.T, projectDir string) {
	t.Helper()

	testMD := `# Test Document

This is a test paragraph with some content.

## Section 2

Another paragraph with more content for testing.

### Code Example

` + "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```" + `

## Conclusion

Final paragraph.
`

	simpleMD := `# Simple

Just one paragraph.
`

	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "test.md"), []byte(testMD), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "simple.md"), []byte(simpleMD), 0644))
}

func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server did not start within %v", timeout)
}

func (env *TestEnv) runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command(env.BinaryPath, args...)
	cmd.Env = append(os.Environ(),
		"CR_DATA_DIR="+env.DataDir,
		"CR_LISTEN_PORT="+env.Port,
	)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (env *TestEnv) postJSON(t *testing.T, path string, data interface{}) *http.Response {
	t.Helper()

	jsonData, err := json.Marshal(data)
	require.NoError(t, err)

	resp, err := http.Post(
		env.BaseURL+path,
		"application/json",
		bytes.NewReader(jsonData),
	)
	require.NoError(t, err)

	return resp
}

func (env *TestEnv) patchJSON(t *testing.T, path string, data interface{}) *http.Response {
	t.Helper()

	jsonData, err := json.Marshal(data)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPatch, env.BaseURL+path, bytes.NewReader(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func (env *TestEnv) delete(t *testing.T, path string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, env.BaseURL+path, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

// Tests

func TestE2E_RegisterProject(t *testing.T) {
	env := setupE2E(t)

	output, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Registered project")
	assert.Contains(t, output, env.ProjectDir)
}

func TestE2E_CommentWorkflow(t *testing.T) {
	env := setupE2E(t)

	// Register project
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a comment via API
	comment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "This needs work",
	}

	resp := env.postJSON(t, "/api/comments", comment)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	assert.NotNil(t, created["id"])
	commentID := int(created["id"].(float64))
	t.Logf("Created comment ID: %d", commentID)

	// Check comment appears in address output
	output, err := env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "This needs work")
	assert.Contains(t, output, "Test Document")
	assert.Contains(t, output, "Found 1 unresolved comment")

	// Update the comment
	updateResp := env.patchJSON(t, fmt.Sprintf("/api/comments/%d", commentID), map[string]string{
		"comment_text": "Updated feedback",
	})
	defer updateResp.Body.Close()
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)

	// Verify update in address
	output, err = env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Updated feedback")
	assert.NotContains(t, output, "This needs work")

	// Resolve comment
	output, err = env.runCLI(t, "resolve", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Resolved 1 comment")

	// Verify comment no longer appears
	output, err = env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "No unresolved comments")
}

func TestE2E_MultipleComments(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create multiple comments
	comments := []map[string]interface{}{
		{
			"project_directory": env.ProjectDir,
			"file_path":         "test.md",
			"line_start":        1,
			"line_end":          1,
			"selected_text":     "Test Document",
			"comment_text":      "First comment",
		},
		{
			"project_directory": env.ProjectDir,
			"file_path":         "test.md",
			"line_start":        5,
			"line_end":          5,
			"selected_text":     "Section 2",
			"comment_text":      "Second comment",
		},
		{
			"project_directory": env.ProjectDir,
			"file_path":         "test.md",
			"line_start":        7,
			"line_end":          7,
			"selected_text":     "Another paragraph",
			"comment_text":      "Third comment",
		},
	}

	for _, c := range comments {
		resp := env.postJSON(t, "/api/comments", c)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Verify all appear in address
	output, err := env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Found 3 unresolved comment")
	assert.Contains(t, output, "First comment")
	assert.Contains(t, output, "Second comment")
	assert.Contains(t, output, "Third comment")

	// Verify ordering by line number
	firstPos := strings.Index(output, "First comment")
	secondPos := strings.Index(output, "Second comment")
	thirdPos := strings.Index(output, "Third comment")
	assert.Less(t, firstPos, secondPos, "Comments should be ordered by line number")
	assert.Less(t, secondPos, thirdPos, "Comments should be ordered by line number")
}

func TestE2E_DeleteComment(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create comment
	comment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "To be deleted",
	}

	resp := env.postJSON(t, "/api/comments", comment)
	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	commentID := int(created["id"].(float64))

	// Delete comment
	deleteResp := env.delete(t, fmt.Sprintf("/api/comments/%d", commentID))
	defer deleteResp.Body.Close()
	assert.Equal(t, http.StatusOK, deleteResp.StatusCode)

	// Verify no longer appears
	output, err := env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "No unresolved comments")
}

func TestE2E_WebInterface_HomePage(t *testing.T) {
	env := setupE2E(t)

	// Register project first
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Visit home page
	resp, err := http.Get(env.BaseURL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	assert.Contains(t, bodyStr, env.ProjectDir)
	assert.Contains(t, bodyStr, "Claude Review")
}

func TestE2E_WebInterface_FileViewer(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Visit file viewer
	url := fmt.Sprintf("%s/projects%s/test.md", env.BaseURL, env.ProjectDir)

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check markdown rendering
	assert.Contains(t, bodyStr, "Test Document")
	assert.Contains(t, bodyStr, "data-line-start", "Should have line number attributes")
	assert.Contains(t, bodyStr, "data-line-end", "Should have line number attributes")

	// Check viewer.js is loaded
	assert.Contains(t, bodyStr, "viewer.js")
}

func TestE2E_WebInterface_DirectoryListing(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Visit project root (directory)
	url := fmt.Sprintf("%s/projects%s/", env.BaseURL, env.ProjectDir)

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should list markdown files
	assert.Contains(t, bodyStr, "test.md")
	assert.Contains(t, bodyStr, "simple.md")
}

func TestE2E_PathTraversal_Security(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a sensitive file outside project directory
	sensitiveFile := filepath.Join(env.TempDir, "secret.txt")
	require.NoError(t, os.WriteFile(sensitiveFile, []byte("SECRET DATA"), 0644))

	maliciousPaths := []string{
		"../../secret.txt",
		"..%2F..%2Fsecret.txt",
		"test.md/../../secret.txt",
		"test.md/../../../secret.txt",
	}

	for _, maliciousPath := range maliciousPaths {
		url := fmt.Sprintf("%s/projects%s/%s", env.BaseURL, env.ProjectDir, maliciousPath)
		t.Logf("Testing path traversal: %s", url)

		resp, err := http.Get(url)
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Should NOT return the secret file
		bodyStr := string(body)
		assert.NotContains(t, bodyStr, "SECRET DATA",
			"Path traversal succeeded for: %s", maliciousPath)

		// Should return 404 or error, not 200
		if resp.StatusCode == http.StatusOK {
			t.Errorf("Path traversal may have succeeded for %s: got status 200", maliciousPath)
		}
	}
}

func TestE2E_InvalidInput(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	t.Run("malformed JSON", func(t *testing.T) {
		resp, err := http.Post(
			env.BaseURL+"/api/comments",
			"application/json",
			strings.NewReader("{invalid json}"),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing required fields", func(t *testing.T) {
		comment := map[string]interface{}{
			"project_directory": env.ProjectDir,
			// Missing other required fields
		}

		resp := env.postJSON(t, "/api/comments", comment)
		defer resp.Body.Close()

		// Should fail (either 400 or 500, depending on validation)
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("update non-existent comment", func(t *testing.T) {
		resp := env.patchJSON(t, "/api/comments/99999", map[string]string{
			"comment_text": "Updated",
		})
		defer resp.Body.Close()

		// Should succeed (UPDATE affects 0 rows but doesn't error)
		// This is acceptable behavior
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("delete non-existent comment", func(t *testing.T) {
		resp := env.delete(t, "/api/comments/99999")
		defer resp.Body.Close()

		// Should succeed (DELETE affects 0 rows but doesn't error)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestE2E_MultipleFiles_Isolation(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create comments on different files
	comment1 := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "Comment on test.md",
	}

	comment2 := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "simple.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Simple",
		"comment_text":      "Comment on simple.md",
	}

	resp1 := env.postJSON(t, "/api/comments", comment1)
	resp1.Body.Close()
	resp2 := env.postJSON(t, "/api/comments", comment2)
	resp2.Body.Close()

	// Verify test.md only shows its comment
	output, err := env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Comment on test.md")
	assert.NotContains(t, output, "Comment on simple.md")

	// Verify simple.md only shows its comment
	output, err = env.runCLI(t, "address", "--file", "simple.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Comment on simple.md")
	assert.NotContains(t, output, "Comment on test.md")

	// Resolve only test.md comments
	output, err = env.runCLI(t, "resolve", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Resolved 1 comment")

	// Verify simple.md comment still unresolved
	output, err = env.runCLI(t, "address", "--file", "simple.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Comment on simple.md")
	assert.Contains(t, output, "Found 1 unresolved comment")
}

func TestE2E_NoComments(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Address with no comments
	output, err := env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "No unresolved comments")

	// Resolve with no comments
	output, err = env.runCLI(t, "resolve", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "No unresolved comments")
}
