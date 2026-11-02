// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package layout

import (
	"math"

	"github.com/gsoultan/proteus-converter/css"
)

type tcell struct {
	box          *Box
	col, colspan int
	rowspan      int
}

type trow struct {
	box    *Box
	group  *Box // the row group the row came from
	cells  []*tcell
	header bool
}

// tableGrid flattens row groups into rows (header, body, footer order) and
// assigns column indexes, honouring colspan and rowspan.
func tableGrid(t *Box) (rows []*trow, ncols int, captions []*Box) {
	var head, body, foot []*Box
	groupOf := map[*Box]*Box{}
	for _, g := range t.Children {
		for _, r := range g.Children {
			groupOf[r] = g
		}
		switch g.Kind {
		case KindCaption:
			captions = append(captions, g)
		case KindRowGroup:
			switch g.RowGroup {
			case "header":
				if head == nil {
					head = g.Children
				} else {
					body = append(body, g.Children...)
				}
			case "footer":
				foot = append(foot, g.Children...)
			default:
				body = append(body, g.Children...)
			}
		}
	}
	occupied := map[[2]int]bool{}
	add := func(rs []*Box, header bool) {
		for _, r := range rs {
			ri := len(rows)
			tr := &trow{box: r, group: groupOf[r], header: header}
			col := 0
			for _, c := range r.Children {
				for occupied[[2]int{ri, col}] {
					col++
				}
				cs, rsp := max(c.ColSpan, 1), max(c.RowSpan, 1)
				tr.cells = append(tr.cells, &tcell{box: c, col: col, colspan: cs, rowspan: rsp})
				for dr := range rsp {
					for dc := range cs {
						occupied[[2]int{ri + dr, col + dc}] = true
					}
				}
				col += cs
				ncols = max(ncols, col)
			}
			rows = append(rows, tr)
		}
	}
	add(head, true)
	add(body, false)
	add(foot, false)
	if t.Style.BorderCollapse == "collapse" && !t.collapsed {
		t.collapsed = true
		collapseBorders(t, rows, ncols)
	}
	return rows, ncols, captions
}

func tableSpacing(s *css.Style) (h, v float64) {
	if s.BorderCollapse == "collapse" {
		return 0, 0
	}
	return s.BorderSpacingH, s.BorderSpacingV
}

// columnWidths computes per-column min and max widths from cell intrinsic widths.
func (l *layouter) columnWidths(rows []*trow, ncols int, hs float64) (mn, mx []float64) {
	mn = make([]float64, ncols)
	mx = make([]float64, ncols)
	var spans []*tcell
	for _, r := range rows {
		for _, c := range r.cells {
			if c.colspan > 1 {
				spans = append(spans, c)
				continue
			}
			cmn, cmx := l.intrinsicWidths(c.box)
			if w := c.box.Style.Width; !w.Auto && !w.HasPct() {
				cmx = math.Max(cmn, cmx)
			}
			mn[c.col] = math.Max(mn[c.col], cmn)
			mx[c.col] = math.Max(mx[c.col], math.Max(cmx, cmn))
		}
	}
	for _, c := range spans {
		cmn, cmx := l.intrinsicWidths(c.box)
		end := min(c.col+c.colspan, ncols)
		var smn, smx float64
		for i := c.col; i < end; i++ {
			smn += mn[i]
			smx += mx[i]
		}
		gaps := hs * float64(end-c.col-1)
		distribute := func(arr []float64, need, have, weight float64, weights []float64) {
			if need <= have {
				return
			}
			extra := need - have
			for i := c.col; i < end; i++ {
				share := 1 / float64(end-c.col)
				if weight > 0 {
					share = weights[i] / weight
				}
				arr[i] += extra * share
			}
		}
		distribute(mn, cmn-gaps, smn, smx, mx)
		distribute(mx, cmx-gaps, smx, smx, mx)
	}
	return mn, mx
}

func (l *layouter) tableIntrinsic(t *Box) (float64, float64) {
	rows, ncols, _ := tableGrid(t)
	s := t.Style // after tableGrid, which rewrites it for collapsed borders
	hs, _ := tableSpacing(s)
	mn, mx := l.columnWidths(rows, ncols, hs)
	var smn, smx float64
	for i := range ncols {
		smn += mn[i]
		smx += mx[i]
	}
	sp := hs * float64(ncols+1)
	if ncols == 0 {
		sp = 0
	}
	if !s.Width.Auto && !s.Width.HasPct() {
		w := math.Max(s.Width.Pt, smn+sp)
		return w, w
	}
	return smn + sp, smx + sp
}

func (l *layouter) layoutTable(t *Box, x, y, cbw float64, pending margins, forced float64) blockResult {
	rows, ncols, captions := tableGrid(t)
	s := t.Style // after tableGrid, which rewrites it for collapsed borders
	hs, vs := tableSpacing(s)
	e := l.hEdges(s, cbw)
	bt, bb := s.BorderUsed(css.Top), s.BorderUsed(css.Bottom)
	pt, pb := s.Padding[css.Top].Resolve(cbw, 0), s.Padding[css.Bottom].Resolve(cbw, 0)
	if s.BorderCollapse == "collapse" {
		e.pl, e.pr, pt, pb = 0, 0, 0, 0
	}
	edge := e.bl + e.br + e.pl + e.pr
	mn, mx := l.columnWidths(rows, ncols, hs)
	pct := columnPercents(rows, ncols)
	var smn, smx, sumPct, freeMx float64
	for i := range ncols {
		smn += mn[i]
		smx += mx[i]
		sumPct += pct[i]
		if pct[i] == 0 {
			freeMx += mx[i]
		}
	}
	if sumPct > 0 {
		// an auto-width table grows so percentage columns get their share
		need := 0.0
		for i := range ncols {
			if pct[i] > 0 {
				need = math.Max(need, mx[i]*100/pct[i])
			}
		}
		if sumPct < 100 {
			need = math.Max(need, freeMx*100/(100-sumPct))
		}
		smx = math.Max(smx, need)
	}
	spacing := 0.0
	if ncols > 0 {
		spacing = hs * float64(ncols+1)
	}
	avail := cbw - e.ml - e.mr
	minW := smn + spacing + edge
	var tableW float64
	switch {
	case forced >= 0:
		tableW = forced
	case !s.Width.Auto:
		tableW = s.Width.Resolve(cbw, 0)
		if s.BoxSizing != "border-box" {
			tableW += edge
		}
	default:
		tableW = math.Min(avail, smx+spacing+edge)
	}
	tableW = math.Max(tableW, minW)
	inner := tableW - edge - spacing
	widths := make([]float64, ncols)
	fixedLayout := s.TableLayout == "fixed" && !s.Width.Auto
	switch {
	case fixedLayout:
		widths = l.fixedColumnWidths(rows, ncols, inner)
	case sumPct > 0:
		widths = distributePercent(mn, mx, pct, inner)
	case inner >= smx:
		extra := inner - smx
		for i := range ncols {
			if smx > 0 {
				widths[i] = mx[i] + extra*mx[i]/smx
			} else {
				widths[i] = inner / float64(ncols)
			}
		}
	case smx > smn:
		f := (inner - smn) / (smx - smn)
		for i := range ncols {
			widths[i] = mn[i] + (mx[i]-mn[i])*f
		}
	default:
		copy(widths, mn)
	}
	ml := e.ml
	if e.mlAuto && e.mrAuto {
		ml = math.Max((cbw-tableW)/2, 0)
	} else if e.mlAuto {
		ml = math.Max(cbw-tableW-e.mr, 0)
	}
	tx := x + ml
	top := l.afterMargin(y, pending)
	startLen := len(l.items)

	// captions above the table box
	for _, c := range captions {
		if c.Style.CaptionSide == "bottom" {
			continue
		}
		r := l.layoutBlock(c, tx, top, tableW, margins{}, true)
		top = r.bottom + r.pending.value()
	}
	bg := &groupItem{}
	l.emit(bg)
	colX := make([]float64, ncols)
	cx := tx + e.bl + e.pl + hs
	for i := range ncols {
		colX[i] = cx
		cx += widths[i] + hs
	}
	rowY := top + bt + pt + vs
	tableTop := top
	var header []Item
	headerH := 0.0
	firstRow := true
	pushedFirst := false
	heights := l.measureRows(rows, colX, widths, hs, vs)
	spanH := func(c *tcell, first int) float64 {
		last := min(first+c.rowspan, len(rows)) - 1
		h := vs * float64(last-first)
		for i := first; i <= last; i++ {
			h += heights[i]
		}
		return h
	}
	for ri, r := range rows {
		h := heights[ri]
		adv := h
		items := l.layoutRow(r, colX, widths, hs, h, func(c *tcell) float64 { return spanH(c, ri) })
		if r.header {
			header = append(header, copyItems(items, 0, headerH)...)
			headerH += adv + vs
		}
		ny := l.place(rowY, h)
		if ny != rowY {
			if firstRow {
				pushedFirst = true
				tableTop = ny
			} else if !r.header && header != nil {
				l.items = append(l.items, copyItems(header, 0, ny)...)
				ny += headerH
			}
		}
		translateItems(items, 0, ny)
		l.items = append(l.items, items...)
		rowY = ny + adv + vs
		firstRow = false
	}
	bottom := rowY + pb + bb
	if len(rows) == 0 {
		bottom = top + bt + pt + pb + bb
	}
	bx := tx
	if pushedFirst && !l.atPageStart(top) {
		// the whole table starts on the next page; captions moved with it is not supported
		top = tableTop
	}
	if !s.BackgroundColor.IsTransparent() {
		bg.items = append(bg.items, &RectItem{X: bx, Y: top, W: tableW, H: bottom - top, Color: s.BackgroundColor})
	}
	saved := l.items
	l.items = nil
	l.emitBorders(s, bx, top, tableW, bottom-top, [4]bool{true, true, true, true})
	bg.items = append(bg.items, l.items...)
	l.items = saved
	for _, c := range captions {
		if c.Style.CaptionSide != "bottom" {
			continue
		}
		r := l.layoutBlock(c, tx, bottom, tableW, margins{}, true)
		bottom = r.bottom + r.pending.value()
	}
	_ = startLen
	return blockResult{top: top, bottom: bottom, pushedFirst: pushedFirst}
}

// columnPercents returns, per column, the largest percentage width of a
// single-column cell in it (0 when none), scaled down so they sum to <= 100.
func columnPercents(rows []*trow, ncols int) []float64 {
	pct := make([]float64, ncols)
	sum := 0.0
	for _, r := range rows {
		for _, c := range r.cells {
			if c.colspan != 1 || c.col >= ncols {
				continue
			}
			if w := c.box.Style.Width; !w.Auto && w.Pct > 0 && w.Pt == 0 {
				pct[c.col] = math.Max(pct[c.col], w.Pct)
			}
		}
	}
	for _, p := range pct {
		sum += p
	}
	if sum > 100 {
		for i := range pct {
			pct[i] *= 100 / sum
		}
	}
	return pct
}

// distributePercent sizes columns when some have percentage widths:
// percentage columns get their share of inner (never below their minimum),
// the rest share what is left in proportion to their max-content widths.
func distributePercent(mn, mx, pct []float64, inner float64) []float64 {
	n := len(mn)
	w := make([]float64, n)
	rest := inner
	var freeMn, freeMx float64
	nFree := 0
	for i := range n {
		if pct[i] > 0 {
			w[i] = math.Max(mn[i], pct[i]/100*inner)
			rest -= w[i]
		} else {
			freeMn += mn[i]
			freeMx += mx[i]
			nFree++
		}
	}
	if nFree == 0 {
		// only percentage columns: spread any remainder proportionally
		if rest > 0 {
			total := inner - rest
			for i := range n {
				if total > 0 {
					w[i] += rest * w[i] / total
				}
			}
		}
		return w
	}
	rest = math.Max(rest, freeMn)
	for i := range n {
		if pct[i] > 0 {
			continue
		}
		switch {
		case rest >= freeMx && freeMx > 0:
			w[i] = mx[i] + (rest-freeMx)*mx[i]/freeMx
		case freeMx > freeMn:
			w[i] = mn[i] + (mx[i]-mn[i])*(rest-freeMn)/(freeMx-freeMn)
		case freeMx == 0:
			w[i] = rest / float64(nFree)
		default:
			w[i] = mn[i]
		}
	}
	return w
}

// fixedColumnWidths implements table-layout: fixed: the first row's cell
// widths (lengths or percentages of the table) set the columns, and columns
// without one share the remaining space equally.
func (l *layouter) fixedColumnWidths(rows []*trow, ncols int, inner float64) []float64 {
	w := make([]float64, ncols)
	set := make([]bool, ncols)
	if len(rows) > 0 {
		for _, c := range rows[0].cells {
			cs := c.box.Style
			if cs.Width.Auto || c.col >= ncols {
				continue
			}
			v := cs.Width.Resolve(inner, 0)
			if cs.BoxSizing != "border-box" {
				v += absPart(cs.Padding[css.Left]) + absPart(cs.Padding[css.Right]) +
					cs.BorderUsed(css.Left) + cs.BorderUsed(css.Right)
			}
			end := min(c.col+c.colspan, ncols)
			for i := c.col; i < end; i++ {
				w[i] = v / float64(end-c.col)
				set[i] = true
			}
		}
	}
	used, nFree := 0.0, 0
	for i := range ncols {
		if set[i] {
			used += w[i]
		} else {
			nFree++
		}
	}
	switch {
	case nFree > 0:
		share := math.Max(inner-used, 0) / float64(nFree)
		for i := range ncols {
			if !set[i] {
				w[i] = share
			}
		}
	case used > 0 && used < inner:
		for i := range ncols {
			w[i] += (inner - used) * w[i] / used
		}
	}
	return w
}

func copyItems(items []Item, dx, dy float64) []Item {
	out := make([]Item, 0, len(items))
	for _, it := range items {
		c := cloneItem(it)
		c.translate(dx, dy)
		out = append(out, c)
	}
	return out
}

func cloneItem(it Item) Item {
	switch v := it.(type) {
	case *RectItem:
		c := *v
		return &c
	case *LineItem:
		c := *v
		return &c
	case *TextItem:
		c := *v
		return &c
	case *ImageItem:
		c := *v
		return &c
	case *PathItem:
		c := *v
		return &c
	case *LinkItem:
		c := *v
		return &c
	case *EllipseItem:
		c := *v
		return &c
	case *AnchorItem:
		c := *v
		c.Name = "" // anchors are not duplicated
		return &c
	case *BookmarkItem:
		c := *v
		c.Title = ""
		return &c
	case *groupItem:
		g := &groupItem{}
		for _, x := range v.items {
			g.items = append(g.items, cloneItem(x))
		}
		return g
	}
	return it
}

// rowCell is a cell of a row with its horizontal placement.
type rowCell struct {
	c    *tcell
	x, w float64
}

func rowCells(r *trow, colX, widths []float64, hs float64) []rowCell {
	var cells []rowCell
	for _, c := range r.cells {
		if c.col >= len(colX) {
			continue
		}
		end := min(c.col+c.colspan, len(colX))
		w := hs * float64(end-c.col-1)
		for i := c.col; i < end; i++ {
			w += widths[i]
		}
		cells = append(cells, rowCell{c: c, x: colX[c.col], w: w})
	}
	return cells
}

// measureRows returns the height of every row. A cell spanning several rows
// that is taller than them together makes the last spanned row taller.
func (l *layouter) measureRows(rows []*trow, colX, widths []float64, hs, vs float64) []float64 {
	heights := make([]float64, len(rows))
	type span struct {
		first, last int
		h           float64
	}
	var spans []span
	for i, r := range rows {
		h := 0.0
		for _, p := range rowCells(r, colX, widths, hs) {
			m := l.sub()
			res := m.layoutBlockW(p.c.box, p.x, 0, p.w, margins{}, true, p.w)
			if p.c.rowspan == 1 {
				h = math.Max(h, res.bottom)
			} else {
				spans = append(spans, span{i, min(i+p.c.rowspan, len(rows)) - 1, res.bottom})
			}
		}
		if rs := r.box.Style; !rs.Height.Auto && !rs.Height.HasPct() {
			h = math.Max(h, rs.Height.Pt)
		}
		heights[i] = h
	}
	for _, sp := range spans {
		have := vs * float64(sp.last-sp.first)
		for i := sp.first; i <= sp.last; i++ {
			have += heights[i]
		}
		if sp.h > have {
			heights[sp.last] += sp.h - have
		}
	}
	return heights
}

// layoutRow lays out one table row at y = 0 with the given height; spanH
// returns the height of a cell spanning several rows. It returns the items.
func (l *layouter) layoutRow(r *trow, colX, widths []float64, hs, rowH float64, spanH func(*tcell) float64) []Item {
	cells := rowCells(r, colX, widths, hs)
	rs := r.box.Style
	out := l.sub()
	if !rs.BackgroundColor.IsTransparent() && len(cells) > 0 {
		x0 := cells[0].x
		x1 := cells[len(cells)-1].x + cells[len(cells)-1].w
		out.emit(&RectItem{X: x0, Y: 0, W: x1 - x0, H: rowH, Color: rs.BackgroundColor})
	}
	for _, p := range cells {
		h := rowH
		if p.c.rowspan > 1 {
			h = spanH(p.c)
		}
		l.e.cellHeight[p.c.box] = h
		out.layoutBlockW(p.c.box, p.x, 0, p.w, margins{}, true, p.w)
		delete(l.e.cellHeight, p.c.box)
	}
	return out.items
}
