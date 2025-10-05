# Architecture Overview

## System Diagram

```
                                      YOUR COMPUTER
┌───────────────────────────────────────────────────────────────────────────────────────────┐
│                                                                                           │
│  ┌─────────────────────────┐  ┌───────────────────────────┐  ┌─────────────────────────┐  │
│  │       Claude Code       │  │        Claude Code        │  │       Claude Code       │  │
│  │                         │  │                           │  │                         │  │
│  │ /home/me/proj-a/PLAN.md │  │ /home/me/proj-b/DESIGN.md │  │ /home/me/proj-c/SPEC.md │  │
│  └────────────┬────────────┘  └─────────────┬─────────────┘  └────────────┬────────────┘  │
│               │                             │                             │               │
│               │ /cr-review                  │ /cr-review                  │ /cr-review    │
│               │ /cr-address                 │ /cr-address                 │ /cr-address   │
│               │                             │                             │               │
│               └─────────────────────────────┴─────────────────────────────┘               │
│                                             │                                             │
│                                             ▼                                             │
│                              ┌──────────────────────────────┐                             │
│                              │     claude-review daemon     │                             │
│                              │   (single global instance)   │                             │
│                              │                              │                             │
│                              │ • HTTP server localhost:4779 │                             │
│                              │ • SQLite database            │                             │
│                              │ • File watcher               │                             │
│                              │ • SSE broadcaster            │                             │
│                              └──────────────┬───────────────┘                             │
│                                             │                                             │
│                                             ▼                                             │
│                 ┌──────────────────────────────────────────────────────────┐              │
│                 │                     Web browser(s)                       │              │
│                 │                                                          │              │
│                 │  http://localhost:4779/projects/home/me/proj-a/PLAN.md   │              │
│                 │  http://localhost:4779/projects/home/me/proj-b/DESIGN.md │              │
│                 │  http://localhost:4779/projects/home/me/proj-c/SPEC.md   │              │
│                 └──────────────────────────────────────────────────────────┘              │
│                                                                                           │
└───────────────────────────────────────────────────────────────────────────────────────────┘
```

## Key Concepts

### Single Global Daemon
- **One server serves all**: A single `claude-review` daemon process runs on port 4779 and serves all Claude Code
  instances on your computer
- **Idempotent startup**: When any Claude Code instance runs `/cr-review`, it checks if the daemon is running. If yes,
  it reuses it. If no, it starts it
- **Shared state**: All projects, comments, and file watches are managed by this single daemon through a shared SQLite
  database at `~/.local/share/claude-review/comments.db`

### Workflow

1. **Starting the server** (from any Claude Code instance):
   ```bash
   /cr-review PLAN.md
   # Starts daemon if not running
   # Registers project root directory in database (for comment association)
   # Generates URL: http://localhost:4779/projects/home/me/proj-a/PLAN.md
   ```

2. **Reviewing & commenting** (in browser):
   - Highlight text in rendered Markdown, add comments
   - Comments stored in global database, associated with line number, context, file path, and project root

3. **Addressing comments** (in Claude Code):
   ```bash
   /cr-address PLAN.md
   # Fetches unresolved comments associated with project and file from daemon
   # Claude Code addresses them and marks as resolved
   ```

4. **Real-time sync**:
   - File changes -> Daemon watches files and sends SSE -> Page content refreshes
   - Comments resolved -> Daemon sends SSE -> Page content refreshes

### Server Process Lifecycle

```bash
# Any Claude Code instance can:
claude-review server --daemon    # Start daemon (idempotent)
claude-review server --status    # Check if running
```

The daemon runs independently of Claude Code instances and persists until explicitly stopped with
`claude-review server --stop`
