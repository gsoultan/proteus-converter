// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gsoultan/gexcel/style"
	"github.com/gsoultan/proteus-converter/css"
)

// The spreadsheet style vocabulary is gexcel's: style.Font, style.Fill,
// style.Border and style.Alignment, combined into a style.Style per cell.
// gexcel deduplicates them into the workbook's style tables on save.

// defFontW is the maximum digit width in pixels of the default font
// (Calibri 11), used to convert pixel widths to column width units.
const defFontW = 7.0

type font = style.Font

// numFmtCode normalises a number format: the empty string and "General" both
// mean the general format.
func numFmtCode(code string) string {
	if code == "General" {
		return ""
	}
	return code
}

// --- CSS to spreadsheet style mapping ---

// genericFonts maps CSS generic families to fonts every spreadsheet has.
var genericFonts = map[string]string{
	"serif":         "Times New Roman",
	"sans-serif":    "Arial",
	"monospace":     "Courier New",
	"cursive":       "Comic Sans MS",
	"fantasy":       "Impact",
	"system-ui":     "Calibri",
	"ui-serif":      "Times New Roman",
	"ui-sans-serif": "Arial",
	"ui-monospace":  "Courier New",
}

// fontOf converts a computed style's text properties to a font.
func fontOf(s *css.Style) font {
	f := font{Size: roundPt(s.FontSize)}
	f.Name = fontName(s.FontFamily)
	f.Bold = s.FontWeight >= 600
	f.Italic = s.FontStyle == "italic" || s.FontStyle == "oblique"
	if s.TextDecorationLine.Underline {
		f.Underline = style.UnderlineSingle
	}
	f.Strike = s.TextDecorationLine.LineThrough
	if c := argb(s.Color); c != "" && c != "FF000000" { // black is automatic, like Excel's defaults
		f.Color = style.RGB(c)
	}
	return f
}

// fontName picks the spreadsheet font for a CSS font-family list: the first
// family that is not a browser-only system keyword, with generic families
// mapped to fonts every spreadsheet application has.
func fontName(families []string) string {
	for _, name := range families {
		lower := strings.ToLower(strings.TrimSpace(name))
		switch {
		case lower == "":
			continue
		case strings.HasPrefix(lower, "-"), lower == "blinkmacsystemfont", lower == "emoji", strings.Contains(lower, "color emoji"):
			continue // -apple-system and friends mean nothing outside a browser
		}
		if g, ok := genericFonts[lower]; ok {
			return g
		}
		return strings.TrimSpace(name)
	}
	return ""
}

func roundPt(v float64) float64 { return math.Round(v*2) / 2 }

// argb returns the colour as Excel ARGB hex, composited over white when
// translucent. Fully transparent returns "".
func argb(c css.Color) string {
	if c.A <= 0 {
		return ""
	}
	return argbOver(c, css.Color{R: 1, G: 1, B: 1, A: 1})
}

// argbOver composites c over bg and returns the opaque ARGB hex.
func argbOver(c, bg css.Color) string {
	a := clamp(c.A)
	r := clamp(c.R*a + bg.R*(1-a))
	g := clamp(c.G*a + bg.G*(1-a))
	b := clamp(c.B*a + bg.B*(1-a))
	return fmt.Sprintf("FF%02X%02X%02X", int(math.Round(r*255)), int(math.Round(g*255)), int(math.Round(b*255)))
}

func clamp(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// blend composites a stack of CSS colours (bottom first) over white and
// returns the visible fill, or an empty fill when nothing paints.
func blend(layers ...css.Color) style.Fill {
	var painted bool
	cur := css.Color{R: 1, G: 1, B: 1, A: 1}
	for _, l := range layers {
		if l.A <= 0 {
			continue
		}
		painted = true
		a := clamp(l.A)
		cur = css.Color{R: l.R*a + cur.R*(1-a), G: l.G*a + cur.G*(1-a), B: l.B*a + cur.B*(1-a), A: 1}
	}
	if !painted {
		return style.Fill{}
	}
	return style.Solid(style.RGB(argb(cur)))
}

// borderSideOf maps a CSS border side to a spreadsheet border style.
func borderSideOf(s *css.Style, side int) style.Line {
	w := s.BorderUsed(side)
	st := s.BorderStyle[side]
	if w <= 0 || st == "none" || st == "hidden" || st == "" {
		return style.Line{}
	}
	var ls style.LineStyle
	switch st {
	case "dashed":
		ls = style.LineDashed
		if w >= 2 {
			ls = style.LineMediumDashed
		}
	case "dotted":
		ls = style.LineDotted
	case "double":
		ls = style.LineDouble
	default: // solid, groove, ridge, inset, outset
		switch {
		case w < 0.75:
			ls = style.LineHair
		case w < 1.5:
			ls = style.LineThin
		case w < 2.5:
			ls = style.LineMedium
		default:
			ls = style.LineThick
		}
	}
	l := style.Line{Style: ls}
	if c := argb(s.BorderColor[side]); c != "" {
		l.Color = style.RGB(c)
	}
	return l
}

func borderOf(s *css.Style) style.Border {
	return style.Border{
		Top:    borderSideOf(s, css.Top),
		Right:  borderSideOf(s, css.Right),
		Bottom: borderSideOf(s, css.Bottom),
		Left:   borderSideOf(s, css.Left),
	}
}

// horizontalOf maps text-align to a spreadsheet alignment. The initial value
// "start" maps to "" (general) so numbers keep their natural alignment.
func horizontalOf(s *css.Style) style.HAlign {
	switch s.TextAlign {
	case "left", "-webkit-left":
		return style.HAlignLeft
	case "right", "-webkit-right", "end":
		return style.HAlignRight
	case "center", "-webkit-center":
		return style.HAlignCenter
	case "justify":
		return style.HAlignJustify
	}
	return style.HAlignGeneral
}

func verticalOf(s *css.Style) style.VAlign {
	switch s.VerticalAlign {
	case "top", "text-top":
		return style.VAlignTop
	case "bottom", "text-bottom", "baseline":
		return style.VAlignBottom
	case "middle":
		return style.VAlignCenter
	}
	return style.VAlignDefault
}

// widthChars converts a CSS pixel width to Excel's column width unit
// (characters of the default font's widest digit), following the formula in
// the OOXML spec: width = Truncate((px - 5) / maxDigitWidth * 100 + 0.5) / 100
// plus the padding term, which for the default 7 px digit is px/7.
func widthChars(px float64, digitW float64) float64 {
	if px <= 0 {
		return 0
	}
	return math.Round(px/digitW*100) / 100
}

// gradientOf maps a CSS linear-gradient() to a spreadsheet gradient fill, or
// returns nil when v is not a gradient a cell can show (repeating gradients,
// stops positioned by length). Translucent stops are composited over under,
// the colours painted below the cell. CSS angles run clockwise from "to
// top"; spreadsheet angles run clockwise from "left to right".
func gradientOf(v string, under ...css.Color) *style.Gradient {
	fn, args, ok := css.Function(strings.TrimSpace(v))
	if !ok {
		return nil
	}
	if fn = strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(fn), "-webkit-"), "-moz-"); fn != "linear-gradient" {
		return nil
	}
	parts := css.SplitCommas(args)
	if len(parts) == 0 {
		return nil
	}
	angle := 180.0
	first := strings.ToLower(strings.TrimSpace(parts[0]))
	if strings.HasPrefix(first, "to ") {
		var dx, dy float64
		for _, d := range strings.Fields(first[3:]) {
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
		if dx == 0 && dy == 0 {
			return nil
		}
		angle = math.Atan2(dx, -dy) * 180 / math.Pi
		parts = parts[1:]
	} else if a, ok := cssAngle(first); ok {
		angle = a
		parts = parts[1:]
	}
	base := css.Color{R: 1, G: 1, B: 1, A: 1}
	for _, l := range under {
		base = over(l, base)
	}
	var stops []style.GradientStop
	var set []bool
	for _, p := range parts {
		toks := css.SplitValue(p)
		if len(toks) == 0 {
			continue
		}
		c, ok := css.ParseColor(toks[0], css.Black)
		if !ok {
			if len(toks) == 1 {
				continue // an interpolation hint
			}
			return nil
		}
		col := style.RGB(argb(over(c, base)))
		if len(toks) == 1 {
			stops, set = append(stops, style.GradientStop{Color: col}), append(set, false)
			continue
		}
		for _, t := range toks[1:] {
			if !strings.HasSuffix(t, "%") {
				return nil
			}
			pct, err := strconv.ParseFloat(strings.TrimSuffix(t, "%"), 64)
			if err != nil {
				return nil
			}
			stops, set = append(stops, style.GradientStop{Position: clamp(pct / 100), Color: col}), append(set, true)
		}
	}
	n := len(stops)
	if n < 2 {
		return nil
	}
	if !set[0] {
		stops[0].Position, set[0] = 0, true
	}
	if !set[n-1] {
		stops[n-1].Position, set[n-1] = 1, true
	}
	for i := 1; i < n; i++ {
		if !set[i] { // spread unpositioned stops evenly up to the next positioned one
			j := i
			for !set[j] {
				j++
			}
			from, to := stops[i-1].Position, math.Max(stops[j].Position, stops[i-1].Position)
			for k := i; k < j; k++ {
				stops[k].Position, set[k] = from+(to-from)*float64(k-i+1)/float64(j-i+1), true
			}
		}
		stops[i].Position = math.Max(stops[i].Position, stops[i-1].Position)
	}
	deg := math.Mod(math.Round(angle)-90, 360)
	if deg < 0 {
		deg += 360
	}
	return &style.Gradient{Degree: deg, Stops: stops}
}

// cssAngle parses a CSS angle into degrees.
func cssAngle(v string) (float64, bool) {
	for _, u := range []struct {
		suffix string
		toDeg  float64
	}{{"deg", 1}, {"grad", 0.9}, {"rad", 180 / math.Pi}, {"turn", 360}} {
		if num, ok := strings.CutSuffix(v, u.suffix); ok {
			// "grad" also ends in "rad": the longer unit is tried first.
			if f, err := strconv.ParseFloat(strings.TrimSpace(num), 64); err == nil {
				return f * u.toDeg, true
			}
		}
	}
	return 0, false
}

// over composites c on the opaque colour bg.
func over(c, bg css.Color) css.Color {
	a := clamp(c.A)
	return css.Color{R: c.R*a + bg.R*(1-a), G: c.G*a + bg.G*(1-a), B: c.B*a + bg.B*(1-a), A: 1}
}
