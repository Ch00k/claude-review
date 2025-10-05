# claude-review

claude-review is a lightweight companion for working on planning documents with Claude Code. It lets you review a
Markdown plan in the browser, leave inline comments, and hand those comments back to the same Claude Code session that
created the plan.

For a detailed overview of the architecture, see [ARCHITECTURE](ARCHITECTURE.md).

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

### Automated (recommended)

```bash
curl -fsSL https://github.com/Ch00k/claude-review/releases/latest/download/install.sh | bash
```

The installer will:
- Download and install the `claude-review` binary to `~/.local/bin/`
- Install the `/cr-review` and `/cr-address` slash commands to `~/.claude/commands/`

### Manual

1. Download the binary for your platform from the [latest release](https://github.com/Ch00k/claude-review/releases/latest)
2. Make it executable and move it to your PATH:
   ```bash
   chmod +x claude-review-<os>-<arch>
   mv claude-review-<os>-<arch> ~/.local/bin/claude-review
   ```
3. Install the slash commands:
   ```bash
   claude-review install
   ```

Make sure `~/.local/bin` is in your `PATH`. If not:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

## How it fits into your workflow
1. Ask Claude Code to create a Markdown document (e.g. `PLAN.md`)
2. Run `/cr-review PLAN.md` in Claude Code and open the URL it returns
3. Highlight portions of the document and add contextual comments
4. Run `/cr-address PLAN.md` in your Claude Code session to get your comments addressed (the HTML page will refresh
   automatically)
5. Repeat steps 3-4 until the document matches your intent

<!--## Example-->

<!--### Ask Claude Code to create a plan-->
<!--![Create a plan](screenshots/create.png)-->

<!--### Review the plan in a browser-->
<!--![Review in browser](screenshots/review.png)-->

<!--### Ask Claude Code to address your review comments-->
<!--![Address comments](screenshots/address.png)-->
