// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/pipeline"
	"golang.org/x/net/html"
)

// extractor walks a styled HTML tree and fills a workbook.
type extractor struct {
	ctx    context.Context
	cfg    *core.Config
	styled *pipeline.Styled
	wb     *workbook
	warn   func(string)
	names  map[string]bool
	images *imageStore
	widths map[*sheet]cellWidths
	// lastHeading is the text of the most recent heading, which names the
	// next table that has no caption.
	lastHeading string
	// omitted counts non-table blocks skipped because tables map to sheets.
	omitted int
}

func (e *extractor) style(n *html.Node) *css.Style {
	if es := e.styled.Styles[n]; es != nil && es.Style != nil {
		return es.Style
	}
	return css.Initial()
}

func (e *extractor) hidden(n *html.Node) bool {
	return n.Type == html.ElementNode && e.style(n).Display == "none"
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// run extracts every table into its own sheet. A document without tables
// becomes a single sheet with one row per block of text.
func (e *extractor) run(root *html.Node) error {
	body := findBody(root)
	if body == nil {
		body = root
	}
	var tables []*html.Node
	e.collectTables(body, &tables, true)
	if len(tables) == 0 {
		sh := e.newSheet("Document", "document")
		e.wb.sheets = append(e.wb.sheets, sh)
		e.blocks(body, sh)
		return e.ctx.Err()
	}
	e.lastHeading = ""
	e.collectTables(body, nil, false)
	if e.omitted > 0 {
		e.warn(fmt.Sprintf("xlsx: %d block(s) of text outside tables were not written; each table becomes a sheet", e.omitted))
	}
	return e.ctx.Err()
}

// collectTables finds top-level tables (tables nested in cells are kept as
// cell text). With count=true it only gathers them; otherwise it converts each
// one to a sheet as it is met, tracking headings for sheet names.
func (e *extractor) collectTables(n *html.Node, out *[]*html.Node, count bool) {
	if e.ctx.Err() != nil {
		return
	}
	if n.Type == html.ElementNode {
		if e.hidden(n) {
			return
		}
		switch n.Data {
		case "script", "style", "template", "head", "noscript":
			return
		case "table":
			if count {
				*out = append(*out, n)
			} else {
				e.table(n)
			}
			return
		case "h1", "h2", "h3", "h4", "h5", "h6":
			if !count {
				e.lastHeading = collapseText(plainText(n, e), "normal")
			}
			return
		}
		if !count && isBlockText(n, e) {
			e.omitted++
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		e.collectTables(c, out, count)
	}
}

// isBlockText reports whether n is a block element holding text directly
// (a paragraph or list item), used only to count what a table-sheet
// conversion leaves out.
func isBlockText(n *html.Node, e *extractor) bool {
	switch n.Data {
	case "p", "li", "pre", "blockquote", "dd", "dt", "figcaption":
		return strings.TrimSpace(plainText(n, e)) != ""
	}
	return false
}

func findBody(root *html.Node) *html.Node {
	var body *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if body != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return body
}

func (e *extractor) newSheet(name, src string) *sheet {
	return &sheet{name: uniqueSheetName(sheetName(name), e.names), src: src}
}

// --- tables ---

// tableRow is an HTML row with its style and section.
type tableRow struct {
	node    *html.Node
	section string // thead, tbody, tfoot
	group   *html.Node
}

func (e *extractor) table(t *html.Node) {
	name := ""
	var caption *html.Node
	var colStyles []*css.Style
	var head, body, foot []tableRow
	collect := func(sec *html.Node, section string) {
		for r := sec.FirstChild; r != nil; r = r.NextSibling {
			if r.Type == html.ElementNode && r.Data == "tr" && !e.hidden(r) {
				switch section {
				case "thead":
					head = append(head, tableRow{r, section, sec})
				case "tfoot":
					foot = append(foot, tableRow{r, section, sec})
				default:
					body = append(body, tableRow{r, section, sec})
				}
			}
		}
	}
	for c := t.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || e.hidden(c) {
			continue
		}
		switch c.Data {
		case "caption":
			if caption == nil {
				caption = c
			}
		case "colgroup":
			span := 1
			if _, err := fmt.Sscanf(attr(c, "span"), "%d", &span); err != nil || span < 1 {
				span = 1
			}
			var cols int
			for col := c.FirstChild; col != nil; col = col.NextSibling {
				if col.Type == html.ElementNode && col.Data == "col" {
					cs := 1
					if _, err := fmt.Sscanf(attr(col, "span"), "%d", &cs); err != nil || cs < 1 {
						cs = 1
					}
					for i := 0; i < cs && len(colStyles) < maxCols; i++ {
						colStyles = append(colStyles, e.style(col))
					}
					cols++
				}
			}
			if cols == 0 {
				for i := 0; i < span && len(colStyles) < maxCols; i++ {
					colStyles = append(colStyles, e.style(c))
				}
			}
		case "col":
			cs := 1
			if _, err := fmt.Sscanf(attr(c, "span"), "%d", &cs); err != nil || cs < 1 {
				cs = 1
			}
			for i := 0; i < cs && len(colStyles) < maxCols; i++ {
				colStyles = append(colStyles, e.style(c))
			}
		case "thead", "tbody", "tfoot":
			collect(c, c.Data)
		case "tr":
			body = append(body, tableRow{c, "tbody", t})
		}
	}
	if caption != nil {
		name = collapseText(plainText(caption, e), "normal")
	}
	for _, k := range []string{"data-sheet", "aria-label", "title", "id"} {
		if name == "" {
			name = strings.TrimSpace(attr(t, k))
		}
	}
	if name == "" {
		name = e.lastHeading
	}
	if name == "" {
		name = fmt.Sprintf("Table %d", len(e.wb.sheets)+1)
	}
	sh := e.newSheet(name, "table")
	e.lastHeading = "" // a heading names only the table that follows it
	e.wb.sheets = append(e.wb.sheets, sh)
	ts := e.style(t)
	collapse := ts.BorderCollapse == "collapse"

	rows := append(append(head, body...), foot...)
	sh.freezeRows = len(head)

	// First pass: place cells on the grid following the HTML table model, so
	// that spans push later cells right and the column count is known.
	type placement struct {
		node             *html.Node
		row              tableRow
		r, c, rows, cols int
	}
	var placed []placement
	occupied := map[[2]int]bool{}
	totalCols := 0
	for ri, tr := range rows {
		ri++
		if ri > maxRows {
			e.warn(fmt.Sprintf("xlsx: sheet %q: more than %d rows, the rest is dropped", sh.name, maxRows))
			rows = rows[:ri-1]
			break
		}
		ci := 0
		for cn := tr.node.FirstChild; cn != nil; cn = cn.NextSibling {
			if cn.Type != html.ElementNode || cn.Data != "td" && cn.Data != "th" || e.hidden(cn) {
				continue
			}
			ci++
			for occupied[[2]int{ri, ci}] {
				ci++
			}
			if ci > maxCols {
				e.warn(fmt.Sprintf("xlsx: sheet %q: more than %d columns, the rest is dropped", sh.name, maxCols))
				break
			}
			colspan, rowspan := spanAttr(cn, "colspan", 1000), spanAttr(cn, "rowspan", 65534)
			if rowspan == 0 { // rowspan=0 means to the end of the row group
				rowspan = 1
				for j := ri; j < len(rows) && rows[j].group == tr.group; j++ {
					rowspan++
				}
			}
			colspan = min(colspan, maxCols-ci+1)
			rowspan = min(rowspan, maxRows-ri+1)
			for r := ri; r < ri+rowspan; r++ {
				for c := ci; c < ci+colspan; c++ {
					occupied[[2]int{r, c}] = true
				}
			}
			placed = append(placed, placement{node: cn, row: tr, r: ri, c: ci, rows: rowspan, cols: colspan})
			ci += colspan - 1
			totalCols = max(totalCols, ci)
		}
	}
	// Rows that got no cell still exist (for heights and ordering).
	for ri, tr := range rows {
		rw := sh.rowAt(ri + 1)
		rs := e.style(tr.node)
		if h := rs.Height; !h.Auto && h.Pt > 0 {
			rw.heightPt = roundPt(h.Pt)
		}
		if rs.Visibility == "collapse" {
			rw.hidden = true
		}
	}

	// Second pass: values and styles.
	for _, p := range placed {
		var colStyle *css.Style
		if p.c-1 < len(colStyles) {
			colStyle = colStyles[p.c-1]
		}
		e.cell(sh, p.node, p.r, p.c, p.rows, p.cols, cellContext{
			table: ts, group: e.style(p.row.group), row: e.style(p.row.node), col: colStyle, collapse: collapse,
			firstRow: p.r == 1, lastRow: p.r+p.rows-1 == len(rows), firstCol: p.c == 1, lastCol: p.c+p.cols-1 == totalCols,
			header: p.node.Data == "th" || p.row.section == "thead",
		})
		if p.cols > 1 || p.rows > 1 {
			sh.merges = append(sh.merges, cellRange{p.r, p.c, p.r + p.rows - 1, p.c + p.cols - 1})
			// Merged ranges take the anchor's style on every cell so borders
			// and fills cover the whole area.
			anchor := sh.cellAt(p.r, p.c)
			for r := p.r; r < p.r+p.rows; r++ {
				for c := p.c; c < p.c+p.cols; c++ {
					if r != p.r || c != p.c {
						sh.cellAt(r, c).xf = anchor.xf
					}
				}
			}
		}
	}
	// column widths: from <col> or first-row cell CSS widths, else from content
	for c := 1; c <= totalCols; c++ {
		var w css.Length
		if c-1 < len(colStyles) && colStyles[c-1] != nil {
			w = colStyles[c-1].Width
		}
		if w.Auto || w.Pt <= 0 {
			w = e.firstCellWidth(sh, c)
		}
		if !w.Auto && w.Pt > 0 && w.Pct == 0 {
			sh.colAt(c).width = widthChars(w.Pt/0.75, e.wb.styles.defFontW)
		} else {
			sh.colAt(c).width = e.fitWidth(sh, c)
		}
		if c-1 < len(colStyles) && colStyles[c-1] != nil && colStyles[c-1].Visibility == "collapse" {
			sh.colAt(c).hidden = true
		}
	}
}

func spanAttr(n *html.Node, key string, limit int) int {
	v := 1
	if s := strings.TrimSpace(attr(n, key)); s != "" {
		if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
			v = 1
		}
	}
	if key == "rowspan" && v == 0 {
		return 0
	}
	if v < 1 {
		v = 1
	}
	if v > limit {
		v = limit
	}
	return v
}

// cellWidths remembers CSS widths of the first cell seen in each column.
type cellWidths map[int]css.Length

func (e *extractor) firstCellWidth(sh *sheet, c int) css.Length {
	if w, ok := e.widths[sh][c]; ok {
		return w
	}
	return css.Auto
}

// fitWidth estimates a column width from its longest line of text, the way
// a spreadsheet's auto-fit does, within 8..80 characters.
func (e *extractor) fitWidth(sh *sheet, c int) float64 {
	longest := 0.0
	for _, r := range sh.rows {
		if r == nil || c-1 >= len(r.cells) || r.cells[c-1] == nil {
			continue
		}
		cl := r.cells[c-1]
		txt := cl.text
		if cl.kind == kindNumber {
			txt = fmt.Sprintf("%g", cl.num)
			if f := e.wb.styles.xfs[cl.xf]; f.numFmtID != 0 {
				txt += "0000" // separators, symbols, decimals
			}
		}
		scale := 1.0
		if f := e.wb.styles.fonts[e.wb.styles.xfs[cl.xf].fontID]; f.sizePt > 0 {
			scale = f.sizePt / 11
			if f.bold {
				scale *= 1.1
			}
		}
		for _, line := range strings.Split(txt, "\n") {
			if n := float64(len([]rune(line))) * scale; n > longest {
				longest = n
			}
		}
	}
	if longest == 0 {
		return 0
	}
	w := longest + 2
	if w < 8 {
		w = 8
	}
	if w > 80 {
		w = 80
	}
	return w
}

// cellContext carries the styles around a cell that affect its look.
type cellContext struct {
	table, group, row, col *css.Style
	collapse               bool
	firstRow, lastRow      bool
	firstCol, lastCol      bool
	header                 bool
}

// cell writes one <td>/<th> into the sheet.
func (e *extractor) cell(sh *sheet, n *html.Node, r, c, rowspan, colspan int, cx cellContext) {
	s := e.style(n)
	if _, ok := e.widths[sh]; !ok {
		e.widths[sh] = cellWidths{}
	}
	if _, ok := e.widths[sh][c]; !ok && colspan == 1 {
		e.widths[sh][c] = s.Width
	}
	content := e.inlineContent(n, s)
	cl := sh.cellAt(r, c)
	// A cell whose whole text shares one font (say a bold link) uses that
	// font directly; only mixed text needs rich runs.
	f := content.font
	if content.mixed {
		f = fontOf(s)
	}

	// value
	text := content.text
	if len([]rune(text)) > maxCellChars {
		e.warn(fmt.Sprintf("xlsx: sheet %q cell %s: text longer than %d characters was cut", sh.name, cellRef(r, c), maxCellChars))
		text = string([]rune(text)[:maxCellChars])
	}
	tv, ok := explicit(attr(n, "data-type"), attr(n, "data-value"), text)
	switch {
	case ok && tv.kind == kindFormula:
		cl.kind = kindFormula
		cl.formula = firstNonEmpty(attr(n, "data-formula"), attr(n, "data-value"), text)
		cl.text = ""
	case ok:
	case attr(n, "data-formula") != "":
		tv = typed{kind: kindFormula}
		cl.kind = kindFormula
		cl.formula = attr(n, "data-formula")
		cl.text = text
	case e.cfg.DetectNumbers && !cx.header:
		tv = detect(text)
	default:
		tv = typed{kind: kindString}
	}
	switch tv.kind {
	case kindString:
		if text != "" {
			cl.kind = kindString
			cl.text = text
			if content.mixed {
				cl.runs = content.runs
			}
		}
	case kindNumber:
		cl.kind, cl.num = kindNumber, tv.num
	case kindBool:
		cl.kind, cl.boolean = kindBool, tv.boolean
	}

	// style
	x := xf{fontID: e.wb.styles.fontID(f)}
	x.numFmtID = e.wb.styles.numFmtID(tv.numFmt)
	var layers []css.Color
	if cx.table != nil {
		layers = append(layers, cx.table.BackgroundColor)
	}
	if cx.col != nil {
		layers = append(layers, cx.col.BackgroundColor)
	}
	if cx.group != nil {
		layers = append(layers, cx.group.BackgroundColor)
	}
	if cx.row != nil {
		layers = append(layers, cx.row.BackgroundColor)
	}
	layers = append(layers, s.BackgroundColor)
	x.fillID = e.wb.styles.fillID(blend(layers...))
	bd := borderOf(s)
	if cx.collapse {
		// Collapsing borders: the cell, its row, row group and (on the outer
		// edges) the table all compete for each edge; the widest wins, ties
		// going to the cell, then the row, then the group, then the table.
		edge := func(side int, outer bool) borderSide {
			cands := []*css.Style{s, cx.row, cx.group}
			if outer {
				cands = append(cands, cx.table)
			}
			var best *css.Style
			bestW := -1.0
			for _, c := range cands {
				if c == nil {
					continue
				}
				st := c.BorderStyle[side]
				if st == "hidden" {
					return borderSide{}
				}
				if w := c.BorderUsed(side); w > bestW && st != "none" && st != "" {
					best, bestW = c, w
				}
			}
			if best == nil {
				return borderSide{}
			}
			return borderSideOf(best, side)
		}
		bd.top = edge(css.Top, cx.firstRow)
		bd.bottom = edge(css.Bottom, cx.lastRow)
		bd.left = pickSide(borderSideOf(s, css.Left), cx.firstCol, cx.table, css.Left)
		bd.right = pickSide(borderSideOf(s, css.Right), cx.lastCol, cx.table, css.Right)
	}
	x.borderID = e.wb.styles.borderID(bd)
	x.align.horizontal = horizontalOf(s)
	if x.align.horizontal == "" && cl.kind == kindString && s.Direction == "rtl" {
		x.align.horizontal = "right"
	}
	x.align.vertical = verticalOf(s)
	x.align.wrap = strings.Contains(cl.text, "\n") || (!s.Width.Auto && s.Width.Pt > 0 && s.WhiteSpace != "nowrap" && s.WhiteSpace != "pre")
	if pl := s.Padding[css.Left]; !pl.Auto && pl.Pt >= 7.5 { // >= 10px reads as an indent level
		x.align.indent = int(pl.Pt / 7.5)
		if x.align.indent > 15 {
			x.align.indent = 15
		}
	}
	cl.xf = e.wb.styles.xfID(x)

	if content.link != "" {
		sh.links = append(sh.links, hyperlink{r: r, c: c, target: content.link, tooltip: content.linkTitle})
	}
	for _, img := range content.images {
		e.placeImage(sh, img, r, c, s)
	}
	// Multi-line text needs a taller row for every line to show.
	if lines := strings.Count(cl.text, "\n") + 1; lines > 1 && rowspan == 1 {
		lh := f.sizePt * 1.3
		if lh == 0 {
			lh = 15
		}
		if h := roundPt(float64(lines) * lh); h > sh.rows[r-1].heightPt {
			sh.rows[r-1].heightPt = h
		}
	}
}

// pickSide returns the cell's own border side, or the table's when the cell
// is on that outer edge and the table's border is wider.
func pickSide(own borderSide, outer bool, table *css.Style, side int) borderSide {
	if !outer || table == nil {
		return own
	}
	ts := borderSideOf(table, side)
	if ts.style == "" {
		return own
	}
	if own.style == "" || table.BorderUsed(side) > sideWidth(own) {
		return ts
	}
	return own
}

// sideWidth gives the nominal width of a spreadsheet border style, for
// comparing against CSS widths.
func sideWidth(b borderSide) float64 {
	switch b.style {
	case "":
		return 0
	case "hair":
		return 0.5
	case "thin", "dashed", "dotted":
		return 1
	case "medium", "mediumDashed", "double":
		return 2
	default:
		return 3
	}
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// --- documents without tables ---

// blocks writes the text blocks of a document to one sheet, one per row.
func (e *extractor) blocks(n *html.Node, sh *sheet) {
	r := 0
	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if e.ctx.Err() != nil || n.Type != html.ElementNode || e.hidden(n) {
			return
		}
		switch n.Data {
		case "script", "style", "template", "head", "noscript":
			return
		}
		s := e.style(n)
		if n.Data == "img" {
			r++
			if r > maxRows {
				return
			}
			sh.rowAt(r)
			for _, img := range e.inlineContent(n, s).images {
				e.placeImage(sh, img, r, 1, s)
			}
			return
		}
		if n.Data == "br" || n.Data == "hr" {
			return
		}
		if isLeafBlock(n, e) {
			content := e.inlineContent(n, s)
			if content.text == "" && len(content.images) == 0 {
				return
			}
			r++
			if r > maxRows {
				return
			}
			cl := sh.cellAt(r, 1)
			f := content.font
			if content.mixed {
				f = fontOf(s)
			}
			cl.kind, cl.text = kindString, content.text
			if content.mixed {
				cl.runs = content.runs
			}
			x := xf{fontID: e.wb.styles.fontID(f), fillID: e.wb.styles.fillID(blend(s.BackgroundColor)),
				borderID: e.wb.styles.borderID(borderOf(s))}
			x.align.horizontal = horizontalOf(s)
			x.align.vertical = "top"
			x.align.wrap = true
			cl.xf = e.wb.styles.xfID(x)
			if content.link != "" {
				sh.links = append(sh.links, hyperlink{r: r, c: 1, target: content.link, tooltip: content.linkTitle})
			}
			for _, img := range content.images {
				e.placeImage(sh, img, r, 1, s)
			}
			if lines := strings.Count(content.text, "\n") + 1; lines > 1 {
				sh.rows[r-1].heightPt = roundPt(float64(lines) * f.sizePt * 1.3)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	sh.colAt(1).width = 100
}

// isLeafBlock reports whether n is a block whose content is inline text
// (so it becomes one row) rather than a container of further blocks.
func isLeafBlock(n *html.Node, e *extractor) bool {
	if n.Type != html.ElementNode {
		return false
	}
	s := e.style(n)
	if !s.IsBlockLevel() && s.Display != "list-item" && s.Display != "table-cell" {
		return false
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && !e.hidden(c) {
			cs := e.style(c)
			if cs.IsBlockLevel() || cs.Display == "list-item" || cs.Display == "table" {
				return false
			}
		}
	}
	return true
}
