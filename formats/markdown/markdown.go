// Package markdown converts CommonMark / GitHub Flavored Markdown to PDF.
// Markdown is rendered to HTML with goldmark and then goes through the same
// CSS and layout pipeline as HTML, styled by a GitHub-like theme.
package markdown

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/pipeline"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
)

//go:embed theme.css
var themeCSS string

// Theme returns the built-in Markdown stylesheet.
func Theme() string { return themeCSS }

// Converter converts Markdown to PDF. It is safe for concurrent use.
type Converter struct {
	md goldmark.Markdown
}

// New returns a Markdown converter with GFM extensions (tables, task lists,
// strikethrough, autolinks), footnotes and heading IDs enabled. Raw HTML in
// the Markdown source is rendered.
func New() *Converter {
	return &Converter{md: goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, extension.DefinitionList),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)}
}

// Format implements core.Converter.
func (*Converter) Format() core.Format { return core.FormatMarkdown }

// Extensions implements core.Converter.
func (*Converter) Extensions() []string { return []string{"md", "markdown", "mdown", "mkd"} }

// ToHTML renders Markdown source to the standalone HTML document that is laid
// out (without the theme stylesheet; see Theme).
func (c *Converter) ToHTML(src []byte) ([]byte, error) {
	doc, err := c.document(src)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := html.Render(&out, doc); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (c *Converter) document(src []byte) (*html.Node, error) {
	var body bytes.Buffer
	if err := c.md.Convert(src, &body); err != nil {
		return nil, fmt.Errorf("proteus: render markdown: %w", err)
	}
	var page bytes.Buffer
	page.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"></head><body><article class=\"markdown-body\">\n")
	page.Write(body.Bytes())
	page.WriteString("</article></body></html>\n")
	doc, err := html.Parse(&page)
	if err != nil {
		return nil, fmt.Errorf("proteus: parse rendered markdown: %w", err)
	}
	markTaskLists(doc)
	return doc, nil
}

// Convert implements core.Converter.
func (c *Converter) Convert(ctx context.Context, src io.Reader, dst io.Writer, opts ...core.Option) error {
	cfg := core.NewConfig(opts...)
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	doc, err := c.document(data)
	if err != nil {
		return err
	}
	var theme []string
	switch {
	case cfg.NoMarkdownDefaults:
	case cfg.MarkdownCSS != "":
		theme = []string{cfg.MarkdownCSS}
	default:
		theme = []string{themeCSS}
	}
	return pipeline.Render(ctx, pipeline.Input{Doc: doc, ThemeCSS: theme}, cfg, dst)
}

// markTaskLists adds GitHub's task-list classes: "task-list-item" on list
// items starting with a checkbox and "contains-task-list" on their list.
func markTaskLists(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "li" {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" {
				continue
			}
			if c.Type == html.ElementNode && c.Data == "p" {
				c = c.FirstChild
				for c != nil && c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" {
					c = c.NextSibling
				}
			}
			if c != nil && c.Type == html.ElementNode && c.Data == "input" {
				addClass(n, "task-list-item")
				if n.Parent != nil {
					addClass(n.Parent, "contains-task-list")
				}
			}
			break
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		markTaskLists(c)
	}
}

func addClass(n *html.Node, class string) {
	for i, a := range n.Attr {
		if a.Key == "class" {
			if !slices.Contains(strings.Fields(a.Val), class) {
				n.Attr[i].Val = strings.TrimSpace(a.Val + " " + class)
			}
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: "class", Val: class})
}
