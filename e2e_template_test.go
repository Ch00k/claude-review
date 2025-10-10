// +build e2e

package main_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_Template_JSONEncoding(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a comment with special characters to ensure proper escaping
	comment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     `Test "with" quotes & symbols`,
		"comment_text":      `This has "quotes" and 'apostrophes' and <html> & symbols`,
	}

	resp := env.postJSON(t, "/api/comments", comment)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Fetch the viewer page
	url := fmt.Sprintf("%s/projects%s/test.md", env.BaseURL, env.ProjectDir)
	viewerResp, err := http.Get(url)
	require.NoError(t, err)
	defer viewerResp.Body.Close()

	assert.Equal(t, http.StatusOK, viewerResp.StatusCode)
	body, _ := io.ReadAll(viewerResp.Body)
	bodyStr := string(body)

	// Extract the <script> tag with template variables
	// More flexible regex that allows for whitespace variations
	scriptRegex := regexp.MustCompile(`(?s)<script>.*?const projectDir = (.+?);.*?const filePath = (.+?);.*?let comments = (.+?);.*?</script>`)
	matches := scriptRegex.FindStringSubmatch(bodyStr)
	if matches == nil {
		t.Logf("HTML body:\n%s\n", bodyStr)
	}
	require.NotNil(t, matches, "Should find template variables script tag")
	require.Len(t, matches, 4, "Should extract projectDir, filePath, and comments")

	projectDirJS := strings.TrimSpace(matches[1])
	filePathJS := strings.TrimSpace(matches[2])
	commentsJS := strings.TrimSpace(matches[3])

	// Test 1: projectDir should be valid JSON string
	var projectDirDecoded string
	err = json.Unmarshal([]byte(projectDirJS), &projectDirDecoded)
	require.NoError(t, err, "projectDir should be valid JSON: %s", projectDirJS)
	assert.Equal(t, env.ProjectDir, projectDirDecoded, "projectDir should match")

	// Test 2: filePath should be valid JSON string
	var filePathDecoded string
	err = json.Unmarshal([]byte(filePathJS), &filePathDecoded)
	require.NoError(t, err, "filePath should be valid JSON: %s", filePathJS)
	assert.Equal(t, "test.md", filePathDecoded, "filePath should match")

	// Test 3: comments should be valid JSON array
	var commentsDecoded []map[string]interface{}
	err = json.Unmarshal([]byte(commentsJS), &commentsDecoded)
	require.NoError(t, err, "comments should be valid JSON: %s", commentsJS)
	require.Len(t, commentsDecoded, 1, "Should have one comment")

	// Test 4: Verify special characters are properly escaped
	actualComment := commentsDecoded[0]
	assert.Equal(t, `Test "with" quotes & symbols`, actualComment["selected_text"])
	assert.Equal(t, `This has "quotes" and 'apostrophes' and <html> & symbols`, actualComment["comment_text"])

	// Test 5: Verify no raw/unescaped strings in JavaScript
	// The script should NOT contain unquoted paths like: const projectDir = /home/user/project;
	unquotedPathRegex := regexp.MustCompile(`const projectDir = [^"'\[{]`)
	assert.False(t, unquotedPathRegex.MatchString(bodyStr),
		"projectDir should be properly quoted as JSON string, not raw JavaScript")
}

func TestE2E_Template_EmptyComments(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Fetch viewer page with no comments
	url := fmt.Sprintf("%s/projects%s/test.md", env.BaseURL, env.ProjectDir)
	viewerResp, err := http.Get(url)
	require.NoError(t, err)
	defer viewerResp.Body.Close()

	assert.Equal(t, http.StatusOK, viewerResp.StatusCode)
	body, _ := io.ReadAll(viewerResp.Body)
	bodyStr := string(body)

	// Extract comments variable
	scriptRegex := regexp.MustCompile(`let comments = (.+?);`)
	matches := scriptRegex.FindStringSubmatch(bodyStr)
	require.NotNil(t, matches, "Should find comments variable")
	require.Len(t, matches, 2)

	commentsJS := strings.TrimSpace(matches[1])

	// Should be valid JSON array (empty or null)
	var commentsDecoded interface{}
	err = json.Unmarshal([]byte(commentsJS), &commentsDecoded)
	require.NoError(t, err, "comments should be valid JSON even when empty: %s", commentsJS)

	// Should be either empty array or null
	if commentsDecoded != nil {
		commentsList, ok := commentsDecoded.([]interface{})
		require.True(t, ok, "comments should be an array")
		assert.Empty(t, commentsList, "comments array should be empty")
	}
}
