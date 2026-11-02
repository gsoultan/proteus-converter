// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package fonts

import (
	"strings"

	"github.com/gsoultan/gpdf/font"
)

// standard14 maps a generic class and style to a standard-14 base font.
func standardFace(class string, bold, italic bool) string {
	switch class {
	case "monospace":
		switch {
		case bold && italic:
			return "Courier-BoldOblique"
		case bold:
			return "Courier-Bold"
		case italic:
			return "Courier-Oblique"
		}
		return "Courier"
	case "sans-serif":
		switch {
		case bold && italic:
			return "Helvetica-BoldOblique"
		case bold:
			return "Helvetica-Bold"
		case italic:
			return "Helvetica-Oblique"
		}
		return "Helvetica"
	}
	switch {
	case bold && italic:
		return "Times-BoldItalic"
	case bold:
		return "Times-Bold"
	case italic:
		return "Times-Italic"
	}
	return "Times-Roman"
}

// winAnsiExtra lists the non-Latin-1 runes in WinAnsiEncoding.
var winAnsiExtra = map[rune]bool{
	'€': true, '‚': true, 'ƒ': true, '„': true, '…': true, '†': true, '‡': true, 'ˆ': true, '‰': true,
	'Š': true, '‹': true, 'Œ': true, 'Ž': true, '‘': true, '’': true, '“': true, '”': true, '•': true,
	'–': true, '—': true, '˜': true, '™': true, 'š': true, '›': true, 'œ': true, 'ž': true, 'Ÿ': true,
}

func standardHas(name string, r rune) bool {
	if strings.HasPrefix(name, "Symbol") || strings.HasPrefix(name, "ZapfDingbats") {
		return false
	}
	return (r >= 0x20 && r <= 0x7e) || (r >= 0xa0 && r <= 0xff) || winAnsiExtra[r]
}

func standardWidth(name, s string, size float64) float64 {
	units := 0
	for _, r := range s {
		switch {
		case strings.HasPrefix(name, "Courier"):
			units += 600
		case strings.HasPrefix(name, "Helvetica"):
			w := font.HelveticaWidths[r]
			if w == 0 {
				w = 556
			}
			if strings.Contains(name, "Bold") {
				w = w * 105 / 100
			}
			units += w
		default:
			// Times: no metrics in gpdf; approximate from Helvetica.
			w := font.HelveticaWidths[r]
			if w == 0 {
				w = 500
			}
			units += w * 90 / 100
		}
	}
	return float64(units) * size / 1000
}

func standardMetrics(name string) Metrics {
	switch {
	case strings.HasPrefix(name, "Courier"):
		return Metrics{Ascent: 0.833, Descent: 0.300, UnderlinePos: 0.1, UnderlineThickness: 0.05, StrikePos: 0.3, XHeight: 0.43}
	case strings.HasPrefix(name, "Helvetica"):
		return Metrics{Ascent: 0.905, Descent: 0.212, LineGap: 0.033, UnderlinePos: 0.106, UnderlineThickness: 0.073, StrikePos: 0.259, XHeight: 0.519}
	}
	return Metrics{Ascent: 0.891, Descent: 0.216, LineGap: 0.042, UnderlinePos: 0.109, UnderlineThickness: 0.049, StrikePos: 0.259, XHeight: 0.448}
}
