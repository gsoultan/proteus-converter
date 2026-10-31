package css

import (
	"math"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func computeDoc(t *testing.T, src string, author string) (*html.Node, map[*html.Node]*ElementStyle) {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	mc := MediaContext{Type: "print", Width: 500, Height: 700}
	sheets := []Sheet{{Parse(UserAgentCSS, mc.Match), OriginUserAgent}}
	if author != "" {
		sheets = append(sheets, Sheet{Parse(author, mc.Match), OriginAuthor})
	}
	c := NewCascade(sheets, 500, 700, func(s string) { t.Log(s) })
	return doc, c.Compute(doc)
}

func find(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := find(c, id); f != nil {
			return f
		}
	}
	return nil
}

func TestCascadeOrder(t *testing.T) {
	doc, st := computeDoc(t, `<html><body><p id="a" class="x" style="margin-top: 5px">hi <b id="b">x</b></p><p id="c" class="x">y</p></body></html>`, `
		p { color: red; margin-top: 10px }
		.x { color: green !important }
		#a { color: blue }
		#c.x { color: purple !important }
		@media screen { p { color: orange !important } }
		@media print { #b { font-weight: normal } }
	`)
	a := st[find(doc, "a")].Style
	if a.Color != (Color{0, 128.0 / 255, 0, 1}) {
		t.Errorf("important class should beat id: %+v", a.Color)
	}
	if a.Margin[Top].Pt != 3.75 {
		t.Errorf("inline style should win: %v", a.Margin[Top])
	}
	if c := st[find(doc, "c")].Style.Color; c != rgb255(128, 0, 128) {
		t.Errorf("more specific important wins: %+v", c)
	}
	if b := st[find(doc, "b")].Style; b.FontWeight != 400 || b.Color != a.Color {
		t.Errorf("print media / inheritance: %+v", b)
	}
}

func TestVarsCalcShorthands(t *testing.T) {
	doc, st := computeDoc(t, `<div id="d" style="--gap: 4px; --c: #ff0000"><span id="s">x</span></div>`, `
		:root { --base: 10px }
		div { padding: calc(var(--base) * 2) var(--gap); border: 2px solid var(--c); font: italic bold 20px/1.5 "Foo Bar", serif }
		span { margin: var(--missing, 1em) }
	`)
	d := st[find(doc, "d")].Style
	if d.Padding[Top].Pt != 15 || d.Padding[Left].Pt != 3 {
		t.Errorf("padding: %+v", d.Padding)
	}
	if d.BorderWidth[Top] != 1.5 || d.BorderStyle[Left] != "solid" || d.BorderColor[Bottom] != rgb255(255, 0, 0) {
		t.Errorf("border: %+v %+v %+v", d.BorderWidth, d.BorderStyle, d.BorderColor)
	}
	if d.FontSize != 15 || d.FontWeight != 700 || d.FontStyle != "italic" || d.LineHeight.Factor != 1.5 || d.FontFamily[0] != "Foo Bar" {
		t.Errorf("font: %+v", d)
	}
	s := st[find(doc, "s")].Style
	if s.Margin[Top].Pt != 15 {
		t.Errorf("var fallback em: %+v", s.Margin)
	}
}

func TestMonospaceQuirkAndHeadings(t *testing.T) {
	doc, st := computeDoc(t, `<h1 id="h">T</h1><pre id="p">x</pre><code id="c" style="font-family: monospace, monospace">y</code>`, "")
	if h := st[find(doc, "h")].Style; h.FontSize != 24 || math.Abs(h.Margin[Top].Pt-16.08) > 1e-6 {
		t.Errorf("h1: %v %v", h.FontSize, h.Margin[Top])
	}
	if p := st[find(doc, "p")].Style; p.FontSize != 9.75 {
		t.Errorf("pre should be 13px: %v", p.FontSize)
	}
	if c := st[find(doc, "c")].Style; c.FontSize != 12 {
		t.Errorf("monospace,monospace should be 16px: %v", c.FontSize)
	}
}

func TestLayoutProperties(t *testing.T) {
	doc, styles := computeDoc(t, `<div id=f style="display:flex;flex:1 0 20px;gap:4px 8px;justify-content:space-between">
<span id=i>x</span></div>
<div id=g style="display:grid;grid-template-columns:100px repeat(2, minmax(10px, 1fr)) auto;grid-area:2 / 1 / span 2 / 3"></div>
<div id=b style="background:url(a.png) no-repeat right 10px top / cover #fff;position:absolute;inset:1px 2px"></div>`, "")
	f := styles[find(doc, "f")].Style
	if f.FlexGrow != 1 || f.FlexShrink != 0 || f.FlexBasis.Pt != 15 || f.RowGap.Pt != 3 || f.ColumnGap.Pt != 6 || f.JustifyContent != "space-between" {
		t.Errorf("flex: %+v %+v %+v %+v %+v", f.FlexGrow, f.FlexShrink, f.FlexBasis, f.RowGap, f.ColumnGap)
	}
	if i := styles[find(doc, "i")].Style; i.Display != "block" {
		t.Errorf("flex item display = %q, want block", i.Display)
	}
	g := styles[find(doc, "g")].Style
	if n := len(g.GridTemplateColumns.Tracks); n != 4 {
		t.Fatalf("tracks = %d", n)
	}
	if tr := g.GridTemplateColumns.Tracks[1]; tr.Min.Kind != TrackLength || tr.Max.Kind != TrackFr || tr.Max.Fr != 1 {
		t.Errorf("minmax track = %+v", tr)
	}
	if p := g.GridPlacement; p[0].Line != 2 || p[1].Line != 1 || p[2].Span != 2 || p[3].Line != 3 {
		t.Errorf("placement = %+v", p)
	}
	b := styles[find(doc, "b")].Style
	if b.BackgroundImage != "a.png" || b.BackgroundSize.Mode != "cover" || b.BackgroundRepeat[0] != "no-repeat" {
		t.Errorf("background = %q %+v %v", b.BackgroundImage, b.BackgroundSize, b.BackgroundRepeat)
	}
	if bp := b.BackgroundPosition; bp[0].Pct != 100 || bp[0].Pt != -7.5 || bp[1].Pct != 0 {
		t.Errorf("background-position = %+v", bp)
	}
	if b.BackgroundColor.R != 1 || b.Inset[0].Pt != 0.75 || b.Inset[1].Pt != 1.5 || b.Inset[3].Pt != 1.5 {
		t.Errorf("color/inset = %+v %+v", b.BackgroundColor, b.Inset)
	}
}
