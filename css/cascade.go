package css

import (
	"slices"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// Origin is the origin of a stylesheet in the cascade.
type Origin int

const (
	OriginUserAgent Origin = iota
	OriginUser
	OriginAuthor
)

// Sheet is a stylesheet with its cascade origin.
type Sheet struct {
	*Stylesheet
	Origin Origin
}

// ElementStyle holds the computed style of an element and its generated pseudo-elements.
type ElementStyle struct {
	Style  *Style
	Before *Style // nil when not generated
	After  *Style
	Marker *Style // ::marker (list items)
}

type compiledRule struct {
	sel    cascadia.Sel
	spec   [4]int // inline, id, class, type
	pseudo string
	decls  []Declaration
	order  int
	origin Origin
}

// Cascade matches rules against an HTML tree and computes styles.
type Cascade struct {
	rules []compiledRule
	byKey map[string][]int // rule indexes keyed by rightmost id/class/tag; "" = universal
	ctx   LengthContext
	warn  func(string)
}

// NewCascade compiles the given sheets. Sheets must be in document order
// within each origin. vw/vh resolve against viewportW/H (points).
func NewCascade(sheets []Sheet, viewportW, viewportH float64, warn func(string)) *Cascade {
	c := &Cascade{byKey: map[string][]int{}, warn: warn}
	c.ctx = LengthContext{ViewportW: viewportW, ViewportH: viewportH}
	order := 0
	for _, sh := range sheets {
		if sh.Stylesheet == nil {
			continue
		}
		for _, r := range sh.Rules {
			var decls []Declaration
			for _, d := range r.Decls {
				decls = append(decls, Expand(d)...)
			}
			if len(decls) == 0 {
				continue
			}
			for _, selText := range SplitCommas(r.Selector) {
				sel, err := cascadia.ParseWithPseudoElement(selText)
				if err != nil {
					if !isIgnorableSelector(selText) && warn != nil {
						warn("css: unsupported selector " + strings.TrimSpace(selText))
					}
					continue
				}
				sp := sel.Specificity()
				c.rules = append(c.rules, compiledRule{
					sel: sel, spec: [4]int{0, sp[0], sp[1], sp[2]}, pseudo: sel.PseudoElement(),
					decls: decls, order: order, origin: sh.Origin,
				})
				key := selectorKey(selText)
				c.byKey[key] = append(c.byKey[key], len(c.rules)-1)
				order++
			}
		}
	}
	return c
}

func isIgnorableSelector(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "::-") || strings.Contains(s, ":-moz") || strings.Contains(s, ":-webkit") ||
		strings.Contains(s, ":-ms") || strings.Contains(s, "::selection") || strings.Contains(s, "::placeholder")
}

// selectorKey extracts the most selective simple selector of the rightmost
// compound (#id, .class or tag) to pre-filter candidate rules.
func selectorKey(sel string) string {
	sel = strings.TrimSpace(sel)
	if i := strings.Index(sel, "::"); i >= 0 {
		sel = sel[:i]
	} else if strings.HasSuffix(strings.ToLower(sel), ":before") || strings.HasSuffix(strings.ToLower(sel), ":after") {
		sel = sel[:strings.LastIndex(sel, ":")]
	}
	// rightmost compound
	last := sel
	depth := 0
	for i := len(sel) - 1; i >= 0; i-- {
		ch := sel[i]
		if ch == ')' || ch == ']' {
			depth++
		} else if ch == '(' || ch == '[' {
			depth--
		} else if depth == 0 && (ch == ' ' || ch == '>' || ch == '+' || ch == '~') {
			last = sel[i+1:]
			break
		}
	}
	if strings.ContainsAny(last, "([\\") {
		return ""
	}
	if i := strings.IndexByte(last, '#'); i >= 0 {
		id := last[i+1:]
		if j := strings.IndexAny(id, ".:#"); j >= 0 {
			id = id[:j]
		}
		return "#" + id
	}
	if i := strings.IndexByte(last, '.'); i >= 0 {
		cl := last[i+1:]
		if j := strings.IndexAny(cl, ".:#"); j >= 0 {
			cl = cl[:j]
		}
		return "." + cl
	}
	tag := last
	if j := strings.IndexAny(tag, ".:#"); j >= 0 {
		tag = tag[:j]
	}
	if tag == "*" || strings.Contains(tag, "|") {
		return ""
	}
	return strings.ToLower(tag)
}

func (c *Cascade) candidates(n *html.Node) []int {
	var out []int
	out = append(out, c.byKey[""]...)
	out = append(out, c.byKey[n.Data]...)
	for _, a := range n.Attr {
		switch a.Key {
		case "id":
			out = append(out, c.byKey["#"+a.Val]...)
		case "class":
			for _, cl := range strings.Fields(a.Val) {
				out = append(out, c.byKey["."+cl]...)
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

type matchedDecl struct {
	Declaration
	rank  int
	spec  [4]int
	order int
}

func rankOf(o Origin, important bool) int {
	if !important {
		return int(o) // UA 0, user 1, author 2
	}
	return 5 - int(o) // author 3, user 4, UA 5
}

// Compute computes styles for every element under root.
func (c *Cascade) Compute(root *html.Node) map[*html.Node]*ElementStyle {
	out := map[*html.Node]*ElementStyle{}
	initial := Initial()
	var walk func(n *html.Node, parent *Style)
	walk = func(n *html.Node, parent *Style) {
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type != html.ElementNode {
				continue
			}
			es := c.computeElement(ch, parent, initial)
			out[ch] = es
			walk(ch, es.Style)
		}
	}
	walk(root, nil)
	return out
}

func (c *Cascade) computeElement(n *html.Node, parent *Style, initial *Style) *ElementStyle {
	byPseudo := map[string][]matchedDecl{}
	add := func(pseudo string, d Declaration, origin Origin, spec [4]int, order int) {
		byPseudo[pseudo] = append(byPseudo[pseudo], matchedDecl{Declaration: d, rank: rankOf(origin, d.Important), spec: spec, order: order})
	}
	for _, d := range PresentationalHints(n) {
		for _, e := range Expand(d) {
			add("", e, OriginAuthor, [4]int{}, -1)
		}
	}
	for _, idx := range c.candidates(n) {
		r := &c.rules[idx]
		if !r.sel.Match(n) {
			continue
		}
		pseudo := r.pseudo
		for _, d := range r.decls {
			add(pseudo, d, r.origin, r.spec, r.order)
		}
	}
	for _, a := range n.Attr {
		if a.Key == "style" {
			for _, d := range ParseDeclarations(a.Val) {
				for _, e := range Expand(d) {
					add("", e, OriginAuthor, [4]int{1, 0, 0, 0}, 1<<30)
				}
			}
		}
	}
	if parent == nil {
		parent = initial
	}
	es := &ElementStyle{}
	isRoot := n.Parent == nil || n.Parent.Type == html.DocumentNode
	es.Style = c.computeStyle(byPseudo[""], parent, initial, isRoot)
	if isRoot {
		c.ctx.RootFontSize = es.Style.FontSize
	}
	for _, ps := range []string{"before", "after", "marker"} {
		decls, ok := byPseudo[ps]
		if !ok && ps != "marker" {
			continue
		}
		st := c.computeStyle(decls, es.Style, initial, false)
		switch ps {
		case "before":
			if !st.ContentNone && st.Content != "" {
				es.Before = st
			}
		case "after":
			if !st.ContentNone && st.Content != "" {
				es.After = st
			}
		case "marker":
			if es.Style.Display == "list-item" {
				st.Display = "inline"
				es.Marker = st
			}
		}
	}
	return es
}

func sortDecls(ds []matchedDecl) {
	slices.SortStableFunc(ds, func(a, b matchedDecl) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		for i := range 4 {
			if a.spec[i] != b.spec[i] {
				return a.spec[i] - b.spec[i]
			}
		}
		return a.order - b.order
	})
}

// computeStyle applies cascaded declarations on top of the inherited style.
func (c *Cascade) computeStyle(decls []matchedDecl, parent, initial *Style, isRoot bool) *Style {
	sortDecls(decls)
	declared := map[string]string{}
	var custom []Declaration
	for _, d := range decls {
		if strings.HasPrefix(d.Property, "--") {
			custom = append(custom, d.Declaration)
			continue
		}
		declared[d.Property] = d.Value
	}
	s := parent.Inherit()
	if isRoot {
		s = initial.Inherit()
	}
	// custom properties (inherited)
	if len(custom) > 0 {
		m := make(map[string]string, len(parent.Custom)+len(custom))
		for k, v := range parent.Custom {
			m[k] = v
		}
		for _, d := range custom {
			v := d.Value
			if strings.Contains(v, "var(") {
				if sub, ok := substituteVars(v, m, 0); ok {
					v = sub
				} else {
					delete(m, d.Property)
					continue
				}
			}
			if isGlobalKeyword(v) {
				switch strings.ToLower(strings.TrimSpace(v)) {
				case "initial":
					delete(m, d.Property)
				default:
					if pv, ok := parent.Custom[d.Property]; ok {
						m[d.Property] = pv
					}
				}
				continue
			}
			m[d.Property] = v
		}
		s.Custom = m
	}
	lc := c.ctx
	lc.FontSize = parent.FontSize
	if lc.RootFontSize == 0 {
		lc.RootFontSize = 12
	}
	cc := &computeCtx{lc: lc}
	apply := func(name string) {
		v, ok := declared[name]
		if !ok {
			return
		}
		if strings.HasPrefix(v, pendingMarker) {
			rest := v[len(pendingMarker):]
			sep := strings.IndexByte(rest, 0)
			short, raw := rest[:sep], rest[sep+1:]
			sub, ok := substituteVars(raw, s.Custom, 0)
			if !ok {
				v = "unset"
			} else if vals := expandShorthand(short, sub); vals != nil {
				if lv, ok := vals[name]; ok {
					v = lv
				} else {
					return
				}
			} else {
				v = "unset"
			}
		} else if strings.Contains(v, "var(") {
			sub, ok := substituteVars(v, s.Custom, 0)
			if !ok {
				v = "unset"
			} else {
				v = sub
			}
		}
		pd := props[name]
		if pd == nil {
			return
		}
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "inherit":
			pd.copy(s, parent)
			return
		case "initial":
			pd.copy(s, initial)
			return
		case "unset", "revert", "revert-layer":
			if pd.inherited {
				pd.copy(s, parent)
			} else {
				pd.copy(s, initial)
			}
			return
		}
		if !pd.apply(s, parent, v, cc) && c.warn != nil {
			c.warn("css: invalid value for " + name + ": " + v)
		}
	}
	apply("font-family")
	apply("font-size")
	if s.keywordSize > 0 {
		if isMonospaceOnly(s.FontFamily) {
			s.FontSize = s.keywordSize * 13 / 16
		} else {
			s.FontSize = s.keywordSize
		}
	}
	cc.lc.FontSize = s.FontSize
	if isRoot {
		cc.lc.RootFontSize = s.FontSize
	}
	apply("color")
	for name := range declared {
		switch name {
		case "font-family", "font-size", "color":
			continue
		}
		apply(name)
	}
	// border colors default to currentColor
	for i := range 4 {
		if _, ok := declared[[]string{"border-top-color", "border-right-color", "border-bottom-color", "border-left-color"}[i]]; !ok {
			s.BorderColor[i] = s.Color
		}
	}
	// blockification of floats/absolutes and the root
	if s.IsAbsolute() {
		s.Float = "none"
	}
	if isRoot || s.Float != "none" || s.IsAbsolute() || (!isRoot && (parent.IsFlex() || parent.IsGrid())) {
		s.Display = blockify(s.Display)
	}
	if !isRoot && (parent.IsFlex() || parent.IsGrid()) {
		s.Float = "none" // floats do not apply to flex and grid items
	}
	return s
}

func blockify(d string) string {
	switch d {
	case "inline", "inline-block", "table-row-group", "table-column", "table-column-group",
		"table-header-group", "table-footer-group", "table-row", "table-cell", "table-caption":
		return "block"
	case "inline-table":
		return "table"
	case "inline-flex":
		return "flex"
	case "inline-grid":
		return "grid"
	}
	return d
}

// substituteVars replaces var(--x, fallback) references.
func substituteVars(v string, custom map[string]string, depth int) (string, bool) {
	if depth > 16 {
		return "", false
	}
	for {
		i := strings.Index(strings.ToLower(v), "var(")
		if i < 0 {
			return v, true
		}
		// find matching paren
		depthP := 0
		end := -1
		for j := i + 3; j < len(v); j++ {
			if v[j] == '(' {
				depthP++
			} else if v[j] == ')' {
				depthP--
				if depthP == 0 {
					end = j
					break
				}
			}
		}
		if end < 0 {
			return "", false
		}
		inner := v[i+4 : end]
		name, fallback, hasFallback := strings.Cut(inner, ",")
		name = strings.TrimSpace(name)
		val, ok := custom[name]
		if !ok || strings.TrimSpace(val) == "" {
			if !hasFallback {
				return "", false
			}
			val = strings.TrimSpace(fallback)
		}
		if strings.Contains(val, "var(") {
			sub, ok := substituteVars(val, custom, depth+1)
			if !ok {
				return "", false
			}
			val = sub
		}
		v = v[:i] + val + v[end+1:]
	}
}
