# claude-review

claude-review is a lightweight companion for working on planning documents with Claude Code. It lets you review a
Markdown plan in the browser, leave inline comments, and hand those comments back to the same Claude Code session that
created the plan.

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
- Start Claude Code in the directory of your project (the hook automatically starts the server and registers the project)
- Ask Claude Code to produce a plan (for example `PLAN.md`)
- Open http://localhost:4779 in your browser, select your project and the `PLAN.md` Markdown file that Claude Code produced
- Highlight portions of the document and add contextual comments
- Run `/address-comments PLAN.md` in your Claude Code session to pull the pending comments, apply edits, and resolve them
- Watch the plan automatically reload in your browser, repeating the cycle until the document matches your intent
