// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package layout

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
)

type fragKind int

const (
	fText fragKind = iota
	fSpace
	fAtomic
	fBreak
	fOpen
	fClose
	fWbr
	fFloat // a float anchored in the line (zero width for line breaking)
	fAbs   // the static position of an absolutely positioned box
)

// inlineCtx is an inline box (span, a, code...) that frags are nested in.
type inlineCtx struct {
	box            *Box
	style          *css.Style
	parent         *inlineCtx
	shift          float64 // cumulative baseline shift (positive = down)
	relX, relY     float64 // cumulative position: relative offset
	ml, mr, bl, br float64
	pl, pr         float64
	href           string
}

// atomicBox is a laid-out atomic inline (image, inline-block, checkbox).
type atomicBox struct {
	box       *Box
	items     []Item  // positioned relative to the margin box top-left
	w, h      float64 // margin box
	baseline  float64 // distance from margin-box top to baseline
	valign    string
	valignLen float64
	image     *ImageItem
}

type frag struct {
	kind     fragKind
	text     string
	style    *css.Style
	runs     []fonts.Run
	width    float64
	ctx      *inlineCtx // innermost inline box
	decos    []Decoration
	atomic   *atomicBox
	breakOK  bool // a line break may follow this frag
	collapse bool // collapsible space (removed at line edges)
	hang     bool // preserved space that hangs at line end (pre-wrap)
	box      *Box // fFloat, fAbs
	placed   bool // fFloat: already positioned
}

type fragBuilder struct {
	l         *layouter
	avail     float64
	frags     []*frag
	lastSpace bool // previous emitted content was a collapsible space (or line start)
	capNext   bool
}

func (fb *fragBuilder) push(f *frag) { fb.frags = append(fb.frags, f) }

// buildFrags flattens the inline content of a block container into frags.
func (l *layouter) buildFrags(children []*Box, blockStyle *css.Style, avail float64, marker *Box) []*frag {
	fb := &fragBuilder{l: l, avail: avail, lastSpace: true, capNext: true}
	if marker != nil {
		fb.walk(marker.Children, nil, marker.Style)
	}
	fb.walk(children, nil, blockStyle)
	return fb.frags
}

func (fb *fragBuilder) walk(boxes []*Box, ctx *inlineCtx, parentStyle *css.Style) {
	for _, b := range boxes {
		switch b.Kind {
		case KindText:
			fb.text(b, ctx)
		case KindBreak:
			fb.push(&frag{kind: fBreak, style: b.Style, ctx: ctx})
			fb.lastSpace = true
		case KindWordBreak:
			fb.push(&frag{kind: fWbr, style: b.Style, ctx: ctx, breakOK: true})
		case KindInline:
			s := b.Style
			c := &inlineCtx{box: b, style: s, parent: ctx, href: b.Href}
			if ctx != nil {
				c.shift = ctx.shift
				c.relX, c.relY = ctx.relX, ctx.relY
				if c.href == "" {
					c.href = ctx.href
				}
			}
			if s.Position == "relative" {
				dx, dy := relativeOffset(s, fb.avail)
				c.relX += dx
				c.relY += dy
			}
			parentSize := parentStyle.FontSize
			c.shift += verticalShift(s, parentSize, fb.l.e.styleMetrics(s).lh)
			c.ml = s.Margin[css.Left].Resolve(fb.avail, 0)
			c.mr = s.Margin[css.Right].Resolve(fb.avail, 0)
			c.bl, c.br = s.BorderUsed(css.Left), s.BorderUsed(css.Right)
			c.pl, c.pr = s.Padding[css.Left].Resolve(fb.avail, 0), s.Padding[css.Right].Resolve(fb.avail, 0)
			fb.push(&frag{kind: fOpen, style: s, ctx: c, width: c.ml + c.bl + c.pl})
			if c.ml+c.bl+c.pl > 0 {
				fb.lastSpace = false
			}
			fb.walk(b.Children, c, s)
			fb.push(&frag{kind: fClose, style: s, ctx: c, width: c.mr + c.br + c.pr})
			if c.mr+c.br+c.pr > 0 {
				fb.lastSpace = false
			}
		default:
			if b.outOfFlow() {
				if b.Style.IsAbsolute() {
					fb.push(&frag{kind: fAbs, style: b.Style, ctx: ctx, box: b})
				} else {
					fb.push(&frag{kind: fFloat, style: b.Style, ctx: ctx, box: b, atomic: fb.l.layoutAtomic(b, fb.avail)})
				}
				continue
			}
			a := fb.l.layoutAtomic(b, fb.avail)
			if a == nil {
				continue
			}
			fb.push(&frag{kind: fAtomic, style: b.Style, ctx: ctx, atomic: a, width: a.w, breakOK: true})
			// a break is also allowed before an atomic inline
			if n := len(fb.frags); n >= 2 && fb.frags[n-2].kind != fOpen {
				fb.frags[n-2].breakOK = true
			}
			fb.lastSpace = false
		}
	}
}

// verticalShift returns the baseline shift (positive = down) for vertical-align.
func verticalShift(s *css.Style, parentSize, lineHeight float64) float64 {
	switch s.VerticalAlign {
	case "sub":
		return parentSize/5 + 0.75
	case "super":
		return -(parentSize/3 + 0.75)
	case "length":
		return -s.VerticalLength.Resolve(lineHeight, 0)
	}
	return 0
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r) || (r >= 0x3000 && r <= 0x303f) || (r >= 0xff00 && r <= 0xffef)
}

func applyTransform(s string, t string, capNext *bool) string {
	switch t {
	case "uppercase":
		return strings.ToUpper(s)
	case "lowercase":
		return strings.ToLower(s)
	case "capitalize":
		var b strings.Builder
		for _, r := range s {
			if *capNext && unicode.IsLetter(r) {
				b.WriteRune(unicode.ToUpper(r))
				*capNext = false
			} else {
				b.WriteRune(r)
			}
			if unicode.IsSpace(r) {
				*capNext = true
			} else if !unicode.IsLetter(r) && r != '\'' {
				*capNext = true
			}
		}
		return b.String()
	}
	return s
}

func (fb *fragBuilder) text(b *Box, ctx *inlineCtx) {
	s := b.Style
	ws := s.WhiteSpace
	collapseSpaces := ws == "normal" || ws == "nowrap" || ws == "pre-line"
	keepNewlines := ws != "normal" && ws != "nowrap"
	wrap := ws != "nowrap" && ws != "pre"
	text := b.Text
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var word strings.Builder
	flushWord := func(breakAfter bool) {
		if word.Len() == 0 {
			return
		}
		w := applyTransform(word.String(), s.TextTransform, &fb.capNext)
		fb.push(fb.l.textFrag(w, s, ctx, b.Decorations, breakAfter && wrap))
		word.Reset()
		fb.lastSpace = false
	}
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r == '\n' && keepNewlines:
			flushWord(false)
			if ws == "pre-line" {
				// drop spaces before the newline
				for n := len(fb.frags); n > 0 && fb.frags[n-1].kind == fSpace && fb.frags[n-1].collapse; n-- {
					fb.frags = fb.frags[:n-1]
				}
			}
			fb.push(&frag{kind: fBreak, style: s, ctx: ctx})
			fb.lastSpace = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\f':
			flushWord(false)
			if collapseSpaces {
				if fb.lastSpace {
					continue
				}
				fb.lastSpace = true
				sp := fb.l.textFrag(" ", s, ctx, b.Decorations, wrap)
				sp.kind = fSpace
				sp.collapse = true
				sp.width += s.WordSpacing
				fb.push(sp)
				fb.capNext = true
				continue
			}
			t := " "
			if r == '\t' {
				t = "        "
			}
			sp := fb.l.textFrag(t, s, ctx, b.Decorations, wrap)
			sp.kind = fSpace
			sp.hang = ws == "pre-wrap"
			sp.width += s.WordSpacing
			fb.push(sp)
			fb.lastSpace = false
		case r == '​':
			flushWord(false)
			fb.push(&frag{kind: fWbr, style: s, ctx: ctx, breakOK: wrap})
		case isCJK(r) && wrap:
			flushWord(false)
			word.WriteRune(r)
			flushWord(true)
		case r == '-' && wrap && word.Len() > 0 && i+1 < len(runes) && unicode.IsLetter(runes[i+1]):
			word.WriteRune(r)
			flushWord(true)
		default:
			word.WriteRune(r)
		}
	}
	flushWord(false)
}

// textFrag measures text in the style's font chain.
func (l *layouter) textFrag(text string, s *css.Style, ctx *inlineCtx, decos []Decoration, breakAfter bool) *frag {
	m := l.e.styleMetrics(s)
	runs := l.e.Fonts.Split(text, m.chain)
	w := 0.0
	for _, r := range runs {
		w += r.Face.Width(r.Text, s.FontSize)
	}
	if s.LetterSpacing != 0 {
		w += s.LetterSpacing * float64(utf8.RuneCountInString(text))
	}
	return &frag{kind: fText, text: text, style: s, runs: runs, width: w, ctx: ctx, decos: decos, breakOK: breakAfter}
}

// splitFragChars splits an unbreakable text frag into single-rune frags
// (overflow-wrap: anywhere / break-word, word-break: break-all).
func (l *layouter) splitFragChars(f *frag) []*frag {
	var out []*frag
	for _, r := range f.text {
		nf := l.textFrag(string(r), f.style, f.ctx, f.decos, true)
		out = append(out, nf)
	}
	if len(out) > 0 {
		out[len(out)-1].breakOK = f.breakOK
	}
	return out
}

func canBreakAnywhere(s *css.Style) bool {
	return s.OverflowWrap == "anywhere" || s.OverflowWrap == "break-word" || s.WordBreak == "break-all" || s.WordBreak == "break-word"
}
