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
	s := t.Style
	rows, ncols, _ := tableGrid(t)
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
	s := t.Style
	rows, ncols, captions := tableGrid(t)
	hs, vs := tableSpacing(s)
	collapse := s.BorderCollapse == "collapse"
	e := l.hEdges(s, cbw)
	bt, bb := s.BorderUsed(css.Top), s.BorderUsed(css.Bottom)
	pt, pb := s.Padding[css.Top].Resolve(cbw, 0), s.Padding[css.Bottom].Resolve(cbw, 0)
	if collapse {
		e.pl, e.pr, pt, pb = 0, 0, 0, 0
	}
	edge := e.bl + e.br + e.pl + e.pr
	mn, mx := l.columnWidths(rows, ncols, hs)
	var smn, smx float64
	for i := range ncols {
		smn += mn[i]
		smx += mx[i]
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
	switch {
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
	top := y + pending.value()
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
	for _, r := range rows {
		items, h, adv := l.layoutRow(r, colX, widths, hs, collapse)
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

// layoutRow lays out one table row at y = 0. It returns the items, the row
// height, and the vertical advance to the next row.
func (l *layouter) layoutRow(r *trow, colX, widths []float64, hs float64, collapse bool) ([]Item, float64, float64) {
	type placed struct {
		c      *tcell
		x, w   float64
		bt, bb float64
	}
	var cells []placed
	rowH := 0.0
	maxBT, maxBB := 0.0, 0.0
	for _, c := range r.cells {
		if c.col >= len(colX) {
			continue
		}
		end := min(c.col+c.colspan, len(colX))
		w := hs * float64(end-c.col-1)
		for i := c.col; i < end; i++ {
			w += widths[i]
		}
		x := colX[c.col]
		cs := c.box.Style
		bt, bb := cs.BorderUsed(css.Top), cs.BorderUsed(css.Bottom)
		cells = append(cells, placed{c: c, x: x, w: w, bt: bt, bb: bb})
		maxBT = math.Max(maxBT, bt)
		maxBB = math.Max(maxBB, bb)
		// measuring pass
		m := l.sub()
		res := m.layoutBlockW(c.box, x, 0, w, margins{}, true, w)
		if c.rowspan == 1 {
			rowH = math.Max(rowH, res.bottom)
		}
	}
	rs := r.box.Style
	if !rs.Height.Auto && !rs.Height.HasPct() {
		rowH = math.Max(rowH, rs.Height.Pt)
	}
	out := l.sub()
	if !rs.BackgroundColor.IsTransparent() && len(cells) > 0 {
		x0 := cells[0].x
		x1 := cells[len(cells)-1].x + cells[len(cells)-1].w
		out.emit(&RectItem{X: x0, Y: 0, W: x1 - x0, H: rowH, Color: rs.BackgroundColor})
	}
	for _, p := range cells {
		l.e.cellHeight[p.c.box] = rowH
		out.layoutBlockW(p.c.box, p.x, 0, p.w, margins{}, true, p.w)
		delete(l.e.cellHeight, p.c.box)
	}
	return out.items, rowH, rowH
}
