package svg

import (
	"math"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// kappa is the cubic Bézier control distance for a quarter circle.
const kappa = 0.5522847498

// geometry returns the path of a basic shape or <path> element.
func (p *parser) geometry(n *html.Node) []Seg {
	switch n.Data {
	case "path":
		d, _ := attr(n, "d")
		return parsePath(d)
	case "rect":
		x, _ := p.len(n, "x", p.vw)
		y, _ := p.len(n, "y", p.vh)
		w, _ := p.len(n, "width", p.vw)
		h, _ := p.len(n, "height", p.vh)
		if w <= 0 || h <= 0 {
			return nil
		}
		rx, okx := p.len(n, "rx", p.vw)
		ry, oky := p.len(n, "ry", p.vh)
		if !okx {
			rx = ry
		}
		if !oky {
			ry = rx
		}
		rx, ry = math.Min(math.Max(rx, 0), w/2), math.Min(math.Max(ry, 0), h/2)
		return roundRect(x, y, w, h, rx, ry)
	case "circle":
		cx, _ := p.len(n, "cx", p.vw)
		cy, _ := p.len(n, "cy", p.vh)
		r, _ := p.len(n, "r", math.Hypot(p.vw, p.vh)/math.Sqrt2)
		if r <= 0 {
			return nil
		}
		return ellipse(cx, cy, r, r)
	case "ellipse":
		cx, _ := p.len(n, "cx", p.vw)
		cy, _ := p.len(n, "cy", p.vh)
		rx, _ := p.len(n, "rx", p.vw)
		ry, _ := p.len(n, "ry", p.vh)
		if rx <= 0 || ry <= 0 {
			return nil
		}
		return ellipse(cx, cy, rx, ry)
	case "line":
		x1, _ := p.len(n, "x1", p.vw)
		y1, _ := p.len(n, "y1", p.vh)
		x2, _ := p.len(n, "x2", p.vw)
		y2, _ := p.len(n, "y2", p.vh)
		return []Seg{{Op: 'M', Pts: [6]float64{x1, y1}}, {Op: 'L', Pts: [6]float64{x2, y2}}}
	case "polyline", "polygon":
		pts, _ := attr(n, "points")
		f := numbers(pts)
		if len(f) < 4 {
			return nil
		}
		segs := []Seg{{Op: 'M', Pts: [6]float64{f[0], f[1]}}}
		for i := 2; i+1 < len(f); i += 2 {
			segs = append(segs, Seg{Op: 'L', Pts: [6]float64{f[i], f[i+1]}})
		}
		if n.Data == "polygon" {
			segs = append(segs, Seg{Op: 'Z'})
		}
		return segs
	}
	return nil
}

func ellipse(cx, cy, rx, ry float64) []Seg {
	kx, ky := rx*kappa, ry*kappa
	return []Seg{
		{Op: 'M', Pts: [6]float64{cx + rx, cy}},
		{Op: 'C', Pts: [6]float64{cx + rx, cy + ky, cx + kx, cy + ry, cx, cy + ry}},
		{Op: 'C', Pts: [6]float64{cx - kx, cy + ry, cx - rx, cy + ky, cx - rx, cy}},
		{Op: 'C', Pts: [6]float64{cx - rx, cy - ky, cx - kx, cy - ry, cx, cy - ry}},
		{Op: 'C', Pts: [6]float64{cx + kx, cy - ry, cx + rx, cy - ky, cx + rx, cy}},
		{Op: 'Z'},
	}
}

// RoundRect returns a rectangle path with elliptical corners.
func RoundRect(x, y, w, h, rx, ry float64) []Seg { return roundRect(x, y, w, h, rx, ry) }

func roundRect(x, y, w, h, rx, ry float64) []Seg {
	if rx <= 0 || ry <= 0 {
		return []Seg{
			{Op: 'M', Pts: [6]float64{x, y}}, {Op: 'L', Pts: [6]float64{x + w, y}},
			{Op: 'L', Pts: [6]float64{x + w, y + h}}, {Op: 'L', Pts: [6]float64{x, y + h}}, {Op: 'Z'},
		}
	}
	kx, ky := rx*kappa, ry*kappa
	return []Seg{
		{Op: 'M', Pts: [6]float64{x + rx, y}},
		{Op: 'L', Pts: [6]float64{x + w - rx, y}},
		{Op: 'C', Pts: [6]float64{x + w - rx + kx, y, x + w, y + ry - ky, x + w, y + ry}},
		{Op: 'L', Pts: [6]float64{x + w, y + h - ry}},
		{Op: 'C', Pts: [6]float64{x + w, y + h - ry + ky, x + w - rx + kx, y + h, x + w - rx, y + h}},
		{Op: 'L', Pts: [6]float64{x + rx, y + h}},
		{Op: 'C', Pts: [6]float64{x + rx - kx, y + h, x, y + h - ry + ky, x, y + h - ry}},
		{Op: 'L', Pts: [6]float64{x, y + ry}},
		{Op: 'C', Pts: [6]float64{x, y + ry - ky, x + rx - kx, y, x + rx, y}},
		{Op: 'Z'},
	}
}

// --- path data --------------------------------------------------------------

type scanner struct {
	s string
	i int
}

func (sc *scanner) eof() bool { return sc.i >= len(sc.s) }

func (sc *scanner) skipSep() {
	for sc.i < len(sc.s) {
		switch sc.s[sc.i] {
		case ' ', '\t', '\n', '\r', '\f', ',':
			sc.i++
		default:
			return
		}
	}
}

func (sc *scanner) skipWS() {
	for sc.i < len(sc.s) {
		switch sc.s[sc.i] {
		case ' ', '\t', '\n', '\r', '\f':
			sc.i++
		default:
			return
		}
	}
}

func (sc *scanner) number() (float64, bool) {
	start := sc.i
	i := sc.i
	if i < len(sc.s) && (sc.s[i] == '+' || sc.s[i] == '-') {
		i++
	}
	digits := false
	for i < len(sc.s) && sc.s[i] >= '0' && sc.s[i] <= '9' {
		i++
		digits = true
	}
	if i < len(sc.s) && sc.s[i] == '.' {
		i++
		for i < len(sc.s) && sc.s[i] >= '0' && sc.s[i] <= '9' {
			i++
			digits = true
		}
	}
	if !digits {
		return 0, false
	}
	if i < len(sc.s) && (sc.s[i] == 'e' || sc.s[i] == 'E') {
		j := i + 1
		if j < len(sc.s) && (sc.s[j] == '+' || sc.s[j] == '-') {
			j++
		}
		if j < len(sc.s) && sc.s[j] >= '0' && sc.s[j] <= '9' {
			for j < len(sc.s) && sc.s[j] >= '0' && sc.s[j] <= '9' {
				j++
			}
			i = j
		}
	}
	f, err := strconv.ParseFloat(sc.s[start:i], 64)
	if err != nil {
		return 0, false
	}
	sc.i = i
	return f, true
}

// flag reads an arc flag, which may be written without a separator.
func (sc *scanner) flag() (float64, bool) {
	sc.skipSep()
	if sc.i < len(sc.s) && (sc.s[sc.i] == '0' || sc.s[sc.i] == '1') {
		v := float64(sc.s[sc.i] - '0')
		sc.i++
		return v, true
	}
	return 0, false
}

// ParsePath parses SVG path data into absolute segments (quadratic curves and
// arcs become cubic curves). Parsing stops at the first error, keeping what was read.
func ParsePath(d string) []Seg { return parsePath(d) }

func parsePath(d string) []Seg {
	var segs []Seg
	sc := scanner{s: d}
	var cx, cy, sx, sy float64 // current and subpath start
	var lcx, lcy float64       // last control point
	var lastCmd byte
	cmd := byte(0)
	nums := func(n int) ([]float64, bool) {
		out := make([]float64, n)
		for k := range n {
			sc.skipSep()
			f, ok := sc.number()
			if !ok {
				return nil, false
			}
			out[k] = f
		}
		return out, true
	}
	for {
		sc.skipSep()
		if sc.eof() {
			break
		}
		c := sc.s[sc.i]
		if strings.IndexByte("MmLlHhVvCcSsQqTtAaZz", c) >= 0 {
			cmd = c
			sc.i++
		} else if cmd == 0 {
			break
		}
		rel := cmd >= 'a'
		ox, oy := 0.0, 0.0
		if rel {
			ox, oy = cx, cy
		}
		switch cmd | 0x20 {
		case 'z':
			segs = append(segs, Seg{Op: 'Z'})
			cx, cy = sx, sy
			lastCmd = 'z'
			cmd = 0
			continue
		case 'm':
			v, ok := nums(2)
			if !ok {
				return segs
			}
			cx, cy = ox+v[0], oy+v[1]
			sx, sy = cx, cy
			segs = append(segs, Seg{Op: 'M', Pts: [6]float64{cx, cy}})
			// subsequent pairs are implicit lineto
			if rel {
				cmd = 'l'
			} else {
				cmd = 'L'
			}
		case 'l':
			v, ok := nums(2)
			if !ok {
				return segs
			}
			cx, cy = ox+v[0], oy+v[1]
			segs = append(segs, Seg{Op: 'L', Pts: [6]float64{cx, cy}})
		case 'h':
			v, ok := nums(1)
			if !ok {
				return segs
			}
			cx = ox + v[0]
			segs = append(segs, Seg{Op: 'L', Pts: [6]float64{cx, cy}})
		case 'v':
			v, ok := nums(1)
			if !ok {
				return segs
			}
			if rel {
				cy += v[0]
			} else {
				cy = v[0]
			}
			segs = append(segs, Seg{Op: 'L', Pts: [6]float64{cx, cy}})
		case 'c':
			v, ok := nums(6)
			if !ok {
				return segs
			}
			segs = append(segs, Seg{Op: 'C', Pts: [6]float64{ox + v[0], oy + v[1], ox + v[2], oy + v[3], ox + v[4], oy + v[5]}})
			lcx, lcy = ox+v[2], oy+v[3]
			cx, cy = ox+v[4], oy+v[5]
		case 's':
			v, ok := nums(4)
			if !ok {
				return segs
			}
			x1, y1 := cx, cy
			if lastCmd == 'c' || lastCmd == 's' {
				x1, y1 = 2*cx-lcx, 2*cy-lcy
			}
			segs = append(segs, Seg{Op: 'C', Pts: [6]float64{x1, y1, ox + v[0], oy + v[1], ox + v[2], oy + v[3]}})
			lcx, lcy = ox+v[0], oy+v[1]
			cx, cy = ox+v[2], oy+v[3]
		case 'q':
			v, ok := nums(4)
			if !ok {
				return segs
			}
			qx, qy := ox+v[0], oy+v[1]
			ex, ey := ox+v[2], oy+v[3]
			segs = append(segs, quad(cx, cy, qx, qy, ex, ey))
			lcx, lcy = qx, qy
			cx, cy = ex, ey
		case 't':
			v, ok := nums(2)
			if !ok {
				return segs
			}
			qx, qy := cx, cy
			if lastCmd == 'q' || lastCmd == 't' {
				qx, qy = 2*cx-lcx, 2*cy-lcy
			}
			ex, ey := ox+v[0], oy+v[1]
			segs = append(segs, quad(cx, cy, qx, qy, ex, ey))
			lcx, lcy = qx, qy
			cx, cy = ex, ey
		case 'a':
			v, ok := nums(3)
			if !ok {
				return segs
			}
			large, ok1 := sc.flag()
			sweep, ok2 := sc.flag()
			end, ok3 := nums(2)
			if !ok1 || !ok2 || !ok3 {
				return segs
			}
			ex, ey := ox+end[0], oy+end[1]
			segs = append(segs, arc(cx, cy, v[0], v[1], v[2], large != 0, sweep != 0, ex, ey)...)
			cx, cy = ex, ey
		}
		lastCmd = cmd | 0x20
	}
	return segs
}

func quad(x0, y0, qx, qy, x, y float64) Seg {
	return Seg{Op: 'C', Pts: [6]float64{x0 + 2.0/3*(qx-x0), y0 + 2.0/3*(qy-y0), x + 2.0/3*(qx-x), y + 2.0/3*(qy-y), x, y}}
}

// arc converts an SVG elliptical arc to cubic curves (SVG 1.1 implementation notes F.6).
func arc(x1, y1, rx, ry, phiDeg float64, large, sweep bool, x2, y2 float64) []Seg {
	if x1 == x2 && y1 == y2 {
		return nil
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		return []Seg{{Op: 'L', Pts: [6]float64{x2, y2}}}
	}
	phi := phiDeg * math.Pi / 180
	cos, sin := math.Cos(phi), math.Sin(phi)
	dx, dy := (x1-x2)/2, (y1-y2)/2
	x1p := cos*dx + sin*dy
	y1p := -sin*dx + cos*dy
	lambda := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry)
	if lambda > 1 {
		s := math.Sqrt(lambda)
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	coef := 0.0
	if den != 0 && num > 0 {
		coef = math.Sqrt(num / den)
	}
	if large == sweep {
		coef = -coef
	}
	cxp := coef * rx * y1p / ry
	cyp := -coef * ry * x1p / rx
	cx := cos*cxp - sin*cyp + (x1+x2)/2
	cy := sin*cxp + cos*cyp + (y1+y2)/2
	angle := func(ux, uy, vx, vy float64) float64 {
		a := math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
		return a
	}
	t1 := angle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	dt := angle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)
	if !sweep && dt > 0 {
		dt -= 2 * math.Pi
	} else if sweep && dt < 0 {
		dt += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dt) / (math.Pi / 2)))
	if n < 1 {
		n = 1
	}
	step := dt / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	var segs []Seg
	pt := func(t float64) (float64, float64) {
		x, y := rx*math.Cos(t), ry*math.Sin(t)
		return cos*x - sin*y + cx, sin*x + cos*y + cy
	}
	deriv := func(t float64) (float64, float64) {
		x, y := -rx*math.Sin(t), ry*math.Cos(t)
		return cos*x - sin*y, sin*x + cos*y
	}
	t := t1
	for range n {
		ax, ay := pt(t)
		bx, by := pt(t + step)
		dax, day := deriv(t)
		dbx, dby := deriv(t + step)
		segs = append(segs, Seg{Op: 'C', Pts: [6]float64{ax + k*dax, ay + k*day, bx - k*dbx, by - k*dby, bx, by}})
		t += step
	}
	// land exactly on the end point
	segs[len(segs)-1].Pts[4], segs[len(segs)-1].Pts[5] = x2, y2
	return segs
}

// parseTransform parses a transform list.
func parseTransform(v string) Matrix {
	m := Identity
	for {
		v = strings.TrimLeft(v, " \t\n\r,")
		open := strings.IndexByte(v, '(')
		if open < 0 {
			return m
		}
		close := strings.IndexByte(v, ')')
		if close < open {
			return m
		}
		name := strings.TrimSpace(v[:open])
		a := numbers(v[open+1 : close])
		v = v[close+1:]
		var t Matrix
		switch name {
		case "matrix":
			if len(a) != 6 {
				continue
			}
			t = Matrix{a[0], a[1], a[2], a[3], a[4], a[5]}
		case "translate":
			if len(a) == 0 {
				continue
			}
			ty := 0.0
			if len(a) > 1 {
				ty = a[1]
			}
			t = Translate(a[0], ty)
		case "scale":
			if len(a) == 0 {
				continue
			}
			sy := a[0]
			if len(a) > 1 {
				sy = a[1]
			}
			t = ScaleXY(a[0], sy)
		case "rotate":
			if len(a) == 0 {
				continue
			}
			r := a[0] * math.Pi / 180
			c, s := math.Cos(r), math.Sin(r)
			t = Matrix{c, s, -s, c, 0, 0}
			if len(a) == 3 {
				t = Translate(a[1], a[2]).Mul(t).Mul(Translate(-a[1], -a[2]))
			}
		case "skewX":
			if len(a) == 0 {
				continue
			}
			t = Matrix{1, 0, math.Tan(a[0] * math.Pi / 180), 1, 0, 0}
		case "skewY":
			if len(a) == 0 {
				continue
			}
			t = Matrix{1, math.Tan(a[0] * math.Pi / 180), 0, 1, 0, 0}
		default:
			continue
		}
		m = m.Mul(t)
	}
}
