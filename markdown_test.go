package main

import (
	"strings"
	"testing"
)

func TestHTMLEscapingInCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string
		notWant  string
	}{
		{
			name:     "angle brackets",
			markdown: "```\n<project-hash>\n```",
			want:     "&lt;project-hash&gt;",
			notWant:  "<code><project-hash>",
		},
		{
			name:     "ampersand",
			markdown: "```\nfoo & bar\n```",
			want:     "foo &amp; bar",
			notWant:  "<code>foo & bar</code>",
		},
		{
			name:     "quotes",
			markdown: "```\n\"quoted\"\n```",
			want:     "&#34;quoted&#34;",
			notWant:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := RenderMarkdownWithLineNumbers([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("RenderMarkdownWithLineNumbers failed: %v", err)
			}

			htmlStr := string(html)

			if !strings.Contains(htmlStr, tt.want) {
				t.Errorf("Expected %q in HTML, got: %s", tt.want, htmlStr)
			}

			if tt.notWant != "" && strings.Contains(htmlStr, tt.notWant) {
				t.Errorf("Did not expect %q in HTML, got: %s", tt.notWant, htmlStr)
			}
		})
	}
}

func TestInlineCodeLineNumbers(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "inline code in list item",
			markdown: "1. If `--remote` provided: add git remote",
			want:     "data-line-start=\"1\"",
		},
		{
			name:     "multiple inline code blocks",
			markdown: "Use `foo` and `bar` together",
			want:     "data-line-start=\"1\"",
		},
		{
			name:     "inline code with special chars",
			markdown: "The `<code>` element is special",
			want:     "data-line-start=\"1\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := RenderMarkdownWithLineNumbers([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("RenderMarkdownWithLineNumbers failed: %v", err)
			}

			htmlStr := string(html)

			if !strings.Contains(htmlStr, tt.want) {
				t.Errorf("Expected %q in HTML, got: %s", tt.want, htmlStr)
			}
		})
	}
}
