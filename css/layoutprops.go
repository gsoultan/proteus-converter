package css

import (
	"strconv"
	"strings"
)

// TrackSize is one bound of a grid track sizing function.
type TrackSize struct {
	Kind TrackKind
	Len  Length  // TrackLength
	Fr   float64 // TrackFr
}

// TrackKind tells how a grid track bound is sized.
type TrackKind uint8

const (
	TrackAuto TrackKind = iota
	TrackLength
	TrackFr
	TrackMinContent
	TrackMaxContent
)

// Track is a grid track sizing function: minmax(Min, Max); a plain size has Min == Max
// (with fr and auto minimums resolved as in CSS Grid: fr means minmax(auto, fr)).
type Track struct {
	Min, Max TrackSize
}

// TrackList is a computed grid-template-columns/rows value.
type TrackList struct {
	Tracks []Track
	// AutoRepeat holds the tracks of a repeat(auto-fill|auto-fit, ...) inserted
	// before Tracks[AutoAt].
	AutoRepeat []Track
	AutoAt     int
	AutoFit    bool
}

// IsNone reports whether no explicit tracks are defined.
func (t TrackList) IsNone() bool { return len(t.Tracks) == 0 && len(t.AutoRepeat) == 0 }

// GridLine is a computed grid-row-start (and friends) value.
type GridLine struct {
	Line int    // explicit line number (1-based; negative counts from the end); 0 = auto
	Span int    // span count (0 = no span)
	Name string // named area or line
}

// IsAuto reports whether the line is 'auto'.
func (g GridLine) IsAuto() bool { return g.Line == 0 && g.Span == 0 && g.Name == "" }

// BackgroundSize is a computed background-size value.
type BackgroundSize struct {
	Mode string // "", "cover" or "contain"; "" uses W and H (Auto keeps the ratio)
	W, H Length
}

func parseTrackSize(tok string, c *computeCtx) (TrackSize, bool) {
	t := strings.ToLower(strings.TrimSpace(tok))
	switch t {
	case "auto":
		return TrackSize{Kind: TrackAuto}, true
	case "min-content":
		return TrackSize{Kind: TrackMinContent}, true
	case "max-content":
		return TrackSize{Kind: TrackMaxContent}, true
	}
	if strings.HasSuffix(t, "fr") {
		f, ok := ParseNumber(t[:len(t)-2])
		if !ok || f < 0 {
			return TrackSize{}, false
		}
		return TrackSize{Kind: TrackFr, Fr: f}, true
	}
	l, ok := ParseLength(t, c.lc, false)
	if !ok || l.Auto || l.Pt < 0 || l.Pct < 0 {
		return TrackSize{}, false
	}
	return TrackSize{Kind: TrackLength, Len: l}, true
}

func parseTrack(tok string, c *computeCtx) (Track, bool) {
	if fn, args, ok := Function(tok); ok {
		switch fn {
		case "minmax":
			parts := SplitCommas(args)
			if len(parts) != 2 {
				return Track{}, false
			}
			mn, ok1 := parseTrackSize(parts[0], c)
			mx, ok2 := parseTrackSize(parts[1], c)
			if !ok1 || !ok2 || mn.Kind == TrackFr {
				return Track{}, false
			}
			return Track{Min: mn, Max: mx}, true
		case "fit-content":
			ts, ok := parseTrackSize(args, c)
			if !ok {
				return Track{}, false
			}
			return Track{Min: TrackSize{Kind: TrackAuto}, Max: ts}, true
		}
		return Track{}, false
	}
	ts, ok := parseTrackSize(tok, c)
	if !ok {
		return Track{}, false
	}
	if ts.Kind == TrackFr {
		return Track{Min: TrackSize{Kind: TrackAuto}, Max: ts}, true
	}
	return Track{Min: ts, Max: ts}, true
}

// splitTrackTokens splits a track list, dropping [line names].
func splitTrackTokens(v string) []string {
	var out []string
	for _, t := range SplitValue(v) {
		if strings.HasPrefix(t, "[") {
			continue
		}
		out = append(out, t)
	}
	// SplitValue keeps "[a b]" groups only when written without spaces; drop leftovers
	var clean []string
	skip := false
	for _, t := range out {
		if skip {
			if strings.HasSuffix(t, "]") {
				skip = false
			}
			continue
		}
		if strings.HasSuffix(t, "]") {
			continue
		}
		clean = append(clean, t)
	}
	return clean
}

// ParseTrackList parses a grid-template-columns/rows value.
func parseTrackList(v string, c *computeCtx) (TrackList, bool) {
	v = strings.TrimSpace(v)
	if strings.EqualFold(v, "none") || v == "" {
		return TrackList{}, true
	}
	var tl TrackList
	for _, tok := range splitTrackTokens(v) {
		if fn, args, ok := Function(tok); ok && fn == "repeat" {
			parts := SplitCommas(args)
			if len(parts) != 2 {
				return TrackList{}, false
			}
			var inner []Track
			for _, it := range splitTrackTokens(parts[1]) {
				tr, ok := parseTrack(it, c)
				if !ok {
					return TrackList{}, false
				}
				inner = append(inner, tr)
			}
			if len(inner) == 0 {
				return TrackList{}, false
			}
			switch strings.ToLower(strings.TrimSpace(parts[0])) {
			case "auto-fill", "auto-fit":
				if tl.AutoRepeat != nil {
					return TrackList{}, false
				}
				tl.AutoRepeat = inner
				tl.AutoAt = len(tl.Tracks)
				tl.AutoFit = strings.EqualFold(strings.TrimSpace(parts[0]), "auto-fit")
			default:
				n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
				if err != nil || n < 1 {
					return TrackList{}, false
				}
				n = min(n, 1000)
				for range n {
					tl.Tracks = append(tl.Tracks, inner...)
				}
			}
			continue
		}
		tr, ok := parseTrack(tok, c)
		if !ok {
			return TrackList{}, false
		}
		tl.Tracks = append(tl.Tracks, tr)
	}
	return tl, true
}

func parseGridLine(v string) (GridLine, bool) {
	toks := strings.Fields(strings.ToLower(strings.TrimSpace(v)))
	if len(toks) == 0 {
		return GridLine{}, false
	}
	if len(toks) == 1 && toks[0] == "auto" {
		return GridLine{}, true
	}
	var g GridLine
	span := false
	for _, t := range toks {
		if t == "span" {
			span = true
			continue
		}
		if n, err := strconv.Atoi(t); err == nil {
			if n == 0 {
				return GridLine{}, false
			}
			if span {
				if n < 0 {
					return GridLine{}, false
				}
				g.Span = n
			} else {
				g.Line = n
			}
			continue
		}
		g.Name = t
	}
	if span && g.Span == 0 {
		g.Span = 1
	}
	return g, true
}

func parseAreas(v string) ([][]string, bool) {
	v = strings.TrimSpace(v)
	if strings.EqualFold(v, "none") {
		return nil, true
	}
	var rows [][]string
	for _, t := range SplitValue(v) {
		if len(t) == 0 || (t[0] != '"' && t[0] != '\'') {
			return nil, false
		}
		var row []string
		for _, cell := range strings.Fields(Unquote(t)) {
			if strings.Trim(cell, ".") == "" {
				cell = "."
			}
			row = append(row, cell)
		}
		if len(rows) > 0 && len(row) != len(rows[0]) {
			return nil, false
		}
		rows = append(rows, row)
	}
	return rows, len(rows) > 0
}

// parsePosition parses a background-position/object-position value into x and y.
func parsePosition(v string, c *computeCtx) ([2]Length, bool) {
	toks := strings.Fields(strings.ToLower(strings.TrimSpace(v)))
	if len(toks) == 0 || len(toks) > 4 {
		return [2]Length{}, false
	}
	kw := map[string]Length{"left": {Pct: 0}, "center": {Pct: 50}, "right": {Pct: 100}, "top": {Pct: 0}, "bottom": {Pct: 100}}
	isX := func(t string) bool { return t == "left" || t == "right" }
	isY := func(t string) bool { return t == "top" || t == "bottom" }
	if len(toks) == 1 {
		t := toks[0]
		if isY(t) {
			return [2]Length{{Pct: 50}, kw[t]}, true
		}
		if l, ok := kw[t]; ok {
			return [2]Length{l, {Pct: 50}}, true
		}
		l, ok := ParseLength(t, c.lc, false)
		if !ok || l.Auto {
			return [2]Length{}, false
		}
		return [2]Length{l, {Pct: 50}}, true
	}
	// four-value form: edge offset edge offset
	if len(toks) >= 3 {
		var out [2]Length
		out[0], out[1] = Length{Pct: 50}, Length{Pct: 50}
		for i := 0; i < len(toks); i++ {
			edge := toks[i]
			if _, ok := kw[edge]; !ok {
				return [2]Length{}, false
			}
			off := Length{}
			if i+1 < len(toks) {
				if l, ok := ParseLength(toks[i+1], c.lc, false); ok && !l.Auto {
					off = l
					i++
				}
			}
			switch edge {
			case "left":
				out[0] = off
			case "right":
				out[0] = Length{Pt: -off.Pt, Pct: 100 - off.Pct}
			case "top":
				out[1] = off
			case "bottom":
				out[1] = Length{Pt: -off.Pt, Pct: 100 - off.Pct}
			case "center":
				// only valid without offset
			}
		}
		return out, true
	}
	a, b := toks[0], toks[1]
	if isY(a) || isX(b) {
		a, b = b, a
	}
	var out [2]Length
	for i, t := range []string{a, b} {
		if l, ok := kw[t]; ok {
			if (i == 0 && isY(t)) || (i == 1 && isX(t)) {
				return [2]Length{}, false
			}
			out[i] = l
			continue
		}
		l, ok := ParseLength(t, c.lc, false)
		if !ok || l.Auto {
			return [2]Length{}, false
		}
		out[i] = l
	}
	return out, true
}

func parseBackgroundSize(v string, c *computeCtx) (BackgroundSize, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	switch v {
	case "cover", "contain":
		return BackgroundSize{Mode: v}, true
	}
	toks := strings.Fields(v)
	if len(toks) == 0 || len(toks) > 2 {
		return BackgroundSize{}, false
	}
	out := BackgroundSize{W: Auto, H: Auto}
	for i, t := range toks {
		l, ok := ParseLength(t, c.lc, false)
		if !ok || (!l.Auto && (l.Pt < 0 || l.Pct < 0)) {
			return BackgroundSize{}, false
		}
		if i == 0 {
			out.W = l
		} else {
			out.H = l
		}
	}
	return out, true
}

func parseRepeat(v string) ([2]string, bool) {
	toks := strings.Fields(strings.ToLower(strings.TrimSpace(v)))
	valid := func(t string) bool {
		switch t {
		case "repeat", "no-repeat", "space", "round":
			return true
		}
		return false
	}
	switch len(toks) {
	case 1:
		switch toks[0] {
		case "repeat-x":
			return [2]string{"repeat", "no-repeat"}, true
		case "repeat-y":
			return [2]string{"no-repeat", "repeat"}, true
		}
		if valid(toks[0]) {
			return [2]string{toks[0], toks[0]}, true
		}
	case 2:
		if valid(toks[0]) && valid(toks[1]) {
			return [2]string{toks[0], toks[1]}, true
		}
	}
	return [2]string{}, false
}

func parseGap(v string, c *computeCtx) (Length, bool) {
	if strings.EqualFold(strings.TrimSpace(v), "normal") {
		return Length{}, true
	}
	l, ok := ParseLength(v, c.lc, false)
	if !ok || l.Auto || l.Pt < 0 || l.Pct < 0 {
		return Length{}, false
	}
	return l, true
}

func registerLayoutProps() {
	for i, side := range []string{"top", "right", "bottom", "left"} {
		i := i
		a, cp := lengthProp(func(s *Style) *Length { return &s.Inset[i] }, true, true)
		def(side, false, a, cp)
	}
	def("z-index", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		v = strings.TrimSpace(v)
		if strings.EqualFold(v, "auto") {
			s.ZIndex = 0
			return true
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return false
		}
		s.ZIndex = n
		return true
	}, func(d, s *Style) { d.ZIndex = s.ZIndex })

	a, cp := keywordProp(func(s *Style) *string { return &s.FlexDirection }, "row", "row-reverse", "column", "column-reverse")
	def("flex-direction", false, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.FlexWrap }, "nowrap", "wrap", "wrap-reverse")
	def("flex-wrap", false, a, cp)
	alignKw := func(get func(*Style) *string) (func(s, p *Style, v string, c *computeCtx) bool, func(d, s *Style)) {
		return func(s, _ *Style, v string, _ *computeCtx) bool {
				v = strings.ToLower(strings.TrimSpace(v))
				// drop overflow-position and baseline-position prefixes
				for _, p := range []string{"safe ", "unsafe ", "first ", "last "} {
					v = strings.TrimPrefix(v, p)
				}
				switch v {
				case "auto", "normal", "stretch", "start", "end", "center", "flex-start", "flex-end", "self-start",
					"self-end", "baseline", "left", "right", "space-between", "space-around", "space-evenly", "legacy":
					*get(s) = v
					return true
				}
				return false
			}, func(d, s *Style) {
				*get(d) = *get(s)
			}
	}
	a, cp = alignKw(func(s *Style) *string { return &s.JustifyContent })
	def("justify-content", false, a, cp)
	a, cp = alignKw(func(s *Style) *string { return &s.AlignItems })
	def("align-items", false, a, cp)
	a, cp = alignKw(func(s *Style) *string { return &s.AlignSelf })
	def("align-self", false, a, cp)
	a, cp = alignKw(func(s *Style) *string { return &s.AlignContent })
	def("align-content", false, a, cp)
	a, cp = alignKw(func(s *Style) *string { return &s.JustifyItems })
	def("justify-items", false, a, cp)
	a, cp = alignKw(func(s *Style) *string { return &s.JustifySelf })
	def("justify-self", false, a, cp)

	num := func(get func(*Style) *float64) (func(s, p *Style, v string, c *computeCtx) bool, func(d, s *Style)) {
		return func(s, _ *Style, v string, _ *computeCtx) bool {
				f, ok := ParseNumber(strings.TrimSpace(v))
				if !ok || f < 0 {
					return false
				}
				*get(s) = f
				return true
			}, func(d, s *Style) {
				*get(d) = *get(s)
			}
	}
	a, cp = num(func(s *Style) *float64 { return &s.FlexGrow })
	def("flex-grow", false, a, cp)
	a, cp = num(func(s *Style) *float64 { return &s.FlexShrink })
	def("flex-shrink", false, a, cp)
	def("flex-basis", false, func(s, _ *Style, v string, c *computeCtx) bool {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "content", "max-content", "min-content", "fit-content":
			s.FlexBasis = Auto
			return true
		}
		l, ok := ParseLength(v, c.lc, false)
		if !ok || (!l.Auto && (l.Pt < 0 || l.Pct < 0)) {
			return false
		}
		s.FlexBasis = l
		return true
	}, func(d, s *Style) { d.FlexBasis = s.FlexBasis })
	def("order", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return false
		}
		s.Order = n
		return true
	}, func(d, s *Style) { d.Order = s.Order })
	def("row-gap", false, func(s, _ *Style, v string, c *computeCtx) bool {
		l, ok := parseGap(v, c)
		if ok {
			s.RowGap = l
		}
		return ok
	}, func(d, s *Style) { d.RowGap = s.RowGap })
	def("column-gap", false, func(s, _ *Style, v string, c *computeCtx) bool {
		l, ok := parseGap(v, c)
		if ok {
			s.ColumnGap = l
		}
		return ok
	}, func(d, s *Style) { d.ColumnGap = s.ColumnGap })

	tracks := func(get func(*Style) *TrackList) (func(s, p *Style, v string, c *computeCtx) bool, func(d, s *Style)) {
		return func(s, _ *Style, v string, c *computeCtx) bool {
				tl, ok := parseTrackList(v, c)
				if ok {
					*get(s) = tl
				}
				return ok
			}, func(d, s *Style) {
				*get(d) = *get(s)
			}
	}
	a, cp = tracks(func(s *Style) *TrackList { return &s.GridTemplateColumns })
	def("grid-template-columns", false, a, cp)
	a, cp = tracks(func(s *Style) *TrackList { return &s.GridTemplateRows })
	def("grid-template-rows", false, a, cp)
	a, cp = tracks(func(s *Style) *TrackList { return &s.GridAutoColumns })
	def("grid-auto-columns", false, a, cp)
	a, cp = tracks(func(s *Style) *TrackList { return &s.GridAutoRows })
	def("grid-auto-rows", false, a, cp)
	def("grid-template-areas", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		areas, ok := parseAreas(v)
		if ok {
			s.GridTemplateAreas = areas
		}
		return ok
	}, func(d, s *Style) { d.GridTemplateAreas = s.GridTemplateAreas })
	def("grid-auto-flow", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		v = strings.ToLower(strings.TrimSpace(v))
		switch {
		case strings.Contains(v, "column"):
			s.GridAutoFlow = "column"
		case v == "row" || v == "dense" || v == "row dense" || v == "dense row":
			s.GridAutoFlow = "row"
		default:
			return false
		}
		return true
	}, func(d, s *Style) { d.GridAutoFlow = s.GridAutoFlow })
	for i, name := range []string{"grid-row-start", "grid-column-start", "grid-row-end", "grid-column-end"} {
		i := i
		def(name, false, func(s, _ *Style, v string, _ *computeCtx) bool {
			g, ok := parseGridLine(v)
			if ok {
				s.GridPlacement[i] = g
			}
			return ok
		}, func(d, s *Style) { d.GridPlacement[i] = s.GridPlacement[i] })
	}

	def("background-size", false, func(s, _ *Style, v string, c *computeCtx) bool {
		bs, ok := parseBackgroundSize(v, c)
		if ok {
			s.BackgroundSize = bs
		}
		return ok
	}, func(d, s *Style) { d.BackgroundSize = s.BackgroundSize })
	def("background-position", false, func(s, _ *Style, v string, c *computeCtx) bool {
		if i := strings.IndexByte(v, ','); i >= 0 {
			v = v[:i]
		}
		p, ok := parsePosition(v, c)
		if ok {
			s.BackgroundPosition = p
		}
		return ok
	}, func(d, s *Style) { d.BackgroundPosition = s.BackgroundPosition })
	def("background-repeat", false, func(s, _ *Style, v string, _ *computeCtx) bool {
		if i := strings.IndexByte(v, ','); i >= 0 {
			v = v[:i]
		}
		r, ok := parseRepeat(v)
		if ok {
			s.BackgroundRepeat = r
		}
		return ok
	}, func(d, s *Style) { d.BackgroundRepeat = s.BackgroundRepeat })
	boxKw := func(get func(*Style) *string) (func(s, p *Style, v string, c *computeCtx) bool, func(d, s *Style)) {
		return func(s, _ *Style, v string, _ *computeCtx) bool {
				v = strings.ToLower(strings.TrimSpace(v))
				if i := strings.IndexByte(v, ','); i >= 0 {
					v = strings.TrimSpace(v[:i])
				}
				switch v {
				case "border-box", "padding-box", "content-box":
					*get(s) = v
					return true
				}
				return false
			}, func(d, s *Style) {
				*get(d) = *get(s)
			}
	}
	a, cp = boxKw(func(s *Style) *string { return &s.BackgroundOrigin })
	def("background-origin", false, a, cp)
	a, cp = boxKw(func(s *Style) *string { return &s.BackgroundClip })
	def("background-clip", false, a, cp)
	a, cp = keywordProp(func(s *Style) *string { return &s.ObjectFit }, "fill", "contain", "cover", "none", "scale-down")
	def("object-fit", false, a, cp)
	def("object-position", false, func(s, _ *Style, v string, c *computeCtx) bool {
		p, ok := parsePosition(v, c)
		if ok {
			s.ObjectPosition = p
		}
		return ok
	}, func(d, s *Style) { d.ObjectPosition = s.ObjectPosition })
}

// layout-related shorthands, expanded by expandLayoutShorthand.
func init() {
	shorthandLonghands["inset"] = []string{"top", "right", "bottom", "left"}
	shorthandLonghands["flex"] = []string{"flex-grow", "flex-shrink", "flex-basis"}
	shorthandLonghands["flex-flow"] = []string{"flex-direction", "flex-wrap"}
	shorthandLonghands["gap"] = []string{"row-gap", "column-gap"}
	shorthandLonghands["grid-gap"] = shorthandLonghands["gap"]
	shorthandLonghands["place-items"] = []string{"align-items", "justify-items"}
	shorthandLonghands["place-content"] = []string{"align-content", "justify-content"}
	shorthandLonghands["place-self"] = []string{"align-self", "justify-self"}
	shorthandLonghands["grid-row"] = []string{"grid-row-start", "grid-row-end"}
	shorthandLonghands["grid-column"] = []string{"grid-column-start", "grid-column-end"}
	shorthandLonghands["grid-area"] = []string{"grid-row-start", "grid-column-start", "grid-row-end", "grid-column-end"}
	shorthandLonghands["grid-template"] = []string{"grid-template-rows", "grid-template-columns", "grid-template-areas"}
	aliases["grid-row-gap"] = "row-gap"
	aliases["grid-column-gap"] = "column-gap"
	aliases["inset-block-start"] = "top"
	aliases["inset-block-end"] = "bottom"
	aliases["inset-inline-start"] = "left"
	aliases["inset-inline-end"] = "right"
	aliases["-webkit-flex-direction"] = "flex-direction"
	aliases["-webkit-flex-wrap"] = "flex-wrap"
	aliases["-webkit-justify-content"] = "justify-content"
	aliases["-webkit-align-items"] = "align-items"
	aliases["-webkit-align-self"] = "align-self"
	aliases["-webkit-flex-grow"] = "flex-grow"
	aliases["-webkit-flex-shrink"] = "flex-shrink"
	aliases["-webkit-flex-basis"] = "flex-basis"
	aliases["-webkit-order"] = "order"
	aliases["-webkit-background-size"] = "background-size"
	shorthandLonghands["-webkit-flex"] = shorthandLonghands["flex"]
}

// expandLayoutShorthand expands the flex, grid, gap and inset shorthands.
func expandLayoutShorthand(name string, toks []string, raw string) map[string]string {
	long := shorthandLonghands[name]
	switch name {
	case "inset":
		return fourSides(long, toks)
	case "flex", "-webkit-flex":
		m := map[string]string{"flex-grow": "0", "flex-shrink": "1", "flex-basis": "auto"}
		if len(toks) == 1 {
			switch strings.ToLower(toks[0]) {
			case "none":
				m["flex-shrink"] = "0"
				return m
			case "auto":
				m["flex-grow"] = "1"
				return m
			case "initial":
				return m
			}
		}
		var nums []string
		basis := ""
		for _, t := range toks {
			if _, ok := ParseNumber(t); ok && len(nums) < 2 {
				nums = append(nums, t)
				continue
			}
			if basis != "" {
				return nil
			}
			basis = t
		}
		if len(nums) == 0 && basis == "" {
			return nil
		}
		if len(nums) > 0 {
			m["flex-grow"] = nums[0]
			if len(nums) > 1 {
				m["flex-shrink"] = nums[1]
			}
			m["flex-basis"] = "0%"
		} else {
			m["flex-grow"] = "1"
		}
		if basis != "" {
			m["flex-basis"] = basis
		}
		return m
	case "flex-flow":
		m := map[string]string{"flex-direction": "row", "flex-wrap": "nowrap"}
		for _, t := range toks {
			switch strings.ToLower(t) {
			case "row", "row-reverse", "column", "column-reverse":
				m["flex-direction"] = t
			case "nowrap", "wrap", "wrap-reverse":
				m["flex-wrap"] = t
			default:
				return nil
			}
		}
		return m
	case "gap", "grid-gap", "place-items", "place-content", "place-self":
		if len(toks) == 1 {
			toks = append(toks, toks[0])
		}
		if len(toks) != 2 {
			return nil
		}
		return map[string]string{long[0]: toks[0], long[1]: toks[1]}
	case "grid-row", "grid-column":
		parts := strings.SplitN(raw, "/", 2)
		start := strings.TrimSpace(parts[0])
		end := "auto"
		if len(parts) == 2 {
			end = strings.TrimSpace(parts[1])
		} else if g, ok := parseGridLine(start); ok && g.Name != "" && g.Span == 0 && g.Line == 0 {
			end = start
		}
		return map[string]string{long[0]: start, long[1]: end}
	case "grid-area":
		parts := strings.Split(raw, "/")
		vals := make([]string, 4)
		for i := range parts {
			if i < 4 {
				vals[i] = strings.TrimSpace(parts[i])
			}
		}
		for i := range 4 {
			if vals[i] != "" {
				continue
			}
			// omitted values copy a custom-ident start, otherwise auto
			src := ""
			if i >= 2 {
				src = vals[i-2]
			} else if i == 1 {
				src = vals[0]
			}
			if g, ok := parseGridLine(src); ok && g.Name != "" && g.Span == 0 && g.Line == 0 {
				vals[i] = src
			} else {
				vals[i] = "auto"
			}
		}
		return map[string]string{long[0]: vals[0], long[1]: vals[1], long[2]: vals[2], long[3]: vals[3]}
	case "grid-template":
		parts := strings.SplitN(raw, "/", 2)
		if len(parts) != 2 || strings.ContainsAny(raw, "\"'") {
			return nil // area-string form is not supported
		}
		return map[string]string{"grid-template-rows": strings.TrimSpace(parts[0]),
			"grid-template-columns": strings.TrimSpace(parts[1]), "grid-template-areas": "none"}
	}
	return nil
}
