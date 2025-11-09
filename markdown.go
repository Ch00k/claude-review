package main

import (
	"bytes"
	"strconv"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// LineAttributeTransformer adds data-line-start and data-line-end attributes to all block nodes
type LineAttributeTransformer struct{}

func (t *LineAttributeTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()

	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Only process block-level nodes
		if node.Type() == ast.TypeBlock {
			// Skip list containers (ul/ol) - their children (li) will have attributes
			if node.Kind() == ast.KindList {
				return ast.WalkContinue, nil
			}

			var startLine, endLine int

			// Special handling for FencedCodeBlock to include the opening fence line
			if node.Kind() == ast.KindFencedCodeBlock {
				fcb := node.(*ast.FencedCodeBlock)
				// Use the Info segment to find the opening fence line
				if fcb.Info != nil {
					infoStart := fcb.Info.Segment.Start
					startLine = bytes.Count(source[:infoStart], []byte{'\n'}) + 1
				} else {
					// No info, use first line of content
					if fcb.Lines().Len() > 0 {
						firstLine := fcb.Lines().At(0)
						// The opening fence is on the line before the first content line
						startLine = bytes.Count(source[:firstLine.Start], []byte{'\n'})
					}
				}

				// End line is after the last content line (includes closing fence)
				if fcb.Lines().Len() > 0 {
					lastLine := fcb.Lines().At(fcb.Lines().Len() - 1)
					endLine = bytes.Count(source[:lastLine.Stop], []byte{'\n'}) + 1
					// Add 1 for the closing fence line
					endLine++
				}
			} else {
				lines := node.Lines()

				if lines.Len() > 0 {
					// Node has direct line info
					firstLine := lines.At(0)
					startLine = bytes.Count(source[:firstLine.Start], []byte{'\n'}) + 1

					lastLine := lines.At(lines.Len() - 1)
					endLine = bytes.Count(source[:lastLine.Stop], []byte{'\n'}) + 1
				} else {
					// Node has no direct line info
					// Calculate from children
					startLine, endLine = getChildLineRange(node, source)
					if startLine == 0 {
						// No line info available from children either
						return ast.WalkContinue, nil
					}
				}
			}

			// Set attributes (goldmark's HTML renderer will automatically render them)
			node.SetAttribute([]byte("data-line-start"), []byte(strconv.Itoa(startLine)))
			node.SetAttribute([]byte("data-line-end"), []byte(strconv.Itoa(endLine)))
		}

		return ast.WalkContinue, nil
	})
}

// getChildLineRange calculates line range from a node's children
func getChildLineRange(node ast.Node, source []byte) (int, int) {
	var startLine, endLine int

	// Walk children to find first and last line numbers
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		lines := child.Lines()
		if lines.Len() > 0 {
			firstLine := lines.At(0)
			childStart := bytes.Count(source[:firstLine.Start], []byte{'\n'}) + 1

			lastLine := lines.At(lines.Len() - 1)
			childEnd := bytes.Count(source[:lastLine.Stop], []byte{'\n'}) + 1

			if startLine == 0 || childStart < startLine {
				startLine = childStart
			}
			if childEnd > endLine {
				endLine = childEnd
			}
		} else {
			// Recursively check grandchildren
			childStart, childEnd := getChildLineRange(child, source)
			if childStart > 0 {
				if startLine == 0 || childStart < startLine {
					startLine = childStart
				}
				if childEnd > endLine {
					endLine = childEnd
				}
			}
		}
	}

	return startLine, endLine
}

// LineAttributeExtension is a goldmark extension that adds line number attributes
type LineAttributeExtension struct{}

func (e *LineAttributeExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&LineAttributeTransformer{}, 100),
		),
	)
}

// customWrapperRenderer writes the <pre> tag with data-line-start and data-line-end attributes
func customWrapperRenderer(w util.BufWriter, context highlighting.CodeBlockContext, entering bool) {
	if entering {
		_, _ = w.WriteString("<pre")

		// Get the line number attributes from the context
		attrs := context.Attributes()
		if attrs != nil {
			if lineStart, ok := attrs.Get([]byte("data-line-start")); ok {
				_, _ = w.WriteString(` data-line-start="`)
				_, _ = w.Write(lineStart.([]byte))
				_, _ = w.WriteString(`"`)
			}
			if lineEnd, ok := attrs.Get([]byte("data-line-end")); ok {
				_, _ = w.WriteString(` data-line-end="`)
				_, _ = w.Write(lineEnd.([]byte))
				_, _ = w.WriteString(`"`)
			}

			// Add other attributes (like tabindex, style, etc.)
			for _, attr := range attrs.All() {
				name := string(attr.Name)
				// Skip our custom attributes as we already handled them
				if name != "data-line-start" && name != "data-line-end" {
					_, _ = w.WriteString(` `)
					_, _ = w.Write(attr.Name)
					_, _ = w.WriteString(`="`)
					if val, ok := attr.Value.([]byte); ok {
						_, _ = w.Write(val)
					}
					_, _ = w.WriteString(`"`)
				}
			}
		}

		_, _ = w.WriteString(">")
	} else {
		_, _ = w.WriteString("</pre>\n")
	}
}

// RenderMarkdownWithLineNumbers renders markdown to HTML with line number attributes
func RenderMarkdownWithLineNumbers(source []byte) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			&LineAttributeExtension{},
			highlighting.NewHighlighting(
				highlighting.WithStyle("friendly"),
				highlighting.WithFormatOptions(
					html.WithClasses(false),          // Use inline styles
					html.PreventSurroundingPre(true), // Don't write <pre>, we handle it in customWrapperRenderer
				),
				highlighting.WithWrapperRenderer(customWrapperRenderer),
			),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(), // Allow raw HTML
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// RenderMarkdown renders markdown to HTML without line number attributes
func RenderMarkdown(source []byte) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("friendly"),
				highlighting.WithFormatOptions(
					html.WithClasses(false), // Use inline styles
				),
			),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(), // Allow raw HTML
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
