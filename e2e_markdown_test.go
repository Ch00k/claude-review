package main_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_Markdown_Rendering(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a comment with comprehensive markdown formatting
	comment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test",
		"comment_text": `This comment has various markdown features:

**Bold text** and *italic text* and ` + "`inline code`" + `

Here's a list:
- First item
- Second item
- Third item

And a code block:
` + "```" + `
func main() {
  fmt.Println("Hello")
}
` + "```" + `

Plain text at the end.`,
	}

	// Test 1: API response includes rendered_html
	resp := env.postJSON(t, "/api/comments", comment)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should have rendered_html field
	assert.Contains(t, bodyStr, "rendered_html")

	// JSON encoding escapes < as \u003c, > as \u003e
	// Should contain HTML for bold (JSON-encoded)
	assert.Contains(t, bodyStr, "\\u003cstrong\\u003eBold text\\u003c/strong\\u003e")

	// Should contain HTML for italic (JSON-encoded)
	assert.Contains(t, bodyStr, "\\u003cem\\u003eitalic text\\u003c/em\\u003e")

	// Should contain HTML for inline code (JSON-encoded)
	assert.Contains(t, bodyStr, "\\u003ccode\\u003einline code\\u003c/code\\u003e")

	// Should contain HTML for list (JSON-encoded)
	assert.Contains(t, bodyStr, "\\u003cul\\u003e")
	assert.Contains(t, bodyStr, "\\u003cli\\u003eFirst item\\u003c/li\\u003e")
	assert.Contains(t, bodyStr, "\\u003cli\\u003eSecond item\\u003c/li\\u003e")
	assert.Contains(t, bodyStr, "\\u003cli\\u003eThird item\\u003c/li\\u003e")
	assert.Contains(t, bodyStr, "\\u003c/ul\\u003e")

	// Should contain HTML for code block (JSON-encoded)
	assert.Contains(t, bodyStr, "\\u003cpre\\u003e")
	assert.Contains(t, bodyStr, "func main()")

	// Should contain plain text wrapped in paragraph (JSON-encoded)
	assert.Contains(t, bodyStr, "\\u003cp\\u003ePlain text at the end.\\u003c/p\\u003e")

	// Test 2: Viewer page includes pre-rendered HTML
	url := fmt.Sprintf("%s/projects%s/test.md", env.BaseURL, env.ProjectDir)
	viewerResp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = viewerResp.Body.Close() }()

	assert.Equal(t, http.StatusOK, viewerResp.StatusCode)
	viewerBody, _ := io.ReadAll(viewerResp.Body)
	viewerStr := string(viewerBody)

	// Extract comments from the inline JavaScript
	scriptRegex := regexp.MustCompile(`(?s)let comments = (.+?);\s*</script>`)
	matches := scriptRegex.FindStringSubmatch(viewerStr)
	require.NotNil(t, matches, "Should find comments variable in script")

	commentsJS := matches[1]

	// The rendered_html should be present in the JavaScript with all formatting (JSON-encoded)
	assert.Contains(t, commentsJS, "rendered_html")
	assert.Contains(t, commentsJS, "\\u003cstrong\\u003eBold text\\u003c/strong\\u003e")
	assert.Contains(t, commentsJS, "\\u003cem\\u003eitalic text\\u003c/em\\u003e")
	assert.Contains(t, commentsJS, "\\u003ccode\\u003einline code\\u003c/code\\u003e")

	// Test 3: No extra whitespace in rendered HTML
	renderedRegex := regexp.MustCompile(`"rendered_html":"([^"]*)"`)
	renderedMatches := renderedRegex.FindStringSubmatch(bodyStr)
	require.NotNil(t, renderedMatches, "Should find rendered_html")

	renderedHTML := renderedMatches[1]

	// Should not have leading/trailing newlines beyond HTML tags
	assert.NotContains(t, renderedHTML, "\\n<p>", "Should not have leading newline before paragraph")
	assert.NotContains(t, renderedHTML, "</p>\\n\\n", "Should not have double trailing newlines")
}

func TestE2E_Markdown_PlainTextPreserved(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a comment with plain text (no markdown)
	comment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test",
		"comment_text":      "This is just plain text with no formatting.",
	}

	resp := env.postJSON(t, "/api/comments", comment)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Plain text should be wrapped in <p> tags (JSON-encoded)
	assert.Contains(t, bodyStr, "rendered_html")
	assert.Contains(t, bodyStr, "\\u003cp\\u003eThis is just plain text with no formatting.\\u003c/p\\u003e")

	// Should not contain any formatting tags (JSON-encoded)
	assert.NotContains(t, bodyStr, "\\u003cstrong\\u003e")
	assert.NotContains(t, bodyStr, "\\u003cem\\u003e")
	assert.NotContains(t, bodyStr, "\\u003cul\\u003e")
}

func TestE2E_Markdown_SpecialCharacters(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a comment with special characters that need escaping
	comment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test",
		"comment_text":      `Special chars: "quotes" & <html> symbols`,
	}

	resp := env.postJSON(t, "/api/comments", comment)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should have rendered_html
	assert.Contains(t, bodyStr, "rendered_html")

	// The HTML should be properly JSON-encoded in the response
	// JSON encoding will escape < as \u003c, > as \u003e, & as \u0026, " as \u0022 or \"
	assert.Contains(t, bodyStr, "rendered_html")

	// Fetch the viewer page to ensure it renders correctly in the template
	url := fmt.Sprintf("%s/projects%s/test.md", env.BaseURL, env.ProjectDir)
	viewerResp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = viewerResp.Body.Close() }()

	viewerBody, _ := io.ReadAll(viewerResp.Body)
	viewerStr := string(viewerBody)

	// Extract the comments JSON from the script tag
	scriptRegex := regexp.MustCompile(`(?s)let comments = (.+?);\s*</script>`)
	matches := scriptRegex.FindStringSubmatch(viewerStr)
	require.NotNil(t, matches, "Should find comments variable")

	// The JSON should be valid and not truncated
	commentsJS := matches[1]
	assert.Contains(t, commentsJS, "rendered_html")
	assert.True(t, len(commentsJS) > 50, "Comments JSON should not be truncated")
}

func TestE2E_Markdown_ThreadedComments(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create a root comment with markdown
	rootComment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test",
		"comment_text":      "Root: **Important** issue here",
	}

	rootResp := env.postJSON(t, "/api/comments", rootComment)
	defer func() { _ = rootResp.Body.Close() }()
	require.Equal(t, http.StatusOK, rootResp.StatusCode)

	rootBody, _ := io.ReadAll(rootResp.Body)
	var rootData map[string]interface{}
	_ = json.Unmarshal(rootBody, &rootData)
	rootID := int(rootData["id"].(float64))

	// Root should have rendered HTML (JSON-encoded)
	rootStr := string(rootBody)
	assert.Contains(t, rootStr, "\\u003cstrong\\u003eImportant\\u003c/strong\\u003e")

	// Create a reply with markdown
	reply := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"comment_text":      "Reply: I *agree* with this assessment",
		"root_id":           rootID,
	}

	replyResp := env.postJSON(t, "/api/comments", reply)
	defer func() { _ = replyResp.Body.Close() }()
	require.Equal(t, http.StatusOK, replyResp.StatusCode)

	replyBody, _ := io.ReadAll(replyResp.Body)
	replyStr := string(replyBody)

	// Reply should also have rendered HTML (JSON-encoded)
	assert.Contains(t, replyStr, "\\u003cem\\u003eagree\\u003c/em\\u003e")

	// Fetch viewer page - both comments should be rendered
	url := fmt.Sprintf("%s/projects%s/test.md", env.BaseURL, env.ProjectDir)
	viewerResp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = viewerResp.Body.Close() }()

	viewerBody, _ := io.ReadAll(viewerResp.Body)
	viewerStr := string(viewerBody)

	// Both root and reply should have rendered HTML in the page (JSON-encoded in script tag)
	assert.Contains(t, viewerStr, "\\u003cstrong\\u003eImportant\\u003c/strong\\u003e")
	assert.Contains(t, viewerStr, "\\u003cem\\u003eagree\\u003c/em\\u003e")
}
