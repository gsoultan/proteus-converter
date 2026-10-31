package css

import (
	"math"
	"strconv"
	"strings"
)

// computeCtx carries what property application needs besides the styles.
type computeCtx struct {
	lc LengthContext
}

type propDef struct {
	inherited bool
	apply     func(s, parent *Style, v string, c *computeCtx) bool
	copy      func(dst, src *Style)
}

var props map[string]*propDef

func lengthProp(get func(*Style) *Length, allowAuto, allowNeg bool) (func(s, p *Style, v string, c *computeCtx) bool, func(d, s *Style)) {
	return func(s, _ *Style, v string, c *computeCtx) bool {
			l, ok := ParseLength(v, c.lc, false)
			if !ok {
				if strings.EqualFold(v, "none") && allowAuto {
					*get(s) = Auto
					return true
				}
				// legacy keywords for widths
				switch strings.ToLower(v) {
				case "fit-content", "max-content", "min-content", "-webkit-fill-available", "stretch":
					if allowAuto {
						*get(s) = Auto
						return true
					}
				}
				return false
			}
			if l.Auto && !allowAuto {
				return false
			}
			if !allowNeg && (l.Pt < 0 || l.Pct < 0) && !l.Auto {
				return false
			}
			*get(s) = l
			return true
		}, func(d, s *Style) {
			*get(d) = *get(s)
		}
}

func keywordProp(get func(*Style) *string, allowed ...string) (func(s, p *Style, v string, c *computeCtx) bool, func(d, s *Style)) {
	return func(s, _ *Style, v string, _ *computeCtx) bool {
			v = strings.ToLower(strings.TrimSpace(v))
			if len(allowed) > 0 {
				ok := false
				for _, a := range allowed {
					if a == v {
						ok = true
						break
					}
				}
				if !ok {
					return false
				}
			}
			*get(s) = v
			return true
		}, func(d, s *Style) {
			*get(d) = *get(s)
		}
}

func def(name string, inherited bool, apply func(s, p *Style, v string, c *computeCtx) bool, cp func(d, s *Style)) {
	props[name] = &propDef{inherited: inherited, apply: apply, copy: cp}
}

func parseBorderWidth(v string, c *computeCtx) (float64, bool) {
	switch strings.ToLower(v) {
	case "thin":
		return 0.75, true
	case "medium":
		return 2.25, true
	case "thick":
		return 3.75, true
	}
	l, ok := ParseLength(v, c.lc, false)
	if !ok || l.Auto || l.Pct != 0 || l.Pt < 0 {
		return 0, false
	}
	// Browsers snap border widths to whole device pixels (at least 1px when non-zero).
	px := l.Pt / 0.75
	if px > 0 && px < 1 {
		px = 1
	} else {
		px = math.Floor(px + 1e-6)
	}
	return px * 0.75, true
}

var borderStyles = map[string]bool{"none": true, "hidden": true, "dotted": true, "dashed": true, "solid": true,
	"double": true, "groove": true, "ridge": true, "inset": true, "outset": true}

func init() {
	props = map[string]*propDef{}
	sides := []string{"top", "right", "bottom", "left"}

	displayApply, cp := keywordProp(func(s *Style) *string { return &s.Display })
	def("display", false, func(s, p *Style, v string, c *computeCtx) bool {
		v = strings.ToLower(strings.TrimSpace(v))
		switch v {
		case "block flow", "flow":
			v = "block"
		case "inline flow":
			v = "inline"
		case "inline flow-root":
			v = "inline-block"
		case "block flex":
			v = "flex"
		case "inline flex":
			v = "inline-flex"
		case "list-item block", "block list-item", "list-item flow":
			v = "list-item"
		}
		return displayApply(s, p, v, c)
	}, cp)
	a, cp := keywordProp(func(s *Style) *string { return &s.Position }, "static", "relative", "absolute", "fixed", "sticky")
	def("position", false, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.Float }, "none", "left", "right", "inline-start", "inline-end")
	def("float", false, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.Clear }, "none", "left", "right", "both", "inline-start", "inline-end")
	def("clear", false, a, cp)

	def("color", true, func(s, p *Style, v string, _ *computeCtx) bool {
		col, ok := ParseColor(v, p.Color)
		if ok {
			s.Color = col
		}
		return ok
	}, func(d, s *Style) { d.Color = s.Color })
	def("background-color", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		col, ok := ParseColor(v, s.Color)
		if ok {
			s.BackgroundColor = col
		}
		return ok
	}, func(d, s *Style) { d.BackgroundColor = s.BackgroundColor })
	def("background-image", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		s.BackgroundImage, s.BackgroundGradient = "", ""
		if strings.EqualFold(v, "none") {
			return true
		}
		// only the first layer is painted
		layers := SplitCommas(strings.TrimSpace(v))
		first := strings.TrimSpace(layers[0])
		if fn, _, ok := Function(first); ok && strings.HasSuffix(fn, "linear-gradient") {
			s.BackgroundGradient = first
			return true
		}
		if u, ok := URLValue(first); ok {
			s.BackgroundImage = u
		}
		return true
	}, func(d, s *Style) { d.BackgroundImage, d.BackgroundGradient = s.BackgroundImage, s.BackgroundGradient })
	def("opacity", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		v = strings.TrimSpace(v)
		var f float64
		var ok bool
		if strings.HasSuffix(v, "%") {
			f, ok = ParseNumber(v[:len(v)-1])
			f /= 100
		} else {
			f, ok = ParseNumber(v)
		}
		if ok {
			s.Opacity = clamp01(f)
		}
		return ok
	}, func(d, s *Style) { d.Opacity = s.Opacity })
	a, cp = keywordProp(func(s *Style) *string { return &s.Visibility }, "visible", "hidden", "collapse")
	def("visibility", true, a, cp)

	def("font-family", true, func(s, _ *Style, v string, _ *computeCtx) bool {
		var fams []string
		for _, f := range SplitCommas(v) {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if f[0] == '"' || f[0] == '\'' {
				fams = append(fams, Unquote(f))
			} else {
				fams = append(fams, strings.Join(strings.Fields(f), " "))
			}
		}
		if len(fams) == 0 {
			return false
		}
		s.FontFamily = fams
		return true
	}, func(d, s *Style) { d.FontFamily = s.FontFamily })
	def("font-size", true, applyFontSize, func(d, s *Style) { d.FontSize, d.keywordSize = s.FontSize, s.keywordSize })
	def("font-weight", true, func(s, p *Style, v string, _ *computeCtx) bool {
		switch strings.ToLower(v) {
		case "normal":
			s.FontWeight = 400
		case "bold":
			s.FontWeight = 700
		case "bolder":
			s.FontWeight = bolder(p.FontWeight)
		case "lighter":
			s.FontWeight = lighter(p.FontWeight)
		default:
			f, ok := ParseNumber(v)
			if !ok || f < 1 || f > 1000 {
				return false
			}
			s.FontWeight = int(f)
		}
		return true
	}, func(d, s *Style) { d.FontWeight = s.FontWeight })
	def("font-style", true, func(s, _ *Style, v string, _ *computeCtx) bool {
		v = strings.ToLower(strings.TrimSpace(v))
		if strings.HasPrefix(v, "oblique") {
			v = "oblique"
		}
		switch v {
		case "normal", "italic", "oblique":
			s.FontStyle = v
			return true
		}
		return false
	}, func(d, s *Style) { d.FontStyle = s.FontStyle })
	a, cp = keywordProp(func(s *Style) *string { return &s.FontVariant })
	def("font-variant", true, a, cp)
	def("line-height", true, func(s, _ *Style, v string, c *computeCtx) bool {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "normal" {
			s.LineHeight = LineHeight{Normal: true}
			return true
		}
		if f, ok := ParseNumber(v); ok {
			if f < 0 {
				return false
			}
			s.LineHeight = LineHeight{Factor: f}
			if f == 0 {
				s.LineHeight = LineHeight{Pt: 0}
			}
			return true
		}
		l, ok := ParseLength(v, c.lc, false)
		if !ok || l.Auto {
			return false
		}
		s.LineHeight = LineHeight{Pt: l.Pt + l.Pct/100*s.FontSize}
		return true
	}, func(d, s *Style) { d.LineHeight = s.LineHeight })
	def("text-align", true, func(s, p *Style, v string, _ *computeCtx) bool {
		v = strings.ToLower(strings.TrimSpace(v))
		switch v {
		case "-webkit-center", "-moz-center":
			v = "center"
		case "-webkit-left":
			v = "left"
		case "-webkit-right":
			v = "right"
		case "match-parent":
			v = p.TextAlign
		}
		switch v {
		case "left", "right", "center", "justify", "start", "end":
			s.TextAlign = v
			return true
		}
		return false
	}, func(d, s *Style) { d.TextAlign = s.TextAlign })
	a, cp = keywordProp(func(s *Style) *string { return &s.TextAlignLast }, "auto", "left", "right", "center", "justify", "start", "end")
	def("text-align-last", true, a, cp)
	a, cp = lengthProp(func(s *Style) *Length { return &s.TextIndent }, false, true)
	def("text-indent", true, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.TextTransform }, "none", "uppercase", "lowercase", "capitalize")
	def("text-transform", true, a, cp)
	def("white-space", true, func(s, _ *Style, v string, _ *computeCtx) bool {
		v = strings.ToLower(strings.TrimSpace(v))
		switch v {
		case "normal", "pre", "nowrap", "pre-wrap", "pre-line", "break-spaces":
			s.WhiteSpace = v
			return true
		case "-pre-wrap", "-o-pre-wrap", "-moz-pre-wrap":
			s.WhiteSpace = "pre-wrap"
			return true
		}
		return false
	}, func(d, s *Style) { d.WhiteSpace = s.WhiteSpace })
	def("letter-spacing", true, spacingProp(func(s *Style) *float64 { return &s.LetterSpacing }), func(d, s *Style) { d.LetterSpacing = s.LetterSpacing })
	def("word-spacing", true, spacingProp(func(s *Style) *float64 { return &s.WordSpacing }), func(d, s *Style) { d.WordSpacing = s.WordSpacing })
	def("vertical-align", false, func(s, _ *Style, v string, c *computeCtx) bool {
		v = strings.ToLower(strings.TrimSpace(v))
		switch v {
		case "baseline", "sub", "super", "top", "text-top", "middle", "bottom", "text-bottom":
			s.VerticalAlign = v
			return true
		}
		l, ok := ParseLength(v, c.lc, false)
		if !ok || l.Auto {
			return false
		}
		s.VerticalAlign = "length"
		s.VerticalLength = l
		return true
	}, func(d, s *Style) { d.VerticalAlign, d.VerticalLength = s.VerticalAlign, s.VerticalLength })
	a, cp = keywordProp(func(s *Style) *string { return &s.OverflowWrap }, "normal", "break-word", "anywhere")
	def("overflow-wrap", true, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.WordBreak }, "normal", "break-all", "keep-all", "break-word")
	def("word-break", true, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.Hyphens }, "none", "manual", "auto")
	def("hyphens", true, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.Direction }, "ltr", "rtl")
	def("direction", true, a, cp)

	def("text-decoration-line", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		d := Decoration{}
		for _, t := range strings.Fields(strings.ToLower(v)) {
			switch t {
			case "none":
			case "underline":
				d.Underline = true
			case "overline":
				d.Overline = true
			case "line-through":
				d.LineThrough = true
			case "blink":
			default:
				return false
			}
		}
		s.TextDecorationLine = d
		return true
	}, func(d, s *Style) { d.TextDecorationLine = s.TextDecorationLine })
	def("text-decoration-color", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		if strings.EqualFold(strings.TrimSpace(v), "currentcolor") {
			s.TextDecorationColor = nil
			return true
		}
		col, ok := ParseColor(v, s.Color)
		if ok {
			s.TextDecorationColor = &col
		}
		return ok
	}, func(d, s *Style) { d.TextDecorationColor = s.TextDecorationColor })
	a, cp = keywordProp(func(s *Style) *string { return &s.TextDecorationStyle }, "solid", "double", "dotted", "dashed", "wavy")
	def("text-decoration-style", false, a, cp)
	def("text-decoration-thickness", false, func(*Style, *Style, string, *computeCtx) bool { return true }, func(*Style, *Style) {})
	def("text-underline-offset", false, func(*Style, *Style, string, *computeCtx) bool { return true }, func(*Style, *Style) {})

	for i, side := range sides {
		i := i
		a, cp = lengthProp(func(s *Style) *Length { return &s.Margin[i] }, true, true)
		def("margin-"+side, false, a, cp)
		a, cp = lengthProp(func(s *Style) *Length { return &s.Padding[i] }, false, false)
		def("padding-"+side, false, a, cp)
		def("border-"+side+"-width", false, func(s, _ *Style, v string, c *computeCtx) bool {
			w, ok := parseBorderWidth(v, c)
			if ok {
				s.BorderWidth[i] = w
			}
			return ok
		}, func(d, s *Style) { d.BorderWidth[i] = s.BorderWidth[i] })
		def("border-"+side+"-style", false, func(s, _ *Style, v string, _ *computeCtx) bool {
			v = strings.ToLower(strings.TrimSpace(v))
			if !borderStyles[v] {
				return false
			}
			s.BorderStyle[i] = v
			return true
		}, func(d, s *Style) { d.BorderStyle[i] = s.BorderStyle[i] })
		def("border-"+side+"-color", false, func(s, _ *Style, v string, _ *computeCtx) bool {
			col, ok := ParseColor(v, s.Color)
			if ok {
				s.BorderColor[i] = col
			}
			return ok
		}, func(d, s *Style) { d.BorderColor[i] = s.BorderColor[i] })
	}
	for i, corner := range []string{"top-left", "top-right", "bottom-right", "bottom-left"} {
		i := i
		def("border-"+corner+"-radius", false, func(s, _ *Style, v string, c *computeCtx) bool {
			toks := SplitValue(v)
			if len(toks) == 0 {
				return false
			}
			l, ok := ParseLength(toks[0], c.lc, false)
			if ok && !l.Auto {
				s.BorderRadius[i] = l
			}
			return ok
		}, func(d, s *Style) { d.BorderRadius[i] = s.BorderRadius[i] })
	}

	a, cp = lengthProp(func(s *Style) *Length { return &s.Width }, true, false)
	def("width", false, a, cp)
	a, cp = lengthProp(func(s *Style) *Length { return &s.Height }, true, false)
	def("height", false, a, cp)
	minWApply, minWCopy := lengthProp(func(s *Style) *Length { return &s.MinWidth }, true, false)
	def("min-width", false, func(s, p *Style, v string, c *computeCtx) bool {
		ok := minWApply(s, p, v, c)
		if ok && s.MinWidth.Auto {
			s.MinWidth = Length{}
		}
		return ok
	}, minWCopy)
	a2, cp2 := lengthProp(func(s *Style) *Length { return &s.MinHeight }, true, false)
	def("min-height", false, func(s, p *Style, v string, c *computeCtx) bool {
		ok := a2(s, p, v, c)
		if ok && s.MinHeight.Auto {
			s.MinHeight = Length{}
		}
		return ok
	}, cp2)
	a, cp = lengthProp(func(s *Style) *Length { return &s.MaxWidth }, true, false)
	def("max-width", false, a, cp)
	a, cp = lengthProp(func(s *Style) *Length { return &s.MaxHeight }, true, false)
	def("max-height", false, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.BoxSizing }, "content-box", "border-box")
	def("box-sizing", false, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.Overflow })
	def("overflow-x", false, a, cp)
	def("overflow-y", false, a, cp)

	def("list-style-type", true, func(s, _ *Style, v string, _ *computeCtx) bool {
		v = strings.TrimSpace(v)
		if len(v) > 0 && (v[0] == '"' || v[0] == '\'') {
			s.ListStyleType = "string:" + Unquote(v)
			return true
		}
		s.ListStyleType = strings.ToLower(v)
		return true
	}, func(d, s *Style) { d.ListStyleType = s.ListStyleType })
	a, cp = keywordProp(func(s *Style) *string { return &s.ListStylePosition }, "inside", "outside")
	def("list-style-position", true, a, cp)
	def("list-style-image", true, func(s, _ *Style, v string, _ *computeCtx) bool {
		u, _ := URLValue(strings.TrimSpace(v))
		s.ListStyleImage = u
		return true
	}, func(d, s *Style) { d.ListStyleImage = s.ListStyleImage })

	a, cp = keywordProp(func(s *Style) *string { return &s.BreakBefore })
	def("break-before", false, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.BreakAfter })
	def("break-after", false, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.BreakInside })
	def("break-inside", false, a, cp)
	def("orphans", true, intProp(func(s *Style) *int { return &s.Orphans }), func(d, s *Style) { d.Orphans = s.Orphans })
	def("widows", true, intProp(func(s *Style) *int { return &s.Widows }), func(d, s *Style) { d.Widows = s.Widows })

	a, cp = keywordProp(func(s *Style) *string { return &s.BorderCollapse }, "separate", "collapse")
	def("border-collapse", true, a, cp)
	def("border-spacing", true, func(s, _ *Style, v string, c *computeCtx) bool {
		toks := strings.Fields(v)
		if len(toks) == 0 || len(toks) > 2 {
			return false
		}
		h, ok := ParseLength(toks[0], c.lc, false)
		if !ok || h.Auto {
			return false
		}
		vv := h
		if len(toks) == 2 {
			if vv, ok = ParseLength(toks[1], c.lc, false); !ok || vv.Auto {
				return false
			}
		}
		s.BorderSpacingH, s.BorderSpacingV = h.Pt, vv.Pt
		return true
	}, func(d, s *Style) { d.BorderSpacingH, d.BorderSpacingV = s.BorderSpacingH, s.BorderSpacingV })
	a, cp = keywordProp(func(s *Style) *string { return &s.TableLayout }, "auto", "fixed")
	def("table-layout", false, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.CaptionSide }, "top", "bottom")
	def("caption-side", true, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.EmptyCells }, "show", "hide")
	def("empty-cells", true, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.PrintColorAdjust }, "economy", "exact")
	def("print-color-adjust", true, a, cp)

	def("content", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		v = strings.TrimSpace(v)
		switch strings.ToLower(v) {
		case "none", "normal":
			s.ContentNone = true
			s.Content = ""
			return true
		}
		s.ContentNone = false
		s.Content = v // resolved by the box builder (strings, attr(), counters)
		return true
	}, func(d, s *Style) { d.Content, d.ContentNone = s.Content, s.ContentNone })
	a, cp = keywordProp(func(s *Style) *string { return &s.CounterReset })
	def("counter-reset", false, func(s, _ *Style, v string, _ *computeCtx) bool { s.CounterReset = v; return true }, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.CounterIncrement })
	def("counter-increment", false, func(s, _ *Style, v string, _ *computeCtx) bool { s.CounterIncrement = v; return true }, cp)
	_ = a
	registerLayoutProps()
}

func spacingProp(get func(*Style) *float64) func(s, p *Style, v string, c *computeCtx) bool {
	return func(s, _ *Style, v string, c *computeCtx) bool {
		if strings.EqualFold(strings.TrimSpace(v), "normal") {
			*get(s) = 0
			return true
		}
		l, ok := ParseLength(v, c.lc, false)
		if !ok || l.Auto {
			return false
		}
		*get(s) = l.Pt + l.Pct/100*s.FontSize
		return true
	}
}

func intProp(get func(*Style) *int) func(s, p *Style, v string, c *computeCtx) bool {
	return func(s, _ *Style, v string, _ *computeCtx) bool {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || n < 1 {
			return false
		}
		*get(s) = n
		return true
	}
}

var fontSizeKeywords = map[string]float64{
	"xx-small": 9, "x-small": 10, "small": 13, "medium": 16, "large": 18,
	"x-large": 24, "xx-large": 32, "xxx-large": 48,
}

// isMonospaceOnly reports whether the family list is just the generic
// 'monospace', which Blink renders at the fixed default size (13px).
func isMonospaceOnly(f []string) bool {
	return len(f) == 1 && strings.EqualFold(f[0], "monospace")
}

func applyFontSize(s, p *Style, v string, c *computeCtx) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	if px, ok := fontSizeKeywords[v]; ok {
		s.keywordSize = px * 0.75
		s.FontSize = s.keywordSize
		return true
	}
	switch v {
	case "smaller":
		s.FontSize = p.FontSize / 1.2
		s.keywordSize = 0
		return true
	case "larger":
		s.FontSize = p.FontSize * 1.2
		s.keywordSize = 0
		return true
	}
	// em and % are relative to the parent's font size.
	ctx := c.lc
	ctx.FontSize = p.FontSize
	l, ok := ParseLength(v, ctx, false)
	if !ok || l.Auto || l.Pt < 0 || l.Pct < 0 {
		return false
	}
	s.FontSize = l.Pt + l.Pct/100*p.FontSize
	s.keywordSize = 0
	return true
}

// bolder/lighter follow the CSS Fonts relative-weight table.
func bolder(w int) int {
	switch {
	case w < 350:
		return 400
	case w < 550:
		return 700
	case w < 900:
		return 900
	}
	return w
}

func lighter(w int) int {
	switch {
	case w < 100:
		return w
	case w < 550:
		return 100
	case w < 750:
		return 400
	}
	return 700
}
