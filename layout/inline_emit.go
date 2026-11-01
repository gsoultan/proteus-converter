package layout

import (
	"math"

	"github.com/gsoultan/proteus-converter/css"
)

type ctxExtent struct {
	ctx           *inlineCtx
	x0, x1        float64
	opens, closes bool
	started       bool
}

// emitLine positions the frags of a placed line and emits their items.
func (l *layouter) emitLine(lb *lineBox, x0, width float64, block *css.Style) {
	frags := lb.trimmed()
	used := 0.0
	spaces := 0
	for i, f := range frags {
		used += f.width
		if f.kind == fSpace && i > 0 && i < len(frags)-1 {
			spaces++
		}
	}
	free := lb.avail - used
	align := block.TextAlign
	if lb.last || lb.forced {
		switch block.TextAlignLast {
		case "auto":
			if align == "justify" {
				align = "start"
			}
		default:
			align = block.TextAlignLast
		}
	}
	if block.Direction == "rtl" {
		switch align {
		case "start":
			align = "right"
		case "end":
			align = "left"
		}
	}
	offset := lb.indent
	extra := 0.0
	switch align {
	case "right", "end":
		offset += math.Max(free, 0)
	case "center":
		offset += math.Max(free, 0) / 2
	case "justify":
		if spaces > 0 && free > 0 {
			extra = free / float64(spaces)
		}
	}
	baseline := lb.top + lb.baseline
	xs := make([]float64, len(frags))
	cx := x0 + offset
	for i, f := range frags {
		xs[i] = cx
		cx += f.width
		if extra > 0 && f.kind == fSpace && i > 0 && i < len(frags)-1 {
			cx += extra
		}
	}

	// inline box extents on this line
	var exts []*ctxExtent
	byCtx := map[*inlineCtx]*ctxExtent{}
	extend := func(c *inlineCtx, a, b float64) {
		for ; c != nil; c = c.parent {
			e := byCtx[c]
			if e == nil {
				e = &ctxExtent{ctx: c, x0: a, x1: b}
				byCtx[c] = e
				exts = append(exts, e)
			}
			e.x0 = math.Min(e.x0, a)
			e.x1 = math.Max(e.x1, b)
		}
	}
	for i, f := range frags {
		w := f.width
		if extra > 0 && f.kind == fSpace && i > 0 && i < len(frags)-1 {
			w += extra
		}
		switch f.kind {
		case fOpen:
			start := xs[i] + f.ctx.ml
			extend(f.ctx, start, start)
			byCtx[f.ctx].opens = true
			byCtx[f.ctx].x0 = start
		case fClose:
			end := xs[i] + f.width - f.ctx.mr
			extend(f.ctx, end, end)
			byCtx[f.ctx].closes = true
			byCtx[f.ctx].x1 = end
		default:
			if f.ctx != nil {
				extend(f.ctx, xs[i], xs[i]+w)
			}
		}
	}
	// backgrounds and borders of inline boxes (outermost first)
	for _, e := range exts {
		s := e.ctx.style
		m := l.e.styleMetrics(s)
		top := baseline + e.ctx.shift - m.ascent - s.Padding[css.Top].Resolve(width, 0) - s.BorderUsed(css.Top)
		bot := baseline + e.ctx.shift + m.descent + s.Padding[css.Bottom].Resolve(width, 0) + s.BorderUsed(css.Bottom)
		x0b, x1b := e.x0+e.ctx.relX, e.x1+e.ctx.relX
		top += e.ctx.relY
		bot += e.ctx.relY
		sides := [4]bool{true, e.closes, true, e.opens}
		if !s.BackgroundColor.IsTransparent() && s.Visibility == "visible" {
			l.emit(&RectItem{X: x0b, Y: top, W: x1b - x0b, H: bot - top, Color: s.BackgroundColor, Radius: s.BorderRadius[0].Resolve(x1b-x0b, 0)})
		}
		l.emitBorders(s, x0b, top, x1b-x0b, bot-top, sides)
	}
	// text, atomics
	for i, f := range frags {
		switch f.kind {
		case fText, fSpace:
			rx, ry := f.rel()
			l.emitTextFrag(f, xs[i]+rx, baseline+ry)
		case fAtomic:
			a := f.atomic
			rx, ry := f.rel()
			shift := 0.0
			if f.ctx != nil {
				shift = f.ctx.shift
			}
			var top float64
			switch a.valign {
			case "top":
				top = lb.top
			case "bottom":
				top = lb.top + lb.height - a.h
			case "middle":
				ps := block
				if f.ctx != nil {
					ps = f.ctx.style
				}
				top = baseline + shift - l.e.styleMetrics(ps).xHeight/2 - a.h/2
			case "text-top", "text-bottom":
				ps := block
				if f.ctx != nil {
					ps = f.ctx.style
				}
				top = baseline + textEdgeTop(l.e.styleMetrics(ps), a, shift)
			default:
				top = baseline + shift + a.valignLen - a.baseline
			}
			top += ry
			for _, it := range a.items {
				it.translate(xs[i]+rx, top)
				l.emit(it)
			}
			if a.box.Href != "" {
				l.emitLink(a.box.Href, xs[i]+rx, top, a.w, a.h)
			}
			if a.box.ID != "" {
				l.emit(&AnchorItem{Name: a.box.ID, Y: top})
			}
		case fAbs:
			l.addAbs(f.box, xs[i], lb.top)
		}
	}
	// decorations
	l.emitDecorations(frags, xs, extra, baseline)
	// links and anchors of inline boxes
	for _, e := range exts {
		if e.ctx.box.Href != "" {
			l.emitLink(e.ctx.box.Href, e.x0, lb.top, e.x1-e.x0, lb.height)
		}
		if e.ctx.box.ID != "" && e.opens {
			l.emit(&AnchorItem{Name: e.ctx.box.ID, Y: lb.top})
		}
	}
}

// rel returns the relative-position offset of the frag's inline box.
func (f *frag) rel() (float64, float64) {
	if f.ctx == nil {
		return 0, 0
	}
	return f.ctx.relX, f.ctx.relY
}

func (l *layouter) emitLink(href string, x, y, w, h float64) {
	if w <= 0 || h <= 0 {
		return
	}
	if len(href) > 1 && href[0] == '#' {
		l.emit(&LinkItem{X: x, Y: y, W: w, H: h, Dest: href[1:]})
		return
	}
	l.emit(&LinkItem{X: x, Y: y, W: w, H: h, URL: href})
}

// emitDecorations draws underline, overline and line-through for text frags.
func (l *layouter) emitDecorations(frags []*frag, xs []float64, extra, baseline float64) {
	type seg struct {
		d         Decoration
		x0, x1, y float64
		thick     float64
		which     int
	}
	var open []*seg
	flush := func(s *seg) {
		if s.x1-s.x0 <= 0 {
			return
		}
		dash := []float64(nil)
		switch s.d.Style {
		case "dashed":
			dash = []float64{3 * s.thick, 3 * s.thick}
		case "dotted":
			dash = []float64{s.thick, s.thick}
		}
		l.emit(&LineItem{X1: s.x0, Y1: s.y, X2: s.x1, Y2: s.y, Width: s.thick, Color: s.d.Color, Dash: dash})
		if s.d.Style == "double" {
			l.emit(&LineItem{X1: s.x0, Y1: s.y + 2*s.thick, X2: s.x1, Y2: s.y + 2*s.thick, Width: s.thick, Color: s.d.Color})
		}
	}
	for i, f := range frags {
		if (f.kind != fText && f.kind != fSpace) || len(f.decos) == 0 || len(f.runs) == 0 {
			continue
		}
		w := f.width
		if extra > 0 && f.kind == fSpace && i > 0 && i < len(frags)-1 {
			w += extra
		}
		shift := 0.0
		if f.ctx != nil {
			shift = f.ctx.shift
		}
		rx, ry := f.rel()
		shift += ry
		fx := xs[i] + rx
		face := f.runs[0].Face
		fm := face.Metrics()
		size := f.style.FontSize
		px := size / 0.75
		thick := math.Max(1, math.Round(fm.UnderlineThickness*px)) * 0.75
		for _, d := range f.decos {
			type line struct {
				on bool
				y  float64
				k  int
			}
			for _, ln := range []line{
				{d.Lines.Underline, baseline + shift + math.Max(1, math.Round(fm.UnderlinePos*px))*0.75 + thick/2, 0},
				{d.Lines.Overline, baseline + shift - m3(fm.Ascent*size), 1},
				{d.Lines.LineThrough, baseline + shift - fm.StrikePos*size + thick/2, 2},
			} {
				if !ln.on {
					continue
				}
				merged := false
				for _, s := range open {
					if s.which == ln.k && s.d == d && math.Abs(s.y-ln.y) < eps && math.Abs(s.x1-fx) < 0.1 {
						s.x1 = fx + w
						merged = true
						break
					}
				}
				if !merged {
					open = append(open, &seg{d: d, x0: fx, x1: fx + w, y: ln.y, thick: thick, which: ln.k})
				}
			}
		}
	}
	for _, s := range open {
		flush(s)
	}
}

func m3(v float64) float64 { return v }
