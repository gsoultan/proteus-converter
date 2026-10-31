// Package pipeline runs the shared conversion pipeline for HTML-shaped
// sources: collect stylesheets, cascade, build boxes, lay out, paginate and
// paint. Format front ends (HTML, Markdown, later DOCX) parse their input
// into an HTML tree and hand it to Render.
package pipeline

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
	"github.com/gsoultan/proteus-converter/layout"
	"github.com/gsoultan/proteus-converter/render"
	"golang.org/x/net/html"
)

// Input is a parsed document ready for rendering.
type Input struct {
	Doc      *html.Node
	BaseURL  string   // overrides cfg.BaseURL when set
	ThemeCSS []string // format-specific author CSS applied before the document's own styles
}

type sourceSheet struct {
	css   string
	base  string
	media string
}

// Render converts a parsed document into a PDF written to w.
func Render(ctx context.Context, in Input, cfg *core.Config, w io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	loader := newCachingLoader(core.DefaultLoader(cfg))
	base := cfg.BaseURL
	if in.BaseURL != "" {
		base = in.BaseURL
	}
	if b := findBase(in.Doc); b != "" {
		base = core.ResolveURL(base, b)
	}
	warn := cfg.Warnf

	// 1. stylesheets in cascade order
	var author []sourceSheet
	for _, t := range in.ThemeCSS {
		author = append(author, sourceSheet{css: t, base: base})
	}
	collectSheets(ctx, in.Doc, base, loader, warn, &author)
	for _, s := range cfg.Stylesheets {
		author = append(author, sourceSheet{css: s, base: base})
	}

	// 2. page geometry: options, then @page rules
	pageSize, marginsBox := cfg.PageSize, cfg.Margins
	parseAll := func(mc css.MediaContext) ([]css.Sheet, []parsedSheet) {
		sheets := []css.Sheet{{Stylesheet: css.Parse(css.UserAgentCSS, mc.Match), Origin: css.OriginUserAgent}}
		var parsed []parsedSheet
		for _, s := range author {
			if s.media != "" && !mc.Match(s.media) {
				continue
			}
			for _, ps := range expandImports(ctx, s.css, s.base, mc, loader, warn, 0) {
				sheets = append(sheets, css.Sheet{Stylesheet: ps.sheet, Origin: css.OriginAuthor})
				parsed = append(parsed, ps)
			}
		}
		return sheets, parsed
	}
	mc := css.MediaContext{Type: cfg.MediaType, Width: pageSize.Width - marginsBox.Left - marginsBox.Right,
		Height: pageSize.Height - marginsBox.Top - marginsBox.Bottom}
	sheets, parsed := parseAll(mc)
	ps, pm := pageRules(parsed, pageSize, marginsBox)
	if !cfg.PageSizeSet {
		pageSize = ps
	}
	if !cfg.MarginsSet {
		marginsBox = pm
	}
	cw := pageSize.Width - marginsBox.Left - marginsBox.Right
	ch := pageSize.Height - marginsBox.Top - marginsBox.Bottom
	if cw <= 10 || ch <= 10 {
		return fmt.Errorf("proteus: page content area too small (%.1fx%.1f pt)", cw, ch)
	}
	if cw != mc.Width || ch != mc.Height {
		mc.Width, mc.Height = cw, ch
		sheets, parsed = parseAll(mc)
	}

	// 3. fonts
	var res *fonts.Resolver
	if cfg.SystemFonts {
		res = fonts.System().Clone()
	} else {
		res = fonts.NewResolver()
	}
	for _, d := range cfg.FontDirs {
		if err := res.AddDir(d); err != nil {
			warn("fonts: " + err.Error())
		}
	}
	for _, f := range cfg.Fonts {
		if err := res.AddData(f.Data, f.Family, f.Weight, f.Italic); err != nil {
			warn(fmt.Sprintf("fonts: %s: %v", f.Family, err))
		}
	}
	loadFontFaces(ctx, parsed, loader, res, warn)
	if err := ctx.Err(); err != nil {
		return err
	}

	// 4. cascade and boxes
	cascade := css.NewCascade(sheets, cw, ch, warn)
	styles := cascade.Compute(in.Doc)
	images := newImageLoader(ctx, loader, base, warn)
	bd := &layout.Builder{Styles: styles, Images: images.load, Warn: warn}
	root := bd.Build(in.Doc)
	if err := ctx.Err(); err != nil {
		return err
	}

	// 5. layout and pagination
	eng := &layout.Engine{Fonts: res, PageWidth: cw, PageHeight: ch, Warn: warn}
	canvas, hasCanvas := eng.CanvasBackground(root)
	items := eng.Layout(root)
	pages := eng.Paginate(items)
	if err := ctx.Err(); err != nil {
		return err
	}

	// 6. paint
	meta := documentMetadata(in.Doc)
	mergeMeta(&meta, cfg.Metadata)
	p := render.NewGPDF(meta, cfg.Outline, warn)
	d := &render.Document{PageWidth: pageSize.Width, PageHeight: pageSize.Height,
		MarginTop: marginsBox.Top, MarginLeft: marginsBox.Left, Pages: pages}
	if hasCanvas {
		d.Background = &canvas
	}
	return render.Paint(d, p, w)
}

func mergeMeta(m *render.Metadata, o core.Metadata) {
	set := func(dst *string, v string) {
		if v != "" {
			*dst = v
		}
	}
	set(&m.Title, o.Title)
	set(&m.Author, o.Author)
	set(&m.Subject, o.Subject)
	set(&m.Keywords, o.Keywords)
	set(&m.Creator, o.Creator)
	set(&m.Producer, o.Producer)
	set(&m.Language, o.Language)
}

func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func walk(n *html.Node, fn func(*html.Node) bool) {
	if !fn(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, fn)
	}
}

func findBase(doc *html.Node) string {
	var out string
	walk(doc, func(n *html.Node) bool {
		if out != "" {
			return false
		}
		if n.Type == html.ElementNode && n.Data == "base" {
			out, _ = attr(n, "href")
		}
		return n.Type != html.ElementNode || n.Data != "body"
	})
	return out
}

func textOf(n *html.Node) string {
	var b strings.Builder
	walk(n, func(c *html.Node) bool {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
		return true
	})
	return b.String()
}

func documentMetadata(doc *html.Node) render.Metadata {
	var m render.Metadata
	walk(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return true
		}
		switch n.Data {
		case "html":
			m.Language, _ = attr(n, "lang")
		case "title":
			if m.Title == "" {
				m.Title = strings.Join(strings.Fields(textOf(n)), " ")
			}
		case "meta":
			name, _ := attr(n, "name")
			content, _ := attr(n, "content")
			switch strings.ToLower(name) {
			case "author":
				m.Author = content
			case "description", "subject":
				m.Subject = content
			case "keywords":
				m.Keywords = content
			case "generator":
				m.Creator = content
			}
		case "body":
			return false
		}
		return true
	})
	return m
}
