// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"fmt"
	"math"
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
