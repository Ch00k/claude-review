#!/usr/bin/env bash
set -e

# Detect installation mode
UPGRADE_MODE=false
CURRENT_VERSION="unknown"
BIN_DIR="$HOME/.local/bin"

if [ -f "$BIN_DIR/claude-review" ]; then
    UPGRADE_MODE=true
    CURRENT_VERSION=$("$BIN_DIR/claude-review" version 2>/dev/null || echo "unknown")
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

# Get download URL
BINARY_URL=$(echo "$RELEASE_INFO" | grep -o "\"browser_download_url\": \"[^\"]*claude-review-${OS}-${ARCH}\"" | cut -d'"' -f4)

if [ -z "$BINARY_URL" ]; then
    echo "Error: Could not find binary for $OS-$ARCH in latest release"
    exit 1
fi

# Check server status before upgrade
SERVER_WAS_RUNNING=false
if [ "$UPGRADE_MODE" = true ]; then
    echo "Checking if server is running..."
    if "$BIN_DIR/claude-review" server --status >/dev/null 2>&1; then
        SERVER_WAS_RUNNING=true
        echo "Stopping claude-review server for upgrade..."
        "$BIN_DIR/claude-review" server --stop || true
        sleep 1
    fi
fi

# Create installation directory
echo "Creating $BIN_DIR"
mkdir -p "$BIN_DIR"

# Download and install binary
echo "Installing claude-review binary..."
curl -sSL "$BINARY_URL" -o "$BIN_DIR/claude-review"
chmod +x "$BIN_DIR/claude-review"

# Install slash command using the binary
echo "Installing slash command..."
"$BIN_DIR/claude-review" install --slash-command

# Restart server if it was running before upgrade
if [ "$SERVER_WAS_RUNNING" = true ]; then
    echo "Restarting server..."
    "$BIN_DIR/claude-review" server --daemon
    sleep 1
    if "$BIN_DIR/claude-review" server --status >/dev/null 2>&1; then
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
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo "Add $BIN_DIR to your PATH:"
    # shellcheck disable=SC2016
    echo '   export PATH='"$BIN_DIR"':$PATH'
    echo
fi

# Show hook installation instructions
echo "Next Steps:"
echo "  1. Install the session-start hook by running 'claude-review install --hook'"
echo "  2. Restart Claude Code (hook will start the server automatically)"
echo "  3. Ask Claude Code to create a plan in PLAN.md"
echo "  4. Open http://localhost:4779, select project and PLAN.md file, add comments"
echo "  5. Run '/address-comments PLAN.md' in Claude Code"
echo
