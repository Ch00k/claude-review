Updated: 2025-10-09T16:30:00Z

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

claude-review is a lightweight companion for working on planning documents with Claude Code. It provides a web-based interface for reviewing Markdown files, leaving inline comments, and integrating those comments back into Claude Code sessions.

### Core Workflow
1. User creates a planning document (e.g., PLAN.md) via Claude Code
2. User runs `/cr-review <file>` slash command to start server and open document in browser
3. User highlights portions of the rendered Markdown and adds comments
4. User runs `/cr-address <file>` slash command in Claude Code to pull unresolved comments
5. Claude Code reviews comment threads and can either:
   - Reply to threads to ask clarifying questions or discuss alternatives
   - Make requested changes and resolve threads when complete
6. User can reply to Claude's responses in the browser to continue the discussion
7. Browser automatically reloads to show updated document and new replies

## Architecture

### Backend (Go)
- **main.go**: Entry point with CLI commands (server, register, review, address, resolve, install, version)
- **handlers.go**: HTTP route handlers for web UI and REST API
- **db.go**: SQLite database layer for projects and comments
- **markdown.go**: Custom goldmark renderer that adds `data-line-start` and `data-line-end` attributes to HTML elements
- **sse.go**: Server-Sent Events hub for real-time browser updates
- **daemon.go**: Daemon process management (start, stop, status, PID tracking)
- **watcher.go**: File system watcher using fsnotify for automatic reload on file changes
- **notify.go**: HTTP client for notifying server about resolved comments
- **version.go**: Version information
- **install.go**: Slash command installation logic
- **embed.go**: Embedded frontend assets

### Frontend
- **frontend/templates/**: Go HTML templates (viewer.html, directory.html, index.html)
- **frontend/static/**: CSS and JavaScript for the web UI
- **viewer.js**: Handles text selection, comment creation/editing, SSE connection for live updates

### Database Schema
SQLite database stored at `~/.local/share/claude-review/comments.db`:
- **projects**: Tracks registered project directories
- **comments**: Stores inline comments with line ranges, selected text, resolved status, threading support
  - `root_id`: Foreign key to parent comment (NULL for root comments, enables threaded discussions)
  - `author`: Name of comment author (user or agent name)
  - `resolved_by`: Name of who resolved the thread

### Custom Slash Commands
- `/cr-review` (in `slash-commands/cr-review.md`): Runs `claude-review review --file <file>` to start the server, register the project, and open the file URL in the browser
- `/cr-address` (in `slash-commands/cr-address.md`): Runs `claude-review address --file <file>` to fetch unresolved comment threads with full conversation history. Instructs Claude Code to either reply to threads for discussion or make changes and resolve threads when requests are clear

## Development Commands

### Build and Run
```bash
make build              # Build binary to dist/claude-review
make dev                # Run with auto-reload using air
make air                # Alias for make dev
```

### Testing
```bash
make test               # Run tests with summary output
make test-verbose       # Run tests with detailed output
make test-one TEST=TestName  # Run single test by name
make test-ci            # Run tests with coverage report
```

### Linting and Formatting
```bash
make lint               # Run prettier and golangci-lint with auto-fix
make prettier           # Format frontend files only
```

### Installation
```bash
make install            # Build binary and install slash commands to ~/.claude/commands/
```

### Release
```bash
make build-release      # Build binary + tar frontend assets
make release-patch      # Create patch version release (x.y.Z)
make release-minor      # Create minor version release (x.Y.0)
make release-major      # Create major version release (X.0.0)
```

## Key Implementation Details

### Line Number Tracking
The markdown.go renderer walks the Goldmark AST and adds `data-line-start` and `data-line-end` attributes to block-level HTML elements. This enables the frontend to map user text selections back to source line numbers.

### Real-time Updates
The server uses multiple mechanisms for real-time updates:
- **SSE (Server-Sent Events)**: Broadcasts events to connected browsers when comments are resolved
- **File System Watching**: Uses fsnotify to watch Markdown files and automatically reload the browser when files are modified
- **Event Broadcasting**: CLI commands (like `resolve`) notify the server via HTTP POST to `/api/events`, which triggers SSE broadcasts to connected clients

### CLI Commands
- `claude-review server`: Start web server on port 4779 in foreground
- `claude-review server --daemon`: Start web server as a background daemon
- `claude-review server --stop`: Stop the running daemon
- `claude-review server --status`: Check if daemon is running and show PID info
- `claude-review register [--project <path>]`: Register a project directory (defaults to current directory)
- `claude-review review --file <path>`: Start server, register project, and show file URL
- `claude-review address --file <path>`: Output unresolved comment threads for file (grouped by conversation)
- `claude-review reply --comment-id <id> --message <text>`: Reply to a comment thread
- `claude-review resolve --file <path>`: Mark all comments as resolved for file
- `claude-review resolve --comment-id <id>`: Mark a specific comment thread as resolved
- `claude-review install`: Install slash commands to ~/.claude/commands/
- `claude-review version`: Show version information

### Daemon Management
The server can run as a background daemon:
- **PID File**: Stored at `~/.local/share/claude-review/server.pid`
- **Log File**: Daemon logs stored at `~/.local/share/claude-review/server.log`
- **Signal Handling**: Graceful shutdown on SIGTERM/SIGINT with automatic cleanup of PID file
- **Double-fork**: Uses Unix double-fork pattern to fully detach from terminal

## Configuration

- **CGO_ENABLED=1**: Required for go-sqlite3
- **DEBUG_SQL=1**: Environment variable to enable SQL query logging
- Port 4779 is hardcoded for the web server
