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
			want:     "&quot;quoted&quot;",
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

func TestSyntaxHighlighting(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string
		notWant  string
	}{
		{
			name:     "go code block",
			markdown: "```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
			want:     "style=",
			notWant:  "",
		},
		{
			name:     "javascript code block",
			markdown: "```javascript\nconst foo = 'bar';\nconsole.log(foo);\n```",
			want:     "style=",
			notWant:  "",
		},
		{
			name:     "python code block",
			markdown: "```python\ndef greet():\n    print('hello')\n```",
			want:     "style=",
			notWant:  "",
		},
		{
			name:     "code block without language",
			markdown: "```\nplain text code\n```",
			want:     "<pre",
			notWant:  "",
		},
		{
			name:     "inline styles not classes",
			markdown: "```go\nfunc test() {}\n```",
			want:     "style=",
			notWant:  "class=\"chroma\"",
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

func TestCodeBlockLineNumbers(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string
		notWant  string
	}{
		{
			name:     "code block with language has line numbers",
			markdown: "```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
			want:     "data-line-start=\"1\"",
		},
		{
			name:     "code block without language has line numbers",
			markdown: "```\nplain text\ncode\n```",
			want:     "data-line-start=\"1\"",
		},
		{
			name:     "code block after other content has correct line numbers",
			markdown: "# Heading\n\nSome text.\n\n```go\nfunc test() {}\n```",
			want:     "data-line-start=\"5\"",
		},
		{
			name:     "code block has end line number",
			markdown: "```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
			want:     "data-line-end=\"6\"",
		},
		{
			name:     "code block has closing pre tag",
			markdown: "```go\nfunc test() {}\n```",
			want:     "</pre>",
		},
		{
			name:     "multiline code block has correct end line",
			markdown: "```python\ndef greet():\n    print('hello')\n    print('world')\n```",
			want:     "data-line-end=\"6\"",
		},
		{
			name:     "single line code block",
			markdown: "```js\nconsole.log('test')\n```",
			want:     "data-line-start=\"1\" data-line-end=\"4\"",
		},
		{
			name:     "code block without double pre tags",
			markdown: "```go\nfunc test() {}\n```",
			notWant:  "<pre><pre",
		},
		{
			name:     "highlighted code block has chroma styles",
			markdown: "```go\nfunc test() {}\n```",
			want:     "style=",
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

func TestCustomWrapperRenderer(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     []string
	}{
		{
			name:     "wrapper includes data-line-start and data-line-end",
			markdown: "```go\nfunc test() {}\n```",
			want:     []string{"<pre", "data-line-start=", "data-line-end=", ">"},
		},
		{
			name:     "wrapper includes closing tag",
			markdown: "```go\nfunc test() {}\n```",
			want:     []string{"</pre>"},
		},
		{
			name:     "plain code block without highlighting still has attributes",
			markdown: "```\nplain\n```",
			want:     []string{"<pre", "data-line-start=", "data-line-end="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := RenderMarkdownWithLineNumbers([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("RenderMarkdownWithLineNumbers failed: %v", err)
			}

			htmlStr := string(html)

			for _, want := range tt.want {
				if !strings.Contains(htmlStr, want) {
					t.Errorf("Expected %q in HTML, got: %s", want, htmlStr)
				}
			}
		})
	}
}
