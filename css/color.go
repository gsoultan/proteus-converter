// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package css

import (
	"math"
	"strconv"
	"strings"
)

// Color is an sRGB color with alpha, each component in [0, 1].
type Color struct {
	R, G, B, A float64
}

// Transparent is the fully transparent color.
var Transparent = Color{}

// Black is opaque black.
var Black = Color{0, 0, 0, 1}

// IsTransparent reports whether the color paints nothing.
func (c Color) IsTransparent() bool { return c.A <= 0 }

func rgb255(r, g, b int) Color {
	return Color{float64(r) / 255, float64(g) / 255, float64(b) / 255, 1}
}

// ParseColor parses a CSS color. currentColor is returned as cur.
func ParseColor(v string, cur Color) (Color, bool) {
	v = strings.TrimSpace(v)
	lv := strings.ToLower(v)
	switch lv {
	case "currentcolor":
		return cur, true
	case "transparent":
		return Transparent, true
	}
	if strings.HasPrefix(lv, "#") {
		return parseHex(lv[1:])
	}
	if fn, args, ok := Function(lv); ok {
		return parseColorFunc(fn, args)
	}
	if c, ok := namedColors[lv]; ok {
		return c, true
	}
	if c, ok := systemColors[lv]; ok {
		return c, true
	}
	return Color{}, false
}

func parseHex(h string) (Color, bool) {
	expand := func(s string) string {
		var b strings.Builder
		for i := 0; i < len(s); i++ {
			b.WriteByte(s[i])
			b.WriteByte(s[i])
		}
		return b.String()
	}
	switch len(h) {
	case 3, 4:
		h = expand(h)
	case 6, 8:
	default:
		return Color{}, false
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return Color{}, false
	}
	if len(h) == 6 {
		return Color{float64(n>>16&0xff) / 255, float64(n>>8&0xff) / 255, float64(n&0xff) / 255, 1}, true
	}
	return Color{float64(n>>24&0xff) / 255, float64(n>>16&0xff) / 255, float64(n>>8&0xff) / 255, float64(n&0xff) / 255}, true
}

// colorArgs splits modern (space / alpha) and legacy (comma) argument syntax.
func colorArgs(args string) []string {
	args = strings.ReplaceAll(args, ",", " ")
	args = strings.ReplaceAll(args, "/", " ")
	return strings.Fields(args)
}

func parseComponent(s string, scale float64) (float64, bool) {
	if s == "none" {
		return 0, true
	}
	if strings.HasSuffix(s, "%") {
		f, err := strconv.ParseFloat(s[:len(s)-1], 64)
		return f / 100, err == nil
	}
	f, err := strconv.ParseFloat(s, 64)
	return f / scale, err == nil
}

func parseAlpha(parts []string, i int) float64 {
	if len(parts) <= i {
		return 1
	}
	a, ok := parseComponent(parts[i], 1)
	if !ok {
		return 1
	}
	return clamp01(a)
}

func clamp01(f float64) float64 { return math.Max(0, math.Min(1, f)) }

func parseColorFunc(fn, args string) (Color, bool) {
	parts := colorArgs(args)
	switch fn {
	case "rgb", "rgba":
		if len(parts) < 3 {
			return Color{}, false
		}
		var c [3]float64
		for i := range 3 {
			v, ok := parseComponent(parts[i], 255)
			if !ok {
				return Color{}, false
			}
			c[i] = clamp01(v)
		}
		return Color{c[0], c[1], c[2], parseAlpha(parts, 3)}, true
	case "hsl", "hsla":
		if len(parts) < 3 {
			return Color{}, false
		}
		h, err := strconv.ParseFloat(strings.TrimSuffix(parts[0], "deg"), 64)
		if err != nil {
			return Color{}, false
		}
		s, ok1 := parseComponent(parts[1], 100)
		l, ok2 := parseComponent(parts[2], 100)
		if !ok1 || !ok2 {
			return Color{}, false
		}
		r, g, b := hslToRGB(h, clamp01(s), clamp01(l))
		return Color{r, g, b, parseAlpha(parts, 3)}, true
	}
	return Color{}, false
}

func hslToRGB(h, s, l float64) (float64, float64, float64) {
	h = math.Mod(math.Mod(h, 360)+360, 360) / 360
	f := func(n float64) float64 {
		k := math.Mod(n+h*12, 12)
		a := s * math.Min(l, 1-l)
		return l - a*math.Max(-1, math.Min(math.Min(k-3, 9-k), 1))
	}
	return f(0), f(8), f(4)
}

var systemColors = map[string]Color{
	"canvas": rgb255(255, 255, 255), "canvastext": rgb255(0, 0, 0), "linktext": rgb255(0, 0, 238),
	"visitedtext": rgb255(85, 26, 139), "graytext": rgb255(128, 128, 128), "buttonface": rgb255(239, 239, 239),
	"buttontext": rgb255(0, 0, 0), "field": rgb255(255, 255, 255), "fieldtext": rgb255(0, 0, 0),
	"highlight": rgb255(181, 213, 255), "highlighttext": rgb255(0, 0, 0), "mark": rgb255(255, 255, 0), "marktext": rgb255(0, 0, 0),
	"-webkit-link": rgb255(0, 0, 238),
}

var namedColors = map[string]Color{
	"aliceblue": rgb255(240, 248, 255), "antiquewhite": rgb255(250, 235, 215), "aqua": rgb255(0, 255, 255),
	"aquamarine": rgb255(127, 255, 212), "azure": rgb255(240, 255, 255), "beige": rgb255(245, 245, 220),
	"bisque": rgb255(255, 228, 196), "black": rgb255(0, 0, 0), "blanchedalmond": rgb255(255, 235, 205),
	"blue": rgb255(0, 0, 255), "blueviolet": rgb255(138, 43, 226), "brown": rgb255(165, 42, 42),
	"burlywood": rgb255(222, 184, 135), "cadetblue": rgb255(95, 158, 160), "chartreuse": rgb255(127, 255, 0),
	"chocolate": rgb255(210, 105, 30), "coral": rgb255(255, 127, 80), "cornflowerblue": rgb255(100, 149, 237),
	"cornsilk": rgb255(255, 248, 220), "crimson": rgb255(220, 20, 60), "cyan": rgb255(0, 255, 255),
	"darkblue": rgb255(0, 0, 139), "darkcyan": rgb255(0, 139, 139), "darkgoldenrod": rgb255(184, 134, 11),
	"darkgray": rgb255(169, 169, 169), "darkgreen": rgb255(0, 100, 0), "darkgrey": rgb255(169, 169, 169),
	"darkkhaki": rgb255(189, 183, 107), "darkmagenta": rgb255(139, 0, 139), "darkolivegreen": rgb255(85, 107, 47),
	"darkorange": rgb255(255, 140, 0), "darkorchid": rgb255(153, 50, 204), "darkred": rgb255(139, 0, 0),
	"darksalmon": rgb255(233, 150, 122), "darkseagreen": rgb255(143, 188, 143), "darkslateblue": rgb255(72, 61, 139),
	"darkslategray": rgb255(47, 79, 79), "darkslategrey": rgb255(47, 79, 79), "darkturquoise": rgb255(0, 206, 209),
	"darkviolet": rgb255(148, 0, 211), "deeppink": rgb255(255, 20, 147), "deepskyblue": rgb255(0, 191, 255),
	"dimgray": rgb255(105, 105, 105), "dimgrey": rgb255(105, 105, 105), "dodgerblue": rgb255(30, 144, 255),
	"firebrick": rgb255(178, 34, 34), "floralwhite": rgb255(255, 250, 240), "forestgreen": rgb255(34, 139, 34),
	"fuchsia": rgb255(255, 0, 255), "gainsboro": rgb255(220, 220, 220), "ghostwhite": rgb255(248, 248, 255),
	"gold": rgb255(255, 215, 0), "goldenrod": rgb255(218, 165, 32), "gray": rgb255(128, 128, 128),
	"green": rgb255(0, 128, 0), "greenyellow": rgb255(173, 255, 47), "grey": rgb255(128, 128, 128),
	"honeydew": rgb255(240, 255, 240), "hotpink": rgb255(255, 105, 180), "indianred": rgb255(205, 92, 92),
	"indigo": rgb255(75, 0, 130), "ivory": rgb255(255, 255, 240), "khaki": rgb255(240, 230, 140),
	"lavender": rgb255(230, 230, 250), "lavenderblush": rgb255(255, 240, 245), "lawngreen": rgb255(124, 252, 0),
	"lemonchiffon": rgb255(255, 250, 205), "lightblue": rgb255(173, 216, 230), "lightcoral": rgb255(240, 128, 128),
	"lightcyan": rgb255(224, 255, 255), "lightgoldenrodyellow": rgb255(250, 250, 210), "lightgray": rgb255(211, 211, 211),
	"lightgreen": rgb255(144, 238, 144), "lightgrey": rgb255(211, 211, 211), "lightpink": rgb255(255, 182, 193),
	"lightsalmon": rgb255(255, 160, 122), "lightseagreen": rgb255(32, 178, 170), "lightskyblue": rgb255(135, 206, 250),
	"lightslategray": rgb255(119, 136, 153), "lightslategrey": rgb255(119, 136, 153), "lightsteelblue": rgb255(176, 196, 222),
	"lightyellow": rgb255(255, 255, 224), "lime": rgb255(0, 255, 0), "limegreen": rgb255(50, 205, 50),
	"linen": rgb255(250, 240, 230), "magenta": rgb255(255, 0, 255), "maroon": rgb255(128, 0, 0),
	"mediumaquamarine": rgb255(102, 205, 170), "mediumblue": rgb255(0, 0, 205), "mediumorchid": rgb255(186, 85, 211),
	"mediumpurple": rgb255(147, 112, 219), "mediumseagreen": rgb255(60, 179, 113), "mediumslateblue": rgb255(123, 104, 238),
	"mediumspringgreen": rgb255(0, 250, 154), "mediumturquoise": rgb255(72, 209, 204), "mediumvioletred": rgb255(199, 21, 133),
	"midnightblue": rgb255(25, 25, 112), "mintcream": rgb255(245, 255, 250), "mistyrose": rgb255(255, 228, 225),
	"moccasin": rgb255(255, 228, 181), "navajowhite": rgb255(255, 222, 173), "navy": rgb255(0, 0, 128),
	"oldlace": rgb255(253, 245, 230), "olive": rgb255(128, 128, 0), "olivedrab": rgb255(107, 142, 35),
	"orange": rgb255(255, 165, 0), "orangered": rgb255(255, 69, 0), "orchid": rgb255(218, 112, 214),
	"palegoldenrod": rgb255(238, 232, 170), "palegreen": rgb255(152, 251, 152), "paleturquoise": rgb255(175, 238, 238),
	"palevioletred": rgb255(219, 112, 147), "papayawhip": rgb255(255, 239, 213), "peachpuff": rgb255(255, 218, 185),
	"peru": rgb255(205, 133, 63), "pink": rgb255(255, 192, 203), "plum": rgb255(221, 160, 221),
	"powderblue": rgb255(176, 224, 230), "purple": rgb255(128, 0, 128), "rebeccapurple": rgb255(102, 51, 153),
	"red": rgb255(255, 0, 0), "rosybrown": rgb255(188, 143, 143), "royalblue": rgb255(65, 105, 225),
	"saddlebrown": rgb255(139, 69, 19), "salmon": rgb255(250, 128, 114), "sandybrown": rgb255(244, 164, 96),
	"seagreen": rgb255(46, 139, 87), "seashell": rgb255(255, 245, 238), "sienna": rgb255(160, 82, 45),
	"silver": rgb255(192, 192, 192), "skyblue": rgb255(135, 206, 235), "slateblue": rgb255(106, 90, 205),
	"slategray": rgb255(112, 128, 144), "slategrey": rgb255(112, 128, 144), "snow": rgb255(255, 250, 250),
	"springgreen": rgb255(0, 255, 127), "steelblue": rgb255(70, 130, 180), "tan": rgb255(210, 180, 140),
	"teal": rgb255(0, 128, 128), "thistle": rgb255(216, 191, 216), "tomato": rgb255(255, 99, 71),
	"turquoise": rgb255(64, 224, 208), "violet": rgb255(238, 130, 238), "wheat": rgb255(245, 222, 179),
	"white": rgb255(255, 255, 255), "whitesmoke": rgb255(245, 245, 245), "yellow": rgb255(255, 255, 0),
	"yellowgreen": rgb255(154, 205, 50),
}
