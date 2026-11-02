// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package layout

import (
	"strconv"
	"strings"

	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/svg"
	"golang.org/x/net/html"
)

// Builder generates the box tree from an HTML tree and computed styles.
type Builder struct {
	Styles map[*html.Node]*css.ElementStyle
	Images ImageLoader
	Warn   func(string)

	lists      []*listState
	quoteDepth int
}

type listState struct {
	next     int
	reversed bool
}

func (bd *Builder) warn(msg string) {
	if bd.Warn != nil {
		bd.Warn(msg)
	}
}

// Build returns the root block box for the document element.
func (bd *Builder) Build(doc *html.Node) *Box {
	root := doc
	if root.Type == html.DocumentNode {
		for c := doc.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				root = c
				break
			}
		}
	}
	es := bd.Styles[root]
	if es == nil {
		return &Box{Kind: KindBlock, Style: css.Initial()}
	}
	boxes := bd.element(root, es.Style, nil)
	if len(boxes) == 0 {
		return &Box{Kind: KindBlock, Style: es.Style}
	}
	rb := boxes[0]
	if rb.Kind != KindBlock {
		rb = &Box{Kind: KindBlock, Style: es.Style, Children: boxes}
	}
	fixTree(rb)
	return rb
}

func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func kindForDisplay(d string) Kind {
	switch d {
	case "inline", "ruby", "ruby-text":
		return KindInline
	case "inline-block", "inline-flex", "inline-grid", "inline-table":
		return KindInlineBlock
	case "table":
		return KindTable
	case "table-row-group", "table-header-group", "table-footer-group":
		return KindRowGroup
	case "table-row":
		return KindRow
	case "table-cell":
		return KindCell
	case "table-caption":
		return KindCaption
	}
	return KindBlock
}

func withDecorations(parent []Decoration, s *css.Style) []Decoration {
	if !s.TextDecorationLine.Any() {
		return parent
	}
	out := make([]Decoration, len(parent), len(parent)+1)
	copy(out, parent)
	return append(out, Decoration{Lines: s.TextDecorationLine, Color: s.DecorationColor(), Style: s.TextDecorationStyle})
}

func (bd *Builder) children(n *html.Node, parentStyle *css.Style, decos []Decoration) []*Box {
	var out []*Box
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			if c.Data != "" {
				out = append(out, &Box{Kind: KindText, Style: parentStyle, Text: c.Data, Decorations: decos})
			}
		case html.ElementNode:
			es := bd.Styles[c]
			if es == nil {
				continue
			}
			out = append(out, bd.element(c, es.Style, decos)...)
		}
	}
	return out
}

func (bd *Builder) element(n *html.Node, s *css.Style, parentDecos []Decoration) []*Box {
	if s.Display == "none" {
		return nil
	}
	es := bd.Styles[n]
	if s.Display == "contents" {
		return bd.children(n, s, parentDecos)
	}
	kind := kindForDisplay(s.Display)
	b := &Box{Kind: kind, Style: s, Node: n}
	if id, ok := attr(n, "id"); ok && id != "" {
		b.ID = id
	} else if name, ok := attr(n, "name"); ok && n.Data == "a" {
		b.ID = name
	}
	switch n.Data {
	case "a":
		if href, ok := attr(n, "href"); ok {
			b.Href = strings.TrimSpace(href)
		}
	case "h1", "h2", "h3", "h4", "h5", "h6":
		b.Heading = int(n.Data[1] - '0')
	case "td", "th":
		b.ColSpan, b.RowSpan = spanAttr(n, "colspan"), spanAttr(n, "rowspan")
	case "br":
		return []*Box{{Kind: KindBreak, Style: s, Node: n}}
	case "wbr":
		return []*Box{{Kind: KindWordBreak, Style: s, Node: n}}
	case "img":
		return []*Box{bd.image(n, s, parentDecos)}
	case "input":
		t, _ := attr(n, "type")
		switch strings.ToLower(t) {
		case "checkbox", "radio":
			_, checked := attr(n, "checked")
			_, disabled := attr(n, "disabled")
			return []*Box{{Kind: KindCheckbox, Style: s, Node: n, Checked: checked, Disabled: disabled}}
		case "hidden":
			return nil
		}
		v, _ := attr(n, "value")
		if v == "" {
			v, _ = attr(n, "placeholder")
		}
		return []*Box{{Kind: KindInlineBlock, Style: s, Node: n, Children: []*Box{{Kind: KindText, Style: s, Text: v}}}}
	case "svg":
		return []*Box{bd.inlineSVG(n, s, parentDecos)}
	case "canvas", "video", "audio", "iframe", "object", "embed", "math":
		bd.warn("layout: <" + n.Data + "> is not supported yet and was skipped")
		return nil
	case "textarea", "select", "button":
		// rendered as inline-block text
	}
	if s.Display == "table-column" || s.Display == "table-column-group" {
		return nil
	}

	if s.BackgroundImage != "" && bd.Images != nil {
		img, err := bd.Images(s.BackgroundImage)
		if err != nil {
			bd.warn("layout: background image " + s.BackgroundImage + ": " + err.Error())
		} else {
			b.BgImage = img
		}
	}

	atomic := kind == KindInlineBlock || kind == KindTable || kind == KindCell || kind == KindCaption
	decos := parentDecos
	if atomic {
		decos = nil
	}
	decos = withDecorations(decos, s)
	b.Decorations = decos

	pushedList := false
	if n.Data == "ol" || n.Data == "ul" || n.Data == "menu" || n.Data == "dir" {
		st := &listState{next: 1}
		if n.Data == "ol" {
			_, st.reversed = attr(n, "reversed")
			if st.reversed {
				st.next = countListItems(n, bd.Styles)
			}
			if v, ok := attr(n, "start"); ok {
				if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					st.next = i
				}
			}
		}
		bd.lists = append(bd.lists, st)
		pushedList = true
	}
	if s.Display == "list-item" {
		b.Marker = bd.marker(n, es, decos)
	}

	if es != nil && es.Before != nil {
		b.Children = append(b.Children, bd.generated(n, es.Before, decos)...)
	}
	b.Children = append(b.Children, bd.children(n, s, decos)...)
	if es != nil && es.After != nil {
		b.Children = append(b.Children, bd.generated(n, es.After, decos)...)
	}
	if pushedList {
		bd.lists = bd.lists[:len(bd.lists)-1]
	}
	if s.Display == "inline-table" {
		inner := *b
		inner.Kind = KindTable
		innerStyle := *s
		innerStyle.Display = "table"
		inner.Style = &innerStyle
		b.Children = []*Box{&inner}
		b.Href, b.ID = "", ""
	}
	return []*Box{b}
}

func spanAttr(n *html.Node, key string) int {
	if v, ok := attr(n, key); ok {
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && i > 0 {
			return min(i, 1000)
		}
	}
	return 1
}

func countListItems(n *html.Node, styles map[*html.Node]*css.ElementStyle) int {
	c := 0
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if es := styles[ch]; es != nil && es.Style.Display == "list-item" {
			c++
		}
	}
	return c
}

func (bd *Builder) image(n *html.Node, s *css.Style, decos []Decoration) *Box {
	src, _ := attr(n, "src")
	alt, _ := attr(n, "alt")
	b := &Box{Kind: KindImage, Style: s, Node: n, Alt: alt, Decorations: decos}
	if src != "" && bd.Images != nil {
		img, err := bd.Images(src)
		if err != nil {
			bd.warn("layout: image " + src + ": " + err.Error())
		} else {
			b.Image = img
		}
	}
	if b.Image == nil {
		if alt == "" {
			return &Box{Kind: KindInline, Style: s, Node: n}
		}
		return &Box{Kind: KindInline, Style: s, Node: n, Children: []*Box{{Kind: KindText, Style: s, Text: alt, Decorations: decos}}}
	}
	return b
}

// inlineSVG builds a replaced box for an inline <svg> element.
func (bd *Builder) inlineSVG(n *html.Node, s *css.Style, decos []Decoration) *Box {
	d := svg.FromNode(n, s.Color)
	for _, w := range d.Warnings {
		bd.warn("layout: " + w)
	}
	img := &Image{SVG: d, Width: int(d.Width), Height: int(d.Height)}
	b := &Box{Kind: KindImage, Style: s, Node: n, Image: img, Decorations: decos}
	if id, ok := attr(n, "id"); ok {
		b.ID = id
	}
	return b
}

// marker builds the ::marker content of a list item.
func (bd *Builder) marker(n *html.Node, es *css.ElementStyle, decos []Decoration) *Box {
	s := es.Style
	ms := es.Marker
	if ms == nil {
		ms = s.Inherit()
		ms.Display = "inline"
	}
	ordinal := 1
	if len(bd.lists) > 0 {
		st := bd.lists[len(bd.lists)-1]
		if v, ok := attr(n, "value"); ok {
			if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				st.next = i
			}
		}
		ordinal = st.next
		if st.reversed {
			st.next--
		} else {
			st.next++
		}
	}
	var text string
	if ms.Content != "" && !ms.ContentNone {
		text = bd.resolveContent(n, ms.Content)
	} else {
		text = MarkerText(s.ListStyleType, ordinal)
	}
	if text == "" {
		return nil
	}
	mb := &Box{Kind: KindInline, Style: ms, Children: []*Box{{Kind: KindText, Style: ms, Text: text, Decorations: nil}}}
	if ms.Content == "" || ms.ContentNone {
		switch s.ListStyleType {
		case "disc", "circle", "square":
			mb.Symbol = s.ListStyleType
		}
	}
	return mb
}

// MarkerText returns the marker string for a list-style-type and ordinal.
func MarkerText(t string, n int) string {
	if strings.HasPrefix(t, "string:") {
		return t[len("string:"):]
	}
	switch t {
	case "none":
		return ""
	case "disc":
		return "• "
	case "circle":
		return "◦ "
	case "square":
		return "▪ "
	case "disclosure-open":
		return "▾ "
	case "disclosure-closed":
		return "▸ "
	case "decimal-leading-zero":
		if n >= 0 && n < 10 {
			return "0" + strconv.Itoa(n) + ". "
		}
		return strconv.Itoa(n) + ". "
	case "lower-alpha", "lower-latin":
		return alpha(n, 'a') + ". "
	case "upper-alpha", "upper-latin":
		return alpha(n, 'A') + ". "
	case "lower-roman":
		return strings.ToLower(roman(n)) + ". "
	case "upper-roman":
		return roman(n) + ". "
	case "lower-greek":
		if n >= 1 && n <= 24 {
			return string(rune('α'+n-1+boolInt(n > 17))) + ". "
		}
	}
	return strconv.Itoa(n) + ". "
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func alpha(n int, base rune) string {
	if n <= 0 {
		return strconv.Itoa(n)
	}
	var out []rune
	for n > 0 {
		n--
		out = append([]rune{base + rune(n%26)}, out...)
		n /= 26
	}
	return string(out)
}

func roman(n int) string {
	if n <= 0 || n >= 4000 {
		return strconv.Itoa(n)
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

// generated builds the box for a ::before or ::after pseudo-element.
func (bd *Builder) generated(n *html.Node, s *css.Style, decos []Decoration) []*Box {
	text := bd.resolveContent(n, s.Content)
	d := withDecorations(decos, s)
	child := &Box{Kind: KindText, Style: s, Text: text, Decorations: d}
	kind := kindForDisplay(s.Display)
	if kind != KindBlock && kind != KindInlineBlock {
		kind = KindInline
	}
	return []*Box{{Kind: kind, Style: s, Children: []*Box{child}, Decorations: d}}
}

// resolveContent evaluates a 'content' value: strings, attr(), quotes and
// counter() for the list-item counter.
func (bd *Builder) resolveContent(n *html.Node, v string) string {
	var b strings.Builder
	for _, tok := range css.SplitValue(v) {
		switch {
		case tok == "," || tok == "/":
		case tok[0] == '"' || tok[0] == '\'':
			b.WriteString(css.Unquote(tok))
		case tok == "open-quote":
			if bd.quoteDepth%2 == 0 {
				b.WriteString("“")
			} else {
				b.WriteString("‘")
			}
			bd.quoteDepth++
		case tok == "close-quote":
			if bd.quoteDepth > 0 {
				bd.quoteDepth--
			}
			if bd.quoteDepth%2 == 0 {
				b.WriteString("”")
			} else {
				b.WriteString("’")
			}
		default:
			if fn, args, ok := css.Function(tok); ok {
				switch fn {
				case "attr":
					val, _ := attr(n, strings.TrimSpace(args))
					b.WriteString(val)
				case "counter":
					b.WriteString("") // only list-item numbering is supported
				}
			}
		}
	}
	return b.String()
}
