package layout

import (
	"math"
	"strings"

	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/svg"
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
	bx, bw        float64 // border box left edge and width (bw 0 when unknown)
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
	oof           bool // floats or absolutely positioned boxes were found
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
		default:
			legacyAlign(s, &e, free)
		}
	}
	return e, cw
}

// legacyAlign shifts a block without auto margins inside a parent whose
// text-align is -webkit-center/-webkit-right (HTML align attributes).
func legacyAlign(s *css.Style, e *edges, free float64) {
	if free <= 0 {
		return
	}
	switch s.ParentAlignLegacy {
	case "center":
		e.ml += free / 2
		e.mr += free / 2
	case "right":
		e.ml += free
	}
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

// layoutBlockW lays out a block-level box. It sets up a new float context for
// block formatting context roots, lays out the absolutely positioned
// descendants of positioned boxes and applies relative offsets.
func (l *layouter) layoutBlockW(b *Box, x, y, cbw float64, pending margins, bfc bool, forced float64) blockResult {
	s := l.style(b)
	startLen := len(l.items)
	newBFC := bfc || isBFC(b)
	savedFC := l.fc
	if newBFC {
		l.fc = &floatCtx{}
	}
	positioned := s.Position != "static" && s.Position != ""
	savedPos := l.pos
	if positioned {
		l.pos = &posCtx{}
	}
	r := l.layoutBlockInner(b, x, y, cbw, pending, newBFC, forced)
	l.fc = savedFC
	if !positioned {
		return r
	}
	pc := l.pos
	l.pos = savedPos
	bx, bw := r.bx, r.bw
	if bw <= 0 {
		bx, bw = x, cbw
	}
	bl, br := s.BorderUsed(css.Left), s.BorderUsed(css.Right)
	bt, bb := s.BorderUsed(css.Top), s.BorderUsed(css.Bottom)
	l.layoutAbsolute(pc, Clip{X: bx + bl, Y: r.top + bt, W: bw - bl - br, H: math.Max(r.bottom-r.top-bt-bb, 0)})
	if s.Position == "relative" || s.Position == "sticky" {
		dx, dy := relativeOffset(s, cbw)
		if dx != 0 || dy != 0 {
			translateItems(l.items[startLen:], dx, dy)
		}
	}
	return r
}

// definiteHeight returns the definite content height of a box, or -1.
func (l *layouter) definiteHeight(b *Box, cbw float64) float64 {
	s := l.style(b)
	extra := s.BorderUsed(css.Top) + s.BorderUsed(css.Bottom) + s.Padding[css.Top].Resolve(cbw, 0) + s.Padding[css.Bottom].Resolve(cbw, 0)
	if t, ok := l.e.forcedH[b]; ok {
		return math.Max(t-extra, 0)
	}
	if s.Height.Auto || s.Height.HasPct() {
		return -1
	}
	h := s.Height.Resolve(0, 0)
	if s.BoxSizing == "border-box" {
		h -= extra
	}
	if !s.MaxHeight.Auto && !s.MaxHeight.HasPct() {
		mh := s.MaxHeight.Resolve(0, 0)
		if s.BoxSizing == "border-box" {
			mh -= extra
		}
		h = math.Min(h, mh)
	}
	if !s.MinHeight.HasPct() {
		mh := s.MinHeight.Resolve(0, 0)
		if s.BoxSizing == "border-box" {
			mh -= extra
		}
		h = math.Max(h, mh)
	}
	return math.Max(h, 0)
}

// relativeOffset returns the offset of a relatively positioned box.
func relativeOffset(s *css.Style, cbw float64) (dx, dy float64) {
	if s.Position != "relative" {
		return 0, 0
	}
	in := s.Inset
	switch {
	case !in[css.Left].Auto:
		dx = in[css.Left].Resolve(cbw, 0)
	case !in[css.Right].Auto:
		dx = -in[css.Right].Resolve(cbw, 0)
	}
	switch {
	case !in[css.Top].Auto:
		dy = in[css.Top].Resolve(0, 0)
	case !in[css.Bottom].Auto:
		dy = -in[css.Bottom].Resolve(0, 0)
	}
	return dx, dy
}

func (l *layouter) layoutBlockInner(b *Box, x, y, cbw float64, pending margins, bfc bool, forced float64) blockResult {
	s := l.style(b)
	startLen := len(l.items)
	fcN, fcTop := l.fc.snapshot()
	posN := 0
	if l.pos != nil {
		posN = len(l.pos.boxes)
	}
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
		top := l.afterMargin(y, pending.add(mt))
		ny := l.place(top, a.h-mt-mb)
		e := l.hEdges(s, cbw)
		left := x
		if e.mlAuto && e.mrAuto {
			left += (cbw - a.w) / 2
		} else if !e.mlAuto && !e.mrAuto {
			le := edges{}
			legacyAlign(s, &le, cbw-a.w)
			left += le.ml
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
			breakAfter: isForcedBreak(s.BreakAfter), bx: left + e.ml, bw: a.w - e.ml - e.mr}
	}

	e, cw := l.blockWidth(s, cbw, forced)
	// children resolve percentage heights against this box's definite height
	defH := l.definiteHeight(b, cbw)
	savedPctH, savedPctOK := l.pctH, l.pctOK
	l.pctH, l.pctOK = defH, defH >= 0
	defer func() { l.pctH, l.pctOK = savedPctH, savedPctOK }()
	bt, bb := s.BorderUsed(css.Top), s.BorderUsed(css.Bottom)
	pt, pb := s.Padding[css.Top].Resolve(cbw, 0), s.Padding[css.Bottom].Resolve(cbw, 0)
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
			top = l.afterMargin(y, pend)
		}
		contentTop = top
	} else {
		top = l.afterMargin(y, pend)
		contentTop = top + bt + pt
		switch {
		case s.IsFlex():
			cr = l.layoutFlex(b, cx, contentTop, cw, defH)
		case s.IsGrid():
			cr = l.layoutGrid(b, cx, contentTop, cw, defH)
		default:
			cr = l.layoutContents(b, cx, contentTop, cw, margins{})
		}
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
		if !cr.oof {
			l.items = l.items[:startLen]
		}
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
		if bfc && heightAuto {
			// a block formatting context root contains its floats
			contentEnd = math.Max(contentEnd, l.fc.bottom())
		}
		ch := math.Max(contentEnd-contentTop, 0)
		natural := ch // content height, before height/min-height apply
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
		if target, ok := l.e.forcedH[b]; ok {
			ch = math.Max(target-bt-pt-pb-bb, 0)
		}
		if target, ok := l.e.cellHeight[b]; ok {
			// a table cell: align the content inside the row height
			if need := math.Max(target-bt-pt-pb-bb, ch); need > natural {
				shift := 0.0
				switch s.VerticalAlign {
				case "middle":
					shift = (need - natural) / 2
				case "bottom":
					shift = need - natural
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
		lastBaseline: cr.lastBaseline, hasBaseline: cr.hasBaseline, breakAfter: isForcedBreak(s.BreakAfter),
		bx: x + e.ml, bw: cw + e.pl + e.pr + e.bl + e.br}
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
			l.fc.restore(fcN, fcTop)
			if l.pos != nil {
				l.pos.boxes = l.pos.boxes[:posN]
			}
			r := l.layoutBlockInner(b, x, restartAt, cbw, margins{}, bfc, forced)
			r.pushedFirst = true
			return r
		}
	}

	// paint background and borders under the content
	bx := x + e.ml
	bw := cw + e.pl + e.pr + e.bl + e.br
	saved := l.items
	l.items = nil
	l.emitBackgroundCB(b, bx, top, bw, bottom-top, cbw)
	l.emitBorders(s, bx, top, bw, bottom-top, [4]bool{true, true, true, true})
	bg.items = append(bg.items, l.items...)
	l.items = saved
	l.emitOutline(s, bx, top, bw, bottom-top)
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
	if !hasBlockChild(b) {
		pend := pending
		start := l.afterMargin(y, pending)
		ir := l.layoutInline(b, x, y, w, &pend)
		if !ir.placed {
			res.pending = pending
			res.oof = ir.oof
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
	// out-of-flow boxes wait until the margins they adjoin are resolved:
	// they sit at the top of the next in-flow box (or the end of the content)
	// once content is placed (or the container does not collapse its top
	// margin) the block offset is known and floats are placed where they occur
	deferOOF := pending != margins{}
	var waiting []*Box
	flushOOF := func(at float64) {
		for _, c := range waiting {
			if c.Style.IsAbsolute() {
				l.addAbs(c, x, at)
			} else {
				l.placeFloat(c, nil, x, at, w)
			}
		}
		waiting = waiting[:0]
	}
	for _, c := range b.Children {
		cs := c.Style
		if c.outOfFlow() {
			waiting = append(waiting, c)
			res.oof = true
			if !deferOOF || res.placed {
				flushOOF(cur + pend.value())
			}
			continue
		}
		mt := cs.Margin[css.Top].Resolve(w, 0)
		if len(waiting) > 0 {
			flushOOF(cur + pend.add(mt).value())
		}
		if cs.Clear != "none" && !l.fc.empty() {
			if cy := l.fc.clearY(cs.Clear); !math.IsInf(cy, -1) && cur+pend.add(mt).value() < cy-eps {
				cur = cy - mt
				pend = margins{}
			}
		}
		cx, cw := x, w
		if !l.fc.empty() && isBFC(c) {
			// a block formatting context root does not overlap floats
			top := cur + pend.add(mt).value()
			if bl, br := l.fc.band(top, 1, x, x+w); bl > x+eps || br < x+w-eps {
				cx, cw = bl, math.Max(br-bl, 0)
			}
		}
		r := l.layoutBlock(c, cx, cur, cw, pend, false)
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
	if len(waiting) > 0 {
		flushOOF(cur + pend.value())
	}
	res.end = cur
	res.pending = pend
	return res
}

// roundedBorder strokes a uniform border with rounded corners as one path.
// It reports false when the border is not uniform or has no radius.
func (l *layouter) roundedBorder(s *css.Style, x, y, w, h float64, sides [4]bool) bool {
	radii, mixed := cornerRadii(s, w, h)
	r := radii[0]
	if (r <= 0 && !mixed) || sides != [4]bool{true, true, true, true} {
		return false
	}
	bw := s.BorderUsed(0)
	if bw <= 0 {
		return false
	}
	col := s.BorderColor[0]
	style := s.BorderStyle[0]
	for side := 1; side < 4; side++ {
		if s.BorderUsed(side) != bw || s.BorderColor[side] != col || s.BorderStyle[side] != style {
			return false
		}
	}
	switch style {
	case "solid", "dashed", "dotted":
	default:
		return false
	}
	if col.IsTransparent() {
		return true
	}
	r = math.Min(r, math.Min(w, h)/2)
	ir := math.Max(r-bw/2, 0)
	c := col
	st := PathStyle{Stroke: &c, StrokeWidth: bw}
	switch style {
	case "dashed":
		st.Dash = []float64{3 * bw, 3 * bw}
	case "dotted":
		st.Dash = []float64{bw, bw}
		st.LineCap = 1
		st.Dash = []float64{0.001, 2 * bw}
	}
	segs := svg.RoundRect(x+bw/2, y+bw/2, w-bw, h-bw, ir, ir)
	if mixed {
		for i := range radii {
			radii[i] = math.Max(radii[i]-bw/2, 0)
		}
		segs = svg.RoundRect4(x+bw/2, y+bw/2, w-bw, h-bw, radii)
	}
	l.emit(&PathItem{M: svg.Identity, Top: y, Style: st, Segs: segs})
	return true
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
	if l.roundedBorder(s, x, y, w, h, sides) {
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
