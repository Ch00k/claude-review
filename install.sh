#!/usr/bin/env bash
set -e

# Detect installation mode
UPGRADE_MODE=false
CURRENT_VERSION="unknown"
INSTALL_DIR="$HOME/.local/bin"
DATA_DIR="$HOME/.local/share/claude-review"
CLAUDE_COMMANDS_DIR="$HOME/.claude/commands"
CLAUDE_SETTINGS="$HOME/.claude/settings.json"
VERSION_FILE="$DATA_DIR/VERSION"

if [ -f "$INSTALL_DIR/claude-review" ] && [ -f "$VERSION_FILE" ]; then
    UPGRADE_MODE=true
    CURRENT_VERSION=$(cat "$VERSION_FILE" 2>/dev/null || echo "unknown")
fi

# Get latest release information
echo "Fetching latest release information..."
RELEASE_INFO=$(curl -sSL https://api.github.com/repos/Ch00k/claude-review/releases/latest)
LATEST_VERSION=$(echo "$RELEASE_INFO" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)

if [ -n "$LATEST_VERSION" ]; then
    echo "Latest release: $LATEST_VERSION"
else
    echo "Error: Could not fetch latest version"
    exit 1
fi

# Show installation mode and ask for confirmation
echo
if [ "$UPGRADE_MODE" = true ]; then
    echo "Existing claude-review installation detected (version: $CURRENT_VERSION)"

    # Check if versions are the same
    if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ]; then
        echo
        echo "You already have the latest version ($CURRENT_VERSION) installed."
        echo "No upgrade needed. Exiting."
        exit 0
    else
        echo "Upgrading claude-review to version: $LATEST_VERSION"
    fi
else
    echo "Installing claude-review version: $LATEST_VERSION"
fi

echo

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
x86_64)
    ARCH="amd64"
    ;;
aarch64 | arm64)
    ARCH="arm64"
    ;;
*)
    echo "Error: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Detected platform: $OS-$ARCH"
echo

# Get download URLs
BINARY_URL=$(echo "$RELEASE_INFO" | grep -o "\"browser_download_url\": \"[^\"]*claude-review-${OS}-${ARCH}\"" | cut -d'"' -f4)
ASSETS_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url": "[^"]*assets\.tar\.gz"' | cut -d'"' -f4)

if [ -z "$BINARY_URL" ]; then
    echo "Error: Could not find binary for $OS-$ARCH in latest release"
    exit 1
fi

if [ -z "$ASSETS_URL" ]; then
    echo "Error: Could not find assets.tar.gz in latest release"
    exit 1
fi

# Check server status before upgrade
SERVER_WAS_RUNNING=false
if [ "$UPGRADE_MODE" = true ]; then
    echo "Checking if server is running..."
    if "$INSTALL_DIR/claude-review" server --status >/dev/null 2>&1; then
        SERVER_WAS_RUNNING=true
        echo "Stopping claude-review server for upgrade..."
        "$INSTALL_DIR/claude-review" server --stop || true
        sleep 1
    fi
fi

# Create installation directory structure
echo "Creating directory structure..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$DATA_DIR"
mkdir -p "$CLAUDE_COMMANDS_DIR"

# Backup current binary during upgrade
if [ "$UPGRADE_MODE" = true ]; then
    BACKUP_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BINARY_BACKUP="$INSTALL_DIR/claude-review.${CURRENT_VERSION}.${BACKUP_TIMESTAMP}"
    cp "$INSTALL_DIR/claude-review" "$BINARY_BACKUP"
    echo "Current binary backed up to: $BINARY_BACKUP"
fi

# Download and install binary
echo "Installing claude-review command line tool..."
curl -sSL "$BINARY_URL" -o "$INSTALL_DIR/claude-review"
chmod +x "$INSTALL_DIR/claude-review"

# Download and extract assets (frontend, slash-commands, hooks)
echo "Downloading and extracting assets..."
curl -sSL "$ASSETS_URL" | tar -xzf - -C "$DATA_DIR"

# Install slash commands to Claude Code directory
if [ -d "$DATA_DIR/slash-commands" ]; then
    mkdir -p "$CLAUDE_COMMANDS_DIR"
    cp "$DATA_DIR/slash-commands/"*.md "$CLAUDE_COMMANDS_DIR/" 2>/dev/null || true
    echo "Slash commands installed to: $CLAUDE_COMMANDS_DIR"
fi

echo "Assets extracted to: $DATA_DIR"

# Hook config file location
HOOK_CONFIG_FILE="$DATA_DIR/hooks/session-start.json"

# Save version file
echo "$LATEST_VERSION" >"$VERSION_FILE"

# Restart server if it was running before upgrade
if [ "$SERVER_WAS_RUNNING" = true ]; then
    echo "Restarting server..."
    "$INSTALL_DIR/claude-review" server --daemon
    sleep 1
    if "$INSTALL_DIR/claude-review" server --status >/dev/null 2>&1; then
        echo "Server restarted successfully"
    else
        echo "Warning: Failed to restart server"
    fi
fi

# Show completion message
echo
if [ "$UPGRADE_MODE" = true ]; then
    echo "claude-review upgraded: $CURRENT_VERSION → $LATEST_VERSION"
else
    echo "claude-review $LATEST_VERSION installed successfully"
fi
echo

# Check if PATH includes install directory
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "add to your PATH (~/.bashrc or ~/.zshrc):"
    # shellcheck disable=SC2016
    echo '   export PATH=$PATH:'"$INSTALL_DIR"
    echo
fi

# Quick start
echo "Quick Start:"
if [ -f "$HOOK_CONFIG_FILE" ]; then
    echo "  1. Setup hook: merge contents of $HOOK_CONFIG_FILE into $CLAUDE_SETTINGS"
fi
echo "  2. Restart Claude Code (hook will start server automatically)"
echo "  3. Ask Claude Code to create a plan in PLAN.md"
echo "  4. Open http://localhost:4779, select project and PLAN.md file, add comments"
echo "  5. Run '/address-comments PLAN.md' in Claude Code"
echo
