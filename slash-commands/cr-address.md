---
description: Summarise unresolved markdown comments for Claude to act on
argument-hint: [file]
allowed-tools: Bash(claude-review address:*), Bash(claude-review resolve:*), Bash(claude-review reply:*), Edit, Read
---

Latest unresolved comments:
!`claude-review address --file "$ARGUMENTS"`

You are working with threaded comments. Each comment may have replies forming a discussion thread. Each thread is
labeled with a comment ID like "## Comment #123". Use this ID when replying to or resolving threads. Within each thread,
replies are displayed in chronological order (oldest first).

For each comment thread above:

1. **Extract the comment ID** from the "## Comment #<ID>" header
2. **Read the full thread context** (root comment and all replies)
3. **Determine the appropriate action**:
   - **Discuss mode**: If the user is asking questions, seeking clarification, or discussing alternatives, respond to
     the thread using `claude-review reply --comment-id <ID> --message "your response"`
   - **Fix mode**: If the user's request is explicit and unambiguous (e.g., "Please change X to Y"), make the requested
     changes to the document AND reply to acknowledge the change
   - **Clarify mode**: If you're unsure what the user wants, ask for clarification using `claude-review reply
     --comment-id <ID> --message "your question"`

4. **After addressing a thread**:
   - If you made changes to the document that fully address the thread, resolve it using `claude-review resolve
     --comment-id <ID>`
   - If you replied but need more input from the user, do NOT resolve the thread

Only use `claude-review resolve --file <FILENAME>` if you want to resolve ALL threads for the file at once.

Report what you did for each thread (replied, made changes, resolved, etc.).
