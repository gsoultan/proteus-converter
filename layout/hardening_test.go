package layout_test

import (
	"testing"

	"github.com/gsoultan/proteus-converter/layout"
)

// Cells of a 1px collapsed-border table keep no right border themselves; the
// table paints the outer half, so the table's own border must be drawn.
func TestCollapsedTablePaintsOuterBorders(t *testing.T) {
	items := layoutHTML(t, reset+`<style>
		table{border-collapse:collapse;width:100%}
		td{border:1px solid #ff0000;padding:0}
	</style><table><tr><td>a</td><td>b</td></tr></table>`)
	right := false
	for _, it := range items {
		if r, ok := it.(*layout.RectItem); ok && same(r.Color, 255, 0, 0) && r.W < 1 && r.X+r.W > pageW-1 {
			right = true
		}
	}
	if !right {
		t.Error("no right table border painted")
	}
}

func TestTablePercentAndFixedColumns(t *testing.T) {
	items := layoutHTML(t, reset+`<style>table{border-spacing:0;width:400pt}td{padding:0}</style>
	<table><tr><td style="width:25%;background:#ff0000">a</td><td style="background:#00ff00">b</td></tr></table>
	<table style="table-layout:fixed"><tr><td style="width:100pt;background:#0000ff">c</td><td style="background:#ffff00">d</td><td>e</td></tr></table>`)
	near(t, "25% column width", rect(t, items, 255, 0, 0).W, 100)
	near(t, "other column width", rect(t, items, 0, 255, 0).W, 300)
	near(t, "fixed first column", rect(t, items, 0, 0, 255).W, 100)
	near(t, "fixed free column", rect(t, items, 255, 255, 0).W, 150)
}

func TestTableRowspanAndVerticalAlign(t *testing.T) {
	items := layoutHTML(t, reset+`<style>table{border-spacing:0}td{padding:0}</style>
	<table><tr><td rowspan=2 style="background:#ff0000">tall</td><td style="height:30px">r1</td></tr>
	<tr><td style="height:30px">r2</td></tr></table>
	<table><tr><td style="height:60px;vertical-align:middle">mid</td><td style="height:60px;vertical-align:bottom">bot</td></tr></table>`)
	near(t, "rowspan cell height", rect(t, items, 255, 0, 0).H, 60*0.75)
	top := 60 * 0.75
	mid, bot := text(t, items, "mid"), text(t, items, "bot")
	// 20px line box: middle sits 20px down, bottom 40px down
	if d := mid.Baseline - top; d < 20*0.75 || d > 40*0.75 {
		t.Errorf("middle baseline offset %.2f not centered", d)
	}
	near(t, "bottom aligned baseline - middle", bot.Baseline-mid.Baseline, 20*0.75)
}

// A margin that crosses a page boundary is truncated: the next block starts
// at the top of the new page, like Chromium.
func TestMarginTruncatedAtPageBreak(t *testing.T) {
	items := pagesHTML(t, reset+`<style>p{margin:0}</style>
	<div style="height:790px"></div><h2 style="margin:40px 0 0;font-size:12px">next</h2>`)
	if len(items) < 2 {
		t.Fatalf("want 2 pages, got %d", len(items))
	}
	h := text(t, items[1], "next")
	if h.Baseline > 20*0.75 {
		t.Errorf("heading baseline %.2f on page 2: top margin not truncated", h.Baseline)
	}
}

func TestPaginateMaxPages(t *testing.T) {
	eng := &layout.Engine{PageWidth: 100, PageHeight: 100, MaxPages: 3}
	_, err := eng.Paginate([]layout.Item{&layout.RectItem{X: 0, Y: 0, W: 10, H: 1e9}})
	if err != layout.ErrTooManyPages {
		t.Fatalf("err = %v, want ErrTooManyPages", err)
	}
	pages, err := eng.Paginate([]layout.Item{&layout.RectItem{X: 0, Y: 150, W: 10, H: 10}})
	if err != nil || len(pages) != 2 {
		t.Fatalf("pages = %d, err = %v", len(pages), err)
	}
}
