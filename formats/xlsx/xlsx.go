// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package xlsx writes parsed HTML documents (from the HTML or Markdown
// front ends) as Excel workbooks.
//
// Every top-level <table> becomes one worksheet, named after its <caption>,
// data-sheet, aria-label, title or id attribute, else the nearest preceding
// heading, else "Table N". Rows and cells map one to one; rowspan and colspan
// become merged ranges; <thead> rows are frozen and repeat on print. Cell
// text keeps CSS white-space rules and text-transform; mixed inline styles
// become rich-text runs. Computed CSS supplies the cell font, colour,
// background (layered table, column, row group, row and cell colours),
// borders, horizontal and vertical alignment, and column widths from
// <col>/cell widths (else auto-fit). Numbers, percentages, currency amounts,
// ISO dates and accounting negatives are stored as typed values with a
// number format that shows the original text; data-type="string|number|
// bool|date|formula", data-value and data-formula attributes override the
// detection. Links become cell hyperlinks and PNG/JPEG/GIF images are
// embedded at their cell.
//
// A document without tables becomes one sheet with a row per block of text.
// Text outside tables in a document that has tables is not written (a
// warning counts the skipped blocks).
package xlsx

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/pipeline"
)

// MIMEType is the media type of .xlsx files.
const MIMEType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// Writer writes documents as XLSX workbooks. The zero value is ready to use
// and safe for concurrent use.
type Writer struct {
	_ sync.Mutex // keeps the type non-comparable so it is passed by pointer
}

// New returns an XLSX writer.
func New() *Writer { return &Writer{} }

// Output implements core.Writer.
func (*Writer) Output() core.OutputFormat { return core.OutputXLSX }

// Extension implements core.Writer.
func (*Writer) Extension() string { return "xlsx" }

// MIMEType implements core.Writer.
func (*Writer) MIMEType() string { return MIMEType }

// Write implements core.Writer.
func (*Writer) Write(ctx context.Context, doc *core.Document, cfg *core.Config, dst io.Writer) (err error) {
	defer core.Recover(&err)
	if doc == nil || doc.Root == nil {
		return fmt.Errorf("proteus: xlsx: nil document")
	}
	if cfg == nil {
		cfg = core.NewConfig()
	}
	styled, err := pipeline.Style(ctx, pipeline.Input{Doc: doc.Root, BaseURL: doc.BaseURL, ThemeCSS: doc.ThemeCSS}, cfg)
	if err != nil {
		return err
	}
	wb := &workbook{styles: newStyleTable(font{name: "Calibri", sizePt: 11}), sst: newSharedStrings()}
	e := &extractor{ctx: ctx, cfg: cfg, styled: styled, wb: wb, warn: cfg.Warnf, names: map[string]bool{},
		images: &imageStore{cache: map[string]*loadedImage{}}, widths: map[*sheet]cellWidths{}}
	if err := e.run(doc.Root); err != nil {
		return err
	}
	for _, sh := range wb.sheets {
		fitImages(sh, wb.styles.defFontW)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return write(wb, styled.Metadata, dst)
}
