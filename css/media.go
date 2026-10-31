package css

import (
	"strings"
)

// MediaContext describes the target medium for @media evaluation.
type MediaContext struct {
	Type   string  // "print" or "screen"
	Width  float64 // page content width in points
	Height float64 // page content height in points
}

// Matcher returns a MediaMatcher for this context.
func (m MediaContext) Matcher() MediaMatcher {
	return m.Match
}

// Match evaluates a comma-separated media query list.
func (m MediaContext) Match(list string) bool {
	list = strings.TrimSpace(strings.ToLower(list))
	if list == "" {
		return true
	}
	for _, q := range SplitCommas(list) {
		if m.matchQuery(q) {
			return true
		}
	}
	return false
}

func (m MediaContext) matchQuery(q string) bool {
	q = strings.TrimSpace(q)
	negate := false
	if strings.HasPrefix(q, "not ") {
		negate = true
		q = strings.TrimSpace(q[4:])
	}
	q = strings.TrimPrefix(q, "only ")
	result := true
	for _, part := range splitAnd(q) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "(") {
			if !m.matchFeature(strings.Trim(part, "()")) {
				result = false
			}
			continue
		}
		switch part {
		case "all":
		case m.Type:
		default:
			result = false
		}
	}
	return result != negate
}

func splitAnd(q string) []string {
	var out []string
	depth := 0
	last := 0
	for i := 0; i < len(q); i++ {
		switch q[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && strings.HasPrefix(q[i:], " and ") {
			out = append(out, q[last:i])
			last = i + 5
			i += 4
		}
	}
	return append(out, q[last:])
}

func (m MediaContext) matchFeature(f string) bool {
	name, val, hasVal := strings.Cut(f, ":")
	name = strings.TrimSpace(name)
	val = strings.TrimSpace(val)
	lc := LengthContext{FontSize: 12, RootFontSize: 12, ViewportW: m.Width, ViewportH: m.Height}
	cmpLen := func(actual float64, op string) bool {
		l, ok := ParseLength(val, lc, false)
		if !ok {
			return false
		}
		switch op {
		case "min":
			return actual+0.01 >= l.Pt
		case "max":
			return actual-0.01 <= l.Pt
		}
		return actual == l.Pt
	}
	switch name {
	case "min-width":
		return cmpLen(m.Width, "min")
	case "max-width":
		return cmpLen(m.Width, "max")
	case "width":
		return cmpLen(m.Width, "")
	case "min-height":
		return cmpLen(m.Height, "min")
	case "max-height":
		return cmpLen(m.Height, "max")
	case "orientation":
		if m.Width > m.Height {
			return val == "landscape"
		}
		return val == "portrait"
	case "color", "min-color", "monochrome":
		return name != "monochrome"
	case "prefers-color-scheme":
		return val == "light"
	case "prefers-reduced-motion":
		return val == "no-preference" || !hasVal
	case "hover", "any-hover":
		return val == "none"
	case "pointer", "any-pointer":
		return val == "none"
	case "scripting":
		return val == "enabled"
	}
	return false
}
