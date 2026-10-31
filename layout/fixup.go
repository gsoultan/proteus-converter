package layout

import (
	"strings"

	"github.com/gsoultan/proteus-converter/css"
)

// fixTree normalizes the box tree: anonymous block wrappers for mixed
// content, block-in-inline handling and anonymous table parts.
func fixTree(b *Box) {
	for _, c := range b.Children {
		fixTree(c)
	}
	if b.Marker != nil {
		fixTree(b.Marker)
	}
	switch b.Kind {
	case KindTable:
		fixTable(b)
		return
	case KindRowGroup:
		b.Children = wrapRows(b, b.Children)
		return
	case KindRow:
		b.Children = wrapCells(b, b.Children)
		return
	case KindInline:
		if hasBlockChild(b) {
			// block-in-inline: the inline box becomes a block container
			b.Kind = KindBlock
		} else {
			return
		}
	case KindText, KindImage, KindBreak, KindCheckbox, KindWordBreak:
		return
	}
	// block containers (block, inline-block, cell, caption)
	normalizeBlockChildren(b)
}

// hasBlockChild reports whether b has an in-flow block-level child.
func hasBlockChild(b *Box) bool {
	for _, c := range b.Children {
		if c.isBlockLevel() && !c.outOfFlow() {
			return true
		}
	}
	return false
}

func anonymousBlock(parent *Box) *Box {
	s := parent.Style.Inherit()
	s.Display = "block"
	return &Box{Kind: KindBlock, Style: s, Decorations: parent.Decorations}
}

func normalizeBlockChildren(b *Box) {
	// stray table parts outside a table get an anonymous table
	var tmp []*Box
	for i := 0; i < len(b.Children); i++ {
		c := b.Children[i]
		if c.Kind == KindRowGroup || c.Kind == KindRow || c.Kind == KindCell {
			s := b.Style.Inherit()
			s.Display = "table"
			t := &Box{Kind: KindTable, Style: s}
			for i < len(b.Children) && (b.Children[i].Kind == KindRowGroup || b.Children[i].Kind == KindRow || b.Children[i].Kind == KindCell) {
				t.Children = append(t.Children, b.Children[i])
				i++
			}
			i--
			fixTable(t)
			tmp = append(tmp, t)
			continue
		}
		tmp = append(tmp, c)
	}
	b.Children = tmp
	if !hasBlockChild(b) {
		return
	}
	var out []*Box
	var run []*Box
	flush := func() {
		if len(run) == 0 {
			return
		}
		if !allCollapsibleSpace(run) {
			a := anonymousBlock(b)
			a.Children = run
			out = append(out, a)
		}
		run = nil
	}
	for i, c := range b.Children {
		switch {
		case c.outOfFlow():
			// floats and absolute boxes join adjacent inline content, if any
			if (len(run) > 0 && !allCollapsibleSpace(run)) || nextInFlowIsInline(b.Children[i+1:]) {
				run = append(run, c)
			} else {
				flush()
				out = append(out, c)
			}
		case c.isBlockLevel():
			flush()
			out = append(out, c)
		default:
			run = append(run, c)
		}
	}
	flush()
	b.Children = out
}

func nextInFlowIsInline(boxes []*Box) bool {
	for _, c := range boxes {
		if c.outOfFlow() {
			continue
		}
		if c.isBlockLevel() {
			return false
		}
		if !allCollapsibleSpace([]*Box{c}) {
			return true
		}
	}
	return false
}

func allCollapsibleSpace(boxes []*Box) bool {
	for _, c := range boxes {
		switch c.Kind {
		case KindText:
			ws := c.Style.WhiteSpace
			if ws == "pre" || ws == "pre-wrap" || ws == "break-spaces" {
				return false
			}
			if ws == "pre-line" && strings.Contains(c.Text, "\n") {
				return false
			}
			if strings.TrimLeft(c.Text, " \t\n\r\f") != "" {
				return false
			}
		case KindInline:
			if !allCollapsibleSpace(c.Children) || c.Style.BorderUsed(css.Left) > 0 || c.Style.BorderUsed(css.Right) > 0 ||
				c.Style.Padding[css.Left].Pt > 0 || c.Style.Padding[css.Right].Pt > 0 || c.ID != "" {
				return false
			}
		case KindWordBreak:
		default:
			return false
		}
	}
	return true
}

func fixTable(t *Box) {
	var out []*Box
	var rows []*Box
	flushRows := func() {
		if len(rows) == 0 {
			return
		}
		s := t.Style.Inherit()
		s.Display = "table-row-group"
		g := &Box{Kind: KindRowGroup, Style: s, RowGroup: "body"}
		g.Children = rows
		out = append(out, g)
		rows = nil
	}
	var cells []*Box
	flushCells := func() {
		if len(cells) == 0 {
			return
		}
		s := t.Style.Inherit()
		s.Display = "table-row"
		rows = append(rows, &Box{Kind: KindRow, Style: s, Children: wrapCells(t, cells)})
		cells = nil
	}
	for _, c := range t.Children {
		switch c.Kind {
		case KindRowGroup:
			flushCells()
			flushRows()
			switch c.Style.Display {
			case "table-header-group":
				c.RowGroup = "header"
			case "table-footer-group":
				c.RowGroup = "footer"
			default:
				c.RowGroup = "body"
			}
			out = append(out, c)
		case KindRow:
			flushCells()
			rows = append(rows, c)
		case KindCaption:
			flushCells()
			flushRows()
			out = append(out, c)
		case KindCell:
			cells = append(cells, c)
		case KindText:
			if strings.TrimSpace(c.Text) == "" {
				continue
			}
			cells = append(cells, c)
		default:
			cells = append(cells, c)
		}
	}
	flushCells()
	flushRows()
	t.Children = out
}

func wrapRows(parent *Box, children []*Box) []*Box {
	var out []*Box
	var cells []*Box
	flush := func() {
		if len(cells) == 0 {
			return
		}
		s := parent.Style.Inherit()
		s.Display = "table-row"
		out = append(out, &Box{Kind: KindRow, Style: s, Children: wrapCells(parent, cells)})
		cells = nil
	}
	for _, c := range children {
		if c.Kind == KindRow {
			flush()
			out = append(out, c)
			continue
		}
		if c.Kind == KindText && strings.TrimSpace(c.Text) == "" {
			continue
		}
		cells = append(cells, c)
	}
	flush()
	return out
}

func wrapCells(parent *Box, children []*Box) []*Box {
	var out []*Box
	var run []*Box
	flush := func() {
		if len(run) == 0 {
			return
		}
		s := parent.Style.Inherit()
		s.Display = "table-cell"
		cell := &Box{Kind: KindCell, Style: s, Children: run, ColSpan: 1, RowSpan: 1}
		normalizeBlockChildren(cell)
		out = append(out, cell)
		run = nil
	}
	for _, c := range children {
		if c.Kind == KindCell {
			flush()
			if c.ColSpan == 0 {
				c.ColSpan = 1
			}
			if c.RowSpan == 0 {
				c.RowSpan = 1
			}
			out = append(out, c)
			continue
		}
		if c.Kind == KindText && strings.TrimSpace(c.Text) == "" {
			continue
		}
		run = append(run, c)
	}
	flush()
	return out
}
