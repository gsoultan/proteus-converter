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
		lines = append(lines, &lineBox{frags: frags[i:j], forced: forced, avail: lineAvail, indent: ind})
		i = j
		first = false
	}
	if len(lines) > 0 {
		lines[len(lines)-1].last = true
	}
	return lines
}

// onlyEdges reports whether frags hold nothing but zero-width box edges and collapsible spaces.
func onlyEdges(frags []*frag) bool {
	for _, f := range frags {
		switch {
		case (f.kind == fOpen || f.kind == fClose) && f.width == 0:
		case f.kind == fSpace && f.collapse:
		case f.kind == fWbr:
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
		if f.kind == fClose || f.kind == fOpen {
			// keep looking through zero-width box edges
			k := end - 1
			for k > 0 && (fr[k].kind == fClose || fr[k].kind == fOpen) {
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
		if mk.inside {
			frags = append(l.buildFrags(mk.box.Children, mk.box.Style, width, nil), frags...)
		} else {
			markerFrags = l.buildFrags(mk.box.Children, mk.box.Style, width, nil)
		}
	}
	if !hasContent(frags) {
		if mk != nil {
			l.marker = mk // give it to the next formatting context
		}
		return inlineResult{}
	}
	y += pending.value()
	*pending = margins{}
	indent := s.TextIndent.Resolve(width, 0)
	lines := l.breakLines(frags, width, indent)
	for _, lb := range lines {
		l.measureLine(lb, s)
	}
	res := inlineResult{placed: true}
	res.pushedFirst = l.placeLines(lines, y, s)
	for i, lb := range lines {
		l.emitLine(lb, x, width, s)
		if i == 0 {
			res.firstBaseline = lb.top + lb.baseline
			if markerFrags != nil {
				l.emitMarker(mk.box, markerFrags, x, lb.top+lb.baseline)
			}
		}
		res.lastBaseline = lb.top + lb.baseline
		res.end = lb.top + lb.height
	}
	return res
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
func (l *layouter) emitMarker(mb *Box, frags []*frag, x, baseline float64) {
	if mb.Symbol != "" {
		s := mb.Style
		if s.Visibility != "visible" {
			return
		}
		m := l.e.styleMetrics(s)
		asc := int(math.Round(m.ascent / 0.75))
		offset := asc * 2 / 3
		bw := float64((offset+1)/2) * 0.75
		bx := x - float64(offset+kMarkerPadding+1)*0.75 + 0.75
		by := baseline - float64(asc)*0.75 + float64(3*(asc-offset)/2)*0.75
		switch mb.Symbol {
		case "disc":
			l.emit(&EllipseItem{X: bx, Y: by, W: bw, H: bw, Color: s.Color})
		case "circle":
			l.emit(&EllipseItem{X: bx + 0.375, Y: by + 0.375, W: bw - 0.75, H: bw - 0.75, Color: s.Color, Stroke: 0.75})
		case "square":
			l.emit(&RectItem{X: bx, Y: by, W: bw, H: bw, Color: s.Color})
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
			l.emit(&TextItem{X: x, Baseline: by, Text: r.Text, Face: r.Face, Size: f.style.FontSize,
				Color: f.style.Color, LetterSpacing: f.style.LetterSpacing})
		}
		x += r.Face.Width(r.Text, f.style.FontSize) + f.style.LetterSpacing*float64(len([]rune(r.Text)))
	}
}

// mergeText appends text to the previous text item when it continues it exactly.
func (l *layouter) mergeText(text string, face *fonts.Face, f *frag, x, baseline float64) bool {
	if len(l.items) == 0 {
		return false
	}
	prev, ok := l.items[len(l.items)-1].(*TextItem)
	if !ok || prev.Face != face || prev.Size != f.style.FontSize || prev.Color != f.style.Color ||
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
