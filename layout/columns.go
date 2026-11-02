// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package layout

import (
	"math"

	"github.com/gsoultan/proteus-converter/css"
)

// columnGeometry returns the used column count, column width and gap of a
// multi-column container with content width w (CSS Multi-column Layout,
// "pseudo-algorithm" in section 3.4).
func columnGeometry(s *css.Style, w float64) (n int, colW, gap float64) {
	gap = s.ColumnGap.Resolve(w, 0)
	if s.ColumnGapNormal {
		gap = s.FontSize // normal = 1em
	}
	n = s.ColumnCount
	if !s.ColumnWidth.Auto {
		cw := s.ColumnWidth.Resolve(w, 0)
		fit := max(1, int(math.Floor((w+gap)/(cw+gap)+1e-9)))
		if n == 0 || fit < n {
			n = fit
		}
	}
	n = max(n, 1)
	colW = math.Max(0, (w-float64(n-1)*gap)/float64(n))
	return n, colW, gap
}

// layoutColumns lays out the content of the multi-column container b in
// balanced columns at (x, y) within content width w. The content is laid
// out as one flow fragmented into columns of a trial height, the way pages
// are; the column height is the smallest one at which the content fits in
// the column count, which is how Blink balances columns.
func (l *layouter) layoutColumns(b *Box, x, y, w float64) contentsResult {
	s := b.Style
	n, colW, gap := columnGeometry(s, w)
	anon := anonymousBlock(b)
	anon.Children = b.Children

	ce := *l.e
	ce.MaxPages = 0
	ce.fixedItems = nil
	run := func(h float64) (*layouter, float64) {
		ce.PageHeight = h
		cl := &layouter{e: &ce, fragment: true, fc: &floatCtx{}, pos: l.pos, pctH: l.pctH, pctOK: false}
		r := cl.layoutBlock(anon, 0, 0, colW, margins{}, true)
		return cl, r.bottom
	}
	// natural height of a single column
	ce.PageHeight = math.Inf(1)
	cl0 := &layouter{e: &ce, fc: &floatCtx{}, pos: l.pos, pctH: l.pctH}
	total := cl0.layoutBlock(anon, 0, 0, colW, margins{}, true).bottom
	if total <= eps {
		return contentsResult{}
	}
	fits := func(h float64) (*layouter, float64, bool) {
		cl, end := run(h)
		return cl, end, end <= float64(n)*h+eps
	}
	lo, hi := total/float64(n), total
	if !s.Height.Auto && !s.Height.HasPct() {
		// a fixed height is filled column by column (column-fill: balance
		// still applies, but the height is an upper bound); overflow adds
		// columns to the right, as in Chromium
		hi = math.Max(s.Height.Resolve(0, 0), 1)
	}
	var best *layouter
	bestH := hi
	if cl, _, ok := fits(lo); ok {
		best, bestH = cl, lo
	} else {
		for i := 0; i < 24 && hi-lo > 0.01; i++ {
			mid := (lo + hi) / 2
			if _, _, ok := fits(mid); ok {
				hi = mid
			} else {
				lo = mid
			}
		}
		// Blink works in LayoutUnits (1/64 px)
		bestH = math.Ceil(hi/0.75*64-1e-6) / 64 * 0.75
		best, _ = run(bestH)
	}

	pages, _ := ce.Paginate(best.items)
	used := 0
	for k, p := range pages {
		if len(p.Items) == 0 {
			continue
		}
		used = k + 1
		dx := x + float64(k)*(colW+gap)
		for _, it := range p.Items {
			it.translate(dx, y)
			l.emit(it)
		}
	}
	if rw := s.ColumnRuleUsed(); rw > 0 && s.ColumnRuleColor.A > 0 {
		var dash []float64
		switch s.ColumnRuleStyle {
		case "dashed":
			dash = []float64{3 * rw, 3 * rw}
		case "dotted":
			dash = []float64{rw, rw}
		}
		for k := 1; k < used; k++ {
			cx := x + float64(k)*(colW+gap) - gap/2
			l.emit(&LineItem{X1: cx, Y1: y, X2: cx, Y2: y + bestH, Width: rw, Color: s.ColumnRuleColor, Dash: dash})
		}
	}
	return contentsResult{placed: true, firstTop: y, end: y + bestH}
}
