# claude-review

claude-review is a lightweight companion for working on planning documents with Claude Code. It lets you review a
Markdown plan in the browser, leave inline comments, and hand those comments back to the same Claude Code session that
created the plan.

## Motivation

When working with Claude Code to generate planning documents in Markdown format, the typical workflow involves switching
back and forth between reviewing the Markdown document and the Claude Code session. You read through the plan, switch
back to the Claude Code session, and manually describe which sections need changes - often copying and pasting text
snippets to provide context.

claude-review streamlines this process by enabling inline comments directly in the browser, similar to Atlassian
Confluence or Google Docs. You can highlight any portion of the rendered Markdown and add contextual feedback. The same
Claude Code instance that generated the plan can then fetch these comments, understand exactly what needs to change, and
update the document automatically. Since the browser view refreshes on file changes, you see your edits immediately
without any manual intervention.

This keeps you in flow: the agent retains full context of the plan it generated, your comments are precisely anchored to
specific sections, and the feedback loop happens in seconds rather than minutes.

## Requirements

- Linux or macOS
- Claude Code CLI
- A modern web browser

## Installation

```bash
curl -fsSL https://github.com/Ch00k/claude-review/releases/latest/download/install.sh | bash
```

The installer will:
- Download and install the `claude-review` binary to `~/.local/bin/`
- Extract assets (frontend, Claude Code slash-commands and hooks) to `~/.local/share/claude-review/`
- Install the `/address-comments` slash command to `~/.claude/commands/`
- Display instructions for setting up the Claude Code hook

After installation, follow the displayed instructions to add the hook configuration to `~/.claude/settings.json`.

## How it fits into your workflow
1. Start Claude Code in the directory of your project (the hook automatically starts the server and registers the
   project)
2. Ask Claude Code to produce a plan (for example `PLAN.md`)
3. Open http://localhost:4779 in your browser, select your project and the `PLAN.md` Markdown file that Claude Code
   produced
4. Highlight portions of the document and add contextual comments
5. Run `/address-comments PLAN.md` in your Claude Code session to pull the pending comments, apply edits, and resolve
   them
6. Watch the plan automatically reload in your browser, repeating the cycle until the document matches your intent
