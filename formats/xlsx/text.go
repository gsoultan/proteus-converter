// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/css"
	"golang.org/x/net/html"
)

// content is the inline content of one cell or block: its text with runs,
// the first hyperlink and the images it holds.
type content struct {
	text      string
	runs      []textRun
	mixed     bool // runs use more than one font
	font      font // the single font of the text when not mixed (may differ from the cell's)
	link      string
	linkTitle string
	images    []imageRef
}

// imageRef is an <img> met while collecting content.
type imageRef struct {
	src, alt string
	style    *css.Style
}

// segment is a piece of raw text before white-space processing.
type segment struct {
	text  string
	font  font
	brk   bool // forced line break (<br> or block boundary)
	pre   bool // keep spaces (white-space: pre*)
	lines bool // keep newlines (pre, pre-wrap, pre-line)
	xform string
}

// inlineContent collects the text of n in document order, applying CSS
// white-space collapsing, text-transform and font changes per run.
func (e *extractor) inlineContent(n *html.Node, cellStyle *css.Style) content {
	var segs []segment
	var out content
	base := fontOf(cellStyle)
	var walk func(n *html.Node, s *css.Style, deco css.Decoration)
	walk = func(n *html.Node, parent *css.Style, deco css.Decoration) {
		switch n.Type {
		case html.TextNode:
			segs = append(segs, segment{text: n.Data, font: runFont(parent, deco), pre: keepsSpaces(parent.WhiteSpace),
				lines: keepsLines(parent.WhiteSpace), xform: parent.TextTransform})
			return
		case html.ElementNode:
		default:
			return
		}
		if e.hidden(n) {
			return
		}
		s := e.style(n)
		d := deco
		d.Underline = d.Underline || s.TextDecorationLine.Underline
		d.LineThrough = d.LineThrough || s.TextDecorationLine.LineThrough
		switch n.Data {
		case "script", "style", "template", "noscript", "head", "title":
			return
		case "br":
			segs = append(segs, segment{brk: true})
			return
		case "img":
			if src := strings.TrimSpace(attr(n, "src")); src != "" {
				out.images = append(out.images, imageRef{src: src, alt: attr(n, "alt"), style: s})
			} else if alt := attr(n, "alt"); alt != "" {
				segs = append(segs, segment{text: alt, font: runFont(parent, deco)})
			}
			return
		case "input":
			switch strings.ToLower(attr(n, "type")) {
			case "checkbox", "radio":
				mark := "☐"
				if attr(n, "checked") != "" || hasAttr(n, "checked") {
					mark = "☑"
				}
				segs = append(segs, segment{text: mark, font: runFont(parent, deco)})
			default:
				if v := attr(n, "value"); v != "" {
					segs = append(segs, segment{text: v, font: runFont(parent, deco)})
				}
			}
			return
		case "a":
			if href := strings.TrimSpace(attr(n, "href")); href != "" && out.link == "" && !strings.HasPrefix(href, "#") {
				out.link = core.ResolveURL(e.styled.BaseURL, href)
				out.linkTitle = attr(n, "title")
			}
		}
		block := (s.IsBlockLevel() || s.Display == "list-item" || s.Display == "table" || s.Display == "table-row") && s.Display != "table-cell"
		if block && len(segs) > 0 {
			segs = append(segs, segment{brk: true})
		}
		if s.Display == "list-item" {
			if m := markerText(n, s, e); m != "" {
				segs = append(segs, segment{text: m + " ", font: runFont(s, d), pre: true})
			}
		}
		if es := e.styled.Styles[n]; es != nil && es.Before != nil && !es.Before.ContentNone && es.Before.Content != "" {
			segs = append(segs, segment{text: es.Before.Content, font: runFont(es.Before, d), pre: true})
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") && c != n {
				// nested table cells are separated by a tab-like gap
				if c.PrevSibling != nil {
					segs = append(segs, segment{text: " ", font: runFont(s, d), pre: true})
				}
			}
			walk(c, s, d)
		}
		if es := e.styled.Styles[n]; es != nil && es.After != nil && !es.After.ContentNone && es.After.Content != "" {
			segs = append(segs, segment{text: es.After.Content, font: runFont(es.After, d), pre: true})
		}
		if block {
			segs = append(segs, segment{brk: true})
		}
	}
	deco := cellStyle.TextDecorationLine
	if cellStyle.Display == "list-item" {
		if m := markerText(n, cellStyle, e); m != "" {
			segs = append(segs, segment{text: m + " ", font: runFont(cellStyle, deco), pre: true})
		}
	}
	if es := e.styled.Styles[n]; es != nil && es.Before != nil && !es.Before.ContentNone && es.Before.Content != "" {
		segs = append(segs, segment{text: es.Before.Content, font: runFont(es.Before, deco), pre: true})
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, cellStyle, deco)
	}
	if es := e.styled.Styles[n]; es != nil && es.After != nil && !es.After.ContentNone && es.After.Content != "" {
		segs = append(segs, segment{text: es.After.Content, font: runFont(es.After, deco), pre: true})
	}
	out.runs = collapseSegments(segs)
	var b strings.Builder
	for _, r := range out.runs {
		b.WriteString(r.text)
	}
	out.text = b.String()
	out.font = base
	first := true
	for _, r := range out.runs {
		if strings.TrimSpace(r.text) == "" {
			continue
		}
		if first {
			out.font, first = r.font, false
		} else if r.font != out.font {
			out.mixed = true
			break
		}
	}
	return out
}

func sscan(s string, v *int) (int, error) { return fmt.Sscanf(strings.TrimSpace(s), "%d", v) }

func itoa(n int) string { return strconv.Itoa(n) }

// prevCell returns the previous td/th sibling of a cell, if any.
func prevCell(n *html.Node) *html.Node {
	for p := n.PrevSibling; p != nil; p = p.PrevSibling {
		if p.Type == html.ElementNode && (p.Data == "td" || p.Data == "th") {
			return p
		}
	}
	return nil
}

func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func keepsSpaces(ws string) bool { return ws == "pre" || ws == "pre-wrap" || ws == "break-spaces" }
func keepsLines(ws string) bool {
	return ws == "pre" || ws == "pre-wrap" || ws == "pre-line" || ws == "break-spaces"
}

// runFont is the font of text inside an element, with the decorations
// propagated from its ancestors.
func runFont(s *css.Style, deco css.Decoration) font {
	f := fontOf(s)
	f.underline = f.underline || deco.Underline
	f.strike = f.strike || deco.LineThrough
	return f
}

// collapseSegments applies CSS white-space processing across runs and
// merges adjacent runs with the same font. Leading and trailing white space
// of the whole content and around forced breaks is removed, as a block
// container would render it.
func collapseSegments(segs []segment) []textRun {
	var runs []textRun
	var cur strings.Builder
	var curFont font
	have := false
	pendingSpace := false
	atLineStart := true
	flush := func() {
		if cur.Len() > 0 {
			runs = append(runs, textRun{text: cur.String(), font: curFont})
			cur.Reset()
		}
	}
	emit := func(f font, s string) {
		if have && f != curFont {
			flush()
		}
		curFont, have = f, true
		cur.WriteString(s)
	}
	trimTrailing := func() {
		// drop spaces before a line break
		if cur.Len() > 0 {
			t := strings.TrimRight(cur.String(), " ")
			cur.Reset()
			cur.WriteString(t)
		} else if len(runs) > 0 {
			runs[len(runs)-1].text = strings.TrimRight(runs[len(runs)-1].text, " ")
		}
	}
	newline := func(f font) {
		trimTrailing()
		pendingSpace = false
		emit(f, "\n")
		atLineStart = true
	}
	for _, sg := range segs {
		if sg.brk {
			if !atLineStart {
				newline(curFont)
			}
			continue
		}
		text := transform(sg.text, sg.xform)
		if sg.pre {
			for _, r := range text {
				switch {
				case r == '\n' || r == '\r':
					if r == '\r' {
						continue
					}
					if sg.lines {
						newline(sg.font)
					} else {
						emit(sg.font, " ")
					}
				case r == '\t':
					emit(sg.font, "    ")
					atLineStart = false
				default:
					emit(sg.font, string(r))
					atLineStart = false
				}
			}
			pendingSpace = false
			continue
		}
		for _, r := range text {
			switch {
			case r == '\n' && sg.lines:
				newline(sg.font)
			case r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f':
				if !atLineStart {
					pendingSpace = true
				}
			default:
				if pendingSpace {
					emit(sg.font, " ")
					pendingSpace = false
				}
				emit(sg.font, string(r))
				atLineStart = false
			}
		}
	}
	trimTrailing()
	flush()
	// drop empty runs and a trailing newline
	var out []textRun
	for _, r := range runs {
		if r.text != "" {
			out = append(out, r)
		}
	}
	for len(out) > 0 {
		last := &out[len(out)-1]
		last.text = strings.TrimRight(last.text, "\n ")
		if last.text != "" {
			break
		}
		out = out[:len(out)-1]
	}
	return out
}

func transform(s, mode string) string {
	switch mode {
	case "uppercase":
		return strings.ToUpper(s)
	case "lowercase":
		return strings.ToLower(s)
	case "capitalize":
		prev := ' '
		return strings.Map(func(r rune) rune {
			out := r
			if !unicode.IsLetter(prev) && !unicode.IsDigit(prev) {
				out = unicode.ToUpper(r)
			}
			prev = r
			return out
		}, s)
	}
	return s
}

// markerText returns the list marker of a list item: a bullet, or its
// ordinal in the parent list.
func markerText(li *html.Node, s *css.Style, e *extractor) string {
	if es := e.styled.Styles[li]; es != nil && es.Marker != nil && es.Marker.Content != "" && !es.Marker.ContentNone {
		return strings.TrimRight(es.Marker.Content, " ")
	}
	switch s.ListStyleType {
	case "none", "":
		return ""
	case "disc":
		return "•"
	case "circle":
		return "◦"
	case "square":
		return "▪"
	}
	n := 1
	if li.Parent != nil {
		if st := attr(li.Parent, "start"); st != "" {
			var v int
			if _, err := sscan(st, &v); err == nil {
				n = v
			}
		}
		for c := li.Parent.FirstChild; c != nil && c != li; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "li" && !e.hidden(c) {
				n++
			}
		}
	}
	if v := attr(li, "value"); v != "" {
		var x int
		if _, err := sscan(v, &x); err == nil {
			n = x
		}
	}
	switch s.ListStyleType {
	case "lower-alpha", "lower-latin":
		return alpha(n, 'a') + "."
	case "upper-alpha", "upper-latin":
		return alpha(n, 'A') + "."
	case "lower-roman":
		return strings.ToLower(roman(n)) + "."
	case "upper-roman":
		return roman(n) + "."
	}
	return itoa(n) + "."
}

func alpha(n int, base rune) string {
	if n < 1 {
		return itoa(n)
	}
	var b []rune
	for n > 0 {
		n--
		b = append([]rune{base + rune(n%26)}, b...)
		n /= 26
	}
	return string(b)
}

func roman(n int) string {
	if n < 1 || n > 3999 {
		return itoa(n)
	}
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(syms[i])
			n -= v
		}
	}
	return b.String()
}

// plainText returns the collapsed text of n, ignoring hidden parts.
func plainText(n *html.Node, e *extractor) string {
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			b.WriteString(n.Data)
		case html.ElementNode:
			if e.hidden(n) {
				return
			}
			switch n.Data {
			case "script", "style", "template", "noscript":
				return
			case "br":
				b.WriteByte(' ')
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
	}
	walk(n)
	return b.String()
}

// collapseText collapses white space the way CSS does for the given
// white-space value and trims the ends.
func collapseText(s, ws string) string {
	if keepsSpaces(ws) {
		return strings.Trim(s, "\n")
	}
	return strings.Join(strings.Fields(s), " ")
}
