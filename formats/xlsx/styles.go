// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"fmt"
	"math"
	"strings"

	"github.com/gsoultan/proteus-converter/css"
)

// font is a cell or run font in spreadsheet terms.
type font struct {
	name      string
	sizePt    float64
	bold      bool
	italic    bool
	underline bool
	strike    bool
	color     string // ARGB hex, "" for automatic
}

type fill struct {
	color string // ARGB hex; "" means no fill
}

type borderSide struct {
	style string // thin, medium, thick, dashed, dotted, double, hair; "" none
	color string
}

type border struct {
	left, right, top, bottom borderSide
}

type alignment struct {
	horizontal string // "", left, center, right, justify
	vertical   string // "", top, center, bottom
	wrap       bool
	indent     int
}

// xf is a cell format: indexes into the style tables plus alignment.
type xf struct {
	fontID, fillID, borderID, numFmtID int
	align                              alignment
}

// styleTable deduplicates fonts, fills, borders, number formats and cell
// formats, producing the indexes used in sheet XML.
type styleTable struct {
	fonts    []font
	fills    []fill
	borders  []border
	numFmts  []string // custom formats; id = 164 + index
	xfs      []xf
	fontIdx  map[string]int
	fillIdx  map[string]int
	bordIdx  map[string]int
	fmtIdx   map[string]int
	xfIdx    map[string]int
	defFont  font
	defFontW float64 // max digit width in px of the default font, for column widths
}

// Built-in number format ids.
const (
	fmtGeneral = 0
	fmtDate    = 14 // m/d/yyyy; replaced by a custom yyyy-mm-dd
)

func newStyleTable(def font) *styleTable {
	t := &styleTable{fontIdx: map[string]int{}, fillIdx: map[string]int{}, bordIdx: map[string]int{},
		fmtIdx: map[string]int{}, xfIdx: map[string]int{}, defFont: def, defFontW: 7}
	t.fontID(def)
	// Excel requires fill 0 = none and fill 1 = gray125.
	t.fills = append(t.fills, fill{}, fill{color: "gray125"})
	t.fillIdx[""] = 0
	t.borderID(border{})
	t.xfID(xf{})
	return t
}

func (f font) key() string {
	return fmt.Sprintf("%s|%g|%t|%t|%t|%t|%s", f.name, f.sizePt, f.bold, f.italic, f.underline, f.strike, f.color)
}

func (t *styleTable) fontID(f font) int {
	k := f.key()
	if id, ok := t.fontIdx[k]; ok {
		return id
	}
	t.fonts = append(t.fonts, f)
	t.fontIdx[k] = len(t.fonts) - 1
	return len(t.fonts) - 1
}

func (t *styleTable) fillID(f fill) int {
	if id, ok := t.fillIdx[f.color]; ok {
		return id
	}
	t.fills = append(t.fills, f)
	t.fillIdx[f.color] = len(t.fills) - 1
	return len(t.fills) - 1
}

func (b border) key() string {
	s := func(x borderSide) string { return x.style + "/" + x.color }
	return s(b.left) + "|" + s(b.right) + "|" + s(b.top) + "|" + s(b.bottom)
}

func (t *styleTable) borderID(b border) int {
	k := b.key()
	if id, ok := t.bordIdx[k]; ok {
		return id
	}
	t.borders = append(t.borders, b)
	t.bordIdx[k] = len(t.borders) - 1
	return len(t.borders) - 1
}

// numFmtID returns the id of a number format code, registering custom ones.
func (t *styleTable) numFmtID(code string) int {
	switch code {
	case "", "General":
		return fmtGeneral
	case "0":
		return 1
	case "0.00":
		return 2
	case "#,##0":
		return 3
	case "#,##0.00":
		return 4
	case "0%":
		return 9
	case "0.00%":
		return 10
	}
	if id, ok := t.fmtIdx[code]; ok {
		return id
	}
	t.numFmts = append(t.numFmts, code)
	id := 164 + len(t.numFmts) - 1
	t.fmtIdx[code] = id
	return id
}

func (x xf) key() string {
	a := x.align
	return fmt.Sprintf("%d|%d|%d|%d|%s|%s|%t|%d", x.fontID, x.fillID, x.borderID, x.numFmtID, a.horizontal, a.vertical, a.wrap, a.indent)
}

func (t *styleTable) xfID(x xf) int {
	k := x.key()
	if id, ok := t.xfIdx[k]; ok {
		return id
	}
	t.xfs = append(t.xfs, x)
	t.xfIdx[k] = len(t.xfs) - 1
	return len(t.xfs) - 1
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
	f := font{sizePt: roundPt(s.FontSize)}
	f.name = fontName(s.FontFamily)
	f.bold = s.FontWeight >= 600
	f.italic = s.FontStyle == "italic" || s.FontStyle == "oblique"
	f.underline = s.TextDecorationLine.Underline
	f.strike = s.TextDecorationLine.LineThrough
	f.color = argb(s.Color)
	if f.color == "FF000000" {
		f.color = "" // automatic, like Excel's defaults
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
func blend(layers ...css.Color) fill {
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
		return fill{}
	}
	return fill{color: argb(cur)}
}

// borderSideOf maps a CSS border side to a spreadsheet border style.
func borderSideOf(s *css.Style, side int) borderSide {
	w := s.BorderUsed(side)
	st := s.BorderStyle[side]
	if w <= 0 || st == "none" || st == "hidden" || st == "" {
		return borderSide{}
	}
	var style string
	switch st {
	case "dashed":
		style = "dashed"
		if w >= 2 {
			style = "mediumDashed"
		}
	case "dotted":
		style = "dotted"
	case "double":
		style = "double"
	default: // solid, groove, ridge, inset, outset
		switch {
		case w < 0.75:
			style = "hair"
		case w < 1.5:
			style = "thin"
		case w < 2.5:
			style = "medium"
		default:
			style = "thick"
		}
	}
	return borderSide{style: style, color: argb(s.BorderColor[side])}
}

func borderOf(s *css.Style) border {
	return border{
		top:    borderSideOf(s, css.Top),
		right:  borderSideOf(s, css.Right),
		bottom: borderSideOf(s, css.Bottom),
		left:   borderSideOf(s, css.Left),
	}
}

func (b border) any() bool {
	return b.left.style != "" || b.right.style != "" || b.top.style != "" || b.bottom.style != ""
}

// horizontalOf maps text-align to a spreadsheet alignment. The initial value
// "start" maps to "" (general) so numbers keep their natural alignment.
func horizontalOf(s *css.Style) string {
	switch s.TextAlign {
	case "left", "-webkit-left":
		return "left"
	case "right", "-webkit-right", "end":
		return "right"
	case "center", "-webkit-center":
		return "center"
	case "justify":
		return "justify"
	}
	return ""
}

func verticalOf(s *css.Style) string {
	switch s.VerticalAlign {
	case "top", "text-top":
		return "top"
	case "bottom", "text-bottom", "baseline":
		return "bottom"
	case "middle":
		return "center"
	}
	return ""
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
