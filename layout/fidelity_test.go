package layout_test

import (
	"testing"

	"github.com/gsoultan/proteus-converter/layout"
)

// td align=center centres a fixed-width nested table (Blink's -webkit-center),
// while the nested table's own cells go back to start alignment.
func TestAlignAttributeCentresNestedTable(t *testing.T) {
	items := layoutHTML(t, reset+`<table width="100%" cellspacing=0 cellpadding=0><tr><td align=center>
		<table width=200 cellspacing=0 cellpadding=0 bgcolor="#ff0000"><tr><td>left</td></tr></table>
	</td></tr></table>`)
	inner := rect(t, items, 255, 0, 0)
	near(t, "nested table x", inner.X, (pageW-150)/2)
	near(t, "cell text x", text(t, items, "left").X, inner.X)
}

// Percentage heights resolve against a definite containing block height.
func TestPercentageHeights(t *testing.T) {
	items := layoutHTML(t, reset+`<div style="height:200px"><div style="height:25%;background:#ff0000"></div></div>
		<div style="display:flex;align-items:flex-end;height:100px"><div style="width:10px;height:40%;background:#00ff00"></div></div>
		<div><div style="height:50%;background:#0000ff">auto</div></div>`)
	near(t, "block 25% of 200px", rect(t, items, 255, 0, 0).H, 50*0.75)
	g := rect(t, items, 0, 255, 0)
	near(t, "flex item 40% of 100px", g.H, 40*0.75)
	near(t, "flex item aligned to the end", g.Y, (200+60)*0.75)
	near(t, "indefinite base acts as auto", rect(t, items, 0, 0, 255).H, 20*0.75)
}

// A rowspan cell taller than its rows spreads the excess in proportion to
// the rows' heights (TablesNG), not all into the last row.
func TestRowspanExcessIsProportional(t *testing.T) {
	items := layoutHTML(t, reset+`<table style="border-spacing:0">
		<tr><td style="padding:0"><div style="width:10px;height:20px;background:#ff0000"></div></td>
		    <td rowspan=2 style="padding:0"><div style="width:10px;height:200px"></div></td></tr>
		<tr><td style="padding:0;vertical-align:top"><div style="width:10px;height:60px;background:#00ff00"></div></td></tr></table>`)
	// rows 20 and 60 share 120px of excess: 30 and 90, so row 2 starts at 50px
	near(t, "second row y", rect(t, items, 0, 255, 0).Y, 50*0.75)
}

// ::first-letter with float makes a drop cap that the paragraph wraps around.
func TestFirstLetterFloat(t *testing.T) {
	items := layoutHTML(t, reset+`<style>p{margin:0;width:200px}p::first-letter{float:left;font-size:40px;line-height:1}</style>
		<p>Every few years someone declares the paper notebook dead.</p>`)
	e := text(t, items, "E")
	if e.Text != "E" || e.Size != 30 {
		t.Fatalf("drop cap = %q at %.1fpt, want E at 30pt", e.Text, e.Size)
	}
	if v := text(t, items, "very"); v.X <= e.X {
		t.Errorf("text x %.2f does not wrap around the drop cap at %.2f", v.X, e.X)
	}
}

// Outer box-shadows are painted under the box (a blur adds layers), and an
// outline is painted outside the border box.
func TestBoxShadowAndOutline(t *testing.T) {
	items := layoutHTML(t, reset+`<div style="margin:20px;width:100px;height:50px;background:#ffffff;
		box-shadow:4px 4px 0 #ff0000;outline:2px solid #00ff00;outline-offset:2px"></div>`)
	s := rect(t, items, 255, 0, 0)
	near(t, "shadow x", s.X, 24*0.75)
	near(t, "shadow w", s.W, 100*0.75)
	found := false
	for _, it := range items {
		if r, ok := it.(*layout.RectItem); ok && r.Color.G == 1 && r.Color.R == 0 && r.Y < 20*0.75 {
			found = true
		}
	}
	if !found {
		t.Error("no outline painted above the box")
	}
}
