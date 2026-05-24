package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFileArg(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}

	tests := []struct {
		name                     string
		inProjectDir             string
		inFilePath               string
		projectDirDefaultedToCwd bool
		wantProjectDir           string
		wantFilePath             string
	}{
		{
			name:                     "strips @ prefix",
			inProjectDir:             "/repo",
			inFilePath:               "@docs/plan.md",
			projectDirDefaultedToCwd: false,
			wantProjectDir:           "/repo",
			wantFilePath:             "docs/plan.md",
		},
		{
			name:                     "expands ~/ when projectDir is default",
			inProjectDir:             "/cwd",
			inFilePath:               "~/.claude/plans/foo.md",
			projectDirDefaultedToCwd: true,
			wantProjectDir:           filepath.Join(home, ".claude/plans"),
			wantFilePath:             "foo.md",
		},
		{
			name:                     "absolute file splits into dir+base when projectDir is default",
			inProjectDir:             "/cwd",
			inFilePath:               "/Users/me/.claude/plans/foo.md",
			projectDirDefaultedToCwd: true,
			wantProjectDir:           "/Users/me/.claude/plans",
			wantFilePath:             "foo.md",
		},
		{
			name:                     "absolute file with explicit projectDir is left relative-style alone",
			inProjectDir:             "/explicit",
			inFilePath:               "/Users/me/file.md",
			projectDirDefaultedToCwd: false,
			wantProjectDir:           "/explicit",
			wantFilePath:             "/Users/me/file.md",
		},
		{
			name:                     "relative path is unchanged",
			inProjectDir:             "/cwd",
			inFilePath:               "sub/plan.md",
			projectDirDefaultedToCwd: true,
			wantProjectDir:           "/cwd",
			wantFilePath:             "sub/plan.md",
		},
		{
			name:                     "combined @ and ~/",
			inProjectDir:             "/cwd",
			inFilePath:               "@~/.claude/plans/foo.md",
			projectDirDefaultedToCwd: true,
			wantProjectDir:           filepath.Join(home, ".claude/plans"),
			wantFilePath:             "foo.md",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := tc.inProjectDir
			filePath := tc.inFilePath
			resolveFileArg(&projectDir, &filePath, tc.projectDirDefaultedToCwd)
			if projectDir != tc.wantProjectDir {
				t.Errorf("projectDir = %q, want %q", projectDir, tc.wantProjectDir)
			}
			if filePath != tc.wantFilePath {
				t.Errorf("filePath = %q, want %q", filePath, tc.wantFilePath)
			}
		})
	}
}
