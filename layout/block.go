package layout

import (
	"math"
	"strings"

	"github.com/gsoultan/proteus-converter/css"
)

type blockResult struct {
	top, bottom   float64 // border box
	pending       margins // margins collapsing below the box
	collapsed     bool    // empty box whose margins collapsed through
	firstBaseline float64
	lastBaseline  float64
	hasBaseline   bool
	pushedFirst   bool // the box moved to a later page because its first content did
	breakAfter    bool
}

type contentsResult struct {
	placed        bool
	firstTop      float64
	end           float64
	pending       margins
	firstBaseline float64
	lastBaseline  float64
	hasBaseline   bool
	pushedFirst   bool
}

// edges holds resolved horizontal box edges.
type edges struct {
	ml, mr, bl, br, pl, pr float64
	mlAuto, mrAuto         bool
}

func (l *layouter) hEdges(s *css.Style, cbw float64) edges {
	e := edges{
		bl: s.BorderUsed(css.Left), br: s.BorderUsed(css.Right),
		pl: s.Padding[css.Left].Resolve(cbw, 0), pr: s.Padding[css.Right].Resolve(cbw, 0),
	}
	e.mlAuto, e.mrAuto = s.Margin[css.Left].Auto, s.Margin[css.Right].Auto
	e.ml = s.Margin[css.Left].Resolve(cbw, 0)
	e.mr = s.Margin[css.Right].Resolve(cbw, 0)
	return e
}

// clampWidth applies min-width/max-width to a content width.
func clampWidth(s *css.Style, w, cbw, boxExtra float64) float64 {
	toContent := func(v float64) float64 {
		if s.BoxSizing == "border-box" {
			return v - boxExtra
		}
		return v
	}
	if !s.MaxWidth.Auto {
		w = math.Min(w, toContent(s.MaxWidth.Resolve(cbw, 0)))
	}
	if !s.MinWidth.Auto {
		w = math.Max(w, toContent(s.MinWidth.Resolve(cbw, 0)))
	}
	return math.Max(w, 0)
}

// blockWidth resolves the content width and margins of a block-level box.
// forced >= 0 imposes a border-box width (shrink-to-fit contexts).
func (l *layouter) blockWidth(s *css.Style, cbw, forced float64) (e edges, cw float64) {
	e = l.hEdges(s, cbw)
	extra := e.bl + e.br + e.pl + e.pr
	switch {
	case forced >= 0:
		cw = forced - extra
	case s.Width.Auto:
		cw = cbw - e.ml - e.mr - extra
	default:
		cw = s.Width.Resolve(cbw, 0)
		if s.BoxSizing == "border-box" {
			cw -= extra
		}
	}
	cw = clampWidth(s, cw, cbw, extra)
	if forced < 0 && (!s.Width.Auto || cw < cbw-e.ml-e.mr-extra-eps) {
		free := cbw - cw - extra - e.ml - e.mr
		switch {
		case e.mlAuto && e.mrAuto:
			e.ml += free / 2
			e.mr += free / 2
		case e.mlAuto:
			e.ml += free
		case e.mrAuto:
			e.mr += free
		}
	}
	return e, cw
}

func isBFC(b *Box) bool {
	s := b.Style
	if b.Kind == KindCell || b.Kind == KindInlineBlock || b.Kind == KindCaption || b.Kind == KindTable {
		return true
	}
	if s.Overflow != "visible" && s.Overflow != "" && s.Overflow != "clip" {
		return true
	}
	switch s.Display {
	case "flow-root", "flex", "grid", "inline-block", "table-cell":
		return true
	}
	return s.Float != "none" || s.Position == "absolute" || s.Position == "fixed"
}

func (l *layouter) layoutBlock(b *Box, x, y, cbw float64, pending margins, bfc bool) blockResult {
	return l.layoutBlockW(b, x, y, cbw, pending, bfc, -1)
}

func (l *layouter) layoutBlockW(b *Box, x, y, cbw float64, pending margins, bfc bool, forced float64) blockResult {
	s := b.Style
	startLen := len(l.items)
	savedMarker := l.marker
	forcedPush := false
	if l.fragment && isForcedBreak(s.BreakBefore) && !l.atPageStart(y) {
		y = l.nextPageStart(y)
		pending = margins{}
		forcedPush = true
	}
	mt := s.Margin[css.Top].Resolve(cbw, 0)
	mb := s.Margin[css.Bottom].Resolve(cbw, 0)

	switch b.Kind {
	case KindTable:
		r := l.layoutTable(b, x, y, cbw, pending.add(mt), forced)
		r.pending = margins{}.add(mb)
		r.pushedFirst = r.pushedFirst || forcedPush
		r.breakAfter = isForcedBreak(s.BreakAfter)
		l.emitIDs(b, r.top)
		return r
	case KindImage, KindCheckbox:
		a := l.layoutAtomic(b, cbw)
		top := y + pending.add(mt).value()
		ny := l.place(top, a.h-mt-mb)
		e := l.hEdges(s, cbw)
		left := x
		if e.mlAuto && e.mrAuto {
			left += (cbw - a.w) / 2
		}
		for _, it := range a.items {
			it.translate(left, ny-mt)
			l.emit(it)
		}
		if b.Href != "" {
			l.emitLink(b.Href, left, ny, a.w, a.h-mt-mb)
		}
		bottom := ny + a.h - mt - mb
		return blockResult{top: ny, bottom: bottom, pending: margins{}.add(mb), pushedFirst: ny != top || forcedPush,
			breakAfter: isForcedBreak(s.BreakAfter)}
	}

	e, cw := l.blockWidth(s, cbw, forced)
	bt, bb := s.BorderUsed(css.Top), s.BorderUsed(css.Bottom)
	pt, pb := s.Padding[css.Top].Resolve(cbw, 0), s.Padding[css.Bottom].Resolve(cbw, 0)
	bfc = bfc || isBFC(b)
	pend := pending.add(mt)
	bg := &groupItem{}
	l.emit(bg)
	if s.Display == "list-item" && b.Marker != nil {
		l.marker = &pendingMarker{box: b.Marker, inside: s.ListStylePosition == "inside"}
	}
	cx := x + e.ml + e.bl + e.pl
	collapseTop := !bfc && bt == 0 && pt == 0
	var top, contentTop float64
	var cr contentsResult
	if collapseTop {
		cr = l.layoutContents(b, cx, y, cw, pend)
		if cr.placed {
			top = cr.firstTop
		} else {
			top = y + pend.value()
		}
		contentTop = top
	} else {
		top = y + pend.value()
		contentTop = top + bt + pt
		cr = l.layoutContents(b, cx, contentTop, cw, margins{})
	}
	// an outside marker of an item without inline content sits on the first line position
	if l.marker != nil && l.marker.box == b.Marker && b.Marker != nil {
		mk := l.takeMarker()
		frags := l.buildFrags(mk.box.Children, mk.box.Style, cw, nil)
		sm := l.e.styleMetrics(mk.box.Style)
		half := (sm.lh - sm.ascent - sm.descent) / 2
		bl := contentTop + half + sm.ascent
		if cr.hasBaseline {
			bl = cr.firstBaseline
		}
		l.emitMarker(mk.box, frags, cx, bl)
	}
	if l.marker == nil {
		l.marker = savedMarker
		if savedMarker != nil && savedMarker.box == b.Marker {
			l.marker = nil
		}
	}

	heightAuto := s.Height.Auto || s.Height.HasPct()
	minH := 0.0
	if !s.MinHeight.HasPct() {
		minH = s.MinHeight.Resolve(0, 0)
	}
	collapseBottom := !bfc && bb == 0 && pb == 0 && heightAuto && minH == 0
	if !cr.placed && collapseTop && collapseBottom {
		l.items = l.items[:startLen]
		l.emitIDs(b, y)
		return blockResult{top: y, bottom: y, collapsed: true, pending: pend.join(cr.pending).add(mb),
			breakAfter: isForcedBreak(s.BreakAfter)}
	}
	contentEnd := contentTop
	if cr.placed {
		contentEnd = cr.end
	}
	var bottom float64
	var out margins
	if collapseBottom && cr.placed {
		bottom = contentEnd
		out = cr.pending.add(mb)
	} else {
		contentEnd += cr.pending.value()
		ch := math.Max(contentEnd-contentTop, 0)
		if !heightAuto {
			h := s.Height.Resolve(0, 0)
			if s.BoxSizing == "border-box" {
				h -= bt + bb + pt + pb
			}
			ch = math.Max(h, 0)
		}
		if !s.MaxHeight.Auto && !s.MaxHeight.HasPct() {
			mh := s.MaxHeight.Resolve(0, 0)
			if s.BoxSizing == "border-box" {
				mh -= bt + bb + pt + pb
			}
			ch = math.Min(ch, mh)
		}
		if minH > 0 {
			if s.BoxSizing == "border-box" {
				minH -= bt + bb + pt + pb
			}
			ch = math.Max(ch, minH)
		}
		if target, ok := l.e.cellHeight[b]; ok {
			if need := target - bt - pt - pb - bb; need > ch {
				shift := 0.0
				switch s.VerticalAlign {
				case "middle":
					shift = (need - ch) / 2
				case "bottom":
					shift = need - ch
				}
				if shift != 0 {
					translateItems(l.items[startLen+1:], 0, shift)
				}
				ch = need
			}
		}
		bottom = contentTop + ch + pb + bb
		out = margins{}.add(mb)
	}

	res := blockResult{top: top, bottom: bottom, pending: out, firstBaseline: cr.firstBaseline,
		lastBaseline: cr.lastBaseline, hasBaseline: cr.hasBaseline, breakAfter: isForcedBreak(s.BreakAfter)}
	res.pushedFirst = forcedPush || (collapseTop && cr.pushedFirst)

	if l.fragment && !forcedPush && !l.atPageStart(top) {
		restartAt := -1.0
		if cr.pushedFirst && !collapseTop && l.pageStart(cr.firstTop) > l.pageStart(top) {
			restartAt = l.pageStart(cr.firstTop)
		}
		if s.BreakInside == "avoid" || s.BreakInside == "avoid-page" {
			if l.pageStart(top) != l.pageStart(bottom-eps) && bottom-top <= l.pageH() {
				restartAt = l.nextPageStart(top)
			}
		}
		if restartAt > top {
			l.items = l.items[:startLen]
			l.marker = savedMarker
			r := l.layoutBlockW(b, x, restartAt, cbw, margins{}, bfc, forced)
			r.pushedFirst = true
			return r
		}
	}

	// paint background and borders under the content
	bx := x + e.ml
	bw := cw + e.pl + e.pr + e.bl + e.br
	if b != l.e.canvasBox && !s.BackgroundColor.IsTransparent() && s.Visibility == "visible" {
		bg.items = append(bg.items, &RectItem{X: bx, Y: top, W: bw, H: bottom - top, Color: s.BackgroundColor,
			Radius: s.BorderRadius[0].Resolve(bw, 0)})
	}
	if s.BackgroundImage != "" {
		l.e.warnOnce("layout: background images are not supported yet")
	}
	saved := l.items
	l.items = nil
	l.emitBorders(s, bx, top, bw, bottom-top, [4]bool{true, true, true, true})
	bg.items = append(bg.items, l.items...)
	l.items = saved
	l.emitIDs(b, top)
	return res
}

// emitIDs emits fragment anchors and outline entries for a box.
func (l *layouter) emitIDs(b *Box, top float64) {
	if b.ID != "" {
		l.emit(&AnchorItem{Name: b.ID, Y: top})
	}
	if b.Heading > 0 {
		if t := strings.Join(strings.Fields(textContent(b)), " "); t != "" {
			l.emit(&BookmarkItem{Title: t, Level: b.Heading, Y: top})
		}
	}
}

func textContent(b *Box) string {
	var sb strings.Builder
	var walk func(*Box)
	walk = func(x *Box) {
		if x.Kind == KindText {
			sb.WriteString(x.Text)
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(b)
	return sb.String()
}

// layoutContents lays out the children of a block container.
func (l *layouter) layoutContents(b *Box, x, y, w float64, pending margins) contentsResult {
	var res contentsResult
	if len(b.Children) == 0 {
		res.pending = pending
		return res
	}
	if !b.Children[0].isBlockLevel() {
		pend := pending
		start := y + pending.value()
		ir := l.layoutInline(b, x, y, w, &pend)
		if !ir.placed {
			res.pending = pending
			return res
		}
		res.placed = true
		res.firstTop = start
		if ir.pushedFirst {
			res.firstTop = l.pageStart(ir.firstBaseline)
		}
		res.end = ir.end
		res.firstBaseline, res.lastBaseline, res.hasBaseline = ir.firstBaseline, ir.lastBaseline, true
		res.pushedFirst = ir.pushedFirst
		return res
	}
	cur := y
	pend := pending
	for _, c := range b.Children {
		r := l.layoutBlock(c, x, cur, w, pend, false)
		if r.collapsed {
			pend = r.pending
			if r.breakAfter && l.fragment {
				cur = l.nextPageStart(cur)
				pend = margins{}
			}
			continue
		}
		if !res.placed {
			res.placed = true
			res.firstTop = r.top
			res.pushedFirst = r.pushedFirst
		}
		if r.hasBaseline {
			if !res.hasBaseline {
				res.firstBaseline = r.firstBaseline
			}
			res.lastBaseline = r.lastBaseline
			res.hasBaseline = true
		}
		cur = r.bottom
		pend = r.pending
		if r.breakAfter && l.fragment {
			cur = l.nextPageStart(cur + pend.value())
			pend = margins{}
		}
	}
	res.end = cur
	res.pending = pend
	return res
}

// dark returns Blink's darkened variant of a color (used for inset/outset/groove/ridge).
func dark(c css.Color) css.Color {
	v := math.Max(c.R, math.Max(c.G, c.B))
	if v == 0 {
		return css.Color{R: 0x54 / 255.0, G: 0x54 / 255.0, B: 0x54 / 255.0, A: c.A}
	}
	m := math.Max(0, (v-0.33)/v)
	return css.Color{R: c.R * m, G: c.G * m, B: c.B * m, A: c.A}
}

// emitBorders draws the borders of a box whose border box is (x, y, w, h).
func (l *layouter) emitBorders(s *css.Style, x, y, w, h float64, sides [4]bool) {
	if s.Visibility != "visible" {
		return
	}
	for side := range 4 {
		bw := s.BorderUsed(side)
		if bw <= 0 || !sides[side] {
			continue
		}
		col := s.BorderColor[side]
		if col.IsTransparent() {
			continue
		}
		style := s.BorderStyle[side]
		switch style {
		case "inset", "groove":
			if side == css.Top || side == css.Left {
				col = dark(col)
			}
		case "outset", "ridge":
			if side == css.Bottom || side == css.Right {
				col = dark(col)
			}
		}
		var rx, ry, rw, rh float64
		switch side {
		case css.Top:
			rx, ry, rw, rh = x, y, w, bw
		case css.Bottom:
			rx, ry, rw, rh = x, y+h-bw, w, bw
		case css.Left:
			rx, ry, rw, rh = x, y, bw, h
		case css.Right:
			rx, ry, rw, rh = x+w-bw, y, bw, h
		}
		switch style {
		case "dashed", "dotted":
			dash := []float64{3 * bw, 3 * bw}
			if style == "dotted" {
				dash = []float64{bw, bw}
			}
			if side == css.Top || side == css.Bottom {
				l.emit(&LineItem{X1: rx, Y1: ry + bw/2, X2: rx + rw, Y2: ry + bw/2, Width: bw, Color: col, Dash: dash})
			} else {
				l.emit(&LineItem{X1: rx + bw/2, Y1: ry, X2: rx + bw/2, Y2: ry + rh, Width: bw, Color: col, Dash: dash})
			}
		case "double":
			t := bw / 3
			if t < 0.75 {
				l.emit(&RectItem{X: rx, Y: ry, W: rw, H: rh, Color: col})
				continue
			}
			if side == css.Top || side == css.Bottom {
				l.emit(&RectItem{X: rx, Y: ry, W: rw, H: t, Color: col})
				l.emit(&RectItem{X: rx, Y: ry + rh - t, W: rw, H: t, Color: col})
			} else {
				l.emit(&RectItem{X: rx, Y: ry, W: t, H: rh, Color: col})
				l.emit(&RectItem{X: rx + rw - t, Y: ry, W: t, H: rh, Color: col})
			}
		default:
			l.emit(&RectItem{X: rx, Y: ry, W: rw, H: rh, Color: col})
		}
	}
}

func (e *Engine) warnOnce(msg string) {
	if e.warned == nil {
		e.warned = map[string]bool{}
	}
	if !e.warned[msg] {
		e.warned[msg] = true
		e.warn(msg)
	}
}
