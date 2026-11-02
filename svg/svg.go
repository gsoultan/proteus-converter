// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package svg parses static SVG documents into flattened vector shapes:
// paths with resolved fill and stroke paint, already mapped through every
// element transform. It covers the subset that documents, icons and charts
// use (shapes, paths, groups, use, transforms, styles and simple text); there
// is no scripting, animation, filters, masks or clipping.
package svg

import (
	"bytes"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/gsoultan/proteus-converter/css"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Matrix is an affine transform [a b c d e f]: x' = a*x + c*y + e, y' = b*x + d*y + f.
type Matrix [6]float64

// Identity is the identity transform.
var Identity = Matrix{1, 0, 0, 1, 0, 0}

// Mul returns the transform that applies n first, then m.
func (m Matrix) Mul(n Matrix) Matrix {
	return Matrix{
		m[0]*n[0] + m[2]*n[1],
		m[1]*n[0] + m[3]*n[1],
		m[0]*n[2] + m[2]*n[3],
		m[1]*n[2] + m[3]*n[3],
		m[0]*n[4] + m[2]*n[5] + m[4],
		m[1]*n[4] + m[3]*n[5] + m[5],
	}
}

// Apply maps a point.
func (m Matrix) Apply(x, y float64) (float64, float64) {
	return m[0]*x + m[2]*y + m[4], m[1]*x + m[3]*y + m[5]
}

// Scale returns the mean scale factor of the transform (for stroke widths and font sizes).
func (m Matrix) Scale() float64 {
	return math.Sqrt(math.Abs(m[0]*m[3] - m[1]*m[2]))
}

// Translate returns a translation.
func Translate(x, y float64) Matrix { return Matrix{1, 0, 0, 1, x, y} }

// ScaleXY returns a scaling.
func ScaleXY(sx, sy float64) Matrix { return Matrix{sx, 0, 0, sy, 0, 0} }

// Seg is a path segment in absolute coordinates: 'M' and 'L' use Pts[0:2],
// 'C' uses all six values, 'Z' none.
type Seg struct {
	Op  byte
	Pts [6]float64
}

// Shape is a painted path.
type Shape struct {
	Segs        []Seg
	Transform   Matrix // user space to SVG viewport space
	Fill        *css.Color
	EvenOdd     bool
	Stroke      *css.Color
	StrokeWidth float64
	LineCap     int // 0 butt, 1 round, 2 square
	LineJoin    int // 0 miter, 1 round, 2 bevel
	MiterLimit  float64
	Dash        []float64
}

// Text is a run of text positioned at a baseline point.
type Text struct {
	X, Y       float64 // anchor point in user space
	Transform  Matrix
	Text       string
	FontFamily []string
	FontSize   float64 // user units
	FontWeight int
	Italic     bool
	Anchor     string // start, middle, end
	Fill       css.Color
}

// Item is a Shape or a Text, in painting order.
type Item struct {
	Shape *Shape
	Text  *Text
}

// Document is a parsed SVG image.
type Document struct {
	Width, Height float64 // intrinsic size in CSS px; 0 when unknown
	ViewBox       [4]float64
	HasViewBox    bool
	Align         string // preserveAspectRatio alignment, e.g. "xMidYMid"; "none" stretches
	Slice         bool
	Items         []Item
	Warnings      []string
}

// Ratio returns height / width of the image, or 0 when unknown.
func (d *Document) Ratio() float64 {
	switch {
	case d.Width > 0 && d.Height > 0:
		return d.Height / d.Width
	case d.HasViewBox && d.ViewBox[2] > 0:
		return d.ViewBox[3] / d.ViewBox[2]
	}
	return 0
}

// ViewportTransform maps SVG user space onto a w x h box at the origin.
func (d *Document) ViewportTransform(w, h float64) Matrix {
	if !d.HasViewBox || d.ViewBox[2] <= 0 || d.ViewBox[3] <= 0 {
		sx, sy := 1.0, 1.0
		if d.Width > 0 {
			sx = w / d.Width
		}
		if d.Height > 0 {
			sy = h / d.Height
		}
		return ScaleXY(sx, sy)
	}
	vx, vy, vw, vh := d.ViewBox[0], d.ViewBox[1], d.ViewBox[2], d.ViewBox[3]
	sx, sy := w/vw, h/vh
	if d.Align != "none" {
		s := math.Min(sx, sy)
		if d.Slice {
			s = math.Max(sx, sy)
		}
		sx, sy = s, s
	}
	tx, ty := -vx*sx, -vy*sy
	align := d.Align
	if align == "" {
		align = "xMidYMid"
	}
	if align != "none" {
		switch {
		case strings.HasPrefix(align, "xMid"):
			tx += (w - vw*sx) / 2
		case strings.HasPrefix(align, "xMax"):
			tx += w - vw*sx
		}
		switch {
		case strings.HasSuffix(align, "YMid"):
			ty += (h - vh*sy) / 2
		case strings.HasSuffix(align, "YMax"):
			ty += h - vh*sy
		}
	}
	return Matrix{sx, 0, 0, sy, tx, ty}
}

// ErrNoSVG is returned when the data holds no <svg> element.
var ErrNoSVG = errors.New("svg: no <svg> element")

// Parse parses an SVG file.
func Parse(data []byte) (*Document, error) {
	// the HTML parser builds the same tree as for inline SVG and tolerates
	// the XML prolog, doctype and comments
	doc, err := html.Parse(bytes.NewReader(append([]byte("<body>"), data...)))
	if err != nil {
		return nil, err
	}
	var root *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if root != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "svg" {
			root = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	if root == nil {
		return nil, ErrNoSVG
	}
	return FromNode(root, css.Black), nil
}

// FromNode builds a document from an <svg> element. color is the value of currentColor.
func FromNode(root *html.Node, color css.Color) *Document {
	p := &parser{ids: map[string]*html.Node{}, warned: map[string]bool{}}
	d := &Document{Align: "xMidYMid"}
	p.doc = d
	p.collect(root)
	vw, vh := 300.0, 150.0
	if vb, ok := attr(root, "viewBox"); ok {
		f := numbers(vb)
		if len(f) == 4 && f[2] > 0 && f[3] > 0 {
			d.ViewBox = [4]float64{f[0], f[1], f[2], f[3]}
			d.HasViewBox = true
			vw, vh = f[2], f[3]
		}
	}
	if w, ok := attr(root, "width"); ok {
		if v, pct := length(w, 0); !pct && v > 0 {
			d.Width = v
		}
	}
	if h, ok := attr(root, "height"); ok {
		if v, pct := length(h, 0); !pct && v > 0 {
			d.Height = v
		}
	}
	if !d.HasViewBox {
		if d.Width > 0 {
			vw = d.Width
		}
		if d.Height > 0 {
			vh = d.Height
		}
	}
	if par, ok := attr(root, "preserveAspectRatio"); ok {
		f := strings.Fields(par)
		if len(f) > 0 {
			d.Align = f[0]
			d.Slice = len(f) > 1 && f[1] == "slice"
		}
	}
	p.vw, p.vh = vw, vh
	st := defaultState(color)
	p.children(root, st, Identity, 0)
	return d
}

// --- style state ------------------------------------------------------------

type state struct {
	fill, stroke             paint
	fillOpacity, strokeOp    float64
	opacity                  float64 // accumulated group opacity
	strokeWidth              float64
	evenOdd                  bool
	cap, join                int
	miter                    float64
	dash                     []float64
	color                    css.Color
	fontFamily               []string
	fontSize                 float64
	fontWeight               int
	italic                   bool
	anchor                   string
	visible                  bool
	display                  bool
	strokeWidthSet, colorSet bool
}

type paint struct {
	none bool
	c    css.Color
	url  string
	cur  bool // currentColor
}

func defaultState(color css.Color) state {
	return state{fill: paint{c: css.Black}, stroke: paint{none: true}, fillOpacity: 1, strokeOp: 1, opacity: 1,
		strokeWidth: 1, miter: 4, color: color, fontFamily: []string{"serif"}, fontSize: 16, fontWeight: 400,
		anchor: "start", visible: true, display: true}
}

type parser struct {
	doc    *Document
	ids    map[string]*html.Node
	rules  []svgRule
	vw, vh float64
	warned map[string]bool
}

type svgRule struct {
	sel   cascadia.Sel
	decls []css.Declaration
}

func (p *parser) warn(msg string) {
	if !p.warned[msg] {
		p.warned[msg] = true
		p.doc.Warnings = append(p.doc.Warnings, msg)
	}
}

func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key && (a.Namespace == "" || key == "href") {
			return a.Val, true
		}
	}
	return "", false
}

// collect indexes ids and <style> sheets.
func (p *parser) collect(n *html.Node) {
	if n.Type == html.ElementNode {
		if id, ok := attr(n, "id"); ok {
			p.ids[id] = n
		}
		if n.Data == "style" {
			var b strings.Builder
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					b.WriteString(c.Data)
				}
			}
			ss := css.Parse(b.String(), nil)
			for _, r := range ss.Rules {
				for _, s := range css.SplitCommas(r.Selector) {
					sel, err := cascadia.Parse(s)
					if err != nil {
						continue
					}
					p.rules = append(p.rules, svgRule{sel: sel, decls: r.Decls})
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		p.collect(c)
	}
}

// style computes the state of element n from its parent state.
func (p *parser) style(n *html.Node, parent state) state {
	s := parent
	s.opacity = parent.opacity
	s.display = true
	decl := map[string]string{}
	for _, a := range n.Attr {
		if a.Namespace == "" {
			decl[a.Key] = a.Val
		}
	}
	for _, r := range p.rules {
		if r.sel.Match(n) {
			for _, d := range r.decls {
				decl[d.Property] = d.Value
			}
		}
	}
	if v, ok := attr(n, "style"); ok {
		for _, d := range css.ParseDeclarations(v) {
			decl[d.Property] = d.Value
		}
	}
	if v, ok := decl["color"]; ok {
		if c, ok := css.ParseColor(v, parent.color); ok {
			s.color = c
		}
	}
	for k, v := range decl {
		v = strings.TrimSpace(v)
		if v == "inherit" {
			continue
		}
		switch k {
		case "fill":
			s.fill = p.parsePaint(v)
		case "stroke":
			s.stroke = p.parsePaint(v)
		case "fill-opacity":
			s.fillOpacity = opacity(v)
		case "stroke-opacity":
			s.strokeOp = opacity(v)
		case "opacity":
			s.opacity *= opacity(v)
		case "stroke-width":
			if w, _ := length(v, math.Hypot(p.vw, p.vh)/math.Sqrt2); w >= 0 {
				s.strokeWidth = w
			}
		case "fill-rule":
			s.evenOdd = v == "evenodd"
		case "stroke-linecap":
			s.cap = map[string]int{"butt": 0, "round": 1, "square": 2}[v]
		case "stroke-linejoin":
			s.join = map[string]int{"miter": 0, "round": 1, "bevel": 2, "miter-clip": 0, "arcs": 0}[v]
		case "stroke-miterlimit":
			if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 1 {
				s.miter = f
			}
		case "stroke-dasharray":
			s.dash = nil
			if v != "none" {
				for _, f := range numbers(v) {
					s.dash = append(s.dash, math.Abs(f))
				}
				if len(s.dash)%2 == 1 {
					s.dash = append(s.dash, s.dash...)
				}
				all0 := true
				for _, f := range s.dash {
					if f > 0 {
						all0 = false
					}
				}
				if all0 {
					s.dash = nil
				}
			}
		case "font-family":
			var fams []string
			for _, f := range css.SplitCommas(v) {
				if f = strings.TrimSpace(css.Unquote(f)); f != "" {
					fams = append(fams, f)
				}
			}
			if len(fams) > 0 {
				s.fontFamily = fams
			}
		case "font-size":
			if f, pct := length(v, parent.fontSize); f > 0 {
				if pct {
					f = parent.fontSize * f / 100
				}
				s.fontSize = f
			}
		case "font-weight":
			switch v {
			case "bold", "bolder":
				s.fontWeight = 700
			case "normal", "lighter":
				s.fontWeight = 400
			default:
				if f, err := strconv.Atoi(v); err == nil {
					s.fontWeight = f
				}
			}
		case "font-style":
			s.italic = v == "italic" || v == "oblique"
		case "text-anchor":
			s.anchor = v
		case "visibility":
			s.visible = v == "visible"
		case "display":
			s.display = v != "none"
		}
	}
	return s
}

func opacity(v string) float64 {
	v = strings.TrimSpace(v)
	f, err := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64)
	if err != nil {
		return 1
	}
	if strings.HasSuffix(v, "%") {
		f /= 100
	}
	return math.Max(0, math.Min(1, f))
}

func (p *parser) parsePaint(v string) paint {
	lv := strings.ToLower(v)
	switch {
	case lv == "none" || lv == "transparent":
		return paint{none: true}
	case lv == "currentcolor":
		return paint{cur: true}
	case strings.HasPrefix(lv, "url("):
		end := strings.IndexByte(v, ')')
		if end < 0 {
			return paint{none: true}
		}
		ref := strings.Trim(strings.TrimSpace(v[4:end]), `"'`)
		pt := paint{url: strings.TrimPrefix(ref, "#")}
		// fallback color after the url
		if rest := strings.TrimSpace(v[end+1:]); rest != "" {
			if c, ok := css.ParseColor(rest, css.Black); ok {
				pt.c = c
			}
		}
		return pt
	}
	if c, ok := css.ParseColor(v, css.Black); ok {
		return paint{c: c}
	}
	return paint{c: css.Black}
}

// resolve returns the color of a paint (gradients use a representative stop color).
func (p *parser) resolve(pt paint, s state, alpha float64) *css.Color {
	if pt.none {
		return nil
	}
	c := pt.c
	if pt.cur {
		c = s.color
	}
	if pt.url != "" {
		g := p.ids[pt.url]
		if g == nil {
			if pt.c.A == 0 {
				return nil
			}
		} else {
			gc, ok := p.gradientColor(g)
			if !ok {
				return nil
			}
			c = gc
		}
	}
	c.A *= alpha
	if c.A <= 0 {
		return nil
	}
	return &c
}

// gradientColor approximates a gradient by the average of its stop colors.
func (p *parser) gradientColor(g *html.Node) (css.Color, bool) {
	switch g.Data {
	case "linearGradient", "radialGradient":
	default:
		p.warn("svg: <" + g.Data + "> paint servers are not supported")
		return css.Color{}, false
	}
	p.warn("svg: gradients are painted with their average color")
	for depth := 0; depth < 8; depth++ {
		var sum css.Color
		n := 0
		for c := g.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode || c.Data != "stop" {
				continue
			}
			st := p.style(c, defaultState(css.Black))
			col := css.Black
			if v, ok := attr(c, "stop-color"); ok {
				col, _ = css.ParseColor(v, css.Black)
			}
			if v, ok := styleProp(c, "stop-color"); ok {
				col, _ = css.ParseColor(v, st.color)
			}
			a := 1.0
			if v, ok := attr(c, "stop-opacity"); ok {
				a = opacity(v)
			}
			if v, ok := styleProp(c, "stop-opacity"); ok {
				a = opacity(v)
			}
			sum.R += col.R
			sum.G += col.G
			sum.B += col.B
			sum.A += col.A * a
			n++
		}
		if n > 0 {
			f := float64(n)
			return css.Color{R: sum.R / f, G: sum.G / f, B: sum.B / f, A: sum.A / f}, true
		}
		// stops inherited through href
		href, ok := attr(g, "href")
		if !ok {
			return css.Color{}, false
		}
		next := p.ids[strings.TrimPrefix(href, "#")]
		if next == nil {
			return css.Color{}, false
		}
		g = next
	}
	return css.Color{}, false
}

func styleProp(n *html.Node, prop string) (string, bool) {
	v, ok := attr(n, "style")
	if !ok {
		return "", false
	}
	for _, d := range css.ParseDeclarations(v) {
		if d.Property == prop {
			return d.Value, true
		}
	}
	return "", false
}

// --- tree walk --------------------------------------------------------------

func (p *parser) children(n *html.Node, s state, m Matrix, depth int) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			p.element(c, s, m, depth)
		}
	}
}

func (p *parser) element(n *html.Node, parent state, m Matrix, depth int) {
	if depth > 64 {
		return
	}
	switch n.Data {
	case "defs", "style", "title", "desc", "metadata", "linearGradient", "radialGradient", "pattern",
		"clipPath", "mask", "marker", "filter", "symbol", "script":
		return
	case "foreignObject", "image", "switch":
		p.warn("svg: <" + n.Data + "> is not supported")
		return
	}
	s := p.style(n, parent)
	if !s.display {
		return
	}
	if t, ok := attr(n, "transform"); ok {
		m = m.Mul(parseTransform(t))
	}
	switch n.Data {
	case "g", "a":
		p.children(n, s, m, depth+1)
	case "svg":
		// nested viewport
		x, _ := p.len(n, "x", p.vw)
		y, _ := p.len(n, "y", p.vh)
		m = m.Mul(Translate(x, y))
		if vb, ok := attr(n, "viewBox"); ok {
			f := numbers(vb)
			w, _ := p.len(n, "width", p.vw)
			h, _ := p.len(n, "height", p.vh)
			if len(f) == 4 && f[2] > 0 && f[3] > 0 && w > 0 && h > 0 {
				sub := &Document{ViewBox: [4]float64{f[0], f[1], f[2], f[3]}, HasViewBox: true, Align: "xMidYMid"}
				m = m.Mul(sub.ViewportTransform(w, h))
			}
		}
		p.children(n, s, m, depth+1)
	case "use":
		href, ok := attr(n, "href")
		if !ok {
			return
		}
		ref := p.ids[strings.TrimPrefix(href, "#")]
		if ref == nil || isAncestor(ref, n) {
			return
		}
		x, _ := p.len(n, "x", p.vw)
		y, _ := p.len(n, "y", p.vh)
		m = m.Mul(Translate(x, y))
		if ref.Data == "symbol" {
			if vb, ok := attr(ref, "viewBox"); ok {
				f := numbers(vb)
				w, _ := p.len(n, "width", p.vw)
				h, _ := p.len(n, "height", p.vh)
				if len(f) == 4 && f[2] > 0 && f[3] > 0 && w > 0 && h > 0 {
					sub := &Document{ViewBox: [4]float64{f[0], f[1], f[2], f[3]}, HasViewBox: true, Align: "xMidYMid"}
					m = m.Mul(sub.ViewportTransform(w, h))
				}
			}
			p.children(ref, p.style(ref, s), m, depth+1)
			return
		}
		p.element(ref, s, m, depth+1)
	case "text":
		p.text(n, s, m)
	default:
		segs := p.geometry(n)
		if len(segs) == 0 || !s.visible {
			return
		}
		sh := &Shape{Segs: segs, Transform: m, EvenOdd: s.evenOdd, StrokeWidth: s.strokeWidth, LineCap: s.cap,
			LineJoin: s.join, MiterLimit: s.miter, Dash: s.dash}
		sh.Fill = p.resolve(s.fill, s, s.fillOpacity*s.opacity)
		if n.Data == "line" {
			sh.Fill = nil
		}
		if s.strokeWidth > 0 {
			sh.Stroke = p.resolve(s.stroke, s, s.strokeOp*s.opacity)
		}
		if sh.Fill == nil && sh.Stroke == nil {
			return
		}
		p.doc.Items = append(p.doc.Items, Item{Shape: sh})
	}
}

func isAncestor(a, n *html.Node) bool {
	for ; n != nil; n = n.Parent {
		if n == a {
			return true
		}
	}
	return false
}

func (p *parser) text(n *html.Node, s state, m Matrix) {
	x, _ := p.len(n, "x", p.vw)
	y, _ := p.len(n, "y", p.vh)
	dx, _ := p.len(n, "dx", p.vw)
	dy, _ := p.len(n, "dy", p.vh)
	x, y = x+dx, y+dy
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(c *html.Node) {
		for ; c != nil; c = c.NextSibling {
			switch c.Type {
			case html.TextNode:
				b.WriteString(c.Data)
			case html.ElementNode:
				if c.DataAtom == atom.Title {
					continue
				}
				walk(c.FirstChild)
			}
		}
	}
	walk(n.FirstChild)
	txt := strings.Join(strings.Fields(b.String()), " ")
	if txt == "" || !s.visible {
		return
	}
	fill := p.resolve(s.fill, s, s.fillOpacity*s.opacity)
	if fill == nil {
		return
	}
	p.doc.Items = append(p.doc.Items, Item{Text: &Text{X: x, Y: y, Transform: m, Text: txt, FontFamily: s.fontFamily,
		FontSize: s.fontSize, FontWeight: s.fontWeight, Italic: s.italic, Anchor: s.anchor, Fill: *fill}})
}

// len reads a length attribute; percentages resolve against basis.
func (p *parser) len(n *html.Node, key string, basis float64) (float64, bool) {
	v, ok := attr(n, key)
	if !ok {
		return 0, false
	}
	f, pct := length(v, 16)
	if pct {
		f = f / 100 * basis
	}
	return f, true
}

// length parses an SVG length in user units (px). pct reports a percentage
// (returned unresolved). em uses fontSize.
func length(v string, fontSize float64) (float64, bool) {
	v = strings.TrimSpace(strings.ToLower(v))
	if strings.HasSuffix(v, "%") {
		f, err := strconv.ParseFloat(v[:len(v)-1], 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	units := map[string]float64{"px": 1, "pt": 4.0 / 3, "pc": 16, "mm": 96 / 25.4, "cm": 96 / 2.54, "in": 96}
	for u, k := range units {
		if strings.HasSuffix(v, u) {
			f, err := strconv.ParseFloat(strings.TrimSpace(v[:len(v)-len(u)]), 64)
			if err != nil {
				return 0, false
			}
			return f * k, false
		}
	}
	if strings.HasSuffix(v, "em") {
		f, err := strconv.ParseFloat(strings.TrimSpace(v[:len(v)-2]), 64)
		if err != nil {
			return 0, false
		}
		return f * fontSize, false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, false
}

// numbers extracts every number of a list separated by spaces and/or commas.
func numbers(v string) []float64 {
	var out []float64
	sc := scanner{s: v}
	for {
		sc.skipSep()
		if sc.eof() {
			return out
		}
		f, ok := sc.number()
		if !ok {
			return out
		}
		out = append(out, f)
	}
}
