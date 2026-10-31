package layout

import (
	"math"

	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/svg"
)

// intrinsicSize returns the natural size of a replaced box in points and
// whether width and height are known; ratio is height/width (0 when unknown).
func intrinsicSize(b *Box) (w, h, ratio float64, hasW, hasH bool) {
	if b.Image == nil {
		return 13 * 0.75, 13 * 0.75, 1, true, true
	}
	img := b.Image
	if img.SVG != nil {
		ratio = img.SVG.Ratio()
		if img.SVG.Width > 0 {
			w, hasW = img.SVG.Width*0.75, true
		}
		if img.SVG.Height > 0 {
			h, hasH = img.SVG.Height*0.75, true
		}
		return w, h, ratio, hasW, hasH
	}
	w, h = float64(img.Width)*0.75, float64(img.Height)*0.75
	if w > 0 {
		ratio = h / w
	}
	return w, h, ratio, true, true
}

// innerRadius returns the clip radius of the content box for a border-box radius.
func innerRadius(s *css.Style, bw float64, inset float64) float64 {
	r := s.BorderRadius[0].Resolve(bw, 0)
	return math.Max(0, r-inset)
}

// emitReplaced paints the content of an image box into its content box (x, y, w, h).
func (l *layouter) emitReplaced(b *Box, x, y, w, h, radius float64) *ImageItem {
	s := b.Style
	img := b.Image
	if img.SVG != nil {
		l.emitSVG(img.SVG, x, y, w, h, Clip{X: x, Y: y, W: w, H: h, Radius: radius})
		return nil
	}
	iw, ih := float64(img.Width)*0.75, float64(img.Height)*0.75
	dw, dh := w, h
	if iw > 0 && ih > 0 {
		switch s.ObjectFit {
		case "contain", "cover", "scale-down", "none":
			k := 1.0
			switch s.ObjectFit {
			case "contain":
				k = math.Min(w/iw, h/ih)
			case "cover":
				k = math.Max(w/iw, h/ih)
			case "scale-down":
				k = math.Min(1, math.Min(w/iw, h/ih))
			}
			dw, dh = iw*k, ih*k
		}
	}
	dx := s.ObjectPosition[0].Resolve(w-dw, 0)
	dy := s.ObjectPosition[1].Resolve(h-dh, 0)
	it := &ImageItem{X: x + dx, Y: y + dy, W: dw, H: dh, Image: img}
	if radius > 0 || dw > w+eps || dh > h+eps || dx < -eps || dy < -eps {
		it.Clip = Clip{X: x, Y: y, W: w, H: h, Radius: radius}
	}
	l.emit(it)
	return it
}

// emitSVG paints an SVG document into the box (x, y, w, h).
func (l *layouter) emitSVG(d *svg.Document, x, y, w, h float64, clip Clip) {
	if w <= 0 || h <= 0 {
		return
	}
	base := svg.Translate(x, y).Mul(d.ViewportTransform(w, h))
	for _, it := range d.Items {
		if sh := it.Shape; sh != nil {
			l.emit(&PathItem{M: base.Mul(sh.Transform), Segs: sh.Segs, Top: y, Clip: clip, Style: PathStyle{
				Fill: sh.Fill, EvenOdd: sh.EvenOdd, Stroke: sh.Stroke, StrokeWidth: sh.StrokeWidth,
				LineCap: sh.LineCap, LineJoin: sh.LineJoin, MiterLimit: sh.MiterLimit, Dash: sh.Dash}})
			continue
		}
		t := it.Text
		m := base.Mul(t.Transform)
		if math.Abs(m[1]) > 1e-6 || math.Abs(m[2]) > 1e-6 {
			l.e.warnOnce("layout: rotated or skewed SVG text is not supported")
			continue
		}
		size := t.FontSize * math.Abs(m[3])
		if size <= 0 {
			continue
		}
		chain := l.e.Fonts.Chain(t.FontFamily, t.FontWeight, t.Italic)
		runs := l.e.Fonts.Split(t.Text, chain)
		width := 0.0
		for _, r := range runs {
			width += r.Face.Width(r.Text, size)
		}
		px, py := m.Apply(t.X, t.Y)
		switch t.Anchor {
		case "middle":
			px -= width / 2
		case "end":
			px -= width
		}
		for _, r := range runs {
			l.emit(&TextItem{X: px, Baseline: py, Text: r.Text, Face: r.Face, Size: size, Color: t.Fill})
			px += r.Face.Width(r.Text, size)
		}
	}
}
