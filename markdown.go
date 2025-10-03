package main

import (
	"bytes"
	"strconv"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
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

// CodeBlockRenderer renders code blocks with data-line attributes
type CodeBlockRenderer struct {
	html.Config
}

func NewCodeBlockRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &CodeBlockRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *CodeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
}

func (r *CodeBlockRenderer) renderCodeBlock(
	w util.BufWriter,
	source []byte,
	node ast.Node,
	entering bool,
) (ast.WalkStatus, error) {
	if entering {
		// Write <pre> with data-line attributes
		_, _ = w.WriteString("<pre")

		// Output data-line-start and data-line-end if they exist
		if lineStart, ok := node.Attribute([]byte("data-line-start")); ok {
			_, _ = w.WriteString(` data-line-start="`)
			_, _ = w.Write(lineStart.([]byte))
			_, _ = w.WriteString(`"`)
		}
		if lineEnd, ok := node.Attribute([]byte("data-line-end")); ok {
			_, _ = w.WriteString(` data-line-end="`)
			_, _ = w.Write(lineEnd.([]byte))
			_, _ = w.WriteString(`"`)
		}

		_, _ = w.WriteString("><code>")

		// Write code content
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			_, _ = w.Write(line.Value(source))
		}

		_, _ = w.WriteString("</code></pre>\n")
	}
	return ast.WalkContinue, nil
}

// RenderMarkdownWithLineNumbers renders markdown to HTML with line number attributes
func RenderMarkdownWithLineNumbers(source []byte) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(&LineAttributeExtension{}),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML
			renderer.WithNodeRenderers(
				util.Prioritized(NewCodeBlockRenderer(), 100),
			),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
