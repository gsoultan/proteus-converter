// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package css

import (
	"math"
	"strconv"
	"strings"
)

// Length is a computed length: an absolute part in points plus a percentage
// of a basis that is only known at layout time (so calc(100% - 2em) is exact).
type Length struct {
	Pt   float64
	Pct  float64
	Auto bool
}

// Auto is the 'auto' length.
var Auto = Length{Auto: true}

// Pt returns an absolute length.
func Pt(v float64) Length { return Length{Pt: v} }

// Resolve returns the length in points against basis (used for percentages).
// 'auto' resolves to def.
func (l Length) Resolve(basis, def float64) float64 {
	if l.Auto {
		return def
	}
	return l.Pt + l.Pct/100*basis
}

// HasPct reports whether the length depends on its basis.
func (l Length) HasPct() bool { return !l.Auto && l.Pct != 0 }

// LengthContext holds what is needed to turn relative units into points.
type LengthContext struct {
	FontSize     float64 // element font size in pt (for em)
	RootFontSize float64 // root font size in pt (for rem)
	ViewportW    float64 // page content width in pt (vw)
	ViewportH    float64 // page content height in pt (vh)
}

// Points per unit for absolute units.
var absUnits = map[string]float64{
	"px": 0.75, "pt": 1, "pc": 12, "in": 72, "cm": 72 / 2.54, "mm": 72 / 25.4, "q": 72 / 101.6,
}

// ParseLength parses a length, percentage, calc()/min()/max()/clamp() or 'auto'.
// Unitless zero is accepted; other unitless numbers are accepted only when
// allowNumber is true and are treated as px.
func ParseLength(v string, ctx LengthContext, allowNumber bool) (Length, bool) {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "auto" {
		return Auto, true
	}
	if fn, args, ok := Function(v); ok {
		switch fn {
		case "calc", "-webkit-calc":
			return evalCalc(args, ctx)
		case "min", "max", "clamp":
			return evalMinMax(fn, args, ctx)
		}
		return Length{}, false
	}
	if strings.HasSuffix(v, "%") {
		f, err := strconv.ParseFloat(v[:len(v)-1], 64)
		if err != nil {
			return Length{}, false
		}
		return Length{Pct: f}, true
	}
	num, unit := splitNumber(v)
	if num == "" {
		return Length{}, false
	}
	f, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return Length{}, false
	}
	if unit == "" {
		if f == 0 || allowNumber {
			return Length{Pt: f * 0.75}, true
		}
		return Length{}, false
	}
	if k, ok := absUnits[unit]; ok {
		return Length{Pt: f * k}, true
	}
	switch unit {
	case "em":
		return Length{Pt: f * ctx.FontSize}, true
	case "rem":
		return Length{Pt: f * ctx.RootFontSize}, true
	case "ex":
		return Length{Pt: f * ctx.FontSize * 0.5}, true
	case "ch":
		return Length{Pt: f * ctx.FontSize * 0.5}, true
	case "vw", "svw", "lvw", "dvw":
		return Length{Pt: f * ctx.ViewportW / 100}, true
	case "vh", "svh", "lvh", "dvh":
		return Length{Pt: f * ctx.ViewportH / 100}, true
	case "vmin":
		return Length{Pt: f * math.Min(ctx.ViewportW, ctx.ViewportH) / 100}, true
	case "vmax":
		return Length{Pt: f * math.Max(ctx.ViewportW, ctx.ViewportH) / 100}, true
	}
	return Length{}, false
}

func splitNumber(v string) (num, unit string) {
	i := 0
	if i < len(v) && (v[i] == '+' || v[i] == '-') {
		i++
	}
	digits := false
	for i < len(v) && ((v[i] >= '0' && v[i] <= '9') || v[i] == '.') {
		i++
		digits = true
	}
	if !digits {
		return "", ""
	}
	if i < len(v) && (v[i] == 'e' || v[i] == 'E') && i+1 < len(v) && (v[i+1] == '-' || v[i+1] == '+' || (v[i+1] >= '0' && v[i+1] <= '9')) {
		j := i + 1
		if v[j] == '-' || v[j] == '+' {
			j++
		}
		if j < len(v) && v[j] >= '0' && v[j] <= '9' {
			i = j
			for i < len(v) && v[i] >= '0' && v[i] <= '9' {
				i++
			}
		}
	}
	return v[:i], v[i:]
}

// ParseNumber parses a plain number.
func ParseNumber(v string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return f, err == nil
}

// calc: a small recursive-descent evaluator over linear lengths.
type calcVal struct {
	l     Length
	isNum bool
	num   float64
}

type calcParser struct {
	s   string
	pos int
	ctx LengthContext
	ok  bool
}

func evalCalc(expr string, ctx LengthContext) (Length, bool) {
	p := &calcParser{s: expr, ctx: ctx, ok: true}
	v := p.sum()
	p.ws()
	if !p.ok || p.pos != len(p.s) {
		return Length{}, false
	}
	if v.isNum {
		if v.num == 0 {
			return Length{}, true
		}
		return Length{Pt: v.num * 0.75}, true
	}
	return v.l, true
}

func (p *calcParser) ws() {
	for p.pos < len(p.s) && (p.s[p.pos] == ' ' || p.s[p.pos] == '\t' || p.s[p.pos] == '\n') {
		p.pos++
	}
}

func (p *calcParser) sum() calcVal {
	v := p.product()
	for p.ok {
		p.ws()
		if p.pos >= len(p.s) || (p.s[p.pos] != '+' && p.s[p.pos] != '-') {
			return v
		}
		op := p.s[p.pos]
		p.pos++
		r := p.product()
		if v.isNum != r.isNum {
			p.ok = false
			return v
		}
		if op == '-' {
			r.num, r.l.Pt, r.l.Pct = -r.num, -r.l.Pt, -r.l.Pct
		}
		v.num += r.num
		v.l.Pt += r.l.Pt
		v.l.Pct += r.l.Pct
	}
	return v
}

func (p *calcParser) product() calcVal {
	v := p.atom()
	for p.ok {
		p.ws()
		if p.pos >= len(p.s) || (p.s[p.pos] != '*' && p.s[p.pos] != '/') {
			return v
		}
		op := p.s[p.pos]
		p.pos++
		r := p.atom()
		switch {
		case op == '*' && r.isNum:
			v.num *= r.num
			v.l.Pt *= r.num
			v.l.Pct *= r.num
		case op == '*' && v.isNum:
			r.l.Pt *= v.num
			r.l.Pct *= v.num
			v = r
		case op == '/' && r.isNum && r.num != 0:
			v.num /= r.num
			v.l.Pt /= r.num
			v.l.Pct /= r.num
		default:
			p.ok = false
		}
	}
	return v
}

func (p *calcParser) atom() calcVal {
	p.ws()
	if p.pos >= len(p.s) {
		p.ok = false
		return calcVal{}
	}
	if p.s[p.pos] == '(' {
		p.pos++
		v := p.sum()
		p.ws()
		if p.pos < len(p.s) && p.s[p.pos] == ')' {
			p.pos++
		} else {
			p.ok = false
		}
		return v
	}
	start := p.pos
	depth := 0
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '(' {
			depth++
		} else if c == ')' {
			if depth == 0 {
				break
			}
			depth--
		} else if depth == 0 && (c == ' ' || c == '*' || c == '/' || ((c == '+' || c == '-') && p.pos > start && !isExpSign(p.s, p.pos))) {
			break
		}
		p.pos++
	}
	tok := p.s[start:p.pos]
	if strings.HasPrefix(tok, "calc(") || tok == "" {
		if tok == "" {
			p.ok = false
			return calcVal{}
		}
	}
	if f, err := strconv.ParseFloat(tok, 64); err == nil {
		return calcVal{isNum: true, num: f}
	}
	l, ok := ParseLength(tok, p.ctx, false)
	if !ok || l.Auto {
		p.ok = false
	}
	return calcVal{l: l}
}

func isExpSign(s string, i int) bool {
	return i > 0 && (s[i-1] == 'e' || s[i-1] == 'E') && i > 1 && s[i-2] >= '0' && s[i-2] <= '9'
}

func evalMinMax(fn, args string, ctx LengthContext) (Length, bool) {
	parts := SplitCommas(args)
	var vals []Length
	for _, a := range parts {
		l, ok := evalCalc(a, ctx)
		if !ok {
			return Length{}, false
		}
		vals = append(vals, l)
	}
	if len(vals) == 0 {
		return Length{}, false
	}
	for _, v := range vals {
		if v.Pct != 0 {
			// Cannot compare without the basis; keep the first (preferred) value.
			if fn == "clamp" && len(vals) == 3 {
				return vals[1], true
			}
			return vals[0], true
		}
	}
	pick := func(a, b Length, greater bool) Length {
		if (a.Pt > b.Pt) == greater {
			return a
		}
		return b
	}
	switch fn {
	case "min":
		out := vals[0]
		for _, v := range vals[1:] {
			out = pick(out, v, false)
		}
		return out, true
	case "max":
		out := vals[0]
		for _, v := range vals[1:] {
			out = pick(out, v, true)
		}
		return out, true
	default:
		if len(vals) != 3 {
			return Length{}, false
		}
		return pick(vals[0], pick(vals[1], vals[2], false), true), true
	}
}
