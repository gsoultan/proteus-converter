// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package css

import (
	"strconv"
	"strings"
)

// SplitValue splits a property value into top-level component values.
// Whitespace separates components; ',' and '/' are returned as their own
// tokens. Strings and function calls are kept intact.
func SplitValue(v string) []string {
	var out []string
	start := -1
	depth := 0
	var quote byte
	flush := func(i int) {
		if start >= 0 {
			out = append(out, v[start:i])
			start = -1
		}
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		if quote != 0 {
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
			continue
		}
		switch {
		case c == '"' || c == '\'':
			if start < 0 {
				start = i
			}
			quote = c
		case c == '(':
			if start < 0 {
				start = i
			}
			depth++
		case c == ')':
			if depth > 0 {
				depth--
			}
		case depth > 0:
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f':
			flush(i)
		case c == ',' || c == '/':
			flush(i)
			out = append(out, string(c))
		default:
			if start < 0 {
				start = i
			}
		}
	}
	flush(len(v))
	return out
}

// SplitCommas splits v at top-level commas.
func SplitCommas(v string) []string {
	var out []string
	depth := 0
	var quote byte
	last := 0
	for i := 0; i < len(v); i++ {
		c := v[i]
		if quote != 0 {
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(v[last:i]))
				last = i + 1
			}
		}
	}
	return append(out, strings.TrimSpace(v[last:]))
}

// Function splits "name(args)" into its lower-case name and argument text.
func Function(tok string) (name, args string, ok bool) {
	open := strings.IndexByte(tok, '(')
	if open <= 0 || !strings.HasSuffix(tok, ")") {
		return "", "", false
	}
	return strings.ToLower(tok[:open]), tok[open+1 : len(tok)-1], true
}

// Unquote removes surrounding quotes and resolves simple escapes.
func Unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		s = s[1 : len(s)-1]
	}
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		j := i + 1
		for j < len(s) && j < i+7 && isHex(s[j]) {
			j++
		}
		if j > i+1 {
			if n, err := strconv.ParseUint(s[i+1:j], 16, 32); err == nil {
				b.WriteRune(rune(n))
			}
			if j < len(s) && s[j] == ' ' {
				j++
			}
			i = j - 1
			continue
		}
		i++
		if s[i] != '\n' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// URLValue extracts the target of url(...) or a quoted string.
func URLValue(tok string) (string, bool) {
	if fn, args, ok := Function(tok); ok {
		if fn == "url" || fn == "src" {
			return Unquote(args), true
		}
		return "", false
	}
	if len(tok) > 0 && (tok[0] == '"' || tok[0] == '\'') {
		return Unquote(tok), true
	}
	return "", false
}
