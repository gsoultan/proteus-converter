// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package layout

import (
	"math"

	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
)

// Engine lays out a box tree into display items.
type Engine struct {
	Fonts      *fonts.Resolver
	PageWidth  float64 // content width of a page, in points
	PageHeight float64 // content height of a page, in points
	MaxPages   int     // Paginate fails beyond this many pages; <= 0 means no limit
	Warn       func(string)

	metrics    map[*css.Style]styleMetrics
	warned     map[string]bool
	cellHeight map[*Box]float64
	canvasBox  *Box // box whose background was propagated to the page canvas
	intrinsic  map[*Box][2]float64
	forcedH    map[*Box]float64 // border-box heights imposed by flex, grid and absolute layout
	forcedW    map[*Box]float64 // border-box widths imposed on replaced flex and grid items

	fixedOK    bool     // position: fixed boxes repeat on every page
	fixedBoxes []absBox // collected during layout
	fixedItems []Item   // laid out against the page area, copied onto every page
}

// CanvasBackground returns the page background propagated from the root or
// body element (as browsers do) and records which box it came from.
func (e *Engine) CanvasBackground(root *Box) (bg css.Color, ok bool) {
	if !root.Style.BackgroundColor.IsTransparent() {
		e.canvasBox = root
		return root.Style.BackgroundColor, true
	}
	for _, c := range root.Children {
		if c.Node != nil && c.Node.Data == "body" {
			if !c.Style.BackgroundColor.IsTransparent() {
				e.canvasBox = c
				return c.Style.BackgroundColor, true
			}
		}
	}
	return css.Color{}, false
}

// Layout lays out root (the document element box) and returns its items.
func (e *Engine) Layout(root *Box) []Item {
	e.metrics = map[*css.Style]styleMetrics{}
	e.intrinsic = map[*Box][2]float64{}
	e.cellHeight = map[*Box]float64{}
	e.forcedH = map[*Box]float64{}
	e.forcedW = map[*Box]float64{}
	e.fixedBoxes, e.fixedItems = nil, nil
	e.fixedOK = true
	l := &layouter{e: e, fragment: true, fc: &floatCtx{}, pos: &posCtx{}}
	l.layoutBlock(root, 0, 0, e.PageWidth, margins{}, true)
	// absolutely positioned boxes without a positioned ancestor use the first page area
	l.layoutAbsolute(l.pos, Clip{W: e.PageWidth, H: e.PageHeight})
	e.fixedOK = false
	if len(e.fixedBoxes) > 0 {
		fl := &layouter{e: e}
		fl.layoutAbsolute(&posCtx{boxes: e.fixedBoxes}, Clip{W: e.PageWidth, H: e.PageHeight})
		e.fixedItems = fl.items
	}
	return l.items
}

func (e *Engine) warn(msg string) {
	if e.Warn != nil {
		e.Warn(msg)
	}
}

// layouter accumulates items for one formatting pass.
type layouter struct {
	e        *Engine
	items    []Item
	fragment bool // apply page breaks
	marker   *pendingMarker
	fc       *floatCtx // floats of the current block formatting context
	pos      *posCtx   // absolutely positioned boxes waiting for their containing block
}

// sub returns a layouter for an independent formatting context laid out at the origin.
func (l *layouter) sub() *layouter { return &layouter{e: l.e, fc: &floatCtx{}} }

func (l *layouter) emit(it Item) { l.items = append(l.items, it) }

// groupItem is a placeholder filled after the content it must be painted under is laid out.
type groupItem struct{ items []Item }

func (g *groupItem) translate(dx, dy float64) {
	for _, it := range g.items {
		it.translate(dx, dy)
	}
}

func translateItems(items []Item, dx, dy float64) {
	for _, it := range items {
		it.translate(dx, dy)
	}
}

// --- pagination helpers -----------------------------------------------------

const eps = 0.01

func (l *layouter) pageH() float64 { return l.e.PageHeight }

func (l *layouter) pageStart(y float64) float64 {
	h := l.pageH()
	return math.Floor((y+eps)/h) * h
}

func (l *layouter) nextPageStart(y float64) float64 { return l.pageStart(y) + l.pageH() }

func (l *layouter) atPageStart(y float64) bool { return math.Abs(y-l.pageStart(y)) < eps }

// place returns where a non-splittable piece of height h starting at y goes:
// unchanged when it fits on the current page, otherwise the next page start.
func (l *layouter) place(y, h float64) float64 {
	if !l.fragment || h <= 0 || l.atPageStart(y) {
		return y
	}
	if h > l.pageH() {
		return y
	}
	if y+h > l.nextPageStart(y)+eps {
		return l.nextPageStart(y)
	}
	return y
}

// afterMargin returns where content starts after the collapsed margin m
// below y. Like browsers, a margin that crosses into a new page is truncated
// (CSS Fragmentation: margins adjoining an unforced break): the content then
// starts at the top of that page instead of part way down it.
func (l *layouter) afterMargin(y float64, m margins) float64 {
	t := y + m.value()
	if l.fragment && m.value() > 0 && y > eps && l.pageStart(t) > l.pageStart(y-eps) {
		return l.pageStart(t)
	}
	return t
}

func isForcedBreak(v string) bool {
	switch v {
	case "page", "always", "left", "right", "recto", "verso", "all":
		return true
	}
	return false
}

// --- margins ------------------------------------------------------------------

// margins is a set of adjoining margins being collapsed.
type margins struct{ pos, neg float64 }

func (m margins) add(v float64) margins {
	if v > m.pos {
		m.pos = v
	}
	if v < m.neg {
		m.neg = v
	}
	return m
}

func (m margins) join(o margins) margins { return m.add(o.pos).add(o.neg) }

func (m margins) value() float64 { return m.pos + m.neg }

// --- font metrics per style -------------------------------------------------

type styleMetrics struct {
	chain   []*fonts.Face
	ascent  float64
	descent float64
	normal  float64 // normal line height
	lh      float64 // used line height
	xHeight float64
}

func (e *Engine) styleMetrics(s *css.Style) styleMetrics {
	if m, ok := e.metrics[s]; ok {
		return m
	}
	chain := e.Fonts.Chain(s.FontFamily, s.FontWeight, s.FontStyle != "normal")
	f := chain[0]
	a, d, n := f.LineMetrics(s.FontSize)
	m := styleMetrics{chain: chain, ascent: a, descent: d, normal: n}
	m.lh = s.LineHeightPt(n)
	m.xHeight = f.Metrics().XHeight * s.FontSize
	e.metrics[s] = m
	return m
}
