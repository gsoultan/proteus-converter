package layout

import (
	"math"
	"sort"

	"github.com/gsoultan/proteus-converter/css"
)

// flexItem is one item of a flex container during layout. Sizes are border-box.
type flexItem struct {
	box      *Box
	s        *css.Style
	base     float64 // flex base size
	hyp      float64 // hypothetical main size
	min, max float64 // main size bounds
	target   float64
	frozen   bool
	// margins: main start/end, cross start/end
	mms, mme, mcs, mce         float64
	autoMS, autoME             bool
	autoCS, autoCE             bool
	cross                      float64 // border-box cross size
	baseline                   float64 // from the border-box top (row containers)
	hasBaseline                bool
	align                      string
	sub                        *layouter
	res                        blockResult
	mainPos, crossPos          float64 // border-box offsets within the content box
	stretched, definiteCross   bool
	definiteMain, isReplaced   bool
	contentMinMain, contentMax float64
}

type flexLine struct {
	items          []*flexItem
	cross          float64
	maxAscent      float64
	pos            float64
	mainUsedOuters float64
}

// flexItems returns the in-flow children of a flex or grid container in
// order-modified document order, wrapping bare inline content in an anonymous block.
func (l *layouter) flexItems(b *Box, x, y float64) []*Box {
	var kids []*Box
	if !hasBlockChild(b) {
		var run []*Box
		for _, c := range b.Children {
			if c.outOfFlow() {
				if c.Style.IsAbsolute() {
					l.addAbs(c, x, y)
				}
				continue
			}
			run = append(run, c)
		}
		if len(run) > 0 && !allCollapsibleSpace(run) {
			a := anonymousBlock(b)
			a.Children = run
			kids = append(kids, a)
		}
		return kids
	}
	for _, c := range b.Children {
		if c.outOfFlow() && c.Style.IsAbsolute() {
			l.addAbs(c, x, y)
			continue
		}
		kids = append(kids, c)
	}
	sort.SliceStable(kids, func(i, j int) bool { return kids[i].Style.Order < kids[j].Style.Order })
	return kids
}

func resolveAlign(container, self string) string {
	a := self
	if a == "auto" || a == "" {
		a = container
	}
	switch a {
	case "normal", "stretch", "":
		return "stretch"
	case "start", "flex-start", "self-start", "left":
		return "start"
	case "end", "flex-end", "self-end", "right":
		return "end"
	}
	return a
}

// layoutFlex lays out the items of flex container b in its content box at
// (x, y) with width cw; defH is the definite content height or -1.
func (l *layouter) layoutFlex(b *Box, x, y, cw, defH float64) contentsResult {
	s := b.Style
	row := s.FlexDirection == "row" || s.FlexDirection == "row-reverse"
	reverse := s.FlexDirection == "row-reverse" || s.FlexDirection == "column-reverse"
	wrap := s.FlexWrap != "nowrap"
	kids := l.flexItems(b, x, y)
	res := contentsResult{placed: true, firstTop: y, end: y}
	if len(kids) == 0 {
		res.end = y + math.Max(defH, 0)
		res.placed = defH > 0
		return res
	}
	mainGap := s.ColumnGap.Resolve(cw, 0)
	crossGap := s.RowGap.Resolve(cw, 0)
	if !row {
		mainGap, crossGap = s.RowGap.Resolve(cw, 0), s.ColumnGap.Resolve(cw, 0)
	}
	availMain := cw
	if !row {
		availMain = defH // -1 when indefinite
	}

	items := make([]*flexItem, len(kids))
	for i, k := range kids {
		items[i] = l.flexItem(k, s, row, cw, defH)
	}

	// collect lines
	var lines []*flexLine
	cur := &flexLine{}
	used := 0.0
	for _, it := range items {
		outer := it.hyp + it.mms + it.mme
		if wrap && availMain >= 0 && len(cur.items) > 0 && used+mainGap+outer > availMain+eps {
			lines = append(lines, cur)
			cur = &flexLine{}
			used = 0
		}
		if len(cur.items) > 0 {
			used += mainGap
		}
		used += outer
		cur.items = append(cur.items, it)
	}
	lines = append(lines, cur)

	// main sizes
	for _, ln := range lines {
		avail := availMain
		if avail < 0 {
			// indefinite main size (column with auto height): items keep their hypothetical sizes
			for _, it := range ln.items {
				it.target = it.hyp
			}
			continue
		}
		resolveFlexible(ln.items, avail-mainGap*float64(len(ln.items)-1))
	}

	// cross sizes: lay out each item at its main size
	for _, it := range items {
		l.flexLayoutItem(it, row, cw, defH)
	}
	for _, ln := range lines {
		maxDesc := 0.0
		for _, it := range ln.items {
			outer := it.cross + it.mcs + it.mce
			if row && it.align == "baseline" && it.hasBaseline {
				asc := it.baseline + it.mcs
				ln.maxAscent = math.Max(ln.maxAscent, asc)
				maxDesc = math.Max(maxDesc, outer-asc)
				continue
			}
			ln.cross = math.Max(ln.cross, outer)
		}
		ln.cross = math.Max(ln.cross, ln.maxAscent+maxDesc)
	}
	containerCross := defH
	if !row {
		containerCross = cw
	}
	if len(lines) == 1 && containerCross >= 0 {
		lines[0].cross = containerCross
	}

	// align-content: distribute the free cross space between lines
	totalCross := crossGap * float64(len(lines)-1)
	for _, ln := range lines {
		totalCross += ln.cross
	}
	crossOffset, crossBetween := 0.0, crossGap
	if containerCross >= 0 && len(lines) > 1 {
		free := containerCross - totalCross
		switch s.AlignContent {
		case "normal", "stretch":
			if free > 0 {
				for _, ln := range lines {
					ln.cross += free / float64(len(lines))
				}
			}
		case "center":
			crossOffset = free / 2
		case "end", "flex-end":
			crossOffset = free
		case "space-between":
			if free > 0 {
				crossBetween += free / float64(len(lines)-1)
			}
		case "space-around":
			if free > 0 {
				crossBetween += free / float64(len(lines))
				crossOffset = free / float64(len(lines)) / 2
			}
		case "space-evenly":
			if free > 0 {
				crossBetween += free / float64(len(lines)+1)
				crossOffset = free / float64(len(lines)+1)
			}
		}
	}
	pos := crossOffset
	for _, ln := range lines {
		ln.pos = pos
		pos += ln.cross + crossBetween
	}
	if s.FlexWrap == "wrap-reverse" {
		total := pos - crossBetween + crossOffset
		if containerCross >= 0 {
			total = containerCross
		}
		for _, ln := range lines {
			ln.pos = total - ln.pos - ln.cross
		}
	}

	// stretch, then main and cross positions
	for _, ln := range lines {
		for _, it := range ln.items {
			if it.align == "stretch" && !it.definiteCross && !it.autoCS && !it.autoCE {
				want := math.Max(ln.cross-it.mcs-it.mce, 0)
				if math.Abs(want-it.cross) > eps {
					it.cross = want
					it.stretched = true
					l.flexLayoutItem(it, row, cw, defH)
					it.cross = want
				}
			}
		}
		l.flexMainPositions(ln, s, row, reverse, availMain, mainGap)
		for _, it := range ln.items {
			free := ln.cross - it.cross - it.mcs - it.mce
			off := 0.0
			switch {
			case it.autoCS && it.autoCE:
				off = math.Max(free, 0) / 2
			case it.autoCS:
				off = math.Max(free, 0)
			case it.autoCE:
			default:
				switch it.align {
				case "end":
					off = free
				case "center":
					off = free / 2
				case "baseline":
					if row && it.hasBaseline {
						off = ln.maxAscent - it.baseline - it.mcs
					}
				}
			}
			it.crossPos = ln.pos + it.mcs + off
		}
	}

	// container main size for columns with auto height
	contentH := defH
	if row {
		if contentH < 0 {
			contentH = math.Max(pos-crossBetween, 0)
			if len(lines) > 0 {
				contentH = lines[len(lines)-1].pos + lines[len(lines)-1].cross
				if s.FlexWrap == "wrap-reverse" {
					contentH = totalCross
				}
			}
		}
	} else if contentH < 0 {
		contentH = 0
		for _, ln := range lines {
			contentH = math.Max(contentH, ln.mainUsedOuters)
		}
	}

	// emit items, pushing rows (or column items) that would straddle a page
	shift := 0.0
	firstLine := true
	for _, ln := range lines {
		if row && l.fragment && !reverse {
			top := y + ln.pos + shift
			if ny := l.place(top, ln.cross); ny != top {
				shift += ny - top
			}
		}
		for _, it := range ln.items {
			var bx, by float64
			if row {
				bx, by = x+it.mainPos, y+it.crossPos+shift
			} else {
				bx, by = x+it.crossPos, y+it.mainPos+shift
				if l.fragment && !reverse {
					if ny := l.place(by-it.mms, it.target+it.mms+it.mme); ny != by-it.mms {
						shift += ny - (by - it.mms)
						by = ny + it.mms
					}
				}
			}
			dx, dy := bx-it.res.bx, by-it.res.top
			translateItems(it.sub.items, dx, dy)
			l.items = append(l.items, it.sub.items...)
			if it.res.hasBaseline && firstLine && !res.hasBaseline {
				res.firstBaseline = it.res.firstBaseline + dy
				res.hasBaseline = true
			}
			if it.res.hasBaseline {
				res.lastBaseline = it.res.lastBaseline + dy
			}
		}
		firstLine = false
	}
	res.end = y + contentH + shift
	if !res.hasBaseline {
		res.firstBaseline, res.lastBaseline = res.end, res.end
	}
	return res
}

// flexItem computes the flex base size and bounds of one item.
func (l *layouter) flexItem(b *Box, cs *css.Style, row bool, cw, defH float64) *flexItem {
	s := l.style(b)
	it := &flexItem{box: b, s: s, isReplaced: b.Kind == KindImage || b.Kind == KindCheckbox}
	it.align = resolveAlign(cs.AlignItems, s.AlignSelf)
	e := l.hEdges(s, cw)
	mt, mb := s.Margin[css.Top].Resolve(cw, 0), s.Margin[css.Bottom].Resolve(cw, 0)
	padW := e.bl + e.br + e.pl + e.pr
	padH := s.BorderUsed(css.Top) + s.BorderUsed(css.Bottom) + s.Padding[css.Top].Resolve(cw, 0) + s.Padding[css.Bottom].Resolve(cw, 0)
	toBorder := func(v, pad float64) float64 {
		if s.BoxSizing == "border-box" {
			return math.Max(v, pad)
		}
		return v + pad
	}
	if row {
		it.mms, it.mme, it.autoMS, it.autoME = e.ml, e.mr, e.mlAuto, e.mrAuto
		it.mcs, it.mce = mt, mb
		it.autoCS, it.autoCE = s.Margin[css.Top].Auto, s.Margin[css.Bottom].Auto
	} else {
		it.mms, it.mme = mt, mb
		it.autoMS, it.autoME = s.Margin[css.Top].Auto, s.Margin[css.Bottom].Auto
		it.mcs, it.mce, it.autoCS, it.autoCE = e.ml, e.mr, e.mlAuto, e.mrAuto
	}
	if it.autoMS {
		it.mms = 0
	}
	if it.autoME {
		it.mme = 0
	}
	if it.autoCS {
		it.mcs = 0
	}
	if it.autoCE {
		it.mce = 0
	}
	mn, mx := l.intrinsicWidths(b)
	marg := absPart(s.Margin[css.Left]) + absPart(s.Margin[css.Right])
	mn, mx = mn-marg, mx-marg
	if row {
		it.definiteCross = !s.Height.Auto && !s.Height.HasPct()
		it.definiteMain = !s.Width.Auto
		basisSet := !s.FlexBasis.Auto && !(s.FlexBasis.HasPct() && cw < 0)
		switch {
		case basisSet:
			it.base = toBorder(s.FlexBasis.Resolve(cw, 0), padW)
		case !s.Width.Auto:
			it.base = toBorder(s.Width.Resolve(cw, 0), padW)
		case it.isReplaced:
			w, _ := l.replacedSize(b, cw)
			it.base = w + padW
		default:
			it.base = mx
		}
		it.min = 0
		switch {
		case !s.MinWidth.Auto && (s.MinWidth.Pt != 0 || s.MinWidth.Pct != 0):
			it.min = toBorder(s.MinWidth.Resolve(cw, 0), padW)
		case s.Overflow == "visible" || s.Overflow == "":
			// automatic minimum size: the min-content size, capped by a specified size
			auto := mn
			if it.isReplaced {
				w, _ := l.replacedSize(b, cw)
				auto = math.Min(auto, w+padW)
			}
			if !s.Width.Auto {
				auto = math.Min(auto, toBorder(s.Width.Resolve(cw, 0), padW))
			}
			it.min = auto
		}
		it.max = math.Inf(1)
		if !s.MaxWidth.Auto {
			it.max = toBorder(s.MaxWidth.Resolve(cw, 0), padW)
		}
	} else {
		it.definiteCross = !s.Width.Auto
		it.definiteMain = !s.Height.Auto && !s.Height.HasPct()
		// content height at the item's cross size
		crossW := l.flexColumnWidth(it, cw, mn, mx, padW)
		basisSet := !s.FlexBasis.Auto && !(s.FlexBasis.HasPct() && defH < 0)
		contentH := -1.0
		measure := func() float64 {
			if contentH < 0 {
				sub := l.sub()
				r := sub.layoutBlockW(b, 0, 0, cw, margins{}, true, crossW)
				contentH = r.bottom - r.top
			}
			return contentH
		}
		switch {
		case basisSet:
			it.base = toBorder(s.FlexBasis.Resolve(math.Max(defH, 0), 0), padH)
		case it.definiteMain:
			it.base = toBorder(s.Height.Resolve(0, 0), padH)
		default:
			it.base = measure()
		}
		it.min = 0
		switch {
		case !s.MinHeight.Auto && (s.MinHeight.Pt != 0):
			it.min = toBorder(s.MinHeight.Resolve(0, 0), padH)
		case (s.Overflow == "visible" || s.Overflow == "") && defH >= 0:
			auto := measure()
			if it.definiteMain {
				auto = math.Min(auto, toBorder(s.Height.Resolve(0, 0), padH))
			}
			it.min = auto
		}
		it.max = math.Inf(1)
		if !s.MaxHeight.Auto && !s.MaxHeight.HasPct() {
			it.max = toBorder(s.MaxHeight.Resolve(0, 0), padH)
		}
	}
	if it.min > it.max {
		it.max = it.min
	}
	it.hyp = math.Min(math.Max(it.base, it.min), it.max)
	return it
}

// flexColumnWidth returns the border-box width of an item in a column container.
func (l *layouter) flexColumnWidth(it *flexItem, cw, mn, mx, padW float64) float64 {
	s := it.s
	avail := cw - it.mcs - it.mce
	if !s.Width.Auto {
		_, w := l.blockWidth(s, cw, -1)
		return w + padW
	}
	if it.isReplaced {
		w, _ := l.replacedSize(it.box, cw)
		return w + padW
	}
	if it.align == "stretch" && !it.autoCS && !it.autoCE {
		return clampWidth(s, avail-padW, cw, padW) + padW
	}
	return math.Min(math.Max(mn, math.Min(avail, mx)), math.Max(avail, mn))
}

// resolveFlexible runs the CSS flexible length resolution for one line.
func resolveFlexible(items []*flexItem, avail float64) {
	sumHyp := 0.0
	for _, it := range items {
		sumHyp += it.hyp + it.mms + it.mme
		it.frozen = false
	}
	growing := sumHyp < avail
	for _, it := range items {
		it.target = it.hyp
		if (growing && it.s.FlexGrow == 0) || (!growing && it.s.FlexShrink == 0) ||
			(growing && it.base > it.hyp) || (!growing && it.base < it.hyp) {
			it.frozen = true
		}
	}
	initialFree := avail
	for _, it := range items {
		if it.frozen {
			initialFree -= it.target + it.mms + it.mme
		} else {
			initialFree -= it.base + it.mms + it.mme
		}
	}
	for range len(items) + 1 {
		free := avail
		sumGrow, sumScaled, sumFactor := 0.0, 0.0, 0.0
		var open []*flexItem
		for _, it := range items {
			if it.frozen {
				free -= it.target + it.mms + it.mme
				continue
			}
			free -= it.base + it.mms + it.mme
			open = append(open, it)
			sumGrow += it.s.FlexGrow
			sumScaled += it.s.FlexShrink * it.base
			if growing {
				sumFactor += it.s.FlexGrow
			} else {
				sumFactor += it.s.FlexShrink
			}
		}
		if len(open) == 0 {
			break
		}
		if sumFactor < 1 {
			if f := initialFree * sumFactor; math.Abs(f) < math.Abs(free) {
				free = f
			}
		}
		total := 0.0
		for _, it := range open {
			t := it.base
			switch {
			case growing && sumGrow > 0:
				t = it.base + free*it.s.FlexGrow/sumGrow
			case !growing && sumScaled > 0:
				t = it.base + free*it.s.FlexShrink*it.base/sumScaled
			}
			c := math.Min(math.Max(t, it.min), it.max)
			c = math.Max(c, 0)
			total += c - t
			it.target = c
		}
		for _, it := range open {
			switch {
			case total > eps:
				if it.target <= it.min+eps {
					it.frozen = true
				}
			case total < -eps:
				if it.target >= it.max-eps {
					it.frozen = true
				}
			default:
				it.frozen = true
			}
		}
	}
}

// flexLayoutItem lays out an item at its target main size (and stretched cross size).
func (l *layouter) flexLayoutItem(it *flexItem, row bool, cw, defH float64) {
	b := it.box
	s := it.s
	padW := s.BorderUsed(css.Left) + s.BorderUsed(css.Right) + s.Padding[css.Left].Resolve(cw, 0) + s.Padding[css.Right].Resolve(cw, 0)
	var width float64
	if row {
		width = it.target
		if it.stretched {
			l.e.forcedH[b] = it.cross
		}
	} else {
		mn, mx := l.intrinsicWidths(b)
		marg := absPart(s.Margin[css.Left]) + absPart(s.Margin[css.Right])
		width = l.flexColumnWidth(it, cw, mn-marg, mx-marg, padW)
		if it.stretched {
			width = it.cross
		}
		l.e.forcedH[b] = it.target
	}
	if it.isReplaced {
		l.e.forcedW[b] = width
	}
	sub := l.sub()
	r := sub.layoutBlockW(b, 0, 0, cw, margins{}, true, width)
	delete(l.e.forcedH, b)
	delete(l.e.forcedW, b)
	it.sub, it.res = sub, r
	if row {
		if !it.stretched {
			it.cross = r.bottom - r.top
		}
		it.hasBaseline = r.hasBaseline
		it.baseline = r.firstBaseline - r.top
		if !r.hasBaseline {
			it.baseline = r.bottom - r.top
			it.hasBaseline = true
		}
	} else if !it.stretched {
		it.cross = width
	}
}

// flexMainPositions positions the items of a line along the main axis.
func (l *layouter) flexMainPositions(ln *flexLine, s *css.Style, row, reverse bool, availMain, gap float64) {
	n := len(ln.items)
	usedOuter := gap * float64(n-1)
	autos := 0
	for _, it := range ln.items {
		usedOuter += it.target + it.mms + it.mme
		if it.autoMS {
			autos++
		}
		if it.autoME {
			autos++
		}
	}
	ln.mainUsedOuters = usedOuter
	if availMain < 0 {
		availMain = usedOuter
	}
	free := availMain - usedOuter
	start, between := 0.0, gap
	autoShare := 0.0
	if autos > 0 && free > 0 {
		autoShare = free / float64(autos)
	} else {
		jc := s.JustifyContent
		if reverse {
			switch jc {
			case "flex-start":
				jc = "flex-end"
			case "flex-end":
				jc = "flex-start"
			}
		}
		switch jc {
		case "flex-end", "end", "right":
			start = free
		case "center":
			start = free / 2
		case "space-between":
			if n > 1 && free > 0 {
				between += free / float64(n-1)
			}
		case "space-around":
			if free > 0 {
				between += free / float64(n)
				start = free / float64(n) / 2
			} else {
				start = free / 2
			}
		case "space-evenly":
			if free > 0 {
				between += free / float64(n+1)
				start = free / float64(n+1)
			} else {
				start = free / 2
			}
		}
		if reverse && (jc == "start" || jc == "end") {
			// start/end refer to the container, not the flex direction
			if jc == "start" {
				start = free
			} else {
				start = 0
			}
		}
	}
	p := start
	for _, it := range ln.items {
		if it.autoMS {
			p += autoShare
		}
		p += it.mms
		it.mainPos = p
		p += it.target + it.mme
		if it.autoME {
			p += autoShare
		}
		p += between
	}
	if reverse {
		for _, it := range ln.items {
			it.mainPos = availMain - it.mainPos - it.target
		}
	}
	_ = row
}
