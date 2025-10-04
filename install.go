package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func installHook() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")

	hookContent, err := hooksFS.ReadFile("hooks/session-start.json")
	if err != nil {
		return fmt.Errorf("failed to read embedded hook: %w", err)
	}

	fmt.Println("To install the session-start hook, you need to merge the following configuration")
	fmt.Println("into your ~/.claude/settings.json file:")
	fmt.Println()
	fmt.Println(string(hookContent))
	fmt.Println("Steps:")
	fmt.Printf("1. Open %s in your editor\n", settingsPath)
	fmt.Println("2. If the file doesn't exist, create it with the content above")
	fmt.Println("3. If the file exists:")
	fmt.Println("   - If it already has a 'hooks.SessionStart' array, append the hook object to it")
	fmt.Println("   - If it doesn't have 'hooks.SessionStart', add the entire hooks section")
	fmt.Println()
	fmt.Println("The hook will automatically start the claude-review server when you start a Claude Code session")

	return nil
}

func installSlashCommand() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	commandsDir := filepath.Join(homeDir, ".claude", "commands")
	commandPath := filepath.Join(commandsDir, "address-comments.md")

	// Create commands directory if it doesn't exist
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	// Read embedded slash command
	commandContent, err := slashCommandsFS.ReadFile("slash-commands/address-comments.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded slash command: %w", err)
	}

	// Write to ~/.claude/commands/address-comments.md
	if err := os.WriteFile(commandPath, commandContent, 0644); err != nil {
		return fmt.Errorf("failed to write slash command: %w", err)
	}

	fmt.Printf("Successfully installed slash command to %s\n", commandPath)
	fmt.Println()
	fmt.Println("You can now use /address-comments <file> in Claude Code to review comments")

	return nil
}
