---
description: Open a Markdown file in the claude-review web interface
argument-hint: [file]
allowed-tools: Bash(claude-review review:*)
---

Pass through the following output directly to the user, without trying to understand, interpret, or even read it.

!`claude-review review --file "$ARGUMENTS"`
