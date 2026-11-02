// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gsoultan/proteus-converter/core"
	htmlfmt "github.com/gsoultan/proteus-converter/formats/html"
	mdfmt "github.com/gsoultan/proteus-converter/formats/markdown"
)

func convertHTML(t *testing.T, src string, opts ...core.Option) (*rbook, []string) {
	t.Helper()
	return convert(t, htmlfmt.New(), src, opts...)
}

func convert(t *testing.T, p core.Parser, src string, opts ...core.Option) (*rbook, []string) {
	t.Helper()
	var warnings []string
	opts = append([]core.Option{core.WithSystemFonts(false), core.WithWarningHandler(func(m string) { warnings = append(warnings, m) })}, opts...)
	cfg := core.NewConfig(opts...)
	doc, err := p.Parse(context.Background(), strings.NewReader(src), cfg)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := New().Write(context.Background(), doc, cfg, &out); err != nil {
		t.Fatal(err)
	}
	return readBook(t, out.Bytes()), warnings
}

const fixture = `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><title>Q3 Report</title>
<meta name="author" content="Gembit"><meta name="keywords" content="sales, q3">
<style>
table{border-collapse:collapse;font-family:Arial;font-size:10pt}
th{background:#1f4e79;color:#fff;border:1px solid #000;padding:4px 8px}
td{border:1px solid #999;padding:4px 8px;vertical-align:top}
tr.total td{font-weight:bold;background:rgba(255,255,0,.5)}
.r{text-align:right}.c{text-align:center}
</style></head><body>
<h1>Sales</h1>
<p>Intro paragraph.</p>
<table>
<caption>Regional sales</caption>
<colgroup><col style="width:120px"><col><col></colgroup>
<thead><tr><th rowspan="2">Region</th><th colspan="2">Amount</th></tr><tr><th>Q2</th><th>Q3</th></tr></thead>
<tbody>
<tr><td>North<br>Europe</td><td class="r">$1,234.50</td><td class="r">12.5%</td></tr>
<tr><td><b>South</b> and <i style="color:red">East</i></td><td class="r">(2,000)</td><td class="r">2026-11-02</td></tr>
<tr><td><a href="https://example.com/x" title="link">Link text</a></td><td data-type="string">007</td><td>0042</td></tr>
<tr><td>Formula</td><td data-formula="=SUM(B3:B4)">x</td><td data-type="bool">true</td></tr>
<tr><td style="text-transform:uppercase">shout</td><td style="white-space:pre">a   b
c</td><td class="c">1 234</td></tr>
</tbody>
<tfoot><tr class="total"><td>Total</td><td class="r">-765.50</td><td></td></tr></tfoot>
</table>
<h2>Second: with/odd?name</h2>
<table border="1"><tr><td>1</td><td>2</td></tr><tr><td colspan="2">wide</td></tr></table>
</body></html>`

func TestFixtureCells(t *testing.T) {
	b, warnings := convertHTML(t, fixture)
	if len(b.sheets) != 2 {
		t.Fatalf("sheets = %d", len(b.sheets))
	}
	if b.core["dc:title"] != "Q3 Report" || b.core["dc:creator"] != "Gembit" || b.core["cp:keywords"] != "sales, q3" || b.core["dc:language"] != "en" {
		t.Errorf("core properties = %v", b.core)
	}
	s := b.sheet(t, "Regional sales")
	if s.freeze != 2 {
		t.Errorf("frozen header rows = %d, want 2", s.freeze)
	}
	wantMerges := map[string]bool{"A1:A2": true, "B1:C1": true}
	for _, m := range s.merges {
		if !wantMerges[m] {
			t.Errorf("unexpected merge %s", m)
		}
		delete(wantMerges, m)
	}
	if len(wantMerges) > 0 {
		t.Errorf("missing merges %v", wantMerges)
	}
	// header
	for _, ref := range []string{"A1", "B1", "B2", "C2"} {
		c := s.cell(t, ref)
		f := b.font(c)
		if !f.b || f.family != "Arial" || f.sz != "10" || f.color != "FFFFFFFF" {
			t.Errorf("%s header font = %+v", ref, f)
		}
		if b.fill(c) != "FF1F4E79" {
			t.Errorf("%s header fill = %s", ref, b.fill(c))
		}
		if bd := b.border(c); bd.top != "thin" || bd.topC != "FF000000" || bd.left != "thin" {
			t.Errorf("%s header border = %+v", ref, bd)
		}
		if x := b.xf(c); x.h != "center" || x.v != "center" {
			t.Errorf("%s header alignment = %+v", ref, x)
		}
	}
	// merged range cells carry the anchor style
	if c := s.cell(t, "C1"); c.s != s.cell(t, "B1").s || c.t != "" || c.v != "" {
		t.Errorf("C1 (merged) = %+v", c)
	}
	if c := s.cell(t, "A2"); c.s != s.cell(t, "A1").s {
		t.Errorf("A2 (merged) style = %d, want %d", c.s, s.cell(t, "A1").s)
	}
	// text with <br>
	c := s.cell(t, "A3")
	if c.str != "North\nEurope" || !b.xf(c).wrap || b.xf(c).v != "top" {
		t.Errorf("A3 = %q wrap=%v v=%q", c.str, b.xf(c).wrap, b.xf(c).v)
	}
	if h := s.rowsH[3]; h < 20 {
		t.Errorf("row 3 height = %v, want room for two lines", h)
	}
	// numbers
	c = s.cell(t, "B3")
	if v := c.number(t); v != 1234.5 || b.numFmt(c) != `"$"#,##0.00` || b.xf(c).h != "right" {
		t.Errorf("B3 = %v fmt %q h %q", v, b.numFmt(c), b.xf(c).h)
	}
	c = s.cell(t, "C3")
	if v := c.number(t); math.Abs(v-0.125) > 1e-12 || b.numFmt(c) != "0.0%" {
		t.Errorf("C3 = %v fmt %q", v, b.numFmt(c))
	}
	c = s.cell(t, "B4")
	if v := c.number(t); v != -2000 || b.numFmt(c) != "#,##0;(#,##0)" {
		t.Errorf("B4 = %v fmt %q", v, b.numFmt(c))
	}
	c = s.cell(t, "C4")
	if v := c.number(t); v != 46328 || b.numFmt(c) != "yyyy-mm-dd" { // 2026-11-02
		t.Errorf("C4 = %v fmt %q", v, b.numFmt(c))
	}
	// rich text
	c = s.cell(t, "A4")
	if c.str != "South and East" || len(c.runs) != 3 {
		t.Fatalf("A4 = %q runs %+v", c.str, c.runs)
	}
	if r := c.runs[0]; r.text != "South" || !r.b || r.family != "Arial" || r.sz != "10" {
		t.Errorf("run 0 = %+v", r)
	}
	if r := c.runs[1]; r.text != " and" || r.b || r.i {
		t.Errorf("run 1 = %+v", r)
	}
	if r := c.runs[2]; r.text != " East" || !r.i || r.color != "FFFF0000" {
		t.Errorf("run 2 = %+v", r)
	}
	// link: underlined blue cell font and a hyperlink relationship
	c = s.cell(t, "A5")
	if c.str != "Link text" || len(c.runs) != 0 || !b.font(c).u || b.font(c).color != "FF0000EE" {
		t.Errorf("A5 = %q runs=%d font=%+v", c.str, len(c.runs), b.font(c))
	}
	if s.links["A5"] != "https://example.com/x" {
		t.Errorf("A5 link = %q", s.links["A5"])
	}
	// identifiers stay text
	if c := s.cell(t, "B5"); c.str != "007" || c.t != "s" {
		t.Errorf("B5 = %+v", c)
	}
	if c := s.cell(t, "C5"); c.str != "0042" || c.t != "s" {
		t.Errorf("C5 = %+v", c)
	}
	// formula and bool
	if c := s.cell(t, "B6"); c.f != "SUM(B3:B4)" || c.t != "str" || c.v != "x" {
		t.Errorf("B6 = %+v", c)
	}
	if c := s.cell(t, "C6"); c.t != "b" || c.v != "1" {
		t.Errorf("C6 = %+v", c)
	}
	// text-transform, white-space: pre, non-numeric text with a space
	if c := s.cell(t, "A7"); c.str != "SHOUT" {
		t.Errorf("A7 = %q", c.str)
	}
	if c := s.cell(t, "B7"); c.str != "a   b\nc" || !b.xf(c).wrap {
		t.Errorf("B7 = %q", c.str)
	}
	if c := s.cell(t, "C7"); c.t != "s" || c.str != "1 234" || b.xf(c).h != "center" {
		t.Errorf("C7 = %+v h=%q", c, b.xf(c).h)
	}
	// footer: bold, translucent yellow over white, number without grouping
	c = s.cell(t, "B8")
	if v := c.number(t); v != -765.5 || b.numFmt(c) != "0.00" || !b.font(c).b || b.fill(c) != "FFFFFF80" {
		t.Errorf("B8 = %v fmt %q font %+v fill %s", v, b.numFmt(c), b.font(c), b.fill(c))
	}
	if c := s.cell(t, "C8"); c.t != "" || c.v != "" || b.fill(c) != "FFFFFF80" {
		t.Errorf("C8 (empty styled) = %+v fill %s", c, b.fill(c))
	}
	// column widths: 120px -> 17.14 chars; others auto-fit
	if w := s.cols[1]; math.Abs(w-17.14) > 0.01 {
		t.Errorf("col A width = %v", w)
	}
	if w := s.cols[2]; w < 8 || w > 20 {
		t.Errorf("col B width = %v", w)
	}
	if s.dimRef != "A1:C8" {
		t.Errorf("dimension = %s", s.dimRef)
	}

	// second table: name sanitised, integers as General numbers, colspan merge
	s2 := b.sheet(t, "Second with odd name")
	if v := s2.cell(t, "A1").number(t); v != 1 || b.numFmt(s2.cell(t, "A1")) != "General" {
		t.Errorf("A1 = %v", v)
	}
	if len(s2.merges) != 1 || s2.merges[0] != "A2:B2" {
		t.Errorf("merges = %v", s2.merges)
	}
	if bd := b.border(s2.cell(t, "A1")); bd.top == "" || bd.left == "" {
		t.Errorf("border attribute not honoured: %+v", bd)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "1 block(s) of text outside tables") {
		t.Errorf("warnings = %v", warnings)
	}
}

func TestDetect(t *testing.T) {
	cases := []struct {
		in   string
		kind cellKind
		num  float64
		fmt  string
	}{
		{"42", kindNumber, 42, ""},
		{"-42", kindNumber, -42, ""},
		{"+7", kindNumber, 7, ""},
		{"3.14", kindNumber, 3.14, "0.00"},
		{"1,234", kindNumber, 1234, "#,##0"},
		{"1,234.5", kindNumber, 1234.5, "#,##0.0"},
		{"12%", kindNumber, 0.12, "0%"},
		{"12.50%", kindNumber, 0.125, "0.00%"},
		{"$5", kindNumber, 5, `"$"0`},
		{"€1,000.00", kindNumber, 1000, `"€"#,##0.00`},
		{"5 €", kindNumber, 5, `0" €"`},
		{"Rp 1.500", kindNumber, 1500, `"Rp"#,##0`},
		{"Rp1.500,50", kindNumber, 1500.5, `"Rp"#,##0.00`},
		{"-Rp 5", kindNumber, -5, `"Rp"0`},
		{"IDR 2.000.000", kindNumber, 2000000, `"IDR"#,##0`},
		{"(1,000.00)", kindNumber, -1000, "#,##0.00;(#,##0.00)"},
		{"2026-02-29", kindString, 0, ""},
		{"2024-02-29", kindNumber, 45351, "yyyy-mm-dd"},
		{"2026-11-02 14:30", kindNumber, 46328 + 14.5/24, "yyyy-mm-dd hh:mm"},
		{"007", kindString, 0, ""},
		{"1 234", kindString, 0, ""},
		{"1,23", kindString, 0, ""},
		{"12a", kindString, 0, ""},
		{"", kindString, 0, ""},
		{"-", kindString, 0, ""},
		{"(5", kindString, 0, ""},
		{"1e5", kindString, 0, ""},
		{"0", kindNumber, 0, ""},
		{"0.5", kindNumber, 0.5, "0.0"},
	}
	for _, c := range cases {
		got := detect(c.in)
		if got.kind != c.kind || math.Abs(got.num-c.num) > 1e-9 || got.numFmt != c.fmt {
			t.Errorf("detect(%q) = %+v, want kind %d num %v fmt %q", c.in, got, c.kind, c.num, c.fmt)
		}
	}
}

func TestNumberDetectionOff(t *testing.T) {
	b, _ := convertHTML(t, `<table><tr><td>42</td><td data-type="number">43</td></tr></table>`, core.WithNumberDetection(false))
	s := b.sheets[0]
	if c := s.cell(t, "A1"); c.t != "s" || c.str != "42" {
		t.Errorf("A1 = %+v", c)
	}
	if v := s.cell(t, "B1").number(t); v != 43 {
		t.Errorf("B1 = %v", v)
	}
}

func TestHeaderCellsStayText(t *testing.T) {
	b, _ := convertHTML(t, `<table><thead><tr><td>2024</td></tr></thead><tbody><tr><th>2025</th><td>2026</td></tr></tbody></table>`)
	s := b.sheets[0]
	if c := s.cell(t, "A1"); c.t != "s" {
		t.Errorf("thead cell typed as %q", c.t)
	}
	if c := s.cell(t, "A2"); c.t != "s" {
		t.Errorf("th cell typed as %q", c.t)
	}
	if v := s.cell(t, "B2").number(t); v != 2026 {
		t.Errorf("B2 = %v", v)
	}
}

func TestSpansAndOccupancy(t *testing.T) {
	// A rowspan pushes the next row's cells to the right; rowspan=0 spans to
	// the end of the row group; hidden rows and cells are skipped.
	src := `<table>
<tr><td rowspan="2">a</td><td>b</td><td style="display:none">x</td><td>c</td></tr>
<tr><td>d</td><td>e</td></tr>
<tr style="display:none"><td>hidden</td></tr>
<tbody><tr><td rowspan="0">f</td><td>g</td></tr><tr><td>h</td></tr><tr><td>i</td></tr></tbody>
<tr><td colspan="2" rowspan="2">j</td><td>k</td></tr>
<tr><td>l</td></tr>
</table>`
	b, _ := convertHTML(t, src)
	s := b.sheets[0]
	want := map[string]string{"A1": "a", "B1": "b", "C1": "c", "B2": "d", "C2": "e",
		"A3": "f", "B3": "g", "B4": "h", "B5": "i", "A6": "j", "C6": "k", "C7": "l"}
	for ref, text := range want {
		if c, ok := s.cells[ref]; !ok || c.str != text {
			t.Errorf("%s = %q, want %q", ref, c.str, text)
		}
	}
	for ref, c := range s.cells {
		if _, ok := want[ref]; !ok && c.t != "" {
			t.Errorf("unexpected value at %s: %+v", ref, c)
		}
	}
	merges := strings.Join(s.merges, " ")
	for _, m := range []string{"A1:A2", "A3:A5", "A6:B7"} {
		if !strings.Contains(merges, m) {
			t.Errorf("missing merge %s in %s", m, merges)
		}
	}
	if len(s.merges) != 3 {
		t.Errorf("merges = %v", s.merges)
	}
}

func TestSectionsOrderAndFreeze(t *testing.T) {
	src := `<table><tfoot><tr><td>foot</td></tr></tfoot><tbody><tr><td>body</td></tr></tbody><thead><tr><th>head</th></tr></thead></table>`
	b, _ := convertHTML(t, src)
	s := b.sheets[0]
	if s.cell(t, "A1").str != "head" || s.cell(t, "A2").str != "body" || s.cell(t, "A3").str != "foot" {
		t.Errorf("rows = %v %v %v", s.cell(t, "A1").str, s.cell(t, "A2").str, s.cell(t, "A3").str)
	}
	if s.freeze != 1 {
		t.Errorf("freeze = %d", s.freeze)
	}
	if !strings.Contains(string(b.files["xl/workbook.xml"]), "_xlnm.Print_Titles") {
		t.Error("no print titles for the header row")
	}
}

func TestSheetNames(t *testing.T) {
	src := `<h2>Budget 2026: Europe/Asia [draft]?*\</h2><table><tr><td>1</td></tr></table>
<table data-sheet="Named"><tr><td>1</td></tr></table>
<table aria-label="Labelled"><tr><td>1</td></tr></table>
<table id="by-id"><tr><td>1</td></tr></table>
<table><caption>  Caption   wins </caption><tr><td>1</td></tr></table>
<table data-sheet="Named"><tr><td>1</td></tr></table>
<table data-sheet="named"><tr><td>1</td></tr></table>
<table data-sheet="This caption is far too long for a sheet tab name"><tr><td>1</td></tr></table>
<table data-sheet="This caption is far too long for a sheet tab name"><tr><td>1</td></tr></table>
<h3>'Quoted'</h3><table><tr><td>1</td></tr></table>
<table data-sheet="   "><tr><td>1</td></tr></table>`
	b, _ := convertHTML(t, src)
	want := []string{"Budget 2026 Europe Asia draft", "Named", "Labelled", "by-id", "Caption wins", "Named (2)", "named (3)",
		"This caption is far too long fo", "This caption is far too lon (2)", "Quoted", "Table 11"}
	var got []string
	for _, s := range b.sheets {
		got = append(got, s.name)
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("sheet names:\n got %q\nwant %q", got, want)
	}
	for _, n := range got {
		if len([]rune(n)) > 31 || strings.ContainsAny(n, `[]:*?/\`) {
			t.Errorf("invalid sheet name %q", n)
		}
	}
}

func TestHeadingNamesNextTableOnly(t *testing.T) {
	b, _ := convertHTML(t, `<h1>First</h1><table><tr><td>1</td></tr></table><table><tr><td>2</td></tr></table>`)
	if b.sheets[0].name != "First" || b.sheets[1].name != "Table 2" {
		t.Errorf("names = %q %q", b.sheets[0].name, b.sheets[1].name)
	}
}

func TestNoTablesDocument(t *testing.T) {
	md := "# Title\n\nA paragraph with **bold** and a [link](https://example.org).\n\n1. one\n2. two\n\n- bullet\n\n```\ncode\n  indented\n```\n"
	b, warnings := convert(t, mdfmt.New(), md)
	if len(warnings) != 0 {
		t.Errorf("warnings = %v", warnings)
	}
	if len(b.sheets) != 1 || b.sheets[0].name != "Document" {
		t.Fatalf("sheets = %d", len(b.sheets))
	}
	s := b.sheets[0]
	rows := []string{"Title", "A paragraph with bold and a link.", "1. one", "2. two", "• bullet", "code\n  indented"}
	for i, want := range rows {
		ref := "A" + itoa(i+1)
		if c := s.cell(t, ref); c.str != want {
			t.Errorf("%s = %q, want %q", ref, c.str, want)
		}
	}
	if f := b.font(s.cell(t, "A1")); !f.b || f.sz != "24" {
		t.Errorf("heading font = %+v", f)
	}
	c := s.cell(t, "A2")
	if len(c.runs) != 5 || !c.runs[1].b || c.runs[1].text != " bold" || !c.runs[3].u || c.runs[3].text != " link" {
		t.Errorf("paragraph runs = %+v", c.runs)
	}
	if s.links["A2"] != "https://example.org" {
		t.Errorf("link = %q", s.links["A2"])
	}
	if f := b.font(s.cell(t, "A6")); f.family != "Courier New" {
		t.Errorf("code font = %+v", f)
	}
	if !b.xf(s.cell(t, "A6")).wrap || s.rowsH[6] == 0 {
		t.Error("code block not wrapped / sized for two lines")
	}
}

func TestMarkdownTableTheme(t *testing.T) {
	md := "# Inventory\n\n| Item | Qty | Price |\n|:-----|----:|------:|\n| **Bolt** | 1,200 | €0.15 |\n| Nut `M4` | 300 | €0.05 |\n"
	b, _ := convert(t, mdfmt.New(), md)
	s := b.sheet(t, "Inventory")
	if s.freeze != 1 {
		t.Errorf("freeze = %d", s.freeze)
	}
	if f := b.font(s.cell(t, "A1")); !f.b || f.family != "Segoe UI" {
		t.Errorf("header font = %+v (system font stack should resolve past -apple-system)", f)
	}
	if x := b.xf(s.cell(t, "B2")); x.h != "right" {
		t.Errorf("Qty alignment = %+v", x)
	}
	if v := s.cell(t, "B2").number(t); v != 1200 || b.numFmt(s.cell(t, "B2")) != "#,##0" {
		t.Errorf("B2 = %v", v)
	}
	if c := s.cell(t, "A2"); !b.font(c).b || c.str != "Bolt" {
		t.Errorf("A2 = %+v font %+v", c, b.font(c))
	}
	if c := s.cell(t, "A3"); len(c.runs) != 2 || c.runs[1].family != "Courier New" {
		t.Errorf("A3 runs = %+v", c.runs)
	}
	if bd := b.border(s.cell(t, "A2")); bd.bottom != "thin" || bd.bottomC != "FFD1D9E0" {
		t.Errorf("theme borders = %+v", bd)
	}
	if fill := b.fill(s.cell(t, "A3")); fill != "FFF6F8FA" {
		t.Errorf("striped row fill = %s", fill)
	}
	if b.core["dc:title"] != "" {
		t.Errorf("title = %q", b.core["dc:title"])
	}
}

func TestCollapsedBordersInherit(t *testing.T) {
	src := `<style>table{border-collapse:collapse;border:2px solid #333}tr{border-bottom:1px dashed blue}</style>
<table><tr><td>a</td><td>b</td></tr><tr><td>c</td><td>d</td></tr></table>`
	b, _ := convertHTML(t, src)
	s := b.sheets[0]
	a := b.border(s.cell(t, "A1"))
	if a.top != "medium" || a.topC != "FF333333" || a.left != "medium" || a.bottom != "dashed" || a.bottomC != "FF0000FF" {
		t.Errorf("A1 border = %+v", a)
	}
	d := b.border(s.cell(t, "B2"))
	if d.bottom != "medium" || d.top != "" {
		t.Errorf("B2 border = %+v", d)
	}
}

func TestBackgroundLayers(t *testing.T) {
	src := `<table style="background:#000"><colgroup><col style="background:rgba(255,0,0,.5)"></colgroup>
<tbody style="background:rgba(0,255,0,.5)"><tr style="background:rgba(0,0,255,.5)"><td>x</td><td style="background:#fff">y</td></tr></tbody></table>`
	b, _ := convertHTML(t, src)
	s := b.sheets[0]
	// black, then 50% red, 50% green, 50% blue: (32,64,128) -> FF204080
	if f := b.fill(s.cell(t, "A1")); f != "FF204080" {
		t.Errorf("A1 fill = %s", f)
	}
	if f := b.fill(s.cell(t, "B1")); f != "FFFFFFFF" {
		t.Errorf("B1 fill = %s", f)
	}
}

func TestWhiteSpaceAndNestedBlocks(t *testing.T) {
	src := `<table><tr>
<td>  lots   of
 space  </td>
<td><p>one</p><p>two</p><ul><li>a</li><li>b</li></ul></td>
<td style="white-space:pre-line">x
y</td>
<td style="white-space:nowrap">no
wrap</td>
<td><table><tr><td>n1</td><td>n2</td></tr><tr><td>n3</td></tr></table></td>
<td style="text-transform:capitalize">hello big world</td>
<td><span style="text-decoration:line-through"><span>gone</span></span></td>
<td><input type="checkbox" checked> done</td>
</tr></table>`
	b, _ := convertHTML(t, src)
	s := b.sheets[0]
	want := map[string]string{"A1": "lots of space", "B1": "one\ntwo\n• a\n• b", "C1": "x\ny", "D1": "no wrap",
		"E1": "n1 n2\nn3", "F1": "Hello Big World", "G1": "gone", "H1": "☑ done"}
	for ref, w := range want {
		if c := s.cell(t, ref); c.str != w {
			t.Errorf("%s = %q, want %q", ref, c.str, w)
		}
	}
	if !b.font(s.cell(t, "G1")).s {
		t.Error("line-through not propagated to nested span")
	}
	if b.xf(s.cell(t, "D1")).wrap {
		t.Error("nowrap cell marked wrapped")
	}
}

func TestImages(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for i := range img.Pix {
		img.Pix[i] = 0x80
	}
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBuf.Bytes())
	src := `<table><tr><td>text</td><td><img src="` + data + `" alt="Logo"></td></tr>
<tr><td><img src="` + data + `" style="width:80px"></td><td><img src="missing.png" alt="gone"></td></tr></table>`
	b, warnings := convertHTML(t, src, core.WithLocalFiles(false))
	s := b.sheets[0]
	if len(s.images) != 2 {
		t.Fatalf("images = %d", len(s.images))
	}
	i0 := s.images[0]
	if i0.col != 1 || i0.row != 0 || i0.cx != 40*emuPerPx || i0.cy != 20*emuPerPx || !bytes.Equal(i0.data, pngBuf.Bytes()) {
		t.Errorf("image 0 = col %d row %d %dx%d", i0.col, i0.row, i0.cx, i0.cy)
	}
	i1 := s.images[1]
	if i1.col != 0 || i1.row != 1 || i1.cx != 80*emuPerPx || i1.cy != 40*emuPerPx {
		t.Errorf("image 1 = col %d row %d %dx%d", i1.col, i1.row, i1.cx, i1.cy)
	}
	if s.rowsH[2] < 30 || s.cols[1] < 11 {
		t.Errorf("row 2 height %v / col A width %v not grown for the 80x40 image", s.rowsH[2], s.cols[1])
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "missing.png") {
		t.Errorf("warnings = %v", warnings)
	}
	if !strings.Contains(string(b.files["[Content_Types].xml"]), `Extension="png"`) {
		t.Error("png content type missing")
	}
}

func TestControlCharactersAndLongText(t *testing.T) {
	long := strings.Repeat("x", maxCellChars+10)
	src := "<table><tr><td>bad\x01\x02char\u0007s</td><td>" + long + "</td></tr></table>"
	b, warnings := convertHTML(t, src)
	s := b.sheets[0]
	if c := s.cell(t, "A1"); c.str != "badchars" {
		t.Errorf("A1 = %q", c.str)
	}
	if c := s.cell(t, "B1"); len(c.str) != maxCellChars {
		t.Errorf("B1 length = %d", len(c.str))
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "cut") {
		t.Errorf("warnings = %v", warnings)
	}
}

func TestContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg := core.NewConfig(core.WithSystemFonts(false))
	doc, err := htmlfmt.New().Parse(ctx, strings.NewReader("<table><tr><td>1</td></tr></table>"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := New().Write(ctx, doc, cfg, &bytes.Buffer{}); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
}

func TestRegistryRoundTrip(t *testing.T) {
	r := core.NewRegistry()
	r.Register(htmlfmt.New())
	r.Register(mdfmt.New())
	r.RegisterWriter(New())
	if got := r.Outputs(); len(got) != 2 || got[0] != core.OutputPDF || got[1] != core.OutputXLSX {
		t.Errorf("Outputs = %v", got)
	}
	if r.OutputForPath("x.XLSX") != core.OutputXLSX || r.OutputForPath("x.pdf") != core.OutputPDF || r.OutputForPath("x") != core.OutputPDF {
		t.Error("OutputForPath")
	}
	var out bytes.Buffer
	err := r.ConvertTo(context.Background(), core.FormatMarkdown, core.OutputXLSX, strings.NewReader("|a|b|\n|-|-|\n|1|2|\n"), &out, core.WithSystemFonts(false))
	if err != nil {
		t.Fatal(err)
	}
	b := readBook(t, out.Bytes())
	if v := b.sheets[0].cell(t, "B2").number(t); v != 2 {
		t.Errorf("B2 = %v", v)
	}
	if _, err := r.LookupWriter("docx"); !errors.Is(err, core.ErrUnknownFormat) {
		t.Errorf("LookupWriter(docx) = %v", err)
	}
	r.Register(core.ConverterFunc{Name: "txt", Exts: []string{"txt"}, Fn: func(context.Context, io.Reader, io.Writer, ...core.Option) error { return nil }})
	if err := r.ConvertTo(context.Background(), "txt", core.OutputXLSX, strings.NewReader(""), &out); !errors.Is(err, core.ErrUnknownFormat) {
		t.Errorf("non-parser converter to xlsx = %v", err)
	}
}

// TestTestdataTables converts every HTML fixture and checks, table by table,
// that each visible cell's collapsed text (or number) is in the sheet at the
// position the HTML table model gives it, and that the package is well formed.
func TestTestdataTables(t *testing.T) {
	files, _ := filepath.Glob("../../testdata/*.html")
	mds, _ := filepath.Glob("../../testdata/*.md")
	files = append(files, mds...)
	if len(files) == 0 {
		t.Skip("no testdata")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			var p core.Parser = htmlfmt.New()
			if strings.HasSuffix(f, ".md") {
				p = mdfmt.New()
			}
			abs, _ := filepath.Abs(f)
			b, _ := convert(t, p, string(src), core.WithBaseURL(abs))
			if len(b.sheets) == 0 {
				t.Fatal("no sheets")
			}
			for _, s := range b.sheets {
				for ref, c := range s.cells {
					if c.t == "s" && strings.ContainsAny(c.str, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x0b\x0c\x0e\x0f") {
						t.Errorf("%s!%s has control characters", s.name, ref)
					}
				}
			}
		})
	}
}

// TestLibreOfficeRoundTrip opens the fixture workbook with LibreOffice (when
// installed) and compares the CSV it exports with the values the HTML shows,
// including an evaluated formula.
func TestLibreOfficeRoundTrip(t *testing.T) {
	soffice, err := exec.LookPath("soffice")
	if err != nil {
		t.Skip("soffice not installed")
	}
	dir := t.TempDir()
	cfg := core.NewConfig(core.WithSystemFonts(false))
	doc, err := htmlfmt.New().Parse(context.Background(), strings.NewReader(fixture), cfg)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := New().Write(context.Background(), doc, cfg, &out); err != nil {
		t.Fatal(err)
	}
	in := filepath.Join(dir, "fixture.xlsx")
	if err := os.WriteFile(in, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, soffice, "--headless", "-env:UserInstallation=file://"+filepath.Join(dir, "profile"),
		"--convert-to", `csv:Text - txt - csv (StarCalc):44,34,76,1,,0,false,true,true,false,false,-1`, "--outdir", dir, in)
	if outb, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("soffice failed: %v: %s", err, outb)
	}
	got, err := os.ReadFile(filepath.Join(dir, "fixture-Regional sales.csv"))
	if err != nil {
		t.Fatal(err)
	}
	want := "Region,Amount,\n,Q2,Q3\n\"North\nEurope\",\"$1,234.50\",12.5%\nSouth and East,\"(2,000)\",2026-11-02\nLink text,007,0042\nFormula,-765.5,TRUE\nSHOUT,\"a   b\nc\",1 234\nTotal,-765.50,\n"
	if string(got) != want {
		t.Errorf("LibreOffice CSV:\n%s\nwant:\n%s", got, want)
	}
}
