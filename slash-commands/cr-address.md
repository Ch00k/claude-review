---
description: Summarise unresolved markdown comments for Claude to act on
argument-hint: [file]
allowed-tools: Bash(claude-review address:*), Bash(claude-review resolve:*), Bash(claude-review reply:*), Edit, Read, Write
---

First, read the file that is being commented on using the Read tool with path "$ARGUMENTS". This gives you the current
state of the document before processing any comments.

--- COMMENTS START ---

!`claude-review address --file "$ARGUMENTS"`

--- COMMENTS END ---

**Note:** The output above only contains UNRESOLVED threads. Resolved threads will not appear.

You are working with threaded comments. Each comment may have replies forming a discussion thread. Each thread is
labeled with a comment ID like "## Comment #123". Use this ID when replying to or resolving threads. Within each thread,
replies are displayed in chronological order (oldest first).

## Thread Format
Each thread starts with a root comment and may contain replies:
- Root comment: "**User:**" followed by the original comment text
- Replies: "**Reply from User:**" or "**Reply from Agent:**" followed by the reply text
- Messages appear in chronological order (oldest first)

For each comment thread above, follow this process:

## Step 1: Extract Context
- Extract the comment ID from the "## Comment #<ID>" header
- Read the ENTIRE thread (root comment + all replies) to understand the full conversation
- Identify who wrote the last message (could be root User comment, a User reply, or an Agent reply)

## Step 2: Determine if Action is Needed

**Skip this thread if:**
- You (Agent) wrote the last message in the thread

**Process this thread if:**
- User wrote the last message (either the root "**User:**" comment or a "**Reply from User:**" message)

**Examples:**

Thread A - SKIP (Agent was last):
```
## Comment #1 (lines 10-12)
> original text here

**User:**
Fix this typo

**Reply from Agent:**
Fixed the typo and replied
```

Thread B - PROCESS (User replied after Agent):
```
## Comment #2 (lines 15-15)
> some selected text

**User:**
Fix this typo

**Reply from Agent:**
Fixed the typo and replied

**Reply from User:**
Thanks, but can you also fix the grammar?
```

Thread C - PROCESS (new thread, no Agent response yet):
```
## Comment #3 (lines 20-25)
> code block or paragraph

**User:**
Add a section about testing
```

Thread D - PROCESS (User replied to their own comment):
```
## Comment #4 (lines 30-32)
> another selection

**User:**
This paragraph needs clarification

**Reply from User:**
Actually, can you also add an example here?
```

## Step 3: Choose Your Action

Follow this decision tree IN ORDER. **If multiple conditions match, use the FIRST matching rule (A beats B beats C).**

### A. Does the thread contain a CLEAR, UNAMBIGUOUS request to modify the document?

A clear request directly states what should change, without hedging or asking questions. Look at the **most recent User message** in the thread to determine the current request.

**Clear requests (YES):**
- "Change X to Y"
- "Add section about Z"
- "Remove this paragraph"
- "Rewrite this as..."
- "Fix this typo"

**Unclear/Discussion (NO - go to B):**
- "This could be better"
- "Maybe change X to Y?"
- "What if we used Y instead of X?"
- "I'm not sure about this"

**Ambiguous short messages (NO - go to C):**
- Single words like "ok", "thanks", "done"
- Emoji-only messages
- Messages with unclear intent

**If YES** -> Make the change, then reply using:
```
claude-review reply --comment-id <ID> --message "Changed [brief description]. Please verify."
```
Do NOT resolve the thread - leave it open for User to verify. (See Step 4 for when to resolve.)

**If NO** -> Go to step B

### B. Is User asking questions, discussing alternatives, or seeking your input?
Examples: "What do you think about...", "Should we...", "Why did you...", "Can you explain..."

**YES** -> Reply to continue the discussion using:
```
claude-review reply --comment-id <ID> --message "your response to the discussion"
```

**NO** -> Go to step C

### C. Are you uncertain what User wants?

This includes:
- Ambiguous or very short messages ("ok", "hmm", "maybe")
- Incomplete thoughts
- Messages where you cannot determine if action is needed

**YES** -> Ask for clarification using:
```
claude-review reply --comment-id <ID> --message "your clarification question"
```

**Example clarification requests:**
- "I see you wrote 'ok' - could you clarify what you'd like me to do here?"
- "I'm not sure I understand. Are you asking me to [X] or [Y]?"
- "Could you provide more details about what change you'd like?"

## Step 4: Resolving Threads

**Default: NEVER resolve threads automatically**

**ONLY resolve a thread if:**
1. User explicitly says "resolve this" in the comment thread itself, OR
2. User tells you in the main chat to resolve specific thread IDs, OR
3. User tells you in the main chat to resolve all threads for the file (use `claude-review resolve --file <FILENAME>`)

When in doubt, DO NOT RESOLVE - leave threads open for User to review.

## Step 5: Report Your Actions

After processing all threads, provide a summary and detailed report.

**Summary line:**
```
Processed N threads: took action on X, skipped Y (no new messages from User)
```

**Detailed report:**
For each thread where you took action, report what you did using this format:

```
[Comment #123]
Selection: "Selected text from the document..."
Last User Message: "The most recent message from User in this thread..."
Action: Made change X and replied
```

**Reporting rules:**
- **Do NOT include threads that were skipped in Step 2** (where Agent was last to speak)
- Only report threads where you actually took action (replied, made changes, asked for clarification, or resolved)
- "Last User Message" should be the last message from User in the thread. This could be:
  - The root "**User:**" comment (if there are no User replies)
  - The most recent "**Reply from User:**" message (if User has replied in the thread)
- Truncate both Selection and Last User Message at 100 characters, adding "..." if truncated
- Always include the Comment ID, Selection, Last User Message, and Action

**Example report:**

```
Processed 5 threads: took action on 2, skipped 3 (no new messages from User)

[Comment #45]
Selection: "The quick brown fox"
Last User Message: "Fix this typo"
Action: Fixed typo and replied

[Comment #67]
Selection: "The architecture consists of three main components: the authentication layer, the data access ..."
Last User Message: "I think we should consider using OAuth instead of custom auth because it provides better secur..."
Action: Replied to discuss alternatives
```
