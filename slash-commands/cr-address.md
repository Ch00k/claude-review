---
description: Summarise unresolved markdown comments for Claude to act on
argument-hint: [file]
allowed-tools: Bash(claude-review address:*), Bash(claude-review resolve:*), Edit, Read
---

Latest unresolved comments:
!`claude-review address --file "$ARGUMENTS"`

Address each comment above. For each comment you addressed give a summary of what the comment was, what it was related
to, and what changes you made.

After you've addressed all comments, mark them all as resolved by running a command
`claude-review resolve --file <FILENAME>`, replacing `<FILENAME>` with the name of the file you are addressing comments
for. Run this command even if there were no comments to address.

Report the result of the `claude-review resolve` command.
