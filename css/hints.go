package css

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// htmlLength converts an HTML dimension attribute ("100", "50%") to CSS.
func htmlLength(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	if strings.HasSuffix(v, "%") {
		if _, err := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64); err == nil {
			return v, true
		}
		return "", false
	}
	end := 0
	for end < len(v) && ((v[end] >= '0' && v[end] <= '9') || v[end] == '.') {
		end++
	}
	if end == 0 {
		return "", false
	}
	return v[:end] + "px", true
}

func htmlColor(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	if _, ok := ParseColor(v, Black); ok {
		return v, true
	}
	if _, ok := ParseColor("#"+v, Black); ok {
		return "#" + v, true
	}
	return "", false
}

func ancestorTable(n *html.Node) *html.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "table" {
			return p
		}
	}
	return nil
}

// PresentationalHints maps legacy HTML attributes to author-level
// declarations with zero specificity, as browsers do.
func PresentationalHints(n *html.Node) []Declaration {
	var out []Declaration
	add := func(p, v string) { out = append(out, Declaration{Property: p, Value: v}) }
	dim := func(attrName, prop string) {
		if v, ok := attr(n, attrName); ok {
			if l, ok := htmlLength(v); ok {
				add(prop, l)
			}
		}
	}
	bg := func() {
		if v, ok := attr(n, "bgcolor"); ok {
			if c, ok := htmlColor(v); ok {
				add("background-color", c)
			}
		}
	}
	// Blink maps align on these elements to the -webkit-* values, which
	// also align block-level children.
	align := func() {
		if v, ok := attr(n, "align"); ok {
			switch strings.ToLower(v) {
			case "left", "right", "center":
				add("text-align", "-webkit-"+strings.ToLower(v))
			case "middle":
				add("text-align", "-webkit-center")
			case "justify":
				add("text-align", "justify")
			}
		}
	}
	switch n.Data {
	case "body":
		bg()
		if v, ok := attr(n, "text"); ok {
			if c, ok := htmlColor(v); ok {
				add("color", c)
			}
		}
	case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "caption", "thead", "tbody", "tfoot", "tr":
		align()
		if n.Data == "tr" || n.Data == "thead" || n.Data == "tbody" || n.Data == "tfoot" {
			bg()
			if v, ok := attr(n, "valign"); ok {
				add("vertical-align", strings.ToLower(v))
			}
		}
	case "img":
		dim("width", "width")
		dim("height", "height")
		if v, ok := attr(n, "border"); ok {
			if l, ok := htmlLength(v); ok {
				add("border", l+" solid")
			}
		}
		if v, ok := attr(n, "align"); ok {
			switch strings.ToLower(v) {
			case "left", "right":
				add("float", strings.ToLower(v))
			case "middle", "absmiddle":
				add("vertical-align", "middle")
			case "top", "bottom":
				add("vertical-align", strings.ToLower(v))
			}
		}
	case "table":
		dim("width", "width")
		dim("height", "height")
		bg()
		if v, ok := attr(n, "cellspacing"); ok {
			if l, ok := htmlLength(v); ok {
				add("border-spacing", l)
			}
		}
		if v, ok := attr(n, "border"); ok {
			w := "1px"
			if l, ok := htmlLength(v); ok {
				w = l
			}
			if w != "0px" {
				add("border", w+" outset")
			}
		}
		if v, ok := attr(n, "align"); ok {
			switch strings.ToLower(v) {
			case "center":
				add("margin-left", "auto")
				add("margin-right", "auto")
			case "left", "right":
				add("float", strings.ToLower(v))
			}
		}
	case "td", "th":
		dim("width", "width")
		dim("height", "height")
		bg()
		align()
		if v, ok := attr(n, "valign"); ok {
			add("vertical-align", strings.ToLower(v))
		}
		if _, ok := attr(n, "nowrap"); ok {
			add("white-space", "nowrap")
		}
		if t := ancestorTable(n); t != nil {
			if v, ok := attr(t, "cellpadding"); ok {
				if l, ok := htmlLength(v); ok {
					add("padding", l)
				}
			}
			if v, ok := attr(t, "border"); ok {
				if l, ok := htmlLength(v); !ok || l != "0px" {
					// width and style only: Blink makes the colour inherit
					// (through tr/tbody) from the table
					add("border-width", "1px")
					add("border-style", "inset")
					add("border-color", "inherit")
				}
			}
		}
	case "col", "colgroup":
		dim("width", "width")
	case "hr":
		dim("width", "width")
		if v, ok := attr(n, "size"); ok {
			if l, ok := htmlLength(v); ok {
				add("height", l)
			}
		}
		if v, ok := attr(n, "color"); ok {
			if c, ok := htmlColor(v); ok {
				add("color", c)
				add("background-color", c)
				add("border-style", "solid")
			}
		}
		if _, ok := attr(n, "noshade"); ok {
			add("border-style", "solid")
			add("background-color", "gray")
		}
		if v, ok := attr(n, "align"); ok {
			switch strings.ToLower(v) {
			case "left":
				add("margin-left", "0")
			case "right":
				add("margin-right", "0")
			}
		}
	case "font":
		if v, ok := attr(n, "color"); ok {
			if c, ok := htmlColor(v); ok {
				add("color", c)
			}
		}
		if v, ok := attr(n, "face"); ok {
			add("font-family", v)
		}
		if v, ok := attr(n, "size"); ok {
			sizes := []string{"x-small", "small", "medium", "large", "x-large", "xx-large", "xxx-large"}
			v = strings.TrimSpace(v)
			base := 3
			if strings.HasPrefix(v, "+") || strings.HasPrefix(v, "-") {
				if d, err := strconv.Atoi(v); err == nil {
					base += d
				}
			} else if d, err := strconv.Atoi(v); err == nil {
				base = d
			}
			base = max(1, min(7, base))
			add("font-size", sizes[base-1])
		}
	case "ol", "ul", "li":
		if v, ok := attr(n, "type"); ok {
			m := map[string]string{"1": "decimal", "a": "lower-alpha", "A": "upper-alpha", "i": "lower-roman", "I": "upper-roman",
				"disc": "disc", "circle": "circle", "square": "square", "none": "none"}
			if t, ok := m[v]; ok {
				add("list-style-type", t)
			}
		}
	}
	return out
}
