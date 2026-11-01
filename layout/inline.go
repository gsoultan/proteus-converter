package layout

import (
	"math"

	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
)

type lineBox struct {
	frags    []*frag
	x        []float64 // frag x offsets (from content left)
	width    float64   // used content width
	avail    float64
	indent   float64
	forced   bool // ended by a forced break
	last     bool // last line of the paragraph
	top      float64
	height   float64
	baseline float64 // from line top
	x0       float64 // left edge of the line box (moves right of left floats)
}

// hasContent reports whether frags produce at least one line box.
func hasContent(frags []*frag) bool {
	for _, f := range frags {
		switch f.kind {
		case fText, fAtomic, fBreak:
			return true
		case fSpace:
			if !f.collapse {
				return true
			}
		case fOpen, fClose:
			if f.width > 0 {
				return true
			}
		}
	}
	return false
}

// breakLines distributes frags over lines no wider than avail.
func (l *layouter) breakLines(frags []*frag, avail, indent float64) []*lineBox {
	var lines []*lineBox
	i := 0
	n := len(frags)
	first := true
	for i < n {
		for i < n && frags[i].kind == fSpace && frags[i].collapse {
			i++
		}
		if i >= n {
			break
		}
		if len(lines) > 0 && onlyEdges(frags[i:]) {
			// trailing box edges after a final forced break do not start a new line
			lb := lines[len(lines)-1]
			lb.frags = append(lb.frags[:len(lb.frags):len(lb.frags)], frags[i:]...)
			break
		}
		lineAvail := avail
		ind := 0.0
		if first {
			ind = indent
			lineAvail -= indent
		}
		var j int
		var forced bool
		frags, j, forced = l.breakOne(frags, i, lineAvail)
		n = len(frags)
		lines = append(lines, &lineBox{frags: frags[i:j], forced: forced, avail: lineAvail, indent: ind})
		i = j
		first = false
	}
	if len(lines) > 0 {
		lines[len(lines)-1].last = true
	}
	return lines
}

// breakOne finds the end of the line starting at frags[i] for a line of
// width lineAvail. Unbreakable text may be split into characters, so the
// (possibly updated) frag slice is returned with the end index.
func (l *layouter) breakOne(frags []*frag, i int, lineAvail float64) ([]*frag, int, bool) {
	n := len(frags)
	width := 0.0
	lastBreak := -1
	j := i
	forced := false
	for j < n {
		f := frags[j]
		if f.kind == fBreak {
			j++
			forced = true
			break
		}
		if f.kind != fSpace && f.kind != fClose && j > i && width+f.width > lineAvail+eps {
			if lastBreak >= i {
				j = lastBreak + 1
				break
			}
			if f.kind == fText && canBreakAnywhere(f.style) && len([]rune(f.text)) > 1 {
				chars := l.splitFragChars(f)
				frags = append(frags[:j], append(chars, frags[j+1:]...)...)
				n = len(frags)
				continue
			}
		} else if f.kind == fText && j == i && f.width > lineAvail+eps && canBreakAnywhere(f.style) && len([]rune(f.text)) > 1 {
			chars := l.splitFragChars(f)
			frags = append(frags[:j], append(chars, frags[j+1:]...)...)
			n = len(frags)
			continue
		}
		width += f.width
		if f.breakOK {
			lastBreak = j
		}
		j++
	}
	return frags, j, forced
}

// onlyEdges reports whether frags hold nothing but zero-width box edges and collapsible spaces.
func onlyEdges(frags []*frag) bool {
	for _, f := range frags {
		switch {
		case (f.kind == fOpen || f.kind == fClose) && f.width == 0:
		case f.kind == fSpace && f.collapse:
		case f.kind == fWbr, f.kind == fFloat, f.kind == fAbs:
		default:
			return false
		}
	}
	return true
}

// trimmed returns the frags of a line without collapsible spaces at its end.
func (lb *lineBox) trimmed() []*frag {
	fr := lb.frags
	end := len(fr)
	for end > 0 {
		f := fr[end-1]
		if f.kind == fBreak || (f.kind == fSpace && (f.collapse || f.hang)) {
			end--
			continue
		}
		if f.kind == fClose || f.kind == fOpen || f.kind == fFloat || f.kind == fAbs {
			// keep looking through zero-width box edges
			k := end - 1
			for k > 0 && (fr[k].kind == fClose || fr[k].kind == fOpen || fr[k].kind == fFloat || fr[k].kind == fAbs) {
				k--
			}
			if k >= 0 && fr[k].kind == fSpace && fr[k].collapse {
				// move the space out: drop it but keep the box edges
				out := make([]*frag, 0, end)
				out = append(out, fr[:k]...)
				out = append(out, fr[k+1:end]...)
				return out
			}
		}
		break
	}
	return fr[:end]
}

// measureLine computes the vertical metrics of a line box.
func (l *layouter) measureLine(lb *lineBox, block *css.Style) {
	sm := l.e.styleMetrics(block)
	half := (sm.lh - (sm.ascent + sm.descent)) / 2
	minTop := -sm.ascent - half
	maxBot := sm.descent + half
	var deferred []*atomicBox
	for _, f := range lb.frags {
		shift := 0.0
		if f.ctx != nil {
			shift = f.ctx.shift
		}
		switch f.kind {
		case fText, fSpace, fOpen, fClose:
			s := f.style
			if (f.kind == fOpen || f.kind == fClose) && f.ctx != nil {
				s = f.ctx.style
			}
			m := l.e.styleMetrics(s)
			h := (m.lh - (m.ascent + m.descent)) / 2
			minTop = math.Min(minTop, shift-m.ascent-h)
			maxBot = math.Max(maxBot, shift+m.descent+h)
		case fAtomic:
			a := f.atomic
			switch a.valign {
			case "top", "bottom":
				deferred = append(deferred, a)
				continue
			case "middle":
				ps := block
				if f.ctx != nil {
					ps = f.ctx.style
				}
				xh := l.e.styleMetrics(ps).xHeight
				top := shift - xh/2 - a.h/2
				minTop = math.Min(minTop, top)
				maxBot = math.Max(maxBot, top+a.h)
				continue
			case "text-top", "text-bottom":
				ps := block
				if f.ctx != nil {
					ps = f.ctx.style
				}
				top := textEdgeTop(l.e.styleMetrics(ps), a, shift)
				minTop = math.Min(minTop, top)
				maxBot = math.Max(maxBot, top+a.h)
				continue
			case "length", "sub", "super":
				shift += a.valignLen
			}
			minTop = math.Min(minTop, shift-a.baseline)
			maxBot = math.Max(maxBot, shift+a.h-a.baseline)
		}
	}
	for _, a := range deferred {
		if a.h > maxBot-minTop {
			if a.valign == "top" {
				maxBot = minTop + a.h
			} else {
				minTop = maxBot - a.h
			}
		}
	}
	lb.baseline = -minTop
	lb.height = maxBot - minTop
}

// textEdgeTop returns the top of an atomic box aligned with vertical-align
// text-top or text-bottom, relative to the baseline, in a parent with metrics pm.
func textEdgeTop(pm styleMetrics, a *atomicBox, shift float64) float64 {
	if a.valign == "text-top" {
		return shift - pm.ascent
	}
	return shift + pm.descent - a.h
}

// placeLines assigns a top position to every line, honouring page breaks,
// orphans and widows. pushedFirst reports that the first line moved to a later page.
func (l *layouter) placeLines(lines []*lineBox, y0 float64, s *css.Style) (pushedFirst bool) {
	forceBefore := map[int]bool{}
	for attempt := 0; attempt < 4; attempt++ {
		y := y0
		pageFirst := 0
		pushedFirst = false
		retry := false
		for k, lb := range lines {
			var ny float64
			if forceBefore[k] {
				ny = l.nextPageStart(y)
			} else {
				ny = l.place(y, lb.height)
			}
			if ny != y && !forceBefore[k] && k > 0 && l.fragment {
				before := k - pageFirst
				after := len(lines) - k
				if before < s.Orphans && pageFirst == 0 && !l.atPageStart(y0) {
					forceBefore[0] = true
					retry = true
					break
				}
				if after < s.Widows {
					nb := k - (s.Widows - after)
					if nb > pageFirst && nb-pageFirst >= s.Orphans {
						forceBefore[nb] = true
						retry = true
						break
					}
				}
			}
			if ny != y {
				pageFirst = k
				if k == 0 {
					pushedFirst = true
				}
			}
			lb.top = ny
			y = ny + lb.height
		}
		if !retry {
			break
		}
	}
	return pushedFirst
}

type inlineResult struct {
	placed        bool
	oof           bool
	end           float64
	firstBaseline float64
	lastBaseline  float64
	pushedFirst   bool
}

// layoutInline lays out the inline content of block container b.
func (l *layouter) layoutInline(b *Box, x, y, width float64, pending *margins) inlineResult {
	s := b.Style
	frags := l.buildFrags(b.Children, s, width, nil)
	mk := l.takeMarker()
	var markerFrags []*frag
	if mk != nil {
		if mk.inside && mk.box.Symbol != "" {
			frags = append([]*frag{l.insideSymbolFrag(mk.box)}, frags...)
		} else if mk.inside {
			frags = append(l.buildFrags(mk.box.Children, mk.box.Style, width, nil), frags...)
		} else {
			markerFrags = l.buildFrags(mk.box.Children, mk.box.Style, width, nil)
		}
	}
	if !hasContent(frags) {
		if mk != nil {
			l.marker = mk // give it to the next formatting context
		}
		return inlineResult{oof: l.placeOutOfFlow(frags, x, l.afterMargin(y, *pending), width)}
	}
	y = l.afterMargin(y, *pending)
	*pending = margins{}
	indent := s.TextIndent.Resolve(width, 0)
	var lines []*lineBox
	res := inlineResult{placed: true}
	if hasFloatFrags(frags) || l.floatsBelow(y, x, x+width) {
		lines = l.linesAroundFloats(frags, x, y, width, indent, s)
		res.oof = true
	} else {
		lines = l.breakLines(frags, width, indent)
		for _, lb := range lines {
			l.measureLine(lb, s)
			lb.x0 = x
		}
		res.pushedFirst = l.placeLines(lines, y, s)
	}
	if len(lines) > 0 && l.fragment && lines[0].top > y+eps && !l.atPageStart(y) && l.atPageStart(lines[0].top) {
		res.pushedFirst = true
	}
	for i, lb := range lines {
		l.emitLine(lb, lb.x0, width, s)
		if i == 0 {
			res.firstBaseline = lb.top + lb.baseline
			if markerFrags != nil {
				l.emitMarker(mk.box, markerFrags, x, lb.top+lb.baseline)
			}
		}
		res.lastBaseline = lb.top + lb.baseline
		res.end = lb.top + lb.height
	}
	if len(lines) == 0 {
		res.placed = false
	}
	return res
}

func hasFloatFrags(frags []*frag) bool {
	for _, f := range frags {
		if f.kind == fFloat {
			return true
		}
	}
	return false
}

// floatsBelow reports whether a float of the current context reaches below y
// and overlaps the horizontal range [x0, x1].
func (l *layouter) floatsBelow(y, x0, x1 float64) bool {
	if l.fc.empty() {
		return false
	}
	for _, f := range l.fc.floats {
		if f.y1 > y+eps && f.x1 > x0+eps && f.x0 < x1-eps {
			return true
		}
	}
	return false
}

// placeOutOfFlow positions the floats and records the absolute boxes of
// inline content that produced no line box.
func (l *layouter) placeOutOfFlow(frags []*frag, x, y, width float64) bool {
	found := false
	for _, f := range frags {
		switch f.kind {
		case fFloat:
			if !f.placed {
				l.placeFloat(f.box, f.atomic, x, y, width)
				f.placed = true
			}
			found = true
		case fAbs:
			l.addAbs(f.box, x, y)
			found = true
		}
	}
	return found
}

// linesAroundFloats breaks and places lines one at a time, shortening each
// line by the floats beside it and positioning floats met in the content.
func (l *layouter) linesAroundFloats(frags []*frag, x, y, width, indent float64, s *css.Style) []*lineBox {
	var lines []*lineBox
	i, n := 0, len(frags)
	first := true
	yCur := y
	strut := l.e.styleMetrics(s).lh
	for i < n {
		for i < n && ((frags[i].kind == fSpace && frags[i].collapse) || frags[i].kind == fFloat) {
			if f := frags[i]; f.kind == fFloat && !f.placed {
				l.placeFloat(f.box, f.atomic, x, yCur, width)
				f.placed = true
			}
			i++
		}
		if i >= n {
			break
		}
		if len(lines) > 0 && onlyEdges(frags[i:]) {
			lb := lines[len(lines)-1]
			lb.frags = append(lb.frags[:len(lb.frags):len(lb.frags)], frags[i:]...)
			l.placeOutOfFlow(frags[i:], x, yCur, width)
			break
		}
		var lb *lineBox
		var j int
		for attempt := 0; ; attempt++ {
			ind := 0.0
			if first {
				ind = indent
			}
			bl, br := l.fc.band(yCur, strut, x, x+width)
			lineAvail := br - bl - ind
			var forced bool
			frags, j, forced = l.breakOne(frags, i, lineAvail)
			n = len(frags)
			if attempt < 100 {
				// a float met inside the line goes on this line when it fits
				if k := firstUnplacedFloat(frags, i, j); k >= 0 {
					used := 0.0
					for _, f := range frags[i:k] {
						used += f.width
					}
					if used+frags[k].atomic.w <= lineAvail+eps {
						l.placeFloat(frags[k].box, frags[k].atomic, x, yCur, width)
						frags[k].placed = true
						continue
					}
				}
			}
			lb = &lineBox{frags: frags[i:j], forced: forced, avail: lineAvail, indent: ind, x0: bl}
			l.measureLine(lb, s)
			if attempt < 100 {
				narrowed := bl > x+eps || br < x+width-eps
				if narrowed && lineWidth(lb) > lineAvail+eps {
					if nb := l.fc.nextBottom(yCur, lb.height); nb > yCur {
						yCur = nb
						continue
					}
				}
				if ny := l.place(yCur, lb.height); ny != yCur {
					yCur = ny
					continue
				}
			}
			break
		}
		lb.top = yCur
		lines = append(lines, lb)
		yCur += lb.height
		// floats that did not fit go below the line
		for k := i; k < j; k++ {
			if f := frags[k]; f.kind == fFloat && !f.placed {
				l.placeFloat(f.box, f.atomic, x, yCur, width)
				f.placed = true
			}
		}
		i = j
		first = false
	}
	if len(lines) > 0 {
		lines[len(lines)-1].last = true
	}
	return lines
}

func firstUnplacedFloat(frags []*frag, i, j int) int {
	for k := i; k < j; k++ {
		if frags[k].kind == fFloat && !frags[k].placed {
			return k
		}
	}
	return -1
}

// lineWidth returns the width of a line's content without trailing spaces.
func lineWidth(lb *lineBox) float64 {
	w := 0.0
	for _, f := range lb.trimmed() {
		w += f.width
	}
	return w
}

// kMarkerPadding is Blink's gap (px) between a symbol marker and the content.
const kMarkerPadding = 7

// pendingMarker is an outside list marker waiting for its first line.
type pendingMarker struct {
	box    *Box
	inside bool
}

func (l *layouter) takeMarker() *pendingMarker {
	m := l.marker
	l.marker = nil
	return m
}

// emitMarker draws an outside marker ending at the content edge x.
// Symbol markers follow Blink's geometry (ListMarker::RelativeSymbolMarkerRect).
// symbolMarker returns the items of a disc, circle or square list marker
// drawn like Blink: a bullet (offset+1)/2 px wide, offset = 2/3 of the ascent,
// relative to the bullet's left edge and the line's ascent top. It also
// returns the ascent in pt and the bullet width.
func (l *layouter) symbolMarker(mb *Box) (items []Item, asc, bw float64) {
	s := mb.Style
	m := l.e.styleMetrics(s)
	ascPx := int(math.Round(m.ascent / 0.75))
	offset := ascPx * 2 / 3
	bw = float64((offset+1)/2) * 0.75
	by := float64(3*(ascPx-offset)/2) * 0.75
	switch mb.Symbol {
	case "disc":
		items = append(items, &EllipseItem{X: 0, Y: by, W: bw, H: bw, Color: s.Color})
	case "circle":
		items = append(items, &EllipseItem{X: 0.375, Y: by + 0.375, W: bw - 0.75, H: bw - 0.75, Color: s.Color, Stroke: 0.75})
	case "square":
		items = append(items, &RectItem{X: 0, Y: by, W: bw, H: bw, Color: s.Color})
	}
	return items, float64(ascPx) * 0.75, bw
}

// insideSymbolFrag returns an inside disc/circle/square marker as an atomic
// inline. Measured against Chromium: the bullet starts 1px in and the text
// follows after 1px + bullet + one font size.
func (l *layouter) insideSymbolFrag(mb *Box) *frag {
	items, asc, bw := l.symbolMarker(mb)
	if mb.Style.Visibility != "visible" {
		items = nil
	}
	for _, it := range items {
		it.translate(0.75, 0)
	}
	w := 0.75 + bw + mb.Style.FontSize
	a := &atomicBox{box: mb, items: items, w: w, h: asc, baseline: asc, valign: "baseline"}
	return &frag{kind: fAtomic, style: mb.Style, atomic: a, width: w}
}

func (l *layouter) emitMarker(mb *Box, frags []*frag, x, baseline float64) {
	if mb.Symbol != "" {
		if mb.Style.Visibility != "visible" {
			return
		}
		items, asc, _ := l.symbolMarker(mb)
		m := l.e.styleMetrics(mb.Style)
		offset := int(math.Round(m.ascent/0.75)) * 2 / 3
		bx := x - float64(offset+kMarkerPadding+1)*0.75 + 0.75
		for _, it := range items {
			it.translate(bx, baseline-asc)
			l.emit(it)
		}
		return
	}
	w := 0.0
	for _, f := range frags {
		w += f.width
	}
	cx := x - w
	for _, f := range frags {
		if f.kind == fText || f.kind == fSpace {
			l.emitTextFrag(f, cx, baseline)
		}
		cx += f.width
	}
}

func (l *layouter) emitTextFrag(f *frag, x, baseline float64) {
	if f.style.Visibility != "visible" {
		return
	}
	shift := 0.0
	if f.ctx != nil {
		shift = f.ctx.shift
	}
	by := baseline + shift
	for _, r := range f.runs {
		merged := l.mergeText(r.Text, r.Face, f, x, by)
		if !merged && f.kind != fSpace {
			sb, si := synthesis(f.style, r.Face)
			l.emit(&TextItem{X: x, Baseline: by, Text: r.Text, Face: r.Face, Size: f.style.FontSize,
				Color: f.style.Color, LetterSpacing: f.style.LetterSpacing, SynthBold: sb, SynthItalic: si})
		}
		x += r.Face.Width(r.Text, f.style.FontSize) + f.style.LetterSpacing*float64(len([]rune(r.Text)))
	}
}

// synthesis reports whether bold and italic must be synthesised because the
// face chosen for s lacks them (Blink: weight >= 600 on a face below 600,
// italic/oblique on an upright face). Standard-14 faces are never synthesised.
func synthesis(s *css.Style, face *fonts.Face) (bold, italic bool) {
	if face == nil || face.IsStandard() {
		return false, false
	}
	bold = s.FontWeight >= 600 && face.Weight < 600
	italic = (s.FontStyle == "italic" || s.FontStyle == "oblique") && !face.Italic
	return bold, italic
}

// mergeText appends text to the previous text item when it continues it exactly.
func (l *layouter) mergeText(text string, face *fonts.Face, f *frag, x, baseline float64) bool {
	if len(l.items) == 0 {
		return false
	}
	prev, ok := l.items[len(l.items)-1].(*TextItem)
	sb, si := synthesis(f.style, face)
	if !ok || prev.Face != face || prev.Size != f.style.FontSize || prev.Color != f.style.Color ||
		prev.SynthBold != sb || prev.SynthItalic != si ||
		prev.LetterSpacing != f.style.LetterSpacing || math.Abs(prev.Baseline-baseline) > eps {
		return false
	}
	end := prev.X + prev.Face.Width(prev.Text, prev.Size) + prev.LetterSpacing*float64(len([]rune(prev.Text)))
	if math.Abs(end-x) > 0.05 {
		return false
	}
	prev.Text += text
	return true
}
