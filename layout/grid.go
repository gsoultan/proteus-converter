// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package layout

import (
	"math"

	"github.com/gsoultan/proteus-converter/css"
)

// gridItem is an item of a grid container during layout.
type gridItem struct {
	box                    *Box
	s                      *css.Style
	r0, r1, c0, c1         int // track ranges [r0, r1) x [c0, c1)
	mn, mx                 float64
	ml, mr, mt, mb         float64
	autoL, autoR           bool
	autoT, autoB           bool
	w, h                   float64 // border box
	sub                    *layouter
	res                    blockResult
	stretchW, stretchH     bool
	justify, align         string
	isReplaced             bool
	placedRow, placedCol   bool
	spanRows, spanCols     int
	definiteW, definiteH   bool
	padW                   float64
	startRowNeg, startColN bool
}

type gridTrack struct {
	css.Track
	size, limit float64
	used        bool // has an item (auto-fit collapsing)
	collapsible bool
}

// expandTracks returns the explicit tracks of a track list for an available size.
func expandTracks(tl css.TrackList, avail, gap float64) (tracks []gridTrack) {
	add := func(ts []css.Track, collapsible bool) {
		for _, t := range ts {
			tracks = append(tracks, gridTrack{Track: t, collapsible: collapsible})
		}
	}
	add(tl.Tracks[:min(tl.AutoAt, len(tl.Tracks))], false)
	if len(tl.AutoRepeat) > 0 {
		fixed := 0.0
		for _, t := range tl.Tracks {
			fixed += fixedTrackSize(t, avail) + gap
		}
		per := 0.0
		for _, t := range tl.AutoRepeat {
			per += fixedTrackSize(t, avail) + gap
		}
		n := 1
		if per > 0 && avail >= 0 {
			n = max(1, int(math.Floor((avail-fixed+gap+eps)/per)))
		}
		n = min(n, 1000)
		for range n {
			add(tl.AutoRepeat, tl.AutoFit)
		}
	}
	add(tl.Tracks[min(tl.AutoAt, len(tl.Tracks)):], false)
	return tracks
}

// fixedTrackSize returns the definite size used to count auto repetitions.
func fixedTrackSize(t css.Track, avail float64) float64 {
	if t.Max.Kind == css.TrackLength {
		v := t.Max.Len.Resolve(avail, 0)
		if t.Min.Kind == css.TrackLength {
			v = math.Max(v, t.Min.Len.Resolve(avail, 0))
		}
		return v
	}
	if t.Min.Kind == css.TrackLength {
		return t.Min.Len.Resolve(avail, 0)
	}
	return 0
}

// layoutGrid lays out the items of grid container b in its content box at
// (x, y) with width cw; defH is the definite content height or -1.
func (l *layouter) layoutGrid(b *Box, x, y, cw, defH float64) contentsResult {
	s := b.Style
	res := contentsResult{placed: true, firstTop: y, end: y}
	kids := l.flexItems(b, x, y)
	colGap := s.ColumnGap.Resolve(cw, 0)
	rowGap := s.RowGap.Resolve(math.Max(defH, 0), 0)
	if s.GridAutoFlow == "column" {
		l.e.warnOnce("layout: grid-auto-flow: column is laid out as row")
	}

	cols := expandTracks(s.GridTemplateColumns, cw, colGap)
	rows := expandTracks(s.GridTemplateRows, defH, rowGap)
	areas := s.GridTemplateAreas
	if n := len(areas); n > 0 {
		for len(rows) < n {
			rows = append(rows, gridTrack{Track: autoTrack(s.GridAutoRows, len(rows))})
		}
		for len(cols) < len(areas[0]) {
			cols = append(cols, gridTrack{Track: autoTrack(s.GridAutoColumns, len(cols))})
		}
	}
	explicitRows, explicitCols := len(rows), len(cols)

	// resolve definite placements
	items := make([]*gridItem, 0, len(kids))
	for _, k := range kids {
		it := &gridItem{box: k, s: k.Style, isReplaced: k.Kind == KindImage || k.Kind == KindCheckbox}
		p := k.Style.GridPlacement
		it.r0, it.spanRows, it.placedRow = resolveLines(p[0], p[2], areas, true, explicitRows)
		it.c0, it.spanCols, it.placedCol = resolveLines(p[1], p[3], areas, false, explicitCols)
		items = append(items, it)
	}
	occupied := map[[2]int]bool{}
	free := func(r, c, rs, cs int) bool {
		for i := r; i < r+rs; i++ {
			for j := c; j < c+cs; j++ {
				if occupied[[2]int{i, j}] {
					return false
				}
			}
		}
		return true
	}
	mark := func(it *gridItem) {
		for i := it.r0; i < it.r0+it.spanRows; i++ {
			for j := it.c0; j < it.c0+it.spanCols; j++ {
				occupied[[2]int{i, j}] = true
			}
		}
	}
	ncols := func() int { return max(len(cols), 1) }
	for _, it := range items {
		if it.placedRow && it.placedCol {
			mark(it)
		}
	}
	// items locked to a row
	for _, it := range items {
		if it.placedRow && !it.placedCol {
			c := 0
			for !free(it.r0, c, it.spanRows, it.spanCols) && c < 10000 {
				c++
			}
			it.c0, it.placedCol = c, true
			mark(it)
		}
	}
	// auto placement (sparse, row-major)
	cr, cc := 0, 0
	for _, it := range items {
		if it.placedRow && it.placedCol {
			continue
		}
		span := min(it.spanCols, max(ncols(), it.spanCols))
		if it.placedCol {
			if it.c0 < cc {
				cr++
			}
			cc = it.c0
			for !free(cr, cc, it.spanRows, it.spanCols) && cr < 100000 {
				cr++
			}
		} else {
			for guard := 0; guard < 1000000; guard++ {
				if cc+span > ncols() && cc > 0 {
					cr++
					cc = 0
					continue
				}
				if free(cr, cc, it.spanRows, span) {
					break
				}
				cc++
			}
			it.c0 = cc
		}
		it.r0 = cr
		it.placedRow, it.placedCol = true, true
		mark(it)
		cc = it.c0 + it.spanCols
	}
	// implicit tracks
	for _, it := range items {
		for len(cols) < it.c0+it.spanCols {
			cols = append(cols, gridTrack{Track: autoTrack(s.GridAutoColumns, len(cols)-explicitCols)})
		}
		for len(rows) < it.r0+it.spanRows {
			rows = append(rows, gridTrack{Track: autoTrack(s.GridAutoRows, len(rows)-explicitRows)})
		}
		it.r1, it.c1 = it.r0+it.spanRows, it.c0+it.spanCols
		for c := it.c0; c < it.c1; c++ {
			cols[c].used = true
		}
		for r := it.r0; r < it.r1; r++ {
			rows[r].used = true
		}
	}
	if len(items) == 0 && len(cols) == 0 {
		res.end = y + math.Max(defH, 0)
		return res
	}

	// item metrics
	for _, it := range items {
		st := it.s
		e := l.hEdges(st, cw)
		it.ml, it.mr, it.autoL, it.autoR = e.ml, e.mr, e.mlAuto, e.mrAuto
		if it.autoL {
			it.ml = 0
		}
		if it.autoR {
			it.mr = 0
		}
		it.mt, it.mb = st.Margin[css.Top].Resolve(cw, 0), st.Margin[css.Bottom].Resolve(cw, 0)
		it.autoT, it.autoB = st.Margin[css.Top].Auto, st.Margin[css.Bottom].Auto
		if it.autoT {
			it.mt = 0
		}
		if it.autoB {
			it.mb = 0
		}
		it.padW = e.bl + e.br + e.pl + e.pr
		mn, mx := l.intrinsicWidths(it.box)
		marg := absPart(st.Margin[css.Left]) + absPart(st.Margin[css.Right])
		it.mn, it.mx = mn-marg, mx-marg
		it.definiteW = !st.Width.Auto
		it.definiteH = !st.Height.Auto && !st.Height.HasPct()
		it.justify = gridSelf(s.JustifyItems, st.JustifySelf, it.isReplaced)
		it.align = gridSelf(s.AlignItems, st.AlignSelf, it.isReplaced)
	}

	// column sizes
	sizeTracks(cols, cw, colGap, s.JustifyContent, func(t int) (mins, maxs []float64) {
		for _, it := range items {
			if it.c0 == t && it.c1 == t+1 {
				mins = append(mins, it.mn+it.ml+it.mr)
				maxs = append(maxs, it.mx+it.ml+it.mr)
			}
		}
		return
	}, func(c0, c1 int) []float64 {
		var out []float64
		for _, it := range items {
			if it.c0 == c0 && it.c1 == c1 {
				out = append(out, it.mn+it.ml+it.mr)
			}
		}
		return out
	})
	colPos := trackPositions(cols, colGap, cw, s.JustifyContent)

	// lay out items at their column widths to size the rows
	for _, it := range items {
		aw := colPos[it.c1-1] + cols[it.c1-1].size - colPos[it.c0]
		w := aw - it.ml - it.mr
		switch {
		case it.definiteW || it.isReplaced:
			w = -1
		case it.justify != "stretch" || it.autoL || it.autoR:
			w = math.Min(math.Max(it.mn, math.Min(w, it.mx)), math.Max(w, it.mn))
		default:
			it.stretchW = true
		}
		l.gridLayoutItem(it, cw, w, -1)
	}
	sizeRows(rows, items, defH, rowGap)
	rowPos := trackPositions(rows, rowGap, defH, s.AlignContent)

	// stretch heights and align
	shift := 0.0
	rowShift := make([]float64, len(rows))
	if l.fragment {
		for r := range rows {
			top := y + rowPos[r] + shift
			if ny := l.place(top, rows[r].size); ny != top && rows[r].size > 0 {
				shift += ny - top
			}
			rowShift[r] = shift
		}
	}
	firstRow := len(rows)
	for _, it := range items {
		ah := rowPos[it.r1-1] + rows[it.r1-1].size - rowPos[it.r0]
		aw := colPos[it.c1-1] + cols[it.c1-1].size - colPos[it.c0]
		if it.align == "stretch" && !it.definiteH && !it.autoT && !it.autoB && !it.isReplaced {
			want := math.Max(ah-it.mt-it.mb, 0)
			if math.Abs(want-it.h) > eps {
				w := it.w
				if it.definiteW {
					w = -1
				}
				l.gridLayoutItem(it, cw, w, want)
			}
		}
		fx := alignOffset(it.justify, aw-it.w-it.ml-it.mr, it.autoL, it.autoR)
		fy := alignOffset(it.align, ah-it.h-it.mt-it.mb, it.autoT, it.autoB)
		bx := x + colPos[it.c0] + it.ml + fx
		by := y + rowPos[it.r0] + rowShift[it.r0] + it.mt + fy
		dx, dy := bx-it.res.bx, by-it.res.top
		translateItems(it.sub.items, dx, dy)
		l.items = append(l.items, it.sub.items...)
		if it.res.hasBaseline && it.r0 <= firstRow && (!res.hasBaseline || it.r0 < firstRow) {
			res.firstBaseline = it.res.firstBaseline + dy
			res.hasBaseline = true
			firstRow = it.r0
		}
	}
	total := 0.0
	if n := len(rows); n > 0 {
		total = rowPos[n-1] + rows[n-1].size
	}
	if defH >= 0 {
		total = defH
	}
	res.end = y + total + shift
	if !res.hasBaseline {
		res.firstBaseline = res.end
	}
	res.lastBaseline = res.firstBaseline
	return res
}

func autoTrack(tl css.TrackList, i int) css.Track {
	if len(tl.Tracks) == 0 {
		return css.Track{}
	}
	i %= len(tl.Tracks)
	if i < 0 {
		i += len(tl.Tracks)
	}
	return tl.Tracks[i]
}

// resolveLines resolves a start/end placement pair to a 0-based start track and a span.
func resolveLines(start, end css.GridLine, areas [][]string, rows bool, explicit int) (int, int, bool) {
	lineOf := func(g css.GridLine, isEnd bool) (int, bool) {
		switch {
		case g.Name != "":
			r0, r1, c0, c1, ok := findArea(areas, g.Name)
			if !ok {
				return 0, false
			}
			if rows {
				if isEnd {
					return r1, true
				}
				return r0, true
			}
			if isEnd {
				return c1, true
			}
			return c0, true
		case g.Line > 0:
			return g.Line - 1, true
		case g.Line < 0:
			return max(explicit+1+g.Line, 0), true
		}
		return 0, false
	}
	s, sok := lineOf(start, false)
	e, eok := lineOf(end, true)
	switch {
	case sok && eok:
		if e < s {
			s, e = e, s
		}
		if e == s {
			e = s + 1
		}
		return s, e - s, true
	case sok:
		span := max(end.Span, 1)
		return s, span, true
	case eok:
		span := max(start.Span, 1)
		return max(e-span, 0), span, true
	}
	span := max(max(start.Span, end.Span), 1)
	return 0, span, false
}

func findArea(areas [][]string, name string) (r0, r1, c0, c1 int, ok bool) {
	r0, c0 = -1, -1
	for i, row := range areas {
		for j, cell := range row {
			if cell != name {
				continue
			}
			if r0 < 0 || i < r0 {
				r0 = i
			}
			if c0 < 0 || j < c0 {
				c0 = j
			}
			r1 = max(r1, i+1)
			c1 = max(c1, j+1)
		}
	}
	return r0, r1, c0, c1, r0 >= 0
}

func gridSelf(container, self string, replaced bool) string {
	a := self
	if a == "auto" || a == "" {
		a = container
	}
	switch a {
	case "normal", "legacy", "":
		if replaced {
			return "start"
		}
		return "stretch"
	case "stretch":
		return "stretch"
	case "start", "flex-start", "self-start", "left", "baseline":
		return "start"
	case "end", "flex-end", "self-end", "right":
		return "end"
	}
	return a
}

func alignOffset(a string, free float64, autoStart, autoEnd bool) float64 {
	switch {
	case autoStart && autoEnd:
		return math.Max(free, 0) / 2
	case autoStart:
		return math.Max(free, 0)
	case autoEnd:
		return 0
	}
	switch a {
	case "center":
		return free / 2
	case "end":
		return free
	}
	return 0
}

// gridLayoutItem lays out an item with border-box width w (-1: its own width)
// and border-box height h (-1: auto).
func (l *layouter) gridLayoutItem(it *gridItem, cw, w, h float64) {
	b := it.box
	if h >= 0 {
		l.e.forcedH[b] = h
	}
	forced := -1.0
	if w >= 0 {
		forced = w
		if it.isReplaced {
			l.e.forcedW[b] = w
		}
	}
	sub := l.sub()
	r := sub.layoutBlockW(b, 0, 0, cw, margins{}, true, forced)
	delete(l.e.forcedH, b)
	delete(l.e.forcedW, b)
	it.sub, it.res = sub, r
	it.h = r.bottom - r.top
	if r.bw > 0 {
		it.w = r.bw
	} else {
		it.w = w
	}
}

// sizeTracks sizes column tracks for the available width.
func sizeTracks(tracks []gridTrack, avail, gap float64, justify string,
	single func(t int) (mins, maxs []float64), spanning func(c0, c1 int) []float64) {
	n := len(tracks)
	if n == 0 {
		return
	}
	for i := range tracks {
		t := &tracks[i]
		mins, maxs := single(i)
		maxOf := func(v []float64) float64 {
			m := 0.0
			for _, x := range v {
				m = math.Max(m, x)
			}
			return m
		}
		switch t.Min.Kind {
		case css.TrackLength:
			t.size = t.Min.Len.Resolve(avail, 0)
		case css.TrackMaxContent:
			t.size = maxOf(maxs)
		default:
			t.size = maxOf(mins)
		}
		switch t.Max.Kind {
		case css.TrackLength:
			t.limit = math.Max(t.Max.Len.Resolve(avail, 0), t.size)
		case css.TrackMinContent:
			t.limit = math.Max(maxOf(mins), t.size)
		case css.TrackFr:
			t.limit = t.size
		default:
			t.limit = math.Max(maxOf(maxs), t.size)
		}
		if t.collapsible && !t.used {
			t.size, t.limit = 0, 0
		}
	}
	gaps := gap * float64(visibleTracks(tracks)-1)
	sum := func() float64 {
		s := 0.0
		for _, t := range tracks {
			s += t.size
		}
		return s
	}
	// grow non-flexible tracks towards their limits
	free := avail - gaps - sum()
	for iter := 0; free > eps && iter < n+1; iter++ {
		var growable []int
		for i, t := range tracks {
			if t.Max.Kind != css.TrackFr && t.limit > t.size+eps {
				growable = append(growable, i)
			}
		}
		if len(growable) == 0 {
			break
		}
		share := free / float64(len(growable))
		for _, i := range growable {
			d := math.Min(share, tracks[i].limit-tracks[i].size)
			tracks[i].size += d
			free -= d
		}
	}
	// flexible tracks
	frSum := 0.0
	for _, t := range tracks {
		if t.Max.Kind == css.TrackFr && !(t.collapsible && !t.used) {
			frSum += t.Max.Fr
		}
	}
	if frSum > 0 {
		inflexible := map[int]bool{}
		for range n + 1 {
			left := avail - gaps
			fr := 0.0
			for i, t := range tracks {
				if t.Max.Kind == css.TrackFr && !inflexible[i] && !(t.collapsible && !t.used) {
					fr += t.Max.Fr
				} else {
					left -= t.size
				}
			}
			if fr <= 0 {
				break
			}
			unit := math.Max(left, 0) / math.Max(fr, 1)
			if fr < 1 {
				unit = math.Max(left, 0)
			}
			again := false
			for i, t := range tracks {
				if t.Max.Kind == css.TrackFr && !inflexible[i] && !(t.collapsible && !t.used) {
					if t.size > unit*t.Max.Fr+eps {
						inflexible[i] = true
						again = true
					}
				}
			}
			if again {
				continue
			}
			for i := range tracks {
				t := &tracks[i]
				if t.Max.Kind == css.TrackFr && !inflexible[i] && !(t.collapsible && !t.used) {
					t.size = math.Max(t.size, unit*t.Max.Fr)
				}
			}
			break
		}
	} else if justify == "normal" || justify == "stretch" {
		// stretch auto tracks into the remaining space
		free = avail - gaps - sum()
		var autos []int
		for i, t := range tracks {
			if t.Max.Kind == css.TrackAuto && !(t.collapsible && !t.used) {
				autos = append(autos, i)
			}
		}
		if free > eps && len(autos) > 0 {
			for _, i := range autos {
				tracks[i].size += free / float64(len(autos))
			}
		}
	}
	// items spanning several tracks: grow the spanned auto tracks if too small
	for c0 := 0; c0 < n; c0++ {
		for c1 := c0 + 2; c1 <= n; c1++ {
			need := 0.0
			for _, v := range spanning(c0, c1) {
				need = math.Max(need, v)
			}
			if need <= 0 {
				continue
			}
			have := gap * float64(c1-c0-1)
			var autos []int
			for i := c0; i < c1; i++ {
				have += tracks[i].size
				if tracks[i].Min.Kind != css.TrackLength {
					autos = append(autos, i)
				}
			}
			if need > have+eps && len(autos) > 0 {
				for _, i := range autos {
					tracks[i].size += (need - have) / float64(len(autos))
				}
			}
		}
	}
}

func visibleTracks(tracks []gridTrack) int {
	n := 0
	for _, t := range tracks {
		if !(t.collapsible && !t.used) {
			n++
		}
	}
	return max(n, 1)
}

// sizeRows sizes the row tracks from the items' heights.
func sizeRows(rows []gridTrack, items []*gridItem, defH, gap float64) {
	content := func(r int) float64 {
		m := 0.0
		for _, it := range items {
			if it.r0 == r && it.r1 == r+1 {
				m = math.Max(m, it.h+it.mt+it.mb)
			}
		}
		return m
	}
	frSum := 0.0
	for i := range rows {
		t := &rows[i]
		c := content(i)
		switch {
		case t.Min.Kind == css.TrackLength && (defH >= 0 || !t.Min.Len.HasPct()):
			t.size = t.Min.Len.Resolve(math.Max(defH, 0), 0)
		default:
			t.size = c
		}
		switch t.Max.Kind {
		case css.TrackLength:
			if defH >= 0 || !t.Max.Len.HasPct() {
				t.size = math.Max(t.size, math.Min(c, t.Max.Len.Resolve(math.Max(defH, 0), 0)))
				if t.Min.Kind == css.TrackLength && t.Max.Kind == css.TrackLength {
					t.size = math.Max(t.Min.Len.Resolve(math.Max(defH, 0), 0), t.Max.Len.Resolve(math.Max(defH, 0), 0))
				}
			}
		case css.TrackFr:
			frSum += t.Max.Fr
		default:
			t.size = math.Max(t.size, c)
		}
	}
	gaps := gap * float64(len(rows)-1)
	if frSum > 0 {
		if defH >= 0 {
			left := defH - gaps
			for _, t := range rows {
				if t.Max.Kind != css.TrackFr {
					left -= t.size
				}
			}
			unit := math.Max(left, 0) / math.Max(frSum, 1)
			for i := range rows {
				if rows[i].Max.Kind == css.TrackFr {
					rows[i].size = math.Max(rows[i].size, unit*rows[i].Max.Fr)
				}
			}
		} else {
			// indefinite height: every fr row gets the largest size per fr
			unit := 0.0
			for _, t := range rows {
				if t.Max.Kind == css.TrackFr && t.Max.Fr > 0 {
					unit = math.Max(unit, t.size/math.Max(t.Max.Fr, 1))
				}
			}
			for i := range rows {
				if rows[i].Max.Kind == css.TrackFr {
					rows[i].size = math.Max(rows[i].size, unit*rows[i].Max.Fr)
				}
			}
		}
	} else if defH >= 0 {
		free := defH - gaps
		var autos []int
		for i, t := range rows {
			free -= t.size
			if t.Max.Kind == css.TrackAuto {
				autos = append(autos, i)
			}
		}
		if free > eps && len(autos) > 0 {
			for _, i := range autos {
				rows[i].size += free / float64(len(autos))
			}
		}
	}
	// items spanning rows
	for _, it := range items {
		if it.r1-it.r0 < 2 {
			continue
		}
		have := gap * float64(it.r1-it.r0-1)
		for r := it.r0; r < it.r1; r++ {
			have += rows[r].size
		}
		if need := it.h + it.mt + it.mb; need > have+eps {
			rows[it.r1-1].size += need - have
		}
	}
}

// trackPositions returns each track's offset, honouring content distribution.
func trackPositions(tracks []gridTrack, gap, avail float64, distribution string) []float64 {
	pos := make([]float64, len(tracks))
	used := gap * float64(visibleTracks(tracks)-1)
	for _, t := range tracks {
		used += t.size
	}
	start, between := 0.0, gap
	if free := avail - used; avail >= 0 && free > eps {
		n := float64(visibleTracks(tracks))
		switch distribution {
		case "center":
			start = free / 2
		case "end", "flex-end", "right":
			start = free
		case "space-between":
			if n > 1 {
				between += free / (n - 1)
			}
		case "space-around":
			between += free / n
			start = free / n / 2
		case "space-evenly":
			between += free / (n + 1)
			start = free / (n + 1)
		}
	}
	p := start
	for i, t := range tracks {
		pos[i] = p
		if t.collapsible && !t.used {
			continue
		}
		p += t.size + between
	}
	return pos
}

// gridIntrinsic returns the min-content and max-content widths of a grid container.
func (l *layouter) gridIntrinsic(b *Box) (float64, float64) {
	s := b.Style
	cols := expandTracks(s.GridTemplateColumns, -1, 0)
	n := len(cols)
	var kids []*Box
	for _, c := range b.Children {
		if !c.outOfFlow() {
			kids = append(kids, c)
		}
	}
	if n == 0 {
		var mn, mx float64
		for _, c := range kids {
			cmn, cmx := l.intrinsicWidths(c)
			mn = math.Max(mn, cmn)
			mx = math.Max(mx, cmx)
		}
		return mn, mx
	}
	mins := make([]float64, n)
	maxs := make([]float64, n)
	for i, c := range kids {
		cmn, cmx := l.intrinsicWidths(c)
		j := i % n
		mins[j] = math.Max(mins[j], cmn)
		maxs[j] = math.Max(maxs[j], cmx)
	}
	gap := absPart(s.ColumnGap) * float64(n-1)
	mn, mx := gap, gap
	for j, t := range cols {
		if t.Min.Kind == css.TrackLength && !t.Min.Len.HasPct() && t.Max.Kind == css.TrackLength {
			mins[j], maxs[j] = t.Min.Len.Pt, math.Max(t.Max.Len.Pt, t.Min.Len.Pt)
		}
		mn += mins[j]
		mx += maxs[j]
	}
	return mn, mx
}
