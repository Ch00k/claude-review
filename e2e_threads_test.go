package main_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_ThreadedComments_BasicReply(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create root comment via API (from user)
	rootComment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "What about in_progress as a status?",
		"author":            "user",
	}

	resp := env.postJSON(t, "/api/comments", rootComment)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	rootID := int(created["id"].(float64))
	t.Logf("Created root comment ID: %d", rootID)

	// Reply to comment via CLI (as agent)
	output, err := env.runCLI(
		t,
		"reply",
		"--comment-id",
		fmt.Sprintf("%d", rootID),
		"--message",
		"in_progress could work. We should also consider...",
	)
	require.NoError(t, err)
	assert.Contains(t, output, "Reply added")

	// Verify thread structure in address output
	output, err = env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "What about in_progress as a status?")
	assert.Contains(t, output, "Reply from Agent:")
	assert.Contains(t, output, "in_progress could work")
}

func TestE2E_ThreadedComments_MultipleReplies(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create root comment
	rootComment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        5,
		"line_end":          5,
		"selected_text":     "Section 2",
		"comment_text":      "Should we support rollback?",
		"author":            "user",
	}

	resp := env.postJSON(t, "/api/comments", rootComment)
	defer func() { _ = resp.Body.Close() }()
	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	rootID := int(created["id"].(float64))

	// Add agent reply
	_, err = env.runCLI(
		t,
		"reply",
		"--comment-id",
		fmt.Sprintf("%d", rootID),
		"--message",
		"Rollback could be useful for failed deployments",
	)
	require.NoError(t, err)

	// Add user reply (simulated via API)
	userReply := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"comment_text":      "Let's add it in v2",
		"author":            "user",
		"root_id":           rootID,
	}
	resp2 := env.postJSON(t, "/api/comments", userReply)
	_ = resp2.Body.Close()

	// Verify thread shows all replies in order
	output, err := env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Check ordering
	rootPos := strings.Index(output, "Should we support rollback?")
	agentPos := strings.Index(output, "Rollback could be useful")
	userPos := strings.Index(output, "Let's add it in v2")

	assert.Less(t, rootPos, agentPos, "Root should appear before agent reply")
	assert.Less(t, agentPos, userPos, "Agent reply should appear before user reply")
}

func TestE2E_ThreadedComments_ResolveByCommentID(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create root comment
	rootComment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "First thread",
		"author":            "user",
	}
	resp := env.postJSON(t, "/api/comments", rootComment)
	defer func() { _ = resp.Body.Close() }()
	var created1 map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created1))
	rootID1 := int(created1["id"].(float64))

	// Add reply to first thread
	_, err = env.runCLI(t, "reply", "--comment-id", fmt.Sprintf("%d", rootID1), "--message", "Reply to first")
	require.NoError(t, err)

	// Create second root comment
	rootComment2 := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        5,
		"line_end":          5,
		"selected_text":     "Section 2",
		"comment_text":      "Second thread",
		"author":            "user",
	}
	resp2 := env.postJSON(t, "/api/comments", rootComment2)
	defer func() { _ = resp2.Body.Close() }()
	var created2 map[string]interface{}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&created2))

	// Verify both threads are unresolved
	output, err := env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "First thread")
	assert.Contains(t, output, "Second thread")

	// Resolve only first thread by comment ID
	output, err = env.runCLI(t, "resolve", "--comment-id", fmt.Sprintf("%d", rootID1))
	require.NoError(t, err)
	assert.Contains(t, output, "Resolved thread")

	// Verify only second thread remains unresolved
	output, err = env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.NotContains(t, output, "First thread")
	assert.Contains(t, output, "Second thread")
	assert.Contains(t, output, "Found 1 unresolved comment")
}

func TestE2E_ThreadedComments_ResolveEntireThread(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create root comment with multiple replies
	rootComment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "Root comment",
		"author":            "user",
	}
	resp := env.postJSON(t, "/api/comments", rootComment)
	defer func() { _ = resp.Body.Close() }()
	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	rootID := int(created["id"].(float64))

	// Add two replies
	_, err = env.runCLI(t, "reply", "--comment-id", fmt.Sprintf("%d", rootID), "--message", "First reply")
	require.NoError(t, err)
	_, err = env.runCLI(t, "reply", "--comment-id", fmt.Sprintf("%d", rootID), "--message", "Second reply")
	require.NoError(t, err)

	// Resolve the thread
	output, err := env.runCLI(t, "resolve", "--comment-id", fmt.Sprintf("%d", rootID))
	require.NoError(t, err)
	assert.Contains(t, output, "Resolved thread")

	// Verify entire thread is resolved (root + replies)
	output, err = env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "No unresolved comments")
}

func TestE2E_ThreadedComments_ReplyValidation(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	t.Run("reply to non-existent comment", func(t *testing.T) {
		output, err := env.runCLI(t, "reply", "--comment-id", "99999", "--message", "This should fail")
		assert.Error(t, err)
		assert.Contains(t, output, "not found")
	})

	t.Run("reply to reply instead of root", func(t *testing.T) {
		// Create root comment
		rootComment := map[string]interface{}{
			"project_directory": env.ProjectDir,
			"file_path":         "test.md",
			"line_start":        1,
			"line_end":          1,
			"selected_text":     "Test Document",
			"comment_text":      "Root",
			"author":            "user",
		}
		resp := env.postJSON(t, "/api/comments", rootComment)
		defer func() { _ = resp.Body.Close() }()
		var created map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
		rootID := int(created["id"].(float64))

		// Create first reply
		_, err := env.runCLI(t, "reply", "--comment-id", fmt.Sprintf("%d", rootID), "--message", "First reply")
		require.NoError(t, err)

		// Extract reply ID from output or query database
		// For now, we'll use API to get the reply
		replyComment := map[string]interface{}{
			"project_directory": env.ProjectDir,
			"file_path":         "test.md",
			"comment_text":      "Manual reply",
			"author":            "agent",
			"root_id":           rootID,
		}
		resp2 := env.postJSON(t, "/api/comments", replyComment)
		defer func() { _ = resp2.Body.Close() }()
		var createdReply map[string]interface{}
		require.NoError(t, json.NewDecoder(resp2.Body).Decode(&createdReply))
		replyID := int(createdReply["id"].(float64))

		// Try to reply to the reply (should fail - two-level structure only)
		output, err := env.runCLI(t, "reply", "--comment-id", fmt.Sprintf("%d", replyID), "--message", "Reply to reply")
		assert.Error(t, err)
		assert.Contains(t, output, "can only reply to root comments")
	})
}

func TestE2E_ThreadedComments_ResolveViaAPI(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create root comment with replies
	rootComment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "Root comment",
		"author":            "user",
	}
	resp := env.postJSON(t, "/api/comments", rootComment)
	defer func() { _ = resp.Body.Close() }()
	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	rootID := int(created["id"].(float64))

	// Add reply
	_, err = env.runCLI(t, "reply", "--comment-id", fmt.Sprintf("%d", rootID), "--message", "Agent reply")
	require.NoError(t, err)

	// Verify thread is unresolved
	output, err := env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Root comment")

	// Resolve thread via API
	resolveResp := env.patchJSON(t, fmt.Sprintf("/api/comments/%d/resolve", rootID), map[string]string{})
	defer func() { _ = resolveResp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resolveResp.StatusCode)

	// Verify thread is resolved
	output, err = env.runCLI(t, "address", "--file", "test.md", "--project", env.ProjectDir)
	require.NoError(t, err)
	assert.Contains(t, output, "No unresolved comments")
}

func TestE2E_ThreadedComments_CannotEditCommentWithReplies(t *testing.T) {
	env := setupE2E(t)
	_, err := env.runCLI(t, "register", "--project", env.ProjectDir)
	require.NoError(t, err)

	// Create root comment
	rootComment := map[string]interface{}{
		"project_directory": env.ProjectDir,
		"file_path":         "test.md",
		"line_start":        1,
		"line_end":          1,
		"selected_text":     "Test Document",
		"comment_text":      "Original comment",
		"author":            "user",
	}
	resp := env.postJSON(t, "/api/comments", rootComment)
	defer func() { _ = resp.Body.Close() }()
	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	rootID := int(created["id"].(float64))

	// Should be able to edit before replies exist
	updatePayload := map[string]string{
		"comment_text": "Updated comment",
	}
	updateResp := env.patchJSON(t, fmt.Sprintf("/api/comments/%d", rootID), updatePayload)
	_ = updateResp.Body.Close()
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)

	// Add a reply
	_, err = env.runCLI(t, "reply", "--comment-id", fmt.Sprintf("%d", rootID), "--message", "Agent reply")
	require.NoError(t, err)

	// Should NOT be able to edit after replies exist
	updatePayload2 := map[string]string{
		"comment_text": "Should not update",
	}
	updateResp2 := env.patchJSON(t, fmt.Sprintf("/api/comments/%d", rootID), updatePayload2)
	defer func() { _ = updateResp2.Body.Close() }()
	assert.Equal(t, http.StatusBadRequest, updateResp2.StatusCode)

	// Verify error message
	var errorResp map[string]string
	err = json.NewDecoder(updateResp2.Body).Decode(&errorResp)
	if err == nil {
		// If we got JSON, check the error message
		t.Logf("Error response: %v", errorResp)
	}
}
