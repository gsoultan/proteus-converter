// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package css parses stylesheets and computes styles with the browser cascade:
// origin and importance, specificity, source order, inheritance, custom
// properties (var()) and calc().
package css

import (
	"strings"
)

// Declaration is one property: value pair.
type Declaration struct {
	Property  string
	Value     string
	Important bool
}

// Rule is a style rule: a selector list and its declarations.
type Rule struct {
	Selector string
	Decls    []Declaration
}

// AtPage is an @page rule.
type AtPage struct {
	Selector string // e.g. ":first"; empty for all pages
	Decls    []Declaration
}

// FontFace is an @font-face rule.
type FontFace struct {
	Decls []Declaration
}

// Import is an @import statement.
type Import struct {
	URL   string
	Media string
}

// Stylesheet is a parsed stylesheet. Conditional groups (@media, @supports)
// are evaluated while parsing, so Rules only contains rules that apply.
type Stylesheet struct {
	Rules     []Rule
	Pages     []AtPage
	FontFaces []FontFace
	Imports   []Import
}

// MediaMatcher decides whether a media query list applies.
type MediaMatcher func(query string) bool

// Parse parses CSS source. media evaluates @media preludes; nil accepts every query.
func Parse(src string, media MediaMatcher) *Stylesheet {
	p := &parser{s: stripComments(src), media: media}
	ss := &Stylesheet{}
	p.parseRules(ss, false)
	return ss
}

// ParseDeclarations parses the body of a style attribute or block.
func ParseDeclarations(src string) []Declaration {
	return parseDecls(stripComments(src))
}

type parser struct {
	s     string
	pos   int
	media MediaMatcher
}

func stripComments(s string) string {
	if !strings.Contains(s, "/*") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			b.WriteByte(c)
			if c == '\\' && i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			b.WriteByte(c)
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				break
			}
			i += end + 3
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func (p *parser) skipWS() {
	for p.pos < len(p.s) {
		switch p.s[p.pos] {
		case ' ', '\t', '\n', '\r', '\f':
			p.pos++
		default:
			if strings.HasPrefix(p.s[p.pos:], "<!--") {
				p.pos += 4
				continue
			}
			if strings.HasPrefix(p.s[p.pos:], "-->") {
				p.pos += 3
				continue
			}
			return
		}
	}
}

// readUntil reads up to (not including) the first top-level byte in stops,
// honouring strings, parentheses and brackets.
func (p *parser) readUntil(stops string) string {
	start := p.pos
	depth := 0
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		switch {
		case c == '"' || c == '\'':
			p.skipString(c)
			continue
		case c == '\\':
			p.pos += 2
			continue
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			if depth > 0 {
				depth--
			}
		case depth == 0 && strings.IndexByte(stops, c) >= 0:
			return p.s[start:p.pos]
		}
		p.pos++
	}
	return p.s[start:]
}

func (p *parser) skipString(q byte) {
	p.pos++
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '\\' {
			p.pos += 2
			continue
		}
		p.pos++
		if c == q || c == '\n' {
			return
		}
	}
}

// readBlock reads a {...} block body; p.pos must be at '{'. Returns the inner text.
func (p *parser) readBlock() string {
	p.pos++ // {
	start := p.pos
	depth := 1
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		switch c {
		case '"', '\'':
			p.skipString(c)
			continue
		case '\\':
			p.pos += 2
			continue
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				body := p.s[start:p.pos]
				p.pos++
				return body
			}
		}
		p.pos++
	}
	return p.s[start:]
}

func (p *parser) parseRules(ss *Stylesheet, nested bool) {
	for {
		p.skipWS()
		if p.pos >= len(p.s) {
			return
		}
		c := p.s[p.pos]
		if c == '}' {
			p.pos++
			if nested {
				return
			}
			continue
		}
		if c == ';' {
			p.pos++
			continue
		}
		if c == '@' {
			p.parseAtRule(ss)
			continue
		}
		prelude := strings.TrimSpace(p.readUntil("{;}"))
		if p.pos >= len(p.s) {
			return
		}
		if p.s[p.pos] != '{' {
			p.pos++
			continue
		}
		body := p.readBlock()
		if prelude != "" {
			ss.Rules = append(ss.Rules, Rule{Selector: prelude, Decls: parseDecls(body)})
		}
	}
}

func (p *parser) parseAtRule(ss *Stylesheet) {
	p.pos++ // @
	start := p.pos
	for p.pos < len(p.s) && isIdentByte(p.s[p.pos]) {
		p.pos++
	}
	name := strings.ToLower(p.s[start:p.pos])
	prelude := strings.TrimSpace(p.readUntil("{;"))
	if p.pos >= len(p.s) {
		if name == "import" {
			ss.Imports = append(ss.Imports, parseImport(prelude))
		}
		return
	}
	if p.s[p.pos] == ';' {
		p.pos++
		if name == "import" {
			ss.Imports = append(ss.Imports, parseImport(prelude))
		}
		return
	}
	switch name {
	case "media":
		body := p.readBlock()
		if p.media == nil || p.media(prelude) {
			sub := &parser{s: body, media: p.media}
			sub.parseRules(ss, false)
		}
	case "supports":
		body := p.readBlock()
		if supportsCondition(prelude) {
			sub := &parser{s: body, media: p.media}
			sub.parseRules(ss, false)
		}
	case "layer", "container", "scope", "document", "-moz-document":
		body := p.readBlock()
		sub := &parser{s: body, media: p.media}
		sub.parseRules(ss, false)
	case "page":
		body := p.readBlock()
		// Margin boxes (@top-center etc.) are skipped; only the page's own declarations are kept.
		ss.Pages = append(ss.Pages, AtPage{Selector: prelude, Decls: parseDecls(stripNestedBlocks(body))})
	case "font-face":
		ss.FontFaces = append(ss.FontFaces, FontFace{Decls: parseDecls(p.readBlock())})
	default:
		p.readBlock()
	}
}

func isIdentByte(c byte) bool {
	return c == '-' || c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c >= 0x80
}

func parseImport(prelude string) Import {
	toks := SplitValue(prelude)
	if len(toks) == 0 {
		return Import{}
	}
	u := toks[0]
	if fn, args, ok := Function(u); ok && fn == "url" {
		u = Unquote(strings.TrimSpace(args))
	} else {
		u = Unquote(u)
	}
	return Import{URL: u, Media: strings.TrimSpace(strings.Join(toks[1:], " "))}
}

// supportsCondition is a conservative @supports evaluator: it accepts any
// condition that does not start with "not".
func supportsCondition(cond string) bool {
	return !strings.HasPrefix(strings.ToLower(strings.TrimSpace(cond)), "not")
}

func stripNestedBlocks(s string) string {
	var b strings.Builder
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '{':
			depth++
			continue
		case '}':
			depth--
			continue
		}
		if depth == 0 {
			b.WriteByte(c)
		}
	}
	// drop the at-keyword preludes (e.g. "@top-center") left behind
	parts := strings.Split(b.String(), ";")
	out := parts[:0]
	for _, part := range parts {
		t := strings.TrimSpace(part)
		if i := strings.LastIndex(t, "@"); i >= 0 && !strings.Contains(t[i:], ":") {
			t = strings.TrimSpace(t[:i])
		}
		if t != "" {
			out = append(out, t)
		}
	}
	return strings.Join(out, ";")
}

func parseDecls(body string) []Declaration {
	p := &parser{s: body}
	var out []Declaration
	for p.pos < len(p.s) {
		p.skipWS()
		if p.pos >= len(p.s) {
			break
		}
		if p.s[p.pos] == ';' {
			p.pos++
			continue
		}
		item := p.readUntil(";{")
		if p.pos < len(p.s) && p.s[p.pos] == '{' {
			p.readBlock() // nested rule; not supported
			continue
		}
		if p.pos < len(p.s) {
			p.pos++
		}
		colon := strings.IndexByte(item, ':')
		if colon <= 0 {
			continue
		}
		name := strings.TrimSpace(item[:colon])
		value := strings.TrimSpace(item[colon+1:])
		if !strings.HasPrefix(name, "--") {
			name = strings.ToLower(name)
		}
		d := Declaration{Property: name}
		if i := strings.LastIndex(value, "!"); i >= 0 && strings.EqualFold(strings.TrimSpace(value[i+1:]), "important") {
			d.Important = true
			value = strings.TrimSpace(value[:i])
		}
		d.Value = value
		if name == "" || (value == "" && !strings.HasPrefix(name, "--")) {
			continue
		}
		out = append(out, d)
	}
	return out
}
