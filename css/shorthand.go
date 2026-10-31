package css

import (
	"strings"
)

const pendingMarker = "\x00pending\x00"

var shorthandLonghands = map[string][]string{}

func sideNames(prefix, suffix string) []string {
	return []string{prefix + "-top" + suffix, prefix + "-right" + suffix, prefix + "-bottom" + suffix, prefix + "-left" + suffix}
}

func init() {
	shorthandLonghands["margin"] = sideNames("margin", "")
	shorthandLonghands["padding"] = sideNames("padding", "")
	shorthandLonghands["border-width"] = sideNames("border", "-width")
	shorthandLonghands["border-style"] = sideNames("border", "-style")
	shorthandLonghands["border-color"] = sideNames("border", "-color")
	shorthandLonghands["border-radius"] = []string{"border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius"}
	all := []string{}
	for _, side := range []string{"top", "right", "bottom", "left"} {
		l := []string{"border-" + side + "-width", "border-" + side + "-style", "border-" + side + "-color"}
		shorthandLonghands["border-"+side] = l
		all = append(all, l...)
	}
	shorthandLonghands["border"] = all
	shorthandLonghands["font"] = []string{"font-style", "font-variant", "font-weight", "font-size", "line-height", "font-family"}
	shorthandLonghands["background"] = []string{"background-color", "background-image", "background-size",
		"background-position", "background-repeat", "background-origin", "background-clip"}
	shorthandLonghands["list-style"] = []string{"list-style-type", "list-style-position", "list-style-image"}
	shorthandLonghands["text-decoration"] = []string{"text-decoration-line", "text-decoration-style", "text-decoration-color"}
	shorthandLonghands["overflow"] = []string{"overflow-x", "overflow-y"}
	shorthandLonghands["margin-block"] = []string{"margin-top", "margin-bottom"}
	shorthandLonghands["margin-inline"] = []string{"margin-left", "margin-right"}
	shorthandLonghands["padding-block"] = []string{"padding-top", "padding-bottom"}
	shorthandLonghands["padding-inline"] = []string{"padding-left", "padding-right"}
	shorthandLonghands["border-block"] = append(append([]string{}, shorthandLonghands["border-top"]...), shorthandLonghands["border-bottom"]...)
	shorthandLonghands["border-inline"] = append(append([]string{}, shorthandLonghands["border-left"]...), shorthandLonghands["border-right"]...)
}

// aliases maps legacy and logical (horizontal-tb, ltr) names to physical longhands.
var aliases = map[string]string{
	"word-wrap":                    "overflow-wrap",
	"-webkit-print-color-adjust":   "print-color-adjust",
	"color-adjust":                 "print-color-adjust",
	"margin-block-start":           "margin-top",
	"margin-block-end":             "margin-bottom",
	"margin-inline-start":          "margin-left",
	"margin-inline-end":            "margin-right",
	"padding-block-start":          "padding-top",
	"padding-block-end":            "padding-bottom",
	"padding-inline-start":         "padding-left",
	"padding-inline-end":           "padding-right",
	"border-block-start":           "border-top",
	"border-block-end":             "border-bottom",
	"border-inline-start":          "border-left",
	"border-inline-end":            "border-right",
	"border-block-start-width":     "border-top-width",
	"border-block-end-width":       "border-bottom-width",
	"border-inline-start-width":    "border-left-width",
	"border-inline-end-width":      "border-right-width",
	"border-block-start-style":     "border-top-style",
	"border-block-end-style":       "border-bottom-style",
	"border-inline-start-style":    "border-left-style",
	"border-inline-end-style":      "border-right-style",
	"border-block-start-color":     "border-top-color",
	"border-block-end-color":       "border-bottom-color",
	"border-inline-start-color":    "border-left-color",
	"border-inline-end-color":      "border-right-color",
	"inline-size":                  "width",
	"block-size":                   "height",
	"min-inline-size":              "min-width",
	"min-block-size":               "min-height",
	"max-inline-size":              "max-width",
	"max-block-size":               "max-height",
	"-webkit-text-decoration-line": "text-decoration-line",
	"-webkit-box-sizing":           "box-sizing",
	"-moz-box-sizing":              "box-sizing",
	"-webkit-hyphens":              "hyphens",
	"-webkit-column-break-inside":  "break-inside",
}

// Expand turns one declaration into longhand declarations. Values that use
// var() are kept pending until computed-value time.
func Expand(d Declaration) []Declaration {
	name := d.Property
	if strings.HasPrefix(name, "--") {
		return []Declaration{d}
	}
	if a, ok := aliases[name]; ok {
		name = a
	}
	switch name {
	case "page-break-before", "page-break-after", "page-break-inside":
		v := strings.ToLower(d.Value)
		if v == "always" {
			v = "page"
		}
		return []Declaration{{Property: "break-" + strings.TrimPrefix(name, "page-break-"), Value: v, Important: d.Important}}
	}
	long, isShort := shorthandLonghands[name]
	if !isShort {
		d.Property = name
		return []Declaration{d}
	}
	if strings.Contains(strings.ToLower(d.Value), "var(") {
		out := make([]Declaration, len(long))
		for i, l := range long {
			out[i] = Declaration{Property: l, Value: pendingMarker + name + "\x00" + d.Value, Important: d.Important}
		}
		return out
	}
	vals := expandShorthand(name, d.Value)
	if vals == nil {
		return nil
	}
	out := make([]Declaration, 0, len(vals))
	for _, l := range long {
		if v, ok := vals[l]; ok {
			out = append(out, Declaration{Property: l, Value: v, Important: d.Important})
		}
	}
	return out
}

func isGlobalKeyword(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "inherit", "initial", "unset", "revert", "revert-layer":
		return true
	}
	return false
}

// expandShorthand returns longhand values for a shorthand, or nil if invalid.
func expandShorthand(name, value string) map[string]string {
	long := shorthandLonghands[name]
	if isGlobalKeyword(value) {
		m := map[string]string{}
		for _, l := range long {
			m[l] = value
		}
		return m
	}
	toks := SplitValue(value)
	switch name {
	case "margin", "padding", "border-width", "border-style", "border-color", "border-radius":
		if name == "border-radius" {
			// only horizontal radii are used
			for i, t := range toks {
				if t == "/" {
					toks = toks[:i]
					break
				}
			}
		}
		return fourSides(long, toks)
	case "margin-block", "margin-inline", "padding-block", "padding-inline":
		if len(toks) == 1 {
			toks = append(toks, toks[0])
		}
		if len(toks) != 2 {
			return nil
		}
		return map[string]string{long[0]: toks[0], long[1]: toks[1]}
	case "overflow":
		if len(toks) == 1 {
			toks = append(toks, toks[0])
		}
		return map[string]string{"overflow-x": toks[0], "overflow-y": toks[1]}
	case "border", "border-top", "border-right", "border-bottom", "border-left", "border-block", "border-inline":
		w, st, col := "medium", "none", "currentcolor"
		for _, t := range toks {
			lt := strings.ToLower(t)
			switch {
			case borderStyles[lt]:
				st = lt
			case lt == "thin" || lt == "medium" || lt == "thick" || startsNumeric(lt) || strings.HasPrefix(lt, "calc("):
				w = t
			default:
				if _, ok := ParseColor(t, Black); !ok {
					return nil
				}
				col = t
			}
		}
		m := map[string]string{}
		for _, l := range long {
			switch {
			case strings.HasSuffix(l, "-width"):
				m[l] = w
			case strings.HasSuffix(l, "-style"):
				m[l] = st
			default:
				m[l] = col
			}
		}
		return m
	case "font":
		return expandFont(toks, value)
	case "background":
		return expandBackground(value)
	case "list-style":
		m := map[string]string{"list-style-type": "disc", "list-style-position": "outside", "list-style-image": "none"}
		nones := 0
		typeSet := false
		for _, t := range toks {
			lt := strings.ToLower(t)
			switch {
			case lt == "inside" || lt == "outside":
				m["list-style-position"] = lt
			case strings.HasPrefix(lt, "url("):
				m["list-style-image"] = t
			case lt == "none":
				nones++
			default:
				m["list-style-type"] = t
				typeSet = true
			}
		}
		if nones > 0 && !typeSet {
			m["list-style-type"] = "none"
		}
		return m
	case "text-decoration":
		m := map[string]string{"text-decoration-line": "none", "text-decoration-style": "solid", "text-decoration-color": "currentcolor"}
		var lines []string
		for _, t := range toks {
			lt := strings.ToLower(t)
			switch lt {
			case "underline", "overline", "line-through", "blink":
				lines = append(lines, lt)
			case "none":
			case "solid", "double", "dotted", "dashed", "wavy":
				m["text-decoration-style"] = lt
			default:
				if _, ok := ParseColor(t, Black); ok {
					m["text-decoration-color"] = t
				} else if !startsNumeric(lt) && lt != "auto" && lt != "from-font" {
					return nil
				}
			}
		}
		if len(lines) > 0 {
			m["text-decoration-line"] = strings.Join(lines, " ")
		}
		return m
	}
	return expandLayoutShorthand(name, toks, value)
}

// expandBackground expands the background shorthand (first layer only).
func expandBackground(value string) map[string]string {
	m := map[string]string{"background-color": "transparent", "background-image": "none", "background-size": "auto",
		"background-position": "0% 0%", "background-repeat": "repeat", "background-origin": "padding-box",
		"background-clip": "border-box"}
	layers := SplitCommas(value)
	// the color may only appear in the last layer; images come from the first
	last := SplitValue(layers[len(layers)-1])
	for _, t := range last {
		if _, ok := ParseColor(t, Black); ok && !strings.Contains(t, "(") || isColorFunc(t) {
			m["background-color"] = t
		}
	}
	var pos, size, rep []string
	inSize := false
	boxes := 0
	for _, t := range SplitValue(layers[0]) {
		lt := strings.ToLower(t)
		switch {
		case t == "/":
			inSize = true
			continue
		case strings.HasPrefix(lt, "url(") || strings.Contains(lt, "gradient("):
			m["background-image"] = t
		case lt == "repeat" || lt == "no-repeat" || lt == "repeat-x" || lt == "repeat-y" || lt == "space" || lt == "round":
			rep = append(rep, lt)
		case lt == "border-box" || lt == "padding-box" || lt == "content-box":
			if boxes == 0 {
				m["background-origin"] = lt
				m["background-clip"] = lt
			} else {
				m["background-clip"] = lt
			}
			boxes++
		case inSize && (lt == "cover" || lt == "contain" || lt == "auto" || startsNumeric(lt) || strings.HasPrefix(lt, "calc(")):
			size = append(size, t)
		case lt == "left" || lt == "right" || lt == "top" || lt == "bottom" || lt == "center" || startsNumeric(lt) || strings.HasPrefix(lt, "calc("):
			inSize = false
			pos = append(pos, t)
		}
	}
	if len(pos) > 0 {
		m["background-position"] = strings.Join(pos, " ")
	}
	if len(size) > 0 {
		m["background-size"] = strings.Join(size, " ")
	}
	if len(rep) > 0 {
		m["background-repeat"] = strings.Join(rep, " ")
	}
	return m
}

func isColorFunc(t string) bool {
	fn, _, ok := Function(t)
	if !ok {
		return false
	}
	switch fn {
	case "rgb", "rgba", "hsl", "hsla", "hwb", "lab", "lch", "oklab", "oklch", "color":
		_, ok := ParseColor(t, Black)
		return ok
	}
	return false
}

func startsNumeric(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return (c >= '0' && c <= '9') || c == '.' || ((c == '-' || c == '+') && len(s) > 1 && (s[1] == '.' || (s[1] >= '0' && s[1] <= '9')))
}

func fourSides(long []string, toks []string) map[string]string {
	var v [4]string
	switch len(toks) {
	case 1:
		v = [4]string{toks[0], toks[0], toks[0], toks[0]}
	case 2:
		v = [4]string{toks[0], toks[1], toks[0], toks[1]}
	case 3:
		v = [4]string{toks[0], toks[1], toks[2], toks[1]}
	case 4:
		v = [4]string{toks[0], toks[1], toks[2], toks[3]}
	default:
		return nil
	}
	return map[string]string{long[0]: v[0], long[1]: v[1], long[2]: v[2], long[3]: v[3]}
}

var systemFonts = map[string]bool{"caption": true, "icon": true, "menu": true, "message-box": true, "small-caption": true, "status-bar": true}

func expandFont(toks []string, raw string) map[string]string {
	if len(toks) == 1 && systemFonts[strings.ToLower(toks[0])] {
		return map[string]string{"font-style": "normal", "font-variant": "normal", "font-weight": "normal",
			"font-size": "13px", "line-height": "normal", "font-family": "system-ui"}
	}
	m := map[string]string{"font-style": "normal", "font-variant": "normal", "font-weight": "normal", "line-height": "normal"}
	i := 0
	for ; i < len(toks); i++ {
		lt := strings.ToLower(toks[i])
		switch {
		case lt == "normal":
		case lt == "italic" || lt == "oblique":
			m["font-style"] = lt
		case lt == "small-caps":
			m["font-variant"] = lt
		case lt == "bold" || lt == "bolder" || lt == "lighter":
			m["font-weight"] = lt
		case isWeightNumber(lt):
			m["font-weight"] = lt
		case strings.HasSuffix(lt, "condensed") || strings.HasSuffix(lt, "expanded"):
		default:
			goto size
		}
	}
size:
	if i >= len(toks) {
		return nil
	}
	m["font-size"] = toks[i]
	i++
	if i < len(toks) && toks[i] == "/" {
		if i+1 >= len(toks) {
			return nil
		}
		m["line-height"] = toks[i+1]
		i += 2
	}
	if i >= len(toks) {
		return nil
	}
	// family is the remaining raw text, commas included
	fam := strings.Join(toks[i:], " ")
	fam = strings.ReplaceAll(fam, " , ", ", ")
	m["font-family"] = fam
	return m
}

func isWeightNumber(s string) bool {
	f, ok := ParseNumber(s)
	return ok && f >= 1 && f <= 1000 && !strings.ContainsAny(s, "abcdefghijklmnopqrstuvwxyz%")
}
