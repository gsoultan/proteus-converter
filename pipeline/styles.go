package pipeline

import (
	"context"
	"strings"
	"sync"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
	"golang.org/x/net/html"
)

type parsedSheet struct {
	sheet *css.Stylesheet
	base  string
}

// cachingLoader memoizes resources for the duration of one conversion.
type cachingLoader struct {
	inner core.ResourceLoader
	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	data []byte
	err  error
}

func newCachingLoader(l core.ResourceLoader) *cachingLoader {
	return &cachingLoader{inner: l, cache: map[string]cached{}}
}

func (c *cachingLoader) Load(ctx context.Context, ref string) ([]byte, error) {
	c.mu.Lock()
	if v, ok := c.cache[ref]; ok {
		c.mu.Unlock()
		return v.data, v.err
	}
	c.mu.Unlock()
	data, err := c.inner.Load(ctx, ref)
	c.mu.Lock()
	c.cache[ref] = cached{data, err}
	c.mu.Unlock()
	return data, err
}

// collectSheets gathers <style> and <link rel=stylesheet> in document order.
func collectSheets(ctx context.Context, doc *html.Node, base string, loader core.ResourceLoader, warn func(string), out *[]sourceSheet) {
	walk(doc, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return true
		}
		switch n.Data {
		case "style":
			if t, ok := attr(n, "type"); ok && t != "" && !strings.EqualFold(t, "text/css") {
				return false
			}
			media, _ := attr(n, "media")
			*out = append(*out, sourceSheet{css: textOf(n), base: base, media: media})
			return false
		case "link":
			rel, _ := attr(n, "rel")
			rels := strings.Fields(strings.ToLower(rel))
			isSheet, alt := false, false
			for _, r := range rels {
				isSheet = isSheet || r == "stylesheet"
				alt = alt || r == "alternate"
			}
			href, _ := attr(n, "href")
			if !isSheet || alt || href == "" {
				return false
			}
			u := core.ResolveURL(base, href)
			data, err := loader.Load(ctx, u)
			if err != nil {
				warn("css: stylesheet " + href + ": " + err.Error())
				return false
			}
			media, _ := attr(n, "media")
			*out = append(*out, sourceSheet{css: string(data), base: u, media: media})
			return false
		case "svg", "template", "script":
			return false
		}
		return true
	})
}

// expandImports parses css and returns its @import-ed sheets followed by itself.
func expandImports(ctx context.Context, src, base string, mc css.MediaContext, loader core.ResourceLoader, warn func(string), depth int) []parsedSheet {
	ss := css.Parse(src, mc.Match)
	var out []parsedSheet
	for _, imp := range ss.Imports {
		if imp.URL == "" || (imp.Media != "" && !mc.Match(imp.Media)) {
			continue
		}
		if depth >= 8 {
			warn("css: @import nesting too deep: " + imp.URL)
			continue
		}
		u := core.ResolveURL(base, imp.URL)
		data, err := loader.Load(ctx, u)
		if err != nil {
			warn("css: @import " + imp.URL + ": " + err.Error())
			continue
		}
		out = append(out, expandImports(ctx, string(data), u, mc, loader, warn, depth+1)...)
	}
	return append(out, parsedSheet{sheet: ss, base: base})
}

var namedPageSizes = map[string]core.PageSize{
	"a5": core.A5, "a4": core.A4, "a3": core.A3, "letter": core.Letter, "legal": core.Legal,
	"ledger": {Width: 792, Height: 1224}, "b5": {Width: 498.9, Height: 708.66}, "b4": {Width: 708.66, Height: 1000.63},
	"jis-b5": {Width: 515.91, Height: 728.5}, "jis-b4": {Width: 728.5, Height: 1031.81},
}

// pageRules applies @page size and margin declarations.
func pageRules(sheets []parsedSheet, size core.PageSize, m core.Margins) (core.PageSize, core.Margins) {
	lc := css.LengthContext{FontSize: 12, RootFontSize: 12, ViewportW: size.Width, ViewportH: size.Height}
	for _, ps := range sheets {
		for _, pg := range ps.sheet.Pages {
			sel := strings.TrimSpace(pg.Selector)
			if sel != "" && !strings.HasPrefix(sel, ":") {
				continue // named pages are not supported
			}
			if sel != "" {
				continue // :first/:left/:right not supported yet
			}
			for _, d := range pg.Decls {
				switch d.Property {
				case "size":
					size = pageSize(d.Value, size, lc)
				case "margin":
					toks := strings.Fields(d.Value)
					var v [4]float64
					cur := [4]float64{m.Top, m.Right, m.Bottom, m.Left}
					ok := true
					for i, t := range toks {
						if i > 3 {
							break
						}
						basis := size.Width
						l, good := css.ParseLength(t, lc, false)
						if !good || l.Auto {
							ok = false
							break
						}
						v[i] = l.Resolve(basis, 0)
					}
					if !ok {
						continue
					}
					switch len(toks) {
					case 1:
						cur = [4]float64{v[0], v[0], v[0], v[0]}
					case 2:
						cur = [4]float64{v[0], v[1], v[0], v[1]}
					case 3:
						cur = [4]float64{v[0], v[1], v[2], v[1]}
					case 4:
						cur = v
					}
					m = core.Margins{Top: cur[0], Right: cur[1], Bottom: cur[2], Left: cur[3]}
				case "margin-top", "margin-right", "margin-bottom", "margin-left":
					l, ok := css.ParseLength(d.Value, lc, false)
					if !ok || l.Auto {
						continue
					}
					v := l.Resolve(size.Width, 0)
					switch d.Property {
					case "margin-top":
						m.Top = v
					case "margin-right":
						m.Right = v
					case "margin-bottom":
						m.Bottom = v
					case "margin-left":
						m.Left = v
					}
				}
			}
		}
	}
	return size, m
}

func pageSize(v string, cur core.PageSize, lc css.LengthContext) core.PageSize {
	toks := strings.Fields(strings.ToLower(v))
	var lens []float64
	orient := ""
	named := core.PageSize{}
	for _, t := range toks {
		switch t {
		case "auto":
			return cur
		case "landscape", "portrait":
			orient = t
			continue
		}
		if s, ok := namedPageSizes[t]; ok {
			named = s
			continue
		}
		if l, ok := css.ParseLength(t, lc, false); ok && !l.Auto && l.Pct == 0 {
			lens = append(lens, l.Pt)
		}
	}
	out := cur
	switch {
	case named.Width > 0:
		out = named
	case len(lens) == 1:
		out = core.PageSize{Width: lens[0], Height: lens[0]}
	case len(lens) >= 2:
		out = core.PageSize{Width: lens[0], Height: lens[1]}
	}
	switch orient {
	case "landscape":
		out = out.Landscape()
	case "portrait":
		if out.Width > out.Height {
			out = core.PageSize{Width: out.Height, Height: out.Width}
		}
	}
	return out
}

// loadFontFaces registers @font-face fonts (TrueType/OpenType-glyf only).
func loadFontFaces(ctx context.Context, sheets []parsedSheet, loader core.ResourceLoader, res *fonts.Resolver, warn func(string)) {
	for _, ps := range sheets {
		for _, ff := range ps.sheet.FontFaces {
			var family, src string
			weight, italic := 400, false
			for _, d := range ff.Decls {
				switch d.Property {
				case "font-family":
					family = css.Unquote(d.Value)
				case "src":
					src = d.Value
				case "font-weight":
					f := strings.Fields(d.Value)
					if len(f) > 0 {
						switch f[0] {
						case "bold":
							weight = 700
						case "normal":
							weight = 400
						default:
							if n, ok := css.ParseNumber(f[0]); ok {
								weight = int(n)
							}
						}
					}
				case "font-style":
					italic = strings.HasPrefix(strings.ToLower(strings.TrimSpace(d.Value)), "italic") ||
						strings.HasPrefix(strings.ToLower(strings.TrimSpace(d.Value)), "oblique")
				}
			}
			if family == "" || src == "" {
				continue
			}
			loaded := false
			for _, alt := range css.SplitCommas(src) {
				toks := css.SplitValue(alt)
				if len(toks) == 0 {
					continue
				}
				u, ok := css.URLValue(toks[0])
				if !ok || !strings.HasPrefix(strings.ToLower(toks[0]), "url(") {
					continue
				}
				format := ""
				if len(toks) > 1 {
					if fn, args, ok := css.Function(toks[1]); ok && fn == "format" {
						format = strings.ToLower(css.Unquote(args))
					}
				}
				lu := strings.ToLower(u)
				if strings.HasPrefix(format, "woff") || strings.HasSuffix(lu, ".woff") || strings.HasSuffix(lu, ".woff2") {
					continue
				}
				data, err := loader.Load(ctx, core.ResolveURL(ps.base, u))
				if err != nil {
					warn("fonts: @font-face " + family + ": " + err.Error())
					continue
				}
				if err := res.AddData(data, family, weight, italic); err != nil {
					warn("fonts: @font-face " + family + ": " + err.Error())
					continue
				}
				loaded = true
				break
			}
			if !loaded {
				warn("fonts: @font-face " + family + ": no usable TrueType source (WOFF/WOFF2 and CFF fonts are not supported yet)")
			}
		}
	}
}
