package layout_test

import (
	"math"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
	"github.com/gsoultan/proteus-converter/layout"
)

// Content area used by every test: 400 x 600 pt.
const pageW, pageH = 400.0, 600.0

// layoutHTML runs cascade, box building and layout for src and returns the
// items of its first page.
func layoutHTML(t *testing.T, src string) []layout.Item {
	t.Helper()
	return pagesHTML(t, src)[0]
}

// pagesHTML lays out src and returns the items of every page.
func pagesHTML(t *testing.T, src string) [][]layout.Item {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	mc := css.MediaContext{Type: "print", Width: pageW, Height: pageH}
	sheets := []css.Sheet{{Stylesheet: css.Parse(css.UserAgentCSS, mc.Match), Origin: css.OriginUserAgent}}
	var styleText strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "style" && n.FirstChild != nil {
			styleText.WriteString(n.FirstChild.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	sheets = append(sheets, css.Sheet{Stylesheet: css.Parse(styleText.String(), mc.Match), Origin: css.OriginAuthor})
	warn := func(string) {}
	styles := css.NewCascade(sheets, pageW, pageH, warn).Compute(doc)
	bd := &layout.Builder{Styles: styles, Warn: warn}
	root := bd.Build(doc)
	eng := &layout.Engine{Fonts: fonts.NewResolver(), PageWidth: pageW, PageHeight: pageH, Warn: warn}
	pages, err := eng.Paginate(eng.Layout(root))
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("no pages")
	}
	out := make([][]layout.Item, len(pages))
	for i, p := range pages {
		out[i] = p.Items
	}
	return out
}

// rect finds the background rect painted in the given 0-255 color.
func rect(t *testing.T, items []layout.Item, r, g, b int) *layout.RectItem {
	t.Helper()
	for _, it := range items {
		if v, ok := it.(*layout.RectItem); ok && same(v.Color, r, g, b) {
			return v
		}
	}
	t.Fatalf("no rect with color %d,%d,%d", r, g, b)
	return nil
}

func same(c css.Color, r, g, b int) bool {
	return math.Abs(c.R*255-float64(r)) < 0.5 && math.Abs(c.G*255-float64(g)) < 0.5 && math.Abs(c.B*255-float64(b)) < 0.5
}

// text finds the first text item whose text contains s.
func text(t *testing.T, items []layout.Item, s string) *layout.TextItem {
	t.Helper()
	for _, it := range items {
		if v, ok := it.(*layout.TextItem); ok && strings.Contains(v.Text, s) {
			return v
		}
	}
	t.Fatalf("no text %q", s)
	return nil
}

func near(t *testing.T, what string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.05 {
		t.Errorf("%s = %.2f, want %.2f", what, got, want)
	}
}

const reset = `<style>body{margin:0;font:12px sans-serif;line-height:20px}</style>`

func TestFloatsWrapTextAndClear(t *testing.T) {
	items := layoutHTML(t, reset+`<style>
		.l{float:left;width:100px;height:60px;background:#ff0000}
		.r{float:right;width:50px;height:30px;background:#00ff00}
		.c{clear:both;height:10px;background:#0000ff}
	</style><div class=l></div><div class=r></div><p style="margin:0">alpha</p><div class=c></div>`)
	l, r, c := rect(t, items, 255, 0, 0), rect(t, items, 0, 255, 0), rect(t, items, 0, 0, 255)
	near(t, "left float x", l.X, 0)
	near(t, "right float x", r.X, pageW-50*0.75)
	near(t, "right float y", r.Y, 0)
	if a := text(t, items, "alpha"); a.X < l.X+l.W-0.01 {
		t.Errorf("text x %.2f overlaps the left float ending at %.2f", a.X, l.X+l.W)
	}
	near(t, "cleared block y", c.Y, 45)
}

func TestAbsoluteAndRelativePositioning(t *testing.T) {
	items := layoutHTML(t, reset+`<style>
		.cb{position:relative;margin:20px;height:100px;background:#0000ff}
		.a{position:absolute;right:10px;bottom:10px;width:40px;height:20px;background:#ff0000}
		.rel{position:relative;left:8px;top:4px;height:10px;background:#00ff00}
	</style><div class=cb><div class=a></div></div><div class=rel></div>`)
	cb, a, rel := rect(t, items, 0, 0, 255), rect(t, items, 255, 0, 0), rect(t, items, 0, 255, 0)
	near(t, "abs x", a.X, cb.X+cb.W-(10+40)*0.75)
	near(t, "abs y", a.Y, cb.Y+cb.H-(10+20)*0.75)
	near(t, "relative x", rel.X, 6)
	near(t, "relative y", rel.Y, (20+100+20+4)*0.75)
}

func TestFlexGrowAndJustify(t *testing.T) {
	items := layoutHTML(t, reset+`<style>
		.f{display:flex;gap:10px;align-items:center;height:80px}
		.a{width:50px;height:20px;background:#ff0000}
		.b{flex:1;height:40px;background:#00ff00}
		.g{display:flex;justify-content:space-between}
		.g div{width:40px;height:10px}
	</style><div class=f><div class=a></div><div class=b></div></div>
	<div class=g><div style="background:#0000ff"></div><div style="background:#ffff00"></div></div>`)
	a, b := rect(t, items, 255, 0, 0), rect(t, items, 0, 255, 0)
	near(t, "fixed item y (centered)", a.Y, 30*0.75)
	near(t, "grown item x", b.X, 60*0.75)
	near(t, "grown item width", b.W, pageW-60*0.75)
	near(t, "space-between last item x", rect(t, items, 255, 255, 0).X, pageW-30)
	near(t, "space-between first item x", rect(t, items, 0, 0, 255).X, 0)
}

func TestGridTracksAndAreas(t *testing.T) {
	items := layoutHTML(t, reset+`<style>
		.g{display:grid;grid-template-columns:100px 1fr 2fr;grid-template-rows:30px auto;column-gap:10px;
		   grid-template-areas:"a b c" "a d d"}
		.g div{height:20px}
	</style><div class=g>
		<div style="grid-area:a;background:#ff0000"></div>
		<div style="grid-area:c;background:#00ff00"></div>
		<div style="grid-area:d;background:#0000ff"></div>
	</div>`)
	free := pageW - (100+20)*0.75 // the fr tracks share what is left
	a, c, d := rect(t, items, 255, 0, 0), rect(t, items, 0, 255, 0), rect(t, items, 0, 0, 255)
	near(t, "area a x", a.X, 0)
	near(t, "area c x", c.X, 110*0.75+free/3+10*0.75)
	near(t, "area c width", c.W, free*2/3)
	near(t, "area d x", d.X, 110*0.75)
	near(t, "area d y", d.Y, 30*0.75)
	near(t, "area d width", d.W, free+10*0.75)
}

func TestGradientBackgroundPaintsPaths(t *testing.T) {
	items := layoutHTML(t, reset+`<div style="height:50px;background:linear-gradient(to right,#ff0000,#0000ff)"></div>`)
	n := 0
	for _, it := range items {
		if _, ok := it.(*layout.PathItem); ok {
			n++
		}
	}
	if n < 2 {
		t.Fatalf("gradient painted %d path strips, want several", n)
	}
}
