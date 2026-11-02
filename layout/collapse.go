// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package layout

import (
	"math"

	"github.com/gsoultan/proteus-converter/css"
)

// cborder is one candidate border in the collapsing border model.
type cborder struct {
	width float64 // pt; 0 when the style is none
	style string
	color css.Color
	rank  int // origin: cell 4, row 3, row group 2, table 1
}

var borderStyleRank = map[string]int{
	"double": 8, "solid": 7, "dashed": 6, "dotted": 5,
	"ridge": 4, "outset": 3, "groove": 2, "inset": 1,
}

func sideBorder(s *css.Style, side, rank int) cborder {
	b := cborder{style: s.BorderStyle[side], color: s.BorderColor[side], rank: rank}
	if b.style != "none" && b.style != "hidden" {
		b.width = s.BorderWidth[side]
	}
	return b
}

// wins reports whether a beats b under CSS 2.1 §17.6.2.1: hidden wins, then
// the wider border, then the stronger style, then the origin closer to the
// cell. Ties keep b, which callers pass as the earlier (top/left) candidate.
func (a cborder) wins(b cborder) bool {
	if b.style == "hidden" {
		return false
	}
	if a.style == "hidden" {
		return true
	}
	if a.width != b.width {
		return a.width > b.width
	}
	if ra, rb := borderStyleRank[a.style], borderStyleRank[b.style]; ra != rb {
		return ra > rb
	}
	return a.rank > b.rank
}

func resolve(cands ...cborder) cborder {
	best := cborder{style: "none"}
	for _, c := range cands {
		if c.wins(best) {
			best = c
		}
	}
	if best.style == "hidden" {
		best.width = 0
	}
	return best
}

// px rounds a width in pt to whole device pixels.
func toPx(pt float64) int { return int(math.Round(pt / 0.75)) }

// collapseBorders resolves the collapsed borders of a table and rewrites the
// used styles of the table, its rows and its cells so the ordinary box code
// lays them out and paints them: like Blink, each grid line of width W px is
// split between its two sides, floor(W/2) to the cell above/left and the rest
// to the cell below/right; the table keeps the outer halves.
func collapseBorders(t *Box, rows []*trow, ncols int) {
	nr := len(rows)
	if nr == 0 || ncols == 0 {
		return
	}
	occ := make([][]*tcell, nr)
	rowOf := map[*tcell]int{}
	for i := range occ {
		occ[i] = make([]*tcell, ncols)
	}
	for ri, r := range rows {
		for _, c := range r.cells {
			rowOf[c] = ri
			for dr := 0; dr < c.rowspan && ri+dr < nr; dr++ {
				for dc := 0; dc < c.colspan && c.col+dc < ncols; dc++ {
					occ[ri+dr][c.col+dc] = c
				}
			}
		}
	}
	ts := t.Style
	// hline(i, col): line above row i (i == nr is the bottom edge).
	hline := func(i, col int) cborder {
		var cands []cborder
		if i > 0 {
			if c := occ[i-1][col]; c != nil {
				cands = append(cands, sideBorder(c.box.Style, css.Bottom, 4))
			}
			cands = append(cands, sideBorder(rows[i-1].box.Style, css.Bottom, 3))
			if g := rows[i-1].group; g != nil && (i == nr || rows[i].group != g) {
				cands = append(cands, sideBorder(g.Style, css.Bottom, 2))
			}
		} else {
			cands = append(cands, sideBorder(ts, css.Top, 1))
		}
		if i < nr {
			if c := occ[i][col]; c != nil {
				cands = append(cands, sideBorder(c.box.Style, css.Top, 4))
			}
			cands = append(cands, sideBorder(rows[i].box.Style, css.Top, 3))
			if g := rows[i].group; g != nil && (i == 0 || rows[i-1].group != g) {
				cands = append(cands, sideBorder(g.Style, css.Top, 2))
			}
		} else {
			cands = append(cands, sideBorder(ts, css.Bottom, 1))
		}
		return resolve(cands...)
	}
	// vline(row, j): line left of column j (j == ncols is the right edge).
	vline := func(ri, j int) cborder {
		var cands []cborder
		if j > 0 {
			if c := occ[ri][j-1]; c != nil {
				cands = append(cands, sideBorder(c.box.Style, css.Right, 4))
			}
		} else {
			cands = append(cands, sideBorder(rows[ri].box.Style, css.Left, 3))
			if g := rows[ri].group; g != nil {
				cands = append(cands, sideBorder(g.Style, css.Left, 2))
			}
			cands = append(cands, sideBorder(ts, css.Left, 1))
		}
		if j < ncols {
			if c := occ[ri][j]; c != nil {
				cands = append(cands, sideBorder(c.box.Style, css.Left, 4))
			}
		} else {
			cands = append(cands, sideBorder(rows[ri].box.Style, css.Right, 3))
			if g := rows[ri].group; g != nil {
				cands = append(cands, sideBorder(g.Style, css.Right, 2))
			}
			cands = append(cands, sideBorder(ts, css.Right, 1))
		}
		return resolve(cands...)
	}
	widest := func(bs []cborder) cborder {
		best := cborder{style: "none"}
		for _, b := range bs {
			if b.width > best.width || best.style == "none" {
				best = b
			}
		}
		return best
	}
	type plan struct{ top, bottom, left, right []cborder }
	plans := map[*tcell]*plan{}
	var outer [4][]cborder
	for ri, r := range rows {
		for _, c := range r.cells {
			p := &plan{}
			last := min(ri+c.rowspan, nr)
			end := min(c.col+c.colspan, ncols)
			for col := c.col; col < end; col++ {
				p.top = append(p.top, hline(ri, col))
				p.bottom = append(p.bottom, hline(last, col))
			}
			for rr := ri; rr < last; rr++ {
				p.left = append(p.left, vline(rr, c.col))
				p.right = append(p.right, vline(rr, end))
			}
			plans[c] = p
			if ri == 0 {
				outer[css.Top] = append(outer[css.Top], p.top...)
			}
			if last == nr {
				outer[css.Bottom] = append(outer[css.Bottom], p.bottom...)
			}
			if c.col == 0 {
				outer[css.Left] = append(outer[css.Left], p.left...)
			}
			if end == ncols {
				outer[css.Right] = append(outer[css.Right], p.right...)
			}
		}
	}
	set := func(s *css.Style, side int, b cborder, px int) {
		s.BorderWidth[side] = float64(px) * 0.75
		s.BorderStyle[side] = b.style
		s.BorderColor[side] = b.color
		if px == 0 {
			s.BorderStyle[side] = "none"
		}
	}
	for c, p := range plans {
		s := *c.box.Style
		top, bottom, left, right := widest(p.top), widest(p.bottom), widest(p.left), widest(p.right)
		set(&s, css.Top, top, toPx(top.width)-toPx(top.width)/2)
		set(&s, css.Bottom, bottom, toPx(bottom.width)/2)
		set(&s, css.Left, left, toPx(left.width)-toPx(left.width)/2)
		set(&s, css.Right, right, toPx(right.width)/2)
		c.box.Style = &s
	}
	for _, r := range rows {
		s := *r.box.Style
		for side := range 4 {
			s.BorderStyle[side] = "none"
		}
		r.box.Style = &s
	}
	s := *ts
	for side := range 4 {
		b := widest(outer[side])
		w := toPx(b.width)
		switch side {
		case css.Top, css.Left:
			w /= 2 // the cell keeps the larger half
		default:
			w -= w / 2
		}
		set(&s, side, b, w)
	}
	t.Style = &s
}
