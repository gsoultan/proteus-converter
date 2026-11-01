package layout

import (
	"math"
	"strings"

	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/svg"
)

// maxTiles bounds the number of repeated background tiles per box.
const maxTiles = 4000

// boxAreas returns the border, padding and content boxes of a box whose
// border box is (x, y, w, h).
func boxAreas(s *css.Style, x, y, w, h, cbw float64) (border, padding, content Clip) {
	bt, br, bb, bl := s.BorderUsed(css.Top), s.BorderUsed(css.Right), s.BorderUsed(css.Bottom), s.BorderUsed(css.Left)
	pt, pr := s.Padding[css.Top].Resolve(cbw, 0), s.Padding[css.Right].Resolve(cbw, 0)
	pb, pl := s.Padding[css.Bottom].Resolve(cbw, 0), s.Padding[css.Left].Resolve(cbw, 0)
	border = Clip{X: x, Y: y, W: w, H: h}
	padding = Clip{X: x + bl, Y: y + bt, W: w - bl - br, H: h - bt - bb}
	content = Clip{X: padding.X + pl, Y: padding.Y + pt, W: padding.W - pl - pr, H: padding.H - pt - pb}
	return
}

func pickArea(kind string, border, padding, content Clip) Clip {
	switch kind {
	case "padding-box":
		return padding
	case "content-box":
		return content
	}
	return border
}

// emitBackground paints the background color, image or gradient of a box
// whose border box is (x, y, w, h).
func (l *layouter) emitBackground(b *Box, x, y, w, h float64) {
	l.emitBackgroundCB(b, x, y, w, h, w)
}

func (l *layouter) emitBackgroundCB(b *Box, x, y, w, h, cbw float64) {
	s := b.Style
	if s.Visibility != "visible" || w <= 0 || h <= 0 {
		return
	}
	border, padding, content := boxAreas(s, x, y, w, h, cbw)
	radius := s.BorderRadius[0].Resolve(w, 0)
	l.emitShadows(s, x, y, w, h, math.Min(radius, math.Min(w, h)/2), b != l.e.canvasBox && s.BackgroundColor.A >= 1)
	clip := pickArea(s.BackgroundClip, border, padding, content)
	clip.Radius = math.Max(0, radius-(clip.X-x))
	if b != l.e.canvasBox && !s.BackgroundColor.IsTransparent() && clip.W > 0 && clip.H > 0 {
		if r, mixed := cornerRadii(s, w, h); mixed {
			in := clip.X - x
			for i := range r {
				r[i] = math.Max(r[i]-in, 0)
			}
			c := s.BackgroundColor
			l.emit(&PathItem{M: svg.Identity, Segs: svg.RoundRect4(clip.X, clip.Y, clip.W, clip.H, r),
				Style: PathStyle{Fill: &c}, Top: clip.Y})
		} else {
			l.emit(&RectItem{X: clip.X, Y: clip.Y, W: clip.W, H: clip.H, Color: s.BackgroundColor, Radius: clip.Radius})
		}
	}
	area := pickArea(s.BackgroundOrigin, border, padding, content)
	if area.W <= 0 || area.H <= 0 || clip.W <= 0 || clip.H <= 0 {
		return
	}
	switch {
	case b.BgImage != nil:
		l.emitBackgroundImage(s, b.BgImage, area, clip)
	case s.BackgroundGradient != "":
		l.emitGradient(s.BackgroundGradient, area, clip)
	}
}

func (l *layouter) emitBackgroundImage(s *css.Style, img *Image, area, clip Clip) {
	iw, ih := float64(img.Width)*0.75, float64(img.Height)*0.75
	ratio := 0.0
	if img.SVG != nil {
		ratio = img.SVG.Ratio()
		iw, ih = img.SVG.Width*0.75, img.SVG.Height*0.75
	} else if iw > 0 {
		ratio = ih / iw
	}
	var tw, th float64
	bs := s.BackgroundSize
	switch bs.Mode {
	case "cover", "contain":
		if ratio <= 0 {
			tw, th = area.W, area.H
			break
		}
		k := math.Max(area.W, area.H/ratio)
		if bs.Mode == "contain" {
			k = math.Min(area.W, area.H/ratio)
		}
		tw, th = k, k*ratio
	default:
		switch {
		case !bs.W.Auto && !bs.H.Auto:
			tw, th = bs.W.Resolve(area.W, 0), bs.H.Resolve(area.H, 0)
		case !bs.W.Auto:
			tw = bs.W.Resolve(area.W, 0)
			th = area.H
			if ratio > 0 {
				th = tw * ratio
			} else if ih > 0 {
				th = ih
			}
		case !bs.H.Auto:
			th = bs.H.Resolve(area.H, 0)
			tw = area.W
			if ratio > 0 {
				tw = th / ratio
			} else if iw > 0 {
				tw = iw
			}
		default:
			tw, th = iw, ih
			switch {
			case iw <= 0 && ih <= 0:
				tw, th = area.W, area.H
				if ratio > 0 {
					tw, th = math.Min(area.W, area.H/ratio), math.Min(area.W, area.H/ratio)*ratio
				}
			case iw <= 0:
				tw = area.W
			case ih <= 0:
				th = area.H
			}
		}
	}
	if tw <= 0.01 || th <= 0.01 {
		return
	}
	x0 := area.X + s.BackgroundPosition[0].Resolve(area.W-tw, 0)
	y0 := area.Y + s.BackgroundPosition[1].Resolve(area.H-th, 0)
	xs := tilePositions(x0, tw, clip.X, clip.W, s.BackgroundRepeat[0] != "no-repeat")
	ys := tilePositions(y0, th, clip.Y, clip.H, s.BackgroundRepeat[1] != "no-repeat")
	if len(xs)*len(ys) > maxTiles {
		l.e.warnOnce("layout: background image tiles limited")
		return
	}
	for _, ty := range ys {
		for _, tx := range xs {
			if img.SVG != nil {
				l.emitSVG(img.SVG, tx, ty, tw, th, clip)
				continue
			}
			l.emit(&ImageItem{X: tx, Y: ty, W: tw, H: th, Image: img, Clip: clip})
		}
	}
}

// tilePositions returns the tile origins along one axis that intersect [start, start+size).
func tilePositions(origin, tile, start, size float64, repeat bool) []float64 {
	if !repeat {
		if origin+tile <= start || origin >= start+size {
			return nil
		}
		return []float64{origin}
	}
	first := origin - math.Ceil((origin-start)/tile)*tile
	var out []float64
	for p := first; p < start+size-eps && len(out) <= maxTiles; p += tile {
		if p+tile > start+eps {
			out = append(out, p)
		}
	}
	return out
}

// --- gradients ----------------------------------------------------------------

type gradStop struct {
	c   css.Color
	pos float64 // fraction of the gradient line
}

// parseLinearGradient parses linear-gradient() / repeating-linear-gradient().
// angle is in degrees (CSS convention: 0 = to top, clockwise).
func parseLinearGradient(v string, w, h float64) (angle float64, stops []gradStop, repeating, ok bool) {
	fn, args, isFn := css.Function(strings.TrimSpace(v))
	if !isFn {
		return
	}
	fn = strings.TrimPrefix(strings.TrimPrefix(fn, "-webkit-"), "-moz-")
	repeating = fn == "repeating-linear-gradient"
	parts := css.SplitCommas(args)
	if len(parts) == 0 {
		return
	}
	angle = 180
	first := strings.ToLower(strings.TrimSpace(parts[0]))
	switch {
	case strings.HasPrefix(first, "to "):
		dir := strings.Fields(first[3:])
		var dx, dy float64
		for _, d := range dir {
			switch d {
			case "left":
				dx = -1
			case "right":
				dx = 1
			case "top":
				dy = -1
			case "bottom":
				dy = 1
			}
		}
		switch {
		case dx == 0 && dy == 0:
			return
		case dx != 0 && dy != 0:
			// corner: the gradient line is perpendicular to the diagonal through the other corners
			angle = math.Atan2(dx*h, -dy*w) * 180 / math.Pi
		default:
			angle = math.Atan2(dx, -dy) * 180 / math.Pi
		}
		parts = parts[1:]
	case isAngle(first):
		angle = parseAngle(first)
		parts = parts[1:]
	}
	gl := math.Abs(w*math.Sin(angle*math.Pi/180)) + math.Abs(h*math.Cos(angle*math.Pi/180))
	type raw struct {
		c   css.Color
		pos []float64
	}
	var rs []raw
	for _, p := range parts {
		toks := css.SplitValue(p)
		if len(toks) == 0 {
			continue
		}
		c, cok := css.ParseColor(toks[0], css.Black)
		if !cok {
			if len(toks) == 1 {
				continue // interpolation hint
			}
			return 0, nil, false, false
		}
		r := raw{c: c}
		for _, t := range toks[1:] {
			l, lok := css.ParseLength(t, css.LengthContext{FontSize: 12, RootFontSize: 12}, false)
			if !lok || l.Auto {
				return 0, nil, false, false
			}
			pos := l.Pct / 100
			if gl > 0 {
				pos += l.Pt / gl
			}
			r.pos = append(r.pos, pos)
		}
		rs = append(rs, r)
	}
	if len(rs) == 0 {
		return
	}
	// expand double positions, then fill in missing positions
	var cs []css.Color
	var ps []float64
	var set []bool
	for _, r := range rs {
		if len(r.pos) == 0 {
			cs, ps, set = append(cs, r.c), append(ps, 0), append(set, false)
			continue
		}
		for _, p := range r.pos {
			cs, ps, set = append(cs, r.c), append(ps, p), append(set, true)
		}
	}
	if !set[0] {
		ps[0], set[0] = 0, true
	}
	if n := len(ps) - 1; !set[n] {
		ps[n], set[n] = 1, true
	}
	maxSoFar := ps[0]
	for i := range ps {
		if set[i] {
			ps[i] = math.Max(ps[i], maxSoFar)
			maxSoFar = ps[i]
		}
	}
	for i := 0; i < len(ps); i++ {
		if set[i] {
			continue
		}
		j := i
		for !set[j] {
			j++
		}
		for k := i; k < j; k++ {
			ps[k] = ps[i-1] + (ps[j]-ps[i-1])*float64(k-i+1)/float64(j-i+1)
			set[k] = true
		}
	}
	for i := range cs {
		stops = append(stops, gradStop{c: cs[i], pos: ps[i]})
	}
	return angle, stops, repeating, true
}

func isAngle(t string) bool {
	for _, u := range []string{"deg", "grad", "rad", "turn"} {
		if strings.HasSuffix(t, u) {
			_, ok := css.ParseNumber(strings.TrimSuffix(t, u))
			return ok
		}
	}
	return false
}

func parseAngle(t string) float64 {
	units := []struct {
		u string
		k float64
	}{{"grad", 0.9}, {"turn", 360}, {"deg", 1}, {"rad", 180 / math.Pi}}
	for _, u := range units {
		if strings.HasSuffix(t, u.u) {
			f, _ := css.ParseNumber(strings.TrimSuffix(t, u.u))
			return f * u.k
		}
	}
	return 180
}

// colorAt samples the gradient at fraction t (premultiplied interpolation).
func colorAt(stops []gradStop, t float64) css.Color {
	if t <= stops[0].pos {
		return stops[0].c
	}
	for i := 1; i < len(stops); i++ {
		a, b := stops[i-1], stops[i]
		if t > b.pos {
			continue
		}
		if b.pos-a.pos < 1e-9 {
			return b.c
		}
		f := (t - a.pos) / (b.pos - a.pos)
		alpha := a.c.A + (b.c.A-a.c.A)*f
		mix := func(x, y float64) float64 {
			if alpha <= 0 {
				return x + (y-x)*f
			}
			return (x*a.c.A + (y*b.c.A-x*a.c.A)*f) / alpha
		}
		return css.Color{R: mix(a.c.R, b.c.R), G: mix(a.c.G, b.c.G), B: mix(a.c.B, b.c.B), A: alpha}
	}
	return stops[len(stops)-1].c
}

// emitGradient paints a linear gradient over area, clipped to clip, as thin
// strips (gpdf has no shading resources for graphic runs).
func (l *layouter) emitGradient(v string, area, clip Clip) {
	angle, stops, repeating, ok := parseLinearGradient(v, area.W, area.H)
	if !ok || len(stops) == 0 {
		l.e.warnOnce("layout: unsupported background gradient " + v)
		return
	}
	rad := angle * math.Pi / 180
	sin, cos := math.Sin(rad), math.Cos(rad)
	gl := math.Abs(area.W*sin) + math.Abs(area.H*cos)
	if gl <= 0 {
		return
	}
	cx, cy := area.X+area.W/2, area.Y+area.H/2
	// (u, v): u along the gradient line from its start, v across it
	m := svg.Matrix{sin, -cos, cos, sin, cx, cy}.Mul(svg.Translate(-gl/2, 0))
	span := math.Hypot(clip.W, clip.H) + math.Hypot(clip.X-cx, clip.Y-cy)*2
	period := stops[len(stops)-1].pos - stops[0].pos
	sample := func(t float64) css.Color {
		if repeating && period > 1e-6 {
			t = stops[0].pos + math.Mod(math.Mod(t-stops[0].pos, period)+period, period)
		}
		return colorAt(stops, t)
	}
	// the gradient line covers [0, gl]; extend to cover the clip area
	ext := span
	steps := int(math.Min(math.Ceil(gl/0.75), 600))
	if steps < 1 {
		steps = 1
	}
	du := gl / float64(steps)
	emitStrip := func(u0, u1 float64, c css.Color) {
		if c.A <= 0 {
			return
		}
		col := c
		l.emit(&PathItem{M: m, Top: area.Y, Clip: clip, Style: PathStyle{Fill: &col},
			Segs: svg.RoundRect(u0, -ext, u1-u0, 2*ext, 0, 0)})
	}
	// solid extensions before and after the line
	emitStrip(-ext, 0.01, sample(0))
	start := 0.0
	cur := sample(du / 2 / gl)
	for i := 1; i <= steps; i++ {
		var next css.Color
		if i < steps {
			next = sample((float64(i) + 0.5) * du / gl)
		}
		if i == steps || next != cur {
			end := float64(i) * du
			emitStrip(start, math.Min(end+1, gl+ext), cur)
			start = end
			cur = next
		}
	}
	emitStrip(gl-0.01, gl+ext, sample(1))
}

// emitShadows paints the outer box-shadows of a border box, last layer
// first. A blur is approximated by stacked translucent rounded rectangles
// whose combined opacity follows the Gaussian edge profile (sigma = blur/2).
// Inset shadows are not painted.
func (l *layouter) emitShadows(s *css.Style, x, y, w, h, radius float64, opaque bool) {
	hole := Clip{X: x, Y: y, W: w, H: h, Radius: radius}
	fill := func(r Clip, c css.Color) {
		if opaque {
			// the box's own background covers the part under it
			l.emit(&RectItem{X: r.X, Y: r.Y, W: r.W, H: r.H, Color: c, Radius: r.Radius})
			return
		}
		l.shadowOutside(r, hole, c)
	}
	for i := len(s.BoxShadow) - 1; i >= 0; i-- {
		sh := s.BoxShadow[i]
		if sh.Inset || sh.Color.A <= 0 {
			continue
		}
		sx, sy := x+sh.X-sh.Spread, y+sh.Y-sh.Spread
		sw, shh := w+2*sh.Spread, h+2*sh.Spread
		r := radius
		if r > 0 {
			r = math.Max(r+sh.Spread, 0)
		}
		if sh.Blur <= 0 {
			if sw > 0 && shh > 0 {
				fill(Clip{X: sx, Y: sy, W: sw, H: shh, Radius: r}, sh.Color)
			}
			continue
		}
		n := int(math.Min(32, math.Max(3, math.Ceil(sh.Blur/0.75/1.5))))
		step := 2 * sh.Blur / float64(n)
		sigma := sh.Blur / 2
		prev := 0.0
		for k := range n {
			t := -sh.Blur + step*float64(k) // inset of this layer (negative = grown)
			target := sh.Color.A * normCDF((t+step)/sigma)
			if k == n-1 {
				target = sh.Color.A
			}
			alpha := 1 - (1-target)/(1-prev)
			prev = target
			lw, lh := sw-2*t, shh-2*t
			if alpha <= 0 || lw <= 0 || lh <= 0 {
				continue
			}
			c := sh.Color
			c.A = math.Min(alpha, 1)
			fill(Clip{X: sx + t, Y: sy + t, W: lw, H: lh, Radius: math.Max(r-t, 0)}, c)
		}
	}
}

// shadowOutside fills the part of r outside the border box hole (an outer
// box-shadow never shows through the box itself).
func (l *layouter) shadowOutside(r, hole Clip, c css.Color) {
	inside := hole.X >= r.X && hole.Y >= r.Y && hole.X+hole.W <= r.X+r.W && hole.Y+hole.H <= r.Y+r.H
	if inside {
		segs := append(svg.RoundRect(r.X, r.Y, r.W, r.H, r.Radius, r.Radius),
			svg.RoundRect(hole.X, hole.Y, hole.W, hole.H, hole.Radius, hole.Radius)...)
		col := c
		l.emit(&PathItem{M: svg.Identity, Segs: segs, Style: PathStyle{Fill: &col, EvenOdd: true}, Top: r.Y})
		return
	}
	// the hole sticks out: fill the bands of r around it (corner radii ignored)
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	hx0, hy0 := math.Max(hole.X, x0), math.Max(hole.Y, y0)
	hx1, hy1 := math.Min(hole.X+hole.W, x1), math.Min(hole.Y+hole.H, y1)
	if hx0 >= hx1 || hy0 >= hy1 {
		l.emit(&RectItem{X: r.X, Y: r.Y, W: r.W, H: r.H, Color: c, Radius: r.Radius})
		return
	}
	band := func(x, y, w, h float64) {
		if w > 0 && h > 0 {
			l.emit(&RectItem{X: x, Y: y, W: w, H: h, Color: c})
		}
	}
	band(x0, y0, x1-x0, hy0-y0)
	band(x0, hy1, x1-x0, y1-hy1)
	band(x0, hy0, hx0-x0, hy1-hy0)
	band(hx1, hy0, x1-hx1, hy1-hy0)
}

// normCDF is the standard normal cumulative distribution function.
func normCDF(x float64) float64 { return 0.5 * (1 + math.Erf(x/math.Sqrt2)) }

// emitOutline paints the outline of a border box (outside it, past
// outline-offset), following the border radius like Chromium.
func (l *layouter) emitOutline(s *css.Style, x, y, w, h float64) {
	ow := s.OutlineUsed()
	if ow <= 0 || s.Visibility != "visible" || s.OutlineColor.IsTransparent() {
		return
	}
	d := s.OutlineOffset + ow
	o := *s
	for i := range 4 {
		o.BorderWidth[i], o.BorderStyle[i], o.BorderColor[i] = ow, s.OutlineStyle, s.OutlineColor
		if r := s.BorderRadius[i].Resolve(w, 0); r > 0 {
			o.BorderRadius[i] = css.Pt(math.Max(r+d, 0))
		}
	}
	l.emitBorders(&o, x-d, y-d, w+2*d, h+2*d, [4]bool{true, true, true, true})
}

// cornerRadii returns the four resolved corner radii of a w x h border box
// and whether they differ from each other.
func cornerRadii(s *css.Style, w, h float64) (r [4]float64, mixed bool) {
	for i := range r {
		r[i] = s.BorderRadius[i].Resolve(w, 0)
		if i > 0 && r[i] != r[0] {
			mixed = true
		}
	}
	return r, mixed
}
