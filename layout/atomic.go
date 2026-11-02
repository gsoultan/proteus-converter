// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package layout

import (
	"math"

	"github.com/gsoultan/proteus-converter/css"
)

// replacedSize returns the content-box size of an image or form control.
func (l *layouter) replacedSize(b *Box, cbw float64) (w, h float64) {
	s := b.Style
	iw, ih, ratio, hasW, hasH := intrinsicSize(b)
	e := l.hEdges(s, cbw)
	extraW := e.pl + e.pr + e.bl + e.br
	extraH := s.Padding[css.Top].Resolve(cbw, 0) + s.Padding[css.Bottom].Resolve(cbw, 0) + s.BorderUsed(css.Top) + s.BorderUsed(css.Bottom)
	content := func(v, extra float64) float64 {
		if s.BoxSizing == "border-box" {
			return math.Max(v-extra, 0)
		}
		return v
	}
	wSet := !s.Width.Auto
	hSet := !s.Height.Auto && !s.Height.HasPct()
	if wSet {
		w = content(s.Width.Resolve(cbw, 0), extraW)
	}
	if hSet {
		h = content(s.Height.Resolve(0, 0), extraH)
	}
	if ratio <= 0 && hasW && hasH && iw > 0 {
		ratio = ih / iw
	}
	switch {
	case wSet && hSet:
	case wSet:
		if ratio > 0 {
			h = w * ratio
		} else if hasH {
			h = ih
		} else {
			h = 150 * 0.75
		}
	case hSet:
		if ratio > 0 {
			w = h / ratio
		} else if hasW {
			w = iw
		} else {
			w = 300 * 0.75
		}
	case hasW && hasH:
		w, h = iw, ih
	case hasW && ratio > 0:
		w, h = iw, iw*ratio
	case hasH && ratio > 0:
		w, h = ih/ratio, ih
	case ratio > 0:
		// no intrinsic size: fill the available width (as browsers do for SVG with only a viewBox)
		w = math.Max(cbw-e.ml-e.mr-extraW, 0)
		if cbw <= 0 {
			w = 300 * 0.75
		}
		h = w * ratio
	default:
		w, h = 300*0.75, 150*0.75
		if hasW {
			w = iw
		}
		if hasH {
			h = ih
		}
	}
	if ratio <= 0 {
		ratio = 1
		if w > 0 {
			ratio = h / w
		}
	}
	fw, fwOK := l.e.forcedW[b]
	fh, fhOK := l.e.forcedH[b]
	if fwOK {
		w = math.Max(fw-extraW, 0)
		if !fhOK && !hSet {
			h = w * ratio
		}
	}
	if fhOK {
		h = math.Max(fh-extraH, 0)
		if !fwOK && !wSet {
			w = h / ratio
		}
	}
	if fwOK || fhOK {
		return w, h
	}
	if !s.MaxWidth.Auto {
		if mw := content(s.MaxWidth.Resolve(cbw, 0), extraW); w > mw {
			w = mw
			if !hSet {
				h = w * ratio
			}
		}
	}
	if !s.MaxHeight.Auto && !s.MaxHeight.HasPct() {
		if mh := content(s.MaxHeight.Resolve(0, 0), extraH); h > mh {
			h = mh
			if !wSet {
				w = h / ratio
			}
		}
	}
	if mw := content(s.MinWidth.Resolve(cbw, 0), extraW); w < mw {
		w = mw
	}
	return w, h
}

// layoutAtomic lays out an atomic inline-level box at the origin (margin box top-left).
func (l *layouter) layoutAtomic(b *Box, avail float64) *atomicBox {
	s := b.Style
	a := &atomicBox{box: b, valign: s.VerticalAlign}
	switch s.VerticalAlign {
	case "sub":
		a.valign, a.valignLen = "length", s.FontSize/5+0.75
	case "super":
		a.valign, a.valignLen = "length", -(s.FontSize/3 + 0.75)
	case "length":
		a.valignLen = -s.VerticalLength.Resolve(l.e.styleMetrics(s).lh, 0)
	}
	e := l.hEdges(s, avail)
	mt, mb := s.Margin[css.Top].Resolve(avail, 0), s.Margin[css.Bottom].Resolve(avail, 0)
	switch b.Kind {
	case KindImage, KindCheckbox:
		w, h := l.replacedSize(b, avail)
		bt, bb := s.BorderUsed(css.Top), s.BorderUsed(css.Bottom)
		pt, pb := s.Padding[css.Top].Resolve(avail, 0), s.Padding[css.Bottom].Resolve(avail, 0)
		bx, by := e.ml, mt
		bw, bh := e.bl+e.pl+w+e.pr+e.br, bt+pt+h+pb+bb
		sub := l.sub()
		sub.emitBackground(b, bx, by, bw, bh)
		sub.emitBorders(s, bx, by, bw, bh, [4]bool{true, true, true, true})
		cx, cy := bx+e.bl+e.pl, by+bt+pt
		if b.Kind == KindImage && s.Visibility == "visible" {
			a.image = sub.emitReplaced(b, cx, cy, w, h, innerRadius(s, bw, math.Max(e.bl+e.pl, bt+pt)))
		} else if b.Kind == KindCheckbox {
			sub.checkbox(b, cx, cy, w, h)
		}
		a.items = sub.items
		a.w = e.ml + bw + e.mr
		a.h = mt + bh + mb
		a.baseline = a.h
		if b.Kind == KindCheckbox {
			a.baseline = mt + bh
		}
		return a
	}
	// inline-block and friends
	forced := -1.0
	if s.Width.Auto {
		mn, mx := l.intrinsicWidths(b)
		shrink := math.Min(math.Max(mn, avail), mx)
		forced = shrink - e.ml - e.mr
		_, cw := l.blockWidth(s, avail, forced)
		forced = cw + e.bl + e.br + e.pl + e.pr
	}
	sub := l.sub()
	r := sub.layoutBlockW(b, 0, 0, avail, margins{}, true, forced)
	ee, cw := sub.blockWidth(s, avail, forced)
	a.items = sub.items
	a.w = ee.ml + ee.bl + ee.pl + cw + ee.pr + ee.br + ee.mr
	a.h = r.bottom + mb
	a.baseline = a.h
	if r.hasBaseline && (s.Overflow == "visible" || s.Overflow == "") {
		a.baseline = r.lastBaseline
	}
	return a
}

// checkbox draws a Chromium-like checkbox or radio control.
func (l *layouter) checkbox(b *Box, x, y, w, h float64) {
	border := css.Color{R: 0x76 / 255.0, G: 0x76 / 255.0, B: 0x76 / 255.0, A: 1}
	blue := css.Color{R: 0, G: 0x75 / 255.0, B: 1, A: 1}
	white := css.Color{R: 1, G: 1, B: 1, A: 1}
	if b.Disabled {
		// Chromium paints disabled controls at reduced contrast
		border = css.Color{R: 0xc5 / 255.0, G: 0xc5 / 255.0, B: 0xc5 / 255.0, A: 1}
		blue = border
	}
	r := 2 * 0.75
	if b.Checked {
		l.emit(&RectItem{X: x, Y: y, W: w, H: h, Color: blue, Radius: r})
		l.emit(&LineItem{X1: x + w*0.22, Y1: y + h*0.52, X2: x + w*0.42, Y2: y + h*0.72, Width: 1.5, Color: white})
		l.emit(&LineItem{X1: x + w*0.42, Y1: y + h*0.72, X2: x + w*0.78, Y2: y + h*0.30, Width: 1.5, Color: white})
		return
	}
	l.emit(&RectItem{X: x, Y: y, W: w, H: h, Color: border, Radius: r})
	l.emit(&RectItem{X: x + 0.75, Y: y + 0.75, W: w - 1.5, H: h - 1.5, Color: white, Radius: r - 0.75})
}

// intrinsicWidths returns the min-content and max-content widths of b's
// margin box (absolute margins only).
func (l *layouter) intrinsicWidths(b *Box) (float64, float64) {
	if v, ok := l.e.intrinsic[b]; ok {
		return v[0], v[1]
	}
	mn, mx := l.computeIntrinsic(b)
	l.e.intrinsic[b] = [2]float64{mn, mx}
	return mn, mx
}

func absPart(v css.Length) float64 {
	if v.Auto || v.Pct != 0 {
		return 0
	}
	return v.Pt
}

func (l *layouter) computeIntrinsic(b *Box) (float64, float64) {
	s := b.Style
	pad := absPart(s.Padding[css.Left]) + absPart(s.Padding[css.Right]) + s.BorderUsed(css.Left) + s.BorderUsed(css.Right)
	marg := absPart(s.Margin[css.Left]) + absPart(s.Margin[css.Right])
	if b.Kind == KindImage || b.Kind == KindCheckbox {
		w, _ := l.replacedSize(b, 0)
		if s.MaxWidth.HasPct() || s.Width.HasPct() {
			return marg + pad, w + pad + marg
		}
		return w + pad + marg, w + pad + marg
	}
	if !s.Width.Auto && !s.Width.HasPct() && b.Kind != KindTable {
		w := s.Width.Pt
		if s.BoxSizing != "border-box" {
			w += pad
		}
		return w + marg, w + marg
	}
	var mn, mx float64
	switch {
	case b.Kind == KindTable:
		mn, mx = l.tableIntrinsic(b)
	case (s.IsFlex() && (s.FlexDirection == "row" || s.FlexDirection == "row-reverse")) && hasBlockChild(b):
		gap := absPart(s.ColumnGap)
		n := 0
		for _, c := range b.Children {
			if c.outOfFlow() {
				continue
			}
			cmn, cmx := l.intrinsicWidths(c)
			if s.FlexWrap == "nowrap" {
				mn += cmn
			} else {
				mn = math.Max(mn, cmn)
			}
			mx += cmx
			n++
		}
		if n > 1 {
			mx += gap * float64(n-1)
			if s.FlexWrap == "nowrap" {
				mn += gap * float64(n-1)
			}
		}
	case s.IsGrid() && hasBlockChild(b):
		mn, mx = l.gridIntrinsic(b)
	case hasBlockChild(b):
		for _, c := range b.Children {
			if c.outOfFlow() && c.Style.IsAbsolute() {
				continue
			}
			cmn, cmx := l.intrinsicWidths(c)
			mn = math.Max(mn, cmn)
			mx = math.Max(mx, cmx)
		}
	default:
		frags := l.buildFrags(b.Children, s, 0, nil)
		if b.Marker != nil && s.ListStylePosition == "inside" {
			frags = append(l.buildFrags(b.Marker.Children, b.Marker.Style, 0, nil), frags...)
		}
		mn, mx = fragIntrinsic(frags)
		mn += math.Max(0, absPart(s.TextIndent))
		mx += absPart(s.TextIndent)
	}
	if !s.MinWidth.Auto && !s.MinWidth.HasPct() {
		mn = math.Max(mn, s.MinWidth.Pt)
		mx = math.Max(mx, s.MinWidth.Pt)
	}
	if !s.MaxWidth.Auto && !s.MaxWidth.HasPct() {
		mn = math.Min(mn, s.MaxWidth.Pt)
		mx = math.Min(mx, s.MaxWidth.Pt)
	}
	return mn + pad + marg, mx + pad + marg
}

func fragIntrinsic(frags []*frag) (mn, mx float64) {
	word, line := 0.0, 0.0
	trailing := 0.0
	for _, f := range frags {
		switch f.kind {
		case fFloat:
			mn = math.Max(mn, f.atomic.w)
			line += f.atomic.w
			continue
		case fAbs:
			continue
		case fBreak:
			mx = math.Max(mx, line-trailing)
			mn = math.Max(mn, word)
			line, word, trailing = 0, 0, 0
			continue
		case fSpace:
			if f.collapse && line == 0 {
				continue
			}
			line += f.width
			trailing += f.width
			if f.breakOK {
				mn = math.Max(mn, word)
				word = 0
			} else {
				word += f.width
			}
			continue
		}
		trailing = 0
		line += f.width
		if f.kind == fText && canBreakAnywhere(f.style) {
			for _, r := range f.text {
				mn = math.Max(mn, f.width/float64(len([]rune(f.text))))
				_ = r
				break
			}
			word = 0
			continue
		}
		word += f.width
		if f.breakOK {
			mn = math.Max(mn, word)
			word = 0
		}
	}
	mx = math.Max(mx, line-trailing)
	mn = math.Max(mn, word)
	return mn, mx
}
