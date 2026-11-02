// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The reader below parses the workbook back from the .xlsx bytes without
// any help from the writer's own types, so that tests check what a
// spreadsheet application would actually read.

type rbook struct {
	sheets  []*rsheet
	strings []rstring
	styles  rstyles
	core    map[string]string
	files   map[string][]byte
}

type rstring struct {
	text string
	runs []rrun
}

type rrun struct {
	text   string
	b, i   bool
	u, s   bool
	sz     string
	color  string
	family string
}

type rsheet struct {
	name     string
	cells    map[string]rcell
	merges   []string
	links    map[string]string // ref -> target
	freeze   int
	cols     map[int]float64
	hiddenC  map[int]bool
	rowsH    map[int]float64
	images   []rimage
	dimRef   string
	rawSheet []byte
}

type rimage struct {
	col, row int
	cx, cy   int64
	target   string
	data     []byte
}

type rcell struct {
	ref     string
	t       string // "", s, b, str
	v       string
	f       string
	s       int
	str     string // resolved string value
	runs    []rrun
	hasCell bool
}

type rstyles struct {
	fonts   []rrun
	fills   []string
	borders []rborder
	xfs     []rxf
	numFmts map[int]string
}

type rborder struct {
	left, right, top, bottom     string
	leftC, rightC, topC, bottomC string
}

type rxf struct {
	font, fill, border, numFmt int
	h, v                       string
	wrap                       bool
	indent                     int
}

func (c rcell) number(t *testing.T) float64 {
	t.Helper()
	if c.t != "" {
		t.Fatalf("cell %s is not a number (t=%q v=%q)", c.ref, c.t, c.v)
	}
	f, err := strconv.ParseFloat(c.v, 64)
	if err != nil {
		t.Fatalf("cell %s: %v", c.ref, err)
	}
	return f
}

func readBook(t *testing.T, data []byte) *rbook {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	b := &rbook{files: map[string][]byte{}, core: map[string]string{}}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b.files[f.Name], _ = io.ReadAll(rc)
		rc.Close()
	}
	for name, body := range b.files {
		if strings.HasSuffix(name, ".xml") || strings.HasSuffix(name, ".rels") {
			d := xml.NewDecoder(bytes.NewReader(body))
			for {
				_, err := d.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("%s: malformed XML: %v", name, err)
				}
			}
		}
	}
	for _, must := range []string{"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", "xl/styles.xml", "xl/sharedStrings.xml", "docProps/core.xml", "docProps/app.xml"} {
		if _, ok := b.files[must]; !ok {
			t.Fatalf("missing part %s", must)
		}
	}
	// core properties
	for _, tag := range []string{"dc:title", "dc:creator", "dc:subject", "cp:keywords", "dc:language"} {
		if m := regexp.MustCompile("<" + tag + ">(.*?)</" + tag + ">").FindSubmatch(b.files["docProps/core.xml"]); m != nil {
			b.core[tag] = unesc(string(m[1]))
		}
	}
	// shared strings
	type xr struct {
		RPr struct {
			B  *struct{} `xml:"b"`
			I  *struct{} `xml:"i"`
			U  *struct{} `xml:"u"`
			S  *struct{} `xml:"strike"`
			Sz struct {
				Val string `xml:"val,attr"`
			} `xml:"sz"`
			Color struct {
				Rgb string `xml:"rgb,attr"`
			} `xml:"color"`
			Font struct {
				Val string `xml:"val,attr"`
			} `xml:"rFont"`
		} `xml:"rPr"`
		T string `xml:"t"`
	}
	var sst struct {
		SI []struct {
			T *string `xml:"t"`
			R []xr    `xml:"r"`
		} `xml:"si"`
	}
	if err := xml.Unmarshal(b.files["xl/sharedStrings.xml"], &sst); err != nil {
		t.Fatal(err)
	}
	for _, si := range sst.SI {
		var rs rstring
		if si.T != nil {
			rs.text = *si.T
		}
		for _, r := range si.R {
			rs.runs = append(rs.runs, rrun{text: r.T, b: r.RPr.B != nil, i: r.RPr.I != nil, u: r.RPr.U != nil, s: r.RPr.S != nil,
				sz: r.RPr.Sz.Val, color: r.RPr.Color.Rgb, family: r.RPr.Font.Val})
			rs.text += r.T
		}
		b.strings = append(b.strings, rs)
	}
	// styles
	var st struct {
		NumFmts []struct {
			ID   int    `xml:"numFmtId,attr"`
			Code string `xml:"formatCode,attr"`
		} `xml:"numFmts>numFmt"`
		Fonts []struct {
			B  *struct{} `xml:"b"`
			I  *struct{} `xml:"i"`
			U  *struct{} `xml:"u"`
			S  *struct{} `xml:"strike"`
			Sz struct {
				Val string `xml:"val,attr"`
			} `xml:"sz"`
			Color struct {
				Rgb string `xml:"rgb,attr"`
			} `xml:"color"`
			Name struct {
				Val string `xml:"val,attr"`
			} `xml:"name"`
		} `xml:"fonts>font"`
		Fills []struct {
			P struct {
				Type string `xml:"patternType,attr"`
				Fg   struct {
					Rgb string `xml:"rgb,attr"`
				} `xml:"fgColor"`
			} `xml:"patternFill"`
		} `xml:"fills>fill"`
		Borders []struct {
			L, R, T, B struct {
				Style string `xml:"style,attr"`
				Color struct {
					Rgb string `xml:"rgb,attr"`
				} `xml:"color"`
			}
		} `xml:"borders>border"`
		Xfs []struct {
			NumFmt int `xml:"numFmtId,attr"`
			Font   int `xml:"fontId,attr"`
			Fill   int `xml:"fillId,attr"`
			Border int `xml:"borderId,attr"`
			Align  *struct {
				H      string `xml:"horizontal,attr"`
				V      string `xml:"vertical,attr"`
				Wrap   int    `xml:"wrapText,attr"`
				Indent int    `xml:"indent,attr"`
			} `xml:"alignment"`
		} `xml:"cellXfs>xf"`
	}
	// borders have named children; decode them by hand
	if err := xml.Unmarshal(b.files["xl/styles.xml"], &st); err != nil {
		t.Fatal(err)
	}
	b.styles.numFmts = map[int]string{}
	for _, nf := range st.NumFmts {
		b.styles.numFmts[nf.ID] = nf.Code
	}
	for _, f := range st.Fonts {
		b.styles.fonts = append(b.styles.fonts, rrun{b: f.B != nil, i: f.I != nil, u: f.U != nil, s: f.S != nil, sz: f.Sz.Val, color: f.Color.Rgb, family: f.Name.Val})
	}
	for _, f := range st.Fills {
		if f.P.Type == "solid" {
			b.styles.fills = append(b.styles.fills, f.P.Fg.Rgb)
		} else {
			b.styles.fills = append(b.styles.fills, "")
		}
	}
	reBorder := regexp.MustCompile(`<border>(.*?)</border>`)
	reSide := regexp.MustCompile(`<(left|right|top|bottom) style="([^"]*)">(?:<color rgb="([^"]*)"/>|<color auto="1"/>)</(?:left|right|top|bottom)>`)
	for _, m := range reBorder.FindAllSubmatch(b.files["xl/styles.xml"], -1) {
		var bd rborder
		for _, sm := range reSide.FindAllSubmatch(m[1], -1) {
			switch string(sm[1]) {
			case "left":
				bd.left, bd.leftC = string(sm[2]), string(sm[3])
			case "right":
				bd.right, bd.rightC = string(sm[2]), string(sm[3])
			case "top":
				bd.top, bd.topC = string(sm[2]), string(sm[3])
			case "bottom":
				bd.bottom, bd.bottomC = string(sm[2]), string(sm[3])
			}
		}
		b.styles.borders = append(b.styles.borders, bd)
	}
	for _, x := range st.Xfs {
		rx := rxf{font: x.Font, fill: x.Fill, border: x.Border, numFmt: x.NumFmt}
		if x.Align != nil {
			rx.h, rx.v, rx.wrap, rx.indent = x.Align.H, x.Align.V, x.Align.Wrap == 1, x.Align.Indent
		}
		b.styles.xfs = append(b.styles.xfs, rx)
	}
	// workbook sheets
	var wbx struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			ID   string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := xml.Unmarshal(b.files["xl/workbook.xml"], &wbx); err != nil {
		t.Fatal(err)
	}
	wbRels := readRels(t, b.files["xl/_rels/workbook.xml.rels"])
	for _, s := range wbx.Sheets {
		target := wbRels[s.ID]
		if target == "" {
			t.Fatalf("sheet %q has no relationship", s.Name)
		}
		path := "xl/" + target
		body, ok := b.files[path]
		if !ok {
			t.Fatalf("sheet part %s missing", path)
		}
		sh := b.readSheet(t, s.Name, path, body)
		b.sheets = append(b.sheets, sh)
	}
	return b
}

func readRels(t *testing.T, body []byte) map[string]string {
	t.Helper()
	out := map[string]string{}
	if body == nil {
		return out
	}
	var rels struct {
		R []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
			Type   string `xml:"Type,attr"`
		} `xml:"Relationship"`
	}
	if err := xml.Unmarshal(body, &rels); err != nil {
		t.Fatal(err)
	}
	for _, r := range rels.R {
		out[r.ID] = r.Target
	}
	return out
}

func (b *rbook) readSheet(t *testing.T, name, path string, body []byte) *rsheet {
	t.Helper()
	sh := &rsheet{name: name, cells: map[string]rcell{}, links: map[string]string{}, cols: map[int]float64{}, hiddenC: map[int]bool{}, rowsH: map[int]float64{}, rawSheet: body}
	var ws struct {
		Dim struct {
			Ref string `xml:"ref,attr"`
		} `xml:"dimension"`
		Pane struct {
			YSplit int    `xml:"ySplit,attr"`
			State  string `xml:"state,attr"`
		} `xml:"sheetViews>sheetView>pane"`
		Cols []struct {
			Min    int     `xml:"min,attr"`
			Max    int     `xml:"max,attr"`
			Width  float64 `xml:"width,attr"`
			Hidden int     `xml:"hidden,attr"`
		} `xml:"cols>col"`
		Rows []struct {
			R  int     `xml:"r,attr"`
			Ht float64 `xml:"ht,attr"`
			C  []struct {
				R string `xml:"r,attr"`
				T string `xml:"t,attr"`
				S int    `xml:"s,attr"`
				V string `xml:"v"`
				F string `xml:"f"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
		Merges []struct {
			Ref string `xml:"ref,attr"`
		} `xml:"mergeCells>mergeCell"`
		Links []struct {
			Ref string `xml:"ref,attr"`
			ID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"hyperlinks>hyperlink"`
		Drawing struct {
			ID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"drawing"`
	}
	if err := xml.Unmarshal(body, &ws); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	sh.dimRef = ws.Dim.Ref
	if ws.Pane.State == "frozen" {
		sh.freeze = ws.Pane.YSplit
	}
	for _, c := range ws.Cols {
		for i := c.Min; i <= c.Max; i++ {
			sh.cols[i] = c.Width
			if c.Hidden == 1 {
				sh.hiddenC[i] = true
			}
		}
	}
	seenRow := 0
	for _, r := range ws.Rows {
		if r.R <= seenRow {
			t.Fatalf("%s: rows out of order at %d", path, r.R)
		}
		seenRow = r.R
		if r.Ht > 0 {
			sh.rowsH[r.R] = r.Ht
		}
		lastCol := 0
		for _, c := range r.C {
			rc := rcell{ref: c.R, t: c.T, v: c.V, f: c.F, s: c.S, hasCell: true}
			rr, cc := parseRef(t, c.R)
			if rr != r.R {
				t.Fatalf("%s: cell %s in row %d", path, c.R, r.R)
			}
			if cc <= lastCol {
				t.Fatalf("%s: cells out of order at %s", path, c.R)
			}
			lastCol = cc
			if c.S >= len(b.styles.xfs) {
				t.Fatalf("%s: cell %s style %d out of range", path, c.R, c.S)
			}
			switch c.T {
			case "s":
				i, err := strconv.Atoi(c.V)
				if err != nil || i >= len(b.strings) {
					t.Fatalf("%s: cell %s bad shared string index %q", path, c.R, c.V)
				}
				rc.str = b.strings[i].text
				rc.runs = b.strings[i].runs
			case "str":
				rc.str = c.V
			}
			sh.cells[c.R] = rc
		}
	}
	for _, m := range ws.Merges {
		sh.merges = append(sh.merges, m.Ref)
	}
	relsPath := strings.Replace(path, "worksheets/", "worksheets/_rels/", 1) + ".rels"
	rels := readRels(t, b.files[relsPath])
	for _, l := range ws.Links {
		target, ok := rels[l.ID]
		if !ok {
			t.Fatalf("%s: hyperlink %s has no relationship", path, l.Ref)
		}
		sh.links[l.Ref] = target
	}
	if ws.Drawing.ID != "" {
		target := rels[ws.Drawing.ID]
		dpath := "xl/" + strings.TrimPrefix(target, "../")
		dbody, ok := b.files[dpath]
		if !ok {
			t.Fatalf("%s: drawing %s missing", path, dpath)
		}
		var dr struct {
			Anchors []struct {
				Col int `xml:"from>col"`
				Row int `xml:"from>row"`
				Ext struct {
					Cx int64 `xml:"cx,attr"`
					Cy int64 `xml:"cy,attr"`
				} `xml:"ext"`
				Pic struct {
					Blip struct {
						Embed string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships embed,attr"`
					} `xml:"blipFill>blip"`
				} `xml:"pic"`
			} `xml:"oneCellAnchor"`
		}
		if err := xml.Unmarshal(dbody, &dr); err != nil {
			t.Fatal(err)
		}
		drels := readRels(t, b.files[strings.Replace(dpath, "drawings/", "drawings/_rels/", 1)+".rels"])
		for _, a := range dr.Anchors {
			target := drels[a.Pic.Blip.Embed]
			mpath := "xl/" + strings.TrimPrefix(target, "../")
			data, ok := b.files[mpath]
			if !ok {
				t.Fatalf("image part %s missing", mpath)
			}
			sh.images = append(sh.images, rimage{col: a.Col, row: a.Row, cx: a.Ext.Cx, cy: a.Ext.Cy, target: target, data: data})
		}
	}
	return sh
}

func parseRef(t *testing.T, ref string) (int, int) {
	t.Helper()
	i := 0
	col := 0
	for i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z' {
		col = col*26 + int(ref[i]-'A'+1)
		i++
	}
	row, err := strconv.Atoi(ref[i:])
	if err != nil || col == 0 {
		t.Fatalf("bad ref %q", ref)
	}
	return row, col
}

func unesc(s string) string {
	var out string
	if err := xml.Unmarshal([]byte("<x>"+s+"</x>"), &struct {
		XMLName xml.Name
		V       string `xml:",chardata"`
	}{}); err != nil {
		return s
	}
	var v struct {
		V string `xml:",chardata"`
	}
	_ = xml.Unmarshal([]byte("<x>"+s+"</x>"), &v)
	out = v.V
	return out
}

func (b *rbook) sheet(t *testing.T, name string) *rsheet {
	t.Helper()
	for _, s := range b.sheets {
		if s.name == name {
			return s
		}
	}
	var names []string
	for _, s := range b.sheets {
		names = append(names, s.name)
	}
	t.Fatalf("no sheet %q in %v", name, names)
	return nil
}

func (b *rbook) xf(c rcell) rxf { return b.styles.xfs[c.s] }

func (b *rbook) font(c rcell) rrun { return b.styles.fonts[b.xf(c).font] }

func (b *rbook) fill(c rcell) string { return b.styles.fills[b.xf(c).fill] }

func (b *rbook) border(c rcell) rborder { return b.styles.borders[b.xf(c).border] }

func (b *rbook) numFmt(c rcell) string {
	id := b.xf(c).numFmt
	switch id {
	case 0:
		return "General"
	case 1:
		return "0"
	case 2:
		return "0.00"
	case 3:
		return "#,##0"
	case 4:
		return "#,##0.00"
	case 9:
		return "0%"
	case 10:
		return "0.00%"
	}
	code, ok := b.styles.numFmts[id]
	if !ok {
		return fmt.Sprintf("unknown(%d)", id)
	}
	return code
}

func (s *rsheet) cell(t *testing.T, ref string) rcell {
	t.Helper()
	c, ok := s.cells[ref]
	if !ok {
		t.Fatalf("sheet %q: no cell %s", s.name, ref)
	}
	return c
}
