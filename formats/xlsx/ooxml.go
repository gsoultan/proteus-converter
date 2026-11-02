// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gsoultan/proteus-converter/core"
)

// sharedStrings deduplicates cell strings into the shared string table.
type sharedStrings struct {
	items []string // serialised <si> bodies
	index map[string]int
	count int
}

func newSharedStrings() *sharedStrings { return &sharedStrings{index: map[string]int{}} }

func (s *sharedStrings) add(si string) int {
	s.count++
	if i, ok := s.index[si]; ok {
		return i
	}
	s.items = append(s.items, si)
	s.index[si] = len(s.items) - 1
	return len(s.items) - 1
}

// write serialises the workbook as an .xlsx (OOXML SpreadsheetML) package.
func write(wb *workbook, meta core.Metadata, w io.Writer) error {
	zw := zip.NewWriter(w)
	now := time.Now().UTC().Format(time.RFC3339)
	add := func(name string, body []byte) error {
		f, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: time.Unix(0, 0)})
		if err != nil {
			return err
		}
		_, err = f.Write(body)
		return err
	}

	var ct bytes.Buffer
	ct.WriteString(xml.Header)
	ct.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	ct.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	ct.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	ct.WriteString(`<Default Extension="png" ContentType="image/png"/>`)
	ct.WriteString(`<Default Extension="jpeg" ContentType="image/jpeg"/>`)
	ct.WriteString(`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`)
	ct.WriteString(`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`)
	ct.WriteString(`<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>`)
	ct.WriteString(`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>`)
	ct.WriteString(`<Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>`)
	for i, sh := range wb.sheets {
		fmt.Fprintf(&ct, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i+1)
		if len(sh.images) > 0 {
			fmt.Fprintf(&ct, `<Override PartName="/xl/drawings/drawing%d.xml" ContentType="application/vnd.openxmlformats-officedocument.drawing+xml"/>`, i+1)
		}
	}
	ct.WriteString(`</Types>`)
	if err := add("[Content_Types].xml", ct.Bytes()); err != nil {
		return err
	}

	rels := xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>` +
		`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>` +
		`</Relationships>`
	if err := add("_rels/.rels", []byte(rels)); err != nil {
		return err
	}

	if err := add("docProps/core.xml", coreProps(meta, now)); err != nil {
		return err
	}
	if err := add("docProps/app.xml", appProps(wb, meta)); err != nil {
		return err
	}

	var wbx, wbr bytes.Buffer
	wbx.WriteString(xml.Header)
	wbx.WriteString(`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	wbx.WriteString(`<workbookPr date1904="0"/>`)
	wbx.WriteString(`<bookViews><workbookView xWindow="0" yWindow="0" windowWidth="20000" windowHeight="12000"/></bookViews><sheets>`)
	wbr.WriteString(xml.Header)
	wbr.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i, sh := range wb.sheets {
		fmt.Fprintf(&wbx, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, esc(sh.name), i+1, i+1)
		fmt.Fprintf(&wbr, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i+1, i+1)
	}
	n := len(wb.sheets)
	fmt.Fprintf(&wbr, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, n+1)
	fmt.Fprintf(&wbr, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>`, n+2)
	wbr.WriteString(`</Relationships>`)
	wbx.WriteString(`</sheets>`)
	// Print titles for frozen header rows.
	var defined []string
	for i, sh := range wb.sheets {
		if sh.freezeRows > 0 {
			defined = append(defined, fmt.Sprintf(`<definedName name="_xlnm.Print_Titles" localSheetId="%d">'%s'!$1:$%d</definedName>`,
				i, strings.ReplaceAll(esc(sh.name), "'", "''"), sh.freezeRows))
		}
	}
	if len(defined) > 0 {
		wbx.WriteString(`<definedNames>` + strings.Join(defined, "") + `</definedNames>`)
	}
	wbx.WriteString(`<calcPr calcId="191029" fullCalcOnLoad="1"/></workbook>`)
	if err := add("xl/workbook.xml", wbx.Bytes()); err != nil {
		return err
	}
	if err := add("xl/_rels/workbook.xml.rels", wbr.Bytes()); err != nil {
		return err
	}

	for i, sh := range wb.sheets {
		body, sheetRels, drawing, drawingRels, media := sheetXML(wb, sh, i+1)
		if err := add(fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), body); err != nil {
			return err
		}
		if sheetRels != nil {
			if err := add(fmt.Sprintf("xl/worksheets/_rels/sheet%d.xml.rels", i+1), sheetRels); err != nil {
				return err
			}
		}
		if drawing != nil {
			if err := add(fmt.Sprintf("xl/drawings/drawing%d.xml", i+1), drawing); err != nil {
				return err
			}
			if err := add(fmt.Sprintf("xl/drawings/_rels/drawing%d.xml.rels", i+1), drawingRels); err != nil {
				return err
			}
			for name, data := range media {
				if err := add("xl/media/"+name, data); err != nil {
					return err
				}
			}
		}
	}

	if err := add("xl/styles.xml", stylesXML(wb.styles)); err != nil {
		return err
	}
	var sst bytes.Buffer
	sst.WriteString(xml.Header)
	fmt.Fprintf(&sst, `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`, wb.sst.count, len(wb.sst.items))
	for _, si := range wb.sst.items {
		sst.WriteString("<si>")
		sst.WriteString(si)
		sst.WriteString("</si>")
	}
	sst.WriteString("</sst>")
	if err := add("xl/sharedStrings.xml", sst.Bytes()); err != nil {
		return err
	}
	return zw.Close()
}

// esc escapes text for XML attribute and element content and drops the
// control characters XML 1.0 forbids (Excel refuses files that contain them).
func esc(s string) string {
	s = cleanXML(s)
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	// EscapeText writes newlines as &#xA; which is fine in attributes and
	// elements alike; tabs and CRs are stripped by cleanXML.
	return b.String()
}

func cleanXML(s string) string {
	if !strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 && r != '\n' || r == 0xFFFE || r == 0xFFFF || r == utf8.RuneError }) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\n' || r == 0xFFFE || r == 0xFFFF || r == utf8.RuneError {
			return -1
		}
		return r
	}, s)
}

func coreProps(m core.Metadata, now string) []byte {
	var b bytes.Buffer
	b.WriteString(xml.Header)
	b.WriteString(`<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:dcmitype="http://purl.org/dc/dcmitype/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">`)
	if m.Title != "" {
		b.WriteString("<dc:title>" + esc(m.Title) + "</dc:title>")
	}
	if m.Subject != "" {
		b.WriteString("<dc:subject>" + esc(m.Subject) + "</dc:subject>")
	}
	if m.Author != "" {
		b.WriteString("<dc:creator>" + esc(m.Author) + "</dc:creator>")
	}
	if m.Keywords != "" {
		b.WriteString("<cp:keywords>" + esc(m.Keywords) + "</cp:keywords>")
	}
	if m.Language != "" {
		b.WriteString("<dc:language>" + esc(m.Language) + "</dc:language>")
	}
	b.WriteString(`<dcterms:created xsi:type="dcterms:W3CDTF">` + now + `</dcterms:created>`)
	b.WriteString(`<dcterms:modified xsi:type="dcterms:W3CDTF">` + now + `</dcterms:modified>`)
	b.WriteString(`</cp:coreProperties>`)
	return b.Bytes()
}

func appProps(wb *workbook, m core.Metadata) []byte {
	var b bytes.Buffer
	b.WriteString(xml.Header)
	b.WriteString(`<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">`)
	app := m.Creator
	if app == "" {
		app = "proteus-converter"
	}
	b.WriteString("<Application>" + esc(app) + "</Application>")
	fmt.Fprintf(&b, `<HeadingPairs><vt:vector size="2" baseType="variant"><vt:variant><vt:lpstr>Worksheets</vt:lpstr></vt:variant><vt:variant><vt:i4>%d</vt:i4></vt:variant></vt:vector></HeadingPairs>`, len(wb.sheets))
	fmt.Fprintf(&b, `<TitlesOfParts><vt:vector size="%d" baseType="lpstr">`, len(wb.sheets))
	for _, sh := range wb.sheets {
		b.WriteString("<vt:lpstr>" + esc(sh.name) + "</vt:lpstr>")
	}
	b.WriteString(`</vt:vector></TitlesOfParts></Properties>`)
	return b.Bytes()
}

func stylesXML(t *styleTable) []byte {
	var b bytes.Buffer
	b.WriteString(xml.Header)
	b.WriteString(`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	if len(t.numFmts) > 0 {
		fmt.Fprintf(&b, `<numFmts count="%d">`, len(t.numFmts))
		for i, code := range t.numFmts {
			fmt.Fprintf(&b, `<numFmt numFmtId="%d" formatCode="%s"/>`, 164+i, esc(code))
		}
		b.WriteString(`</numFmts>`)
	}
	fmt.Fprintf(&b, `<fonts count="%d">`, len(t.fonts))
	for _, f := range t.fonts {
		b.WriteString(fontXML(f))
	}
	b.WriteString(`</fonts>`)
	fmt.Fprintf(&b, `<fills count="%d">`, len(t.fills))
	for i, f := range t.fills {
		switch {
		case i == 0:
			b.WriteString(`<fill><patternFill patternType="none"/></fill>`)
		case i == 1:
			b.WriteString(`<fill><patternFill patternType="gray125"/></fill>`)
		default:
			fmt.Fprintf(&b, `<fill><patternFill patternType="solid"><fgColor rgb="%s"/><bgColor indexed="64"/></patternFill></fill>`, f.color)
		}
	}
	b.WriteString(`</fills>`)
	fmt.Fprintf(&b, `<borders count="%d">`, len(t.borders))
	for _, bd := range t.borders {
		b.WriteString("<border>")
		for _, s := range []struct {
			tag  string
			side borderSide
		}{{"left", bd.left}, {"right", bd.right}, {"top", bd.top}, {"bottom", bd.bottom}} {
			if s.side.style == "" {
				b.WriteString("<" + s.tag + "/>")
				continue
			}
			fmt.Fprintf(&b, `<%s style="%s">`, s.tag, s.side.style)
			if s.side.color != "" {
				fmt.Fprintf(&b, `<color rgb="%s"/>`, s.side.color)
			} else {
				b.WriteString(`<color auto="1"/>`)
			}
			b.WriteString("</" + s.tag + ">")
		}
		b.WriteString("<diagonal/></border>")
	}
	b.WriteString(`</borders>`)
	b.WriteString(`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>`)
	fmt.Fprintf(&b, `<cellXfs count="%d">`, len(t.xfs))
	for _, x := range t.xfs {
		fmt.Fprintf(&b, `<xf numFmtId="%d" fontId="%d" fillId="%d" borderId="%d" xfId="0"`, x.numFmtID, x.fontID, x.fillID, x.borderID)
		if x.numFmtID != 0 {
			b.WriteString(` applyNumberFormat="1"`)
		}
		if x.fontID != 0 {
			b.WriteString(` applyFont="1"`)
		}
		if x.fillID != 0 {
			b.WriteString(` applyFill="1"`)
		}
		if x.borderID != 0 {
			b.WriteString(` applyBorder="1"`)
		}
		a := x.align
		if a.horizontal != "" || a.vertical != "" || a.wrap || a.indent > 0 {
			b.WriteString(` applyAlignment="1"><alignment`)
			if a.horizontal != "" {
				b.WriteString(` horizontal="` + a.horizontal + `"`)
			}
			if a.vertical != "" {
				b.WriteString(` vertical="` + a.vertical + `"`)
			}
			if a.wrap {
				b.WriteString(` wrapText="1"`)
			}
			if a.indent > 0 {
				fmt.Fprintf(&b, ` indent="%d"`, a.indent)
			}
			b.WriteString(`/></xf>`)
		} else {
			b.WriteString(`/>`)
		}
	}
	b.WriteString(`</cellXfs>`)
	b.WriteString(`<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>`)
	b.WriteString(`<dxfs count="0"/>`)
	b.WriteString(`</styleSheet>`)
	return b.Bytes()
}

// fontXML serialises a font for styles.xml (<font>) or, with run=true, for a
// rich text run (<rPr>), whose child order and name element differ.
func fontXML(f font) string { return fontXMLTag(f, "font", "name") }

func runPropsXML(f font) string { return fontXMLTag(f, "rPr", "rFont") }

func fontXMLTag(f font, tag, nameTag string) string {
	var b strings.Builder
	b.WriteString("<" + tag + ">")
	if f.bold {
		b.WriteString("<b/>")
	}
	if f.italic {
		b.WriteString("<i/>")
	}
	if f.strike {
		b.WriteString("<strike/>")
	}
	if f.underline {
		b.WriteString("<u/>")
	}
	if f.sizePt > 0 {
		b.WriteString(`<sz val="` + strconv.FormatFloat(f.sizePt, 'f', -1, 64) + `"/>`)
	}
	if f.color != "" {
		b.WriteString(`<color rgb="` + f.color + `"/>`)
	}
	if f.name != "" {
		b.WriteString("<" + nameTag + ` val="` + esc(f.name) + `"/>`)
	}
	b.WriteString("</" + tag + ">")
	return b.String()
}

// sheetXML serialises one worksheet and its related parts.
func sheetXML(wb *workbook, sh *sheet, n int) (body, rels, drawing, drawingRels []byte, media map[string][]byte) {
	var b bytes.Buffer
	b.WriteString(xml.Header)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	maxR, maxC := 0, 0
	for i, r := range sh.rows {
		if r == nil {
			continue
		}
		for j, c := range r.cells {
			if c != nil && c.kind != kindEmpty || c != nil && c.xf != 0 {
				maxR, maxC = max(maxR, i+1), max(maxC, j+1)
			}
		}
	}
	if maxR == 0 || maxC == 0 {
		b.WriteString(`<dimension ref="A1"/>`)
	} else {
		fmt.Fprintf(&b, `<dimension ref="A1:%s"/>`, cellRef(maxR, maxC))
	}
	b.WriteString(`<sheetViews><sheetView workbookViewId="0"`)
	if n == 1 {
		b.WriteString(` tabSelected="1"`)
	}
	b.WriteString(`>`)
	if sh.freezeRows > 0 {
		fmt.Fprintf(&b, `<pane ySplit="%d" topLeftCell="A%d" activePane="bottomLeft" state="frozen"/><selection pane="bottomLeft"/>`, sh.freezeRows, sh.freezeRows+1)
	}
	b.WriteString(`</sheetView></sheetViews>`)
	b.WriteString(`<sheetFormatPr defaultRowHeight="15"/>`)
	if len(sh.cols) > 0 {
		var cols bytes.Buffer
		for i, c := range sh.cols {
			if c.width <= 0 && !c.hidden {
				continue
			}
			fmt.Fprintf(&cols, `<col min="%d" max="%d"`, i+1, i+1)
			if c.width > 0 {
				fmt.Fprintf(&cols, ` width="%s" customWidth="1"`, strconv.FormatFloat(c.width, 'f', -1, 64))
			}
			if c.hidden {
				cols.WriteString(` hidden="1"`)
			}
			cols.WriteString(`/>`)
		}
		if cols.Len() > 0 {
			b.WriteString(`<cols>`)
			b.Write(cols.Bytes())
			b.WriteString(`</cols>`)
		}
	}
	b.WriteString(`<sheetData>`)
	for i, r := range sh.rows {
		if r == nil {
			continue
		}
		fmt.Fprintf(&b, `<row r="%d"`, i+1)
		if r.heightPt > 0 {
			fmt.Fprintf(&b, ` ht="%s" customHeight="1"`, strconv.FormatFloat(r.heightPt, 'f', -1, 64))
		}
		if r.hidden {
			b.WriteString(` hidden="1"`)
		}
		b.WriteString(`>`)
		for j, c := range r.cells {
			if c == nil || c.kind == kindEmpty && c.xf == 0 {
				continue
			}
			ref := cellRef(i+1, j+1)
			switch c.kind {
			case kindEmpty:
				fmt.Fprintf(&b, `<c r="%s" s="%d"/>`, ref, c.xf)
			case kindString:
				idx := wb.sst.add(siXML(c))
				fmt.Fprintf(&b, `<c r="%s" s="%d" t="s"><v>%d</v></c>`, ref, c.xf, idx)
			case kindNumber:
				fmt.Fprintf(&b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, c.xf, strconv.FormatFloat(c.num, 'G', -1, 64))
			case kindBool:
				v := 0
				if c.boolean {
					v = 1
				}
				fmt.Fprintf(&b, `<c r="%s" s="%d" t="b"><v>%d</v></c>`, ref, c.xf, v)
			case kindFormula:
				fmt.Fprintf(&b, `<c r="%s" s="%d" t="str"><f>%s</f>`, ref, c.xf, esc(strings.TrimPrefix(c.formula, "=")))
				if c.text != "" {
					b.WriteString(`<v>` + esc(c.text) + `</v>`)
				}
				b.WriteString(`</c>`)
			}
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData>`)
	if len(sh.merges) > 0 {
		fmt.Fprintf(&b, `<mergeCells count="%d">`, len(sh.merges))
		for _, m := range sh.merges {
			fmt.Fprintf(&b, `<mergeCell ref="%s"/>`, m)
		}
		b.WriteString(`</mergeCells>`)
	}
	var rb bytes.Buffer
	relID := 0
	if len(sh.links) > 0 {
		b.WriteString(`<hyperlinks>`)
		for _, l := range sh.links {
			relID++
			fmt.Fprintf(&b, `<hyperlink ref="%s" r:id="rId%d"`, cellRef(l.r, l.c), relID)
			if l.tooltip != "" {
				b.WriteString(` tooltip="` + esc(l.tooltip) + `"`)
			}
			b.WriteString(`/>`)
			fmt.Fprintf(&rb, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="%s" TargetMode="External"/>`, relID, esc(l.target))
		}
		b.WriteString(`</hyperlinks>`)
	}
	b.WriteString(`<pageMargins left="0.7" right="0.7" top="0.75" bottom="0.75" header="0.3" footer="0.3"/>`)
	if len(sh.images) > 0 {
		relID++
		fmt.Fprintf(&b, `<drawing r:id="rId%d"/>`, relID)
		fmt.Fprintf(&rb, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/drawing%d.xml"/>`, relID, n)
		drawing, drawingRels, media = drawingXML(sh, n)
	}
	b.WriteString(`</worksheet>`)
	if rb.Len() > 0 {
		rels = []byte(xml.Header + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rb.String() + `</Relationships>`)
	}
	return b.Bytes(), rels, drawing, drawingRels, media
}

// siXML serialises a string cell as a shared string item body: plain text,
// or rich text runs when the cell mixes fonts.
func siXML(c *cell) string {
	if len(c.runs) == 0 {
		return `<t xml:space="preserve">` + esc(c.text) + `</t>`
	}
	var b strings.Builder
	for _, r := range c.runs {
		b.WriteString("<r>")
		b.WriteString(runPropsXML(r.font))
		b.WriteString(`<t xml:space="preserve">` + esc(r.text) + `</t>`)
		b.WriteString("</r>")
	}
	return b.String()
}

const emuPerPx = 9525

func drawingXML(sh *sheet, n int) (drawing, rels []byte, media map[string][]byte) {
	var b, rb bytes.Buffer
	media = map[string][]byte{}
	b.WriteString(xml.Header)
	b.WriteString(`<xdr:wsDr xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">`)
	rb.WriteString(xml.Header)
	rb.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i, img := range sh.images {
		name := fmt.Sprintf("image%d_%d.%s", n, i+1, img.ext)
		media[name] = img.data
		fmt.Fprintf(&rb, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/%s"/>`, i+1, name)
		cx, cy := int64(img.wPx*emuPerPx), int64(img.hPx*emuPerPx)
		fmt.Fprintf(&b, `<xdr:oneCellAnchor><xdr:from><xdr:col>%d</xdr:col><xdr:colOff>%d</xdr:colOff><xdr:row>%d</xdr:row><xdr:rowOff>%d</xdr:rowOff></xdr:from>`,
			img.c-1, img.offXEMU, img.r-1, img.offYEMU)
		fmt.Fprintf(&b, `<xdr:ext cx="%d" cy="%d"/>`, cx, cy)
		fmt.Fprintf(&b, `<xdr:pic><xdr:nvPicPr><xdr:cNvPr id="%d" name="%s"/><xdr:cNvPicPr><a:picLocks noChangeAspect="1"/></xdr:cNvPicPr></xdr:nvPicPr>`, i+2, esc(img.name))
		fmt.Fprintf(&b, `<xdr:blipFill><a:blip xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:embed="rId%d"/><a:stretch><a:fillRect/></a:stretch></xdr:blipFill>`, i+1)
		fmt.Fprintf(&b, `<xdr:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></xdr:spPr></xdr:pic>`, cx, cy)
		b.WriteString(`<xdr:clientData/></xdr:oneCellAnchor>`)
	}
	b.WriteString(`</xdr:wsDr>`)
	rb.WriteString(`</Relationships>`)
	return b.Bytes(), rb.Bytes(), media
}
