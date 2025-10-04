package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func installSlashCommands() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	commandsDir := filepath.Join(homeDir, ".claude", "commands")

	// Create commands directory if it doesn't exist
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	// Install all slash commands from embedded FS
	var installed []string
	err = fs.WalkDir(slashCommandsFS, "slash-commands", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Read embedded slash command
		commandContent, err := slashCommandsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Write to ~/.claude/commands/
		filename := filepath.Base(path)
		commandPath := filepath.Join(commandsDir, filename)
		if err := os.WriteFile(commandPath, commandContent, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}

		installed = append(installed, filename)
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Successfully installed %d slash command(s) to %s:\n", len(installed), commandsDir)
	for _, name := range installed {
		cmdName := strings.TrimSuffix(name, ".md")
		fmt.Printf("  /%s\n", cmdName)
	}

	return nil
}
