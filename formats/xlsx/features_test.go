// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/css"
)

func pngDataURL(t *testing.T, w, h int) (string, []byte) {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), buf.Bytes()
}

func TestImagesInCells(t *testing.T) {
	src, data := pngDataURL(t, 40, 20)
	doc := `<table><tr><th>Logo</th><th>Mixed</th></tr>
<tr><td><img src="` + src + `" alt="Acme logo"></td><td>text <img src="` + src + `"></td></tr></table>`

	// Off by default: both pictures float.
	b, _ := convertHTML(t, doc)
	if n := len(b.sheets[0].images); n != 2 {
		t.Fatalf("default: %d floating images, want 2", n)
	}
	if _, ok := b.files["xl/richData/rdrichvalue.xml"]; ok {
		t.Fatal("default: rich value parts written")
	}

	b, _ = convertHTML(t, doc, core.WithImagesInCells(true))
	s := b.sheets[0]
	// The image-only cell holds its picture; the cell with text keeps a floating one.
	if n := len(s.images); n != 1 || s.images[0].col != 1 {
		t.Fatalf("floating images = %+v, want one in column B", s.images)
	}
	if raw := string(s.rawSheet); !strings.Contains(raw, `<c r="A2"`) || !strings.Contains(raw, `vm="1"`) {
		t.Errorf("A2 is not a rich value cell: %s", raw)
	}
	if rv := string(b.files["xl/richData/rdrichvalue.xml"]); !strings.Contains(rv, "Acme logo") {
		t.Errorf("rich value lacks the alt text: %s", rv)
	}
	var media int
	for name, f := range b.files {
		if strings.HasPrefix(name, "xl/media/") {
			media++
			if !bytes.Equal(f, data) {
				t.Errorf("%s differs from the source image", name)
			}
		}
	}
	if media == 0 {
		t.Error("no media part")
	}
	if s.rowsH[2] < 15 {
		t.Errorf("row 2 height = %v, want room for the 20px picture", s.rowsH[2])
	}
}

func TestInternalLinks(t *testing.T) {
	b, _ := convertHTML(t, `<h2 id="totals">Totals</h2>
<table><tr><td><a href="#row-b" title="Jump">to B</a></td><td><a href="#nowhere">lost</a></td></tr></table>
<table id="detail"><caption>Bob's detail</caption>
<tr><td>a</td><td><a href="#totals">back</a></td></tr>
<tr><td id="row-b">b</td><td><a href="#detail">top</a></td></tr></table>`)
	first, second := b.sheets[0], b.sheets[1]
	if len(first.links) != 1 || first.links["A1"] != "#'Bob''s detail'!A2" {
		t.Errorf("sheet 1 links = %v (the link to a missing id must be dropped)", first.links)
	}
	if !strings.Contains(string(first.rawSheet), `tooltip="Jump"`) {
		t.Error("the link title is not the tooltip")
	}
	// A heading id points at the sheet of the table it names; a table id at its sheet.
	if second.links["B1"] != "#'Totals'!A1" || second.links["B2"] != "#'Bob''s detail'!A1" {
		t.Errorf("sheet 2 links = %v", second.links)
	}
}

func TestGradientFill(t *testing.T) {
	b, _ := convertHTML(t, `<table><tr>
<td style="background: linear-gradient(to right, #ff0000, #0000ff)">a</td>
<td style="background: linear-gradient(#fff, #000 25%, rgba(0,0,255,.5))">b</td>
<td style="background: repeating-linear-gradient(red, blue 10px)">c</td></tr></table>`)
	st := string(b.files["xl/styles.xml"])
	for _, want := range []string{
		`<gradientFill><stop position="0"><color rgb="FFFF0000"/></stop><stop position="1"><color rgb="FF0000FF"/></stop></gradientFill>`,
		`<gradientFill degree="90"><stop position="0"><color rgb="FFFFFFFF"/></stop><stop position="0.25"><color rgb="FF000000"/></stop><stop position="1"><color rgb="FF8080FF"/></stop></gradientFill>`,
	} {
		if !strings.Contains(st, want) {
			t.Errorf("styles lack %s\n%s", want, st[strings.Index(st, "<fills"):strings.Index(st, "</fills>")])
		}
	}
	if n := strings.Count(st, "<gradientFill"); n != 2 {
		t.Errorf("%d gradient fills, want 2 (the repeating gradient is not one)", n)
	}
}

func TestGradientOf(t *testing.T) {
	for in, deg := range map[string]float64{
		"linear-gradient(red, blue)":                  90,
		"linear-gradient(to top, red, blue)":          270,
		"linear-gradient(to left, red, blue)":         180,
		"linear-gradient(45deg, red, blue)":           315,
		"linear-gradient(0.25turn, red, blue)":        0,
		"linear-gradient(to bottom right, red, blue)": 45,
		"-webkit-linear-gradient(200grad, red, blue)": 90,
	} {
		g := gradientOf(in)
		if g == nil || g.Degree != deg || len(g.Stops) != 2 {
			t.Errorf("%s = %+v, want degree %v", in, g, deg)
		}
	}
	g := gradientOf("linear-gradient(red, lime, blue, black 90%, white)")
	var pos []float64
	for _, s := range g.Stops {
		pos = append(pos, s.Position)
	}
	if want := []float64{0, 0.3, 0.6, 0.9, 1}; len(pos) != 5 || !near(pos, want) {
		t.Errorf("positions = %v, want %v", pos, want)
	}
	// A translucent stop shows the colour under the cell.
	g = gradientOf("linear-gradient(rgba(0,0,0,.5), black)", css.Color{R: 1, A: 1})
	if got := g.Stops[0].Color; got.ARGB != "FF800000" {
		t.Errorf("stop over red = %+v", got)
	}
	for _, in := range []string{"", "url(x.png)", "linear-gradient(red)", "linear-gradient(red 10px, blue)", "radial-gradient(red, blue)", "linear-gradient(to nowhere, red, blue)"} {
		if g := gradientOf(in); g != nil {
			t.Errorf("%q = %+v, want nil", in, g)
		}
	}
}

func near(a, b []float64) bool {
	for i := range a {
		if d := a[i] - b[i]; d > 1e-9 || d < -1e-9 {
			return false
		}
	}
	return true
}

func TestFormulaCachedValues(t *testing.T) {
	b, _ := convertHTML(t, `<table>
<tr><td>2</td><td>3</td><td data-formula="=A1*B1"></td></tr>
<tr><td data-formula="=NOSUCHFUNCTION(A1)">shown</td><td data-formula="=NOSUCHFUNCTION(B1)"></td><td data-type="formula">=C1&amp;"!"</td></tr></table>`)
	s := b.sheets[0]
	if c := s.cell(t, "C1"); c.f != "A1*B1" || c.v != "6" {
		t.Errorf("C1 = %+v, want the calculated 6", c)
	}
	if c := s.cell(t, "C2"); c.t != "str" || c.v != "6!" {
		t.Errorf("C2 = %+v, want the calculated text", c)
	}
	// A function the engine lacks keeps the cell's text, or no value, never #NAME?.
	if c := s.cell(t, "A2"); c.f != "NOSUCHFUNCTION(A1)" || c.t != "str" || c.v != "shown" {
		t.Errorf("A2 = %+v", c)
	}
	if c := s.cell(t, "B2"); c.f != "NOSUCHFUNCTION(B1)" || c.v != "" {
		t.Errorf("B2 = %+v", c)
	}
	if !strings.Contains(string(b.files["xl/workbook.xml"]), `fullCalcOnLoad="1"`) {
		t.Error("Excel is no longer told to recalculate on load")
	}
}
