// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gsoultan/gexcel"
	"github.com/gsoultan/gexcel/model"
	"github.com/gsoultan/proteus-converter/core"
)

// emuPerPx is the drawing unit per CSS pixel at 96 DPI.
const emuPerPx = model.EMUPerPixel

// write builds a gexcel workbook from the extracted sheets and saves it as
// an .xlsx package.
func write(wb *workbook, meta core.Metadata, w io.Writer) error {
	out := gexcel.NewEmpty()
	for i, sh := range wb.sheets {
		gs, err := out.AddSheet(sh.name)
		if err != nil {
			return fmt.Errorf("proteus: xlsx: sheet %q: %w", sh.name, err)
		}
		if i == 0 {
			_ = out.SetActiveSheet(sh.name)
		}
		if err := fillSheet(gs, sh); err != nil {
			return fmt.Errorf("proteus: xlsx: sheet %q: %w", sh.name, err)
		}
	}
	props := out.Properties()
	props.Title, props.Subject, props.Creator = cleanXML(meta.Title), cleanXML(meta.Subject), cleanXML(meta.Author)
	props.Keywords, props.Language = cleanXML(meta.Keywords), cleanXML(meta.Language)
	props.Application = cleanXML(meta.Creator)
	if props.Application == "" {
		props.Application = "proteus-converter"
	}
	now := time.Now().UTC().Truncate(time.Second)
	props.Created, props.Modified = now, now
	out.SetFullCalcOnLoad(true)
	return out.Save(w)
}

func fillSheet(gs gexcel.Sheet, sh *sheet) error {
	for i, r := range sh.rows {
		if r == nil {
			continue
		}
		var gr gexcel.Row
		if r.heightPt > 0 || r.hidden {
			gr = gs.Row(i + 1)
			if r.heightPt > 0 {
				gr.SetHeight(r.heightPt)
			}
			if r.hidden {
				gr.SetHidden(true)
			}
		}
		for j, c := range r.cells {
			if c == nil || c.kind == kindEmpty && !c.styled {
				continue
			}
			if gr == nil {
				gr = gs.Row(i + 1)
			}
			gc := gr.Cell(j + 1)
			switch c.kind {
			case kindString:
				if len(c.runs) > 0 {
					runs := make([]model.RichRun, 0, len(c.runs))
					for _, run := range c.runs {
						f := run.font
						runs = append(runs, model.RichRun{Text: cleanXML(run.text), Font: &f})
					}
					gc.SetRich(runs...)
				} else {
					gc.Set(cleanXML(c.text))
				}
			case kindNumber:
				gc.Set(c.num)
			case kindBool:
				gc.Set(c.boolean)
			case kindFormula:
				f := strings.TrimPrefix(c.formula, "=")
				if c.text != "" {
					gc.SetFormulaResult(f, cleanXML(c.text))
				} else {
					gc.SetFormula(f)
				}
			}
			if c.styled {
				gc.SetStyle(c.st)
			}
		}
	}
	for i, col := range sh.cols {
		if col.width > 0 {
			gs.SetColWidth(i+1, i+1, col.width)
		}
		if col.hidden {
			gs.SetColHidden(i+1, i+1, true)
		}
	}
	for _, m := range sh.merges {
		if err := gs.Merge(m.String()); err != nil {
			return err
		}
	}
	if sh.freezeRows > 0 {
		gs.Freeze(sh.freezeRows, 0)
		gs.SetPrintTitleRows(1, sh.freezeRows)
	}
	for _, l := range sh.links {
		if err := gs.AddHyperlink(cellRef(l.r, l.c), l.target, cleanXML(l.tooltip)); err != nil {
			return err
		}
	}
	for _, img := range sh.images {
		err := gs.AddImage(cellRef(img.r, img.c), model.Image{
			Name:   cleanXML(img.name),
			Format: img.ext,
			Data:   img.data,
			Anchor: model.Anchor{
				Kind:   model.AnchorOneCell,
				From:   model.CellOffset{Row: img.r, Col: img.c, RowOff: img.offYEMU, ColOff: img.offXEMU},
				Width:  int64(img.wPx * emuPerPx),
				Height: int64(img.hPx * emuPerPx),
			},
			LockAspect: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// cleanXML drops the control characters XML 1.0 forbids (Excel refuses files
// that contain them) and invalid UTF-8.
func cleanXML(s string) string {
	bad := func(r rune) bool { return r < 0x20 && r != '\n' || r == 0xFFFE || r == 0xFFFF || r == utf8.RuneError }
	if !strings.ContainsFunc(s, bad) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if bad(r) {
			return -1
		}
		return r
	}, s)
}
