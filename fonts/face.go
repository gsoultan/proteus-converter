// Package fonts resolves CSS font families to concrete faces, measures text
// and exposes the faces to the PDF painter. TrueType faces come from the
// caller, @font-face rules or the OS font directories; the PDF standard 14
// fonts are the last resort.
package fonts

import (
	"math"
	"os"
	"sync"

	"github.com/gsoultan/gpdf/font/truetype"
)

// Face is one concrete font face.
type Face struct {
	Family string
	Weight int
	Italic bool

	// Standard is the PDF standard-14 base font name when the face is not embedded.
	Standard string

	path   string
	data   []byte
	offset uint32 // font offset inside a collection
	index  int    // index inside a collection
	info   *sfntInfo

	once sync.Once
	ttf  *truetype.Font
	err  error
}

// Metrics are em-relative vertical metrics (fractions of the font size).
type Metrics struct {
	Ascent, Descent, LineGap float64 // descent is positive
	UnderlinePos             float64 // distance below the baseline (positive = below)
	UnderlineThickness       float64
	StrikePos                float64 // distance above the baseline
	XHeight                  float64
}

// IsStandard reports whether the face is a PDF standard-14 font.
func (f *Face) IsStandard() bool { return f.Standard != "" }

// TrueType loads (once) and returns the parsed font, or nil for standard faces.
func (f *Face) TrueType() (*truetype.Font, error) {
	if f.IsStandard() {
		return nil, nil
	}
	f.once.Do(func() {
		data := f.data
		if data == nil {
			data, f.err = os.ReadFile(f.path)
			if f.err != nil {
				return
			}
		}
		if f.offset != 0 || f.index > 0 {
			var all []*truetype.Font
			all, f.err = truetype.ParseCollection(data)
			if f.err == nil {
				if f.index < len(all) {
					f.ttf = all[f.index]
				} else {
					f.err = errNotSfnt
				}
			}
			return
		}
		f.ttf, f.err = truetype.Parse(data)
	})
	return f.ttf, f.err
}

// PDFName is the font name used when drawing with gpdf (PostScript name or standard-14 name).
func (f *Face) PDFName() string {
	if f.IsStandard() {
		return f.Standard
	}
	if t, err := f.TrueType(); err == nil && t != nil {
		return t.PostScriptName()
	}
	return ""
}

// Has reports whether the face maps r to a glyph.
func (f *Face) Has(r rune) bool {
	if f.IsStandard() {
		return standardHas(f.Standard, r)
	}
	t, err := f.TrueType()
	if err != nil || t == nil {
		return false
	}
	return t.Contains(r)
}

// Width returns the advance width of s at size points, matching how gpdf
// positions the glyphs: ligatures and pair kerning applied (gpdf writes the
// kerning as TJ adjustments, and Chromium kerns by default too).
func (f *Face) Width(s string, size float64) float64 {
	if s == "" {
		return 0
	}
	if f.IsStandard() {
		return standardWidth(f.Standard, s, size)
	}
	t, err := f.TrueType()
	if err != nil || t == nil {
		return 0
	}
	return t.TextWidth(s, size)
}

// Metrics returns em-relative vertical metrics.
func (f *Face) Metrics() Metrics {
	if f.IsStandard() {
		return standardMetrics(f.Standard)
	}
	in := f.info
	upm := float64(in.unitsPerEm)
	m := Metrics{
		Ascent:             float64(in.ascent) / upm,
		Descent:            float64(in.descent) / upm,
		LineGap:            float64(max(in.lineGap, 0)) / upm,
		UnderlinePos:       -float64(in.underlinePos) / upm,
		UnderlineThickness: float64(in.underlineThk) / upm,
		StrikePos:          float64(in.strikePos) / upm,
		XHeight:            float64(in.xHeight) / upm,
	}
	if m.UnderlineThickness <= 0 {
		m.UnderlineThickness = 0.05
	}
	if m.UnderlinePos == 0 {
		m.UnderlinePos = 0.1
	}
	if m.StrikePos == 0 {
		m.StrikePos = 0.3
	}
	if m.XHeight == 0 {
		m.XHeight = 0.5
	}
	return m
}

// LineMetrics returns ascent, descent and normal line height in points for
// size, rounded to CSS pixels the way Blink does.
func (f *Face) LineMetrics(size float64) (ascent, descent, normal float64) {
	m := f.Metrics()
	px := size / 0.75
	a := math.Round(m.Ascent * px)
	d := math.Round(m.Descent * px)
	g := math.Round(m.LineGap * px)
	return a * 0.75, d * 0.75, (a + d + g) * 0.75
}
