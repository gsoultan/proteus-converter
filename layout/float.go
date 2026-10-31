package layout

import (
	"math"
	"sort"

	"github.com/gsoultan/proteus-converter/css"
)

// floatBox is a placed float's margin box.
type floatBox struct {
	left           bool
	x0, x1, y0, y1 float64
}

// floatCtx holds the floats of one block formatting context, in the
// coordinates of the layouter that owns it.
type floatCtx struct {
	floats  []floatBox
	lastTop float64 // a float may not be placed above an earlier float
}

func (fc *floatCtx) empty() bool { return fc == nil || len(fc.floats) == 0 }

// overlaps reports whether f intersects the vertical range [y, y+h).
func (f floatBox) overlaps(y, h float64) bool {
	if h < eps {
		h = eps
	}
	return f.y0 < y+h-eps && f.y1 > y+eps
}

// band returns the horizontal space [left, right) left free by floats over
// [y, y+h) inside the container edges [x0, x1].
func (fc *floatCtx) band(y, h, x0, x1 float64) (left, right float64) {
	left, right = x0, x1
	if fc == nil {
		return
	}
	for _, f := range fc.floats {
		if !f.overlaps(y, h) || f.x1-f.x0 <= 0 && f.y1-f.y0 <= 0 {
			continue
		}
		if f.left {
			left = math.Max(left, f.x1)
		} else {
			right = math.Min(right, f.x0)
		}
	}
	return
}

// nextBottom returns the lowest float bottom below y among floats overlapping
// [y, y+h), or -1 when none overlaps.
func (fc *floatCtx) nextBottom(y, h float64) float64 {
	best := -1.0
	if fc == nil {
		return best
	}
	for _, f := range fc.floats {
		if f.overlaps(y, h) && f.y1 > y+eps && (best < 0 || f.y1 < best) {
			best = f.y1
		}
	}
	return best
}

// place finds the position of a float with margin box w x h placed no higher
// than y inside the container [x0, x1], records it and returns its top-left.
func (fc *floatCtx) place(left bool, w, h, y, x0, x1 float64) (fx, fy float64) {
	y = math.Max(y, fc.lastTop)
	for range 1000 {
		l, r := fc.band(y, h, x0, x1)
		if r-l >= w-eps || (l <= x0+eps && r >= x1-eps) {
			if left {
				fx = l
			} else {
				fx = r - w
			}
			fc.add(left, fx, y, w, h)
			return fx, y
		}
		nb := fc.nextBottom(y, h)
		if nb < 0 {
			break
		}
		y = nb
	}
	if left {
		fx = x0
	} else {
		fx = x1 - w
	}
	fc.add(left, fx, y, w, h)
	return fx, y
}

func (fc *floatCtx) add(left bool, x, y, w, h float64) {
	fc.floats = append(fc.floats, floatBox{left: left, x0: x, x1: x + w, y0: y, y1: y + h})
	fc.lastTop = math.Max(fc.lastTop, y)
}

// clearY returns the position below the floats that clear (left, right, both) must avoid.
func (fc *floatCtx) clearY(clear string) float64 {
	y := math.Inf(-1)
	if fc == nil {
		return y
	}
	for _, f := range fc.floats {
		switch clear {
		case "left", "inline-start":
			if !f.left {
				continue
			}
		case "right", "inline-end":
			if f.left {
				continue
			}
		case "both":
		default:
			continue
		}
		y = math.Max(y, f.y1)
	}
	return y
}

// bottom returns the lowest float bottom, or -inf.
func (fc *floatCtx) bottom() float64 { return fc.clearY("both") }

// snapshot and restore support re-layout after a page push.
func (fc *floatCtx) snapshot() (int, float64) {
	if fc == nil {
		return 0, 0
	}
	return len(fc.floats), fc.lastTop
}

func (fc *floatCtx) restore(n int, lastTop float64) {
	if fc == nil {
		return
	}
	fc.floats = fc.floats[:n]
	fc.lastTop = lastTop
}

// --- floats and absolute boxes in block flow ------------------------------------

// placeFloat lays out float box b and places it at or below y within the
// container [x, x+cbw]. It returns the float's margin box bottom.
func (l *layouter) placeFloat(b *Box, a *atomicBox, x, y, cbw float64) float64 {
	if l.fc == nil {
		l.fc = &floatCtx{}
	}
	if a == nil {
		a = l.layoutAtomic(b, cbw)
	}
	left := b.Style.Float == "left" || b.Style.Float == "inline-start"
	n, lt := l.fc.snapshot()
	fx, fy := l.fc.place(left, a.w, a.h, y, x, x+cbw)
	if debugFloats {
		println("float", left, fx, fy, a.w, a.h, "container", x, cbw)
	}
	if l.fragment {
		if ny := l.place(fy, a.h); ny != fy {
			l.fc.restore(n, lt)
			fx, fy = l.fc.place(left, a.w, a.h, ny, x, x+cbw)
		}
	}
	for _, it := range a.items {
		it.translate(fx, fy)
		l.emit(it)
	}
	if b.Href != "" {
		l.emitLink(b.Href, fx, fy, a.w, a.h)
	}
	l.emitIDs(b, fy)
	return fy + a.h
}

// --- absolutely positioned boxes ----------------------------------------------

// absBox is an absolutely positioned box waiting for its containing block.
type absBox struct {
	box    *Box
	sx, sy float64 // static position (margin box top-left)
}

// posCtx collects the absolutely positioned descendants of a containing block.
type posCtx struct {
	boxes []absBox
}

func (l *layouter) addAbs(b *Box, sx, sy float64) {
	if b.Style.Position == "fixed" && l.e.fixedOK {
		l.e.fixedBoxes = append(l.e.fixedBoxes, absBox{box: b, sx: sx, sy: sy})
		return
	}
	if l.pos == nil {
		l.pos = &posCtx{}
	}
	l.pos.boxes = append(l.pos.boxes, absBox{box: b, sx: sx, sy: sy})
}

// layoutAbsolute lays out the boxes of pc against the containing block cb (a
// padding box) and emits their items.
func (l *layouter) layoutAbsolute(pc *posCtx, cb Clip) {
	if pc == nil {
		return
	}
	boxes := pc.boxes
	sort.SliceStable(boxes, func(i, j int) bool { return boxes[i].box.Style.ZIndex < boxes[j].box.Style.ZIndex })
	for _, ab := range boxes {
		items := l.absItems(ab, cb)
		if ab.box.Style.ZIndex < 0 {
			l.items = append(append([]Item{}, items...), l.items...)
			continue
		}
		l.items = append(l.items, items...)
	}
}

// absItems lays out one absolutely positioned box and returns its items.
func (l *layouter) absItems(ab absBox, cb Clip) []Item {
	b := ab.box
	s := b.Style
	cbw, cbh := cb.W, cb.H
	in := s.Inset
	e := l.hEdges(s, cbw)
	if e.mlAuto {
		e.ml = 0
	}
	if e.mrAuto {
		e.mr = 0
	}
	mt := s.Margin[css.Top].Resolve(cbw, 0)
	mb := s.Margin[css.Bottom].Resolve(cbw, 0)
	extra := e.bl + e.br + e.pl + e.pr
	forced := -1.0
	switch {
	case b.Kind == KindImage || b.Kind == KindCheckbox:
	case !s.Width.Auto:
	case !in[css.Left].Auto && !in[css.Right].Auto:
		forced = cbw - in[css.Left].Resolve(cbw, 0) - in[css.Right].Resolve(cbw, 0) - e.ml - e.mr
	default:
		avail := cbw - in[css.Left].Resolve(cbw, 0) - in[css.Right].Resolve(cbw, 0) - e.ml - e.mr
		mn, mx := l.intrinsicWidths(b)
		mn -= absPart(s.Margin[css.Left]) + absPart(s.Margin[css.Right])
		mx -= absPart(s.Margin[css.Left]) + absPart(s.Margin[css.Right])
		forced = math.Min(math.Max(mn, avail), mx)
	}
	if forced >= 0 {
		forced = math.Max(forced, extra)
	}
	// a definite height from top + bottom
	if s.Height.Auto && !in[css.Top].Auto && !in[css.Bottom].Auto && b.Kind != KindImage {
		bt, bb := s.BorderUsed(css.Top), s.BorderUsed(css.Bottom)
		h := cbh - in[css.Top].Resolve(cbh, 0) - in[css.Bottom].Resolve(cbh, 0) - mt - mb
		l.e.forcedH[b] = math.Max(h, bt+bb)
		defer delete(l.e.forcedH, b)
	}
	sub := l.sub()
	r := sub.layoutBlockW(b, 0, 0, cbw, margins{}, true, forced)
	var bw float64
	if r.bw > 0 {
		bw = r.bw
	} else {
		_, cw := sub.blockWidth(s, cbw, forced)
		bw = cw + extra
	}
	mw := e.ml + bw + e.mr
	mh := r.bottom + mb
	// the box was laid out with its margin box at the origin (left edge at
	// x = 0 + ml when margins are not auto)
	ox := r.bx - e.ml
	var x, y float64
	switch {
	case !in[css.Left].Auto:
		x = cb.X + in[css.Left].Resolve(cbw, 0)
	case !in[css.Right].Auto:
		x = cb.X + cbw - in[css.Right].Resolve(cbw, 0) - mw
	default:
		x = ab.sx
	}
	if !in[css.Left].Auto && !in[css.Right].Auto && !s.Width.Auto && s.Margin[css.Left].Auto && s.Margin[css.Right].Auto {
		free := cbw - in[css.Left].Resolve(cbw, 0) - in[css.Right].Resolve(cbw, 0) - bw
		x = cb.X + in[css.Left].Resolve(cbw, 0) + math.Max(free, 0)/2
	}
	switch {
	case !in[css.Top].Auto:
		y = cb.Y + in[css.Top].Resolve(cbh, 0)
	case !in[css.Bottom].Auto:
		y = cb.Y + cbh - in[css.Bottom].Resolve(cbh, 0) - mh
	default:
		y = ab.sy
	}
	if !in[css.Top].Auto && !in[css.Bottom].Auto && !s.Height.Auto && s.Margin[css.Top].Auto && s.Margin[css.Bottom].Auto {
		free := cbh - in[css.Top].Resolve(cbh, 0) - in[css.Bottom].Resolve(cbh, 0) - (mh - mt - mb)
		y = cb.Y + in[css.Top].Resolve(cbh, 0) + math.Max(free, 0)/2
	}
	translateItems(sub.items, x-ox, y)
	if b.Href != "" {
		sub.emitLink(b.Href, x+e.ml, y+mt, bw, mh-mt-mb)
	}
	return sub.items
}

var debugFloats = false
