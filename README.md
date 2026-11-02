# proteus-converter

Pure-Go conversion of **HTML** and **Markdown** to **PDF** (built on
[gpdf](https://github.com/gsoultan/gpdf)) and to **Excel (XLSX)**. No browser,
LibreOffice or cgo at runtime. Microsoft Word (DOCX) input is planned; the API
is shaped so new formats plug in without changes for callers.

```go
import proteus "github.com/gsoultan/proteus-converter"

// File to file; the format is picked from the extension and relative
// stylesheets/images resolve against the input's directory.
err := proteus.ConvertFile(ctx, "invoice.html", "invoice.pdf")

// Bytes in, bytes out, with options.
pdf, err := proteus.ConvertBytes(ctx, proteus.FormatMarkdown, src,
	proteus.WithPageSize(proteus.Letter),
	proteus.WithMargins(proteus.UniformMargins(proteus.Mm(15))),
	proteus.WithTitle("Release notes"),
)

// Streams, for servers.
err = proteus.Convert(ctx, proteus.FormatHTML, r.Body, w, proteus.WithBaseURL("https://example.com/"))

// Other output formats: the output extension picks it for files, or name it.
err = proteus.ConvertFile(ctx, "report.md", "report.xlsx")
xlsx, err := proteus.ConvertBytesTo(ctx, proteus.FormatHTML, proteus.OutputXLSX, src)
```

CLI: `go run ./cmd/proteus [-page A4|Letter[-landscape]] [-margin mm] [-css extra.css] [-http] in.html|in.md [out.pdf|out.xlsx]`

## Design

| Pattern | Where | Why |
| --- | --- | --- |
| Strategy | `core.Converter` (`Format`, `Extensions`, `Convert`) | one implementation per input format; DOCX becomes another strategy |
| Strategy (outputs) | `core.Parser` (input → `core.Document`) + `core.Writer` (`Output`, `Write`) | an input that parses to an HTML tree feeds every writer; XLSX is the first non-PDF writer, more plug in with `RegisterWriter` |
| Registry / factory | `core.Registry`, `proteus.Register`, `proteus.Lookup`, `FormatForPath`, `RegisterWriter`, `OutputForPath` | pick a converter by format or file extension; callers can register their own |
| Functional options | `core.Option` (`WithPageSize`, `WithStylesheet`, …) | defaults that work, settings that do not break when new ones are added |
| Adapter | `render.Painter`, `render.GPDF` | layout never touches gpdf directly, so the PDF backend can change |
| Pipeline | parse → cascade → box tree → layout → paginate → paint | each stage is a package with its own tests |

Packages: `core` (API types), `css` (parser, cascade, values), `fonts`
(TrueType discovery and metrics), `layout` (box tree, block/inline/table
layout, pagination), `render` (painter), `pipeline` (glue), `formats/html`,
`formats/markdown`, `formats/xlsx` (spreadsheet writer).

## Excel output

`formats/xlsx` writes a workbook straight from the styled document (cascade
only, no page layout), with no third-party dependency:

- Every top-level `<table>` becomes a worksheet named after its `<caption>`,
  `data-sheet`, `aria-label`, `title` or `id`, else the heading before it,
  else `Table N` (names are made unique and valid for Excel). A document
  without tables becomes one `Document` sheet with a row per block of text.
- Rows and cells map one to one; `rowspan`/`colspan` (including `rowspan=0`)
  follow the HTML table model and become merged ranges. `<thead>` rows are
  frozen and repeat on printed pages. `display:none` rows and cells are
  skipped.
- Text keeps CSS `white-space` rules, `<br>`, nested blocks (one line each),
  list markers, `text-transform` and `::before`/`::after` content. Mixed
  inline styles become rich-text runs; a uniformly styled cell gets that
  font directly.
- Computed CSS gives the font (generic families map to Arial, Times New
  Roman, Courier New, …; browser-only names like `-apple-system` are
  skipped), size, bold, italic, underline, strike and colour; the fill is the
  table → column → row group → row → cell background stack composited over
  white; borders map to thin/medium/thick/dashed/dotted/double with the
  collapsing model resolved against row, row group and table borders;
  `text-align` and `vertical-align` set the alignment, `padding-left` an
  indent, and `<col>`/cell widths the column widths (else auto-fit).
- Numeric-looking text is stored as a typed cell with a number format that
  shows exactly what the document showed: `1,234.50`, `12.5%`, `$1,234.50`,
  `5 €`, `Rp 1.500`, `(2,000)` (accounting negative), ISO dates and
  date-times. Identifiers like `007` stay text, as do header cells. Authors
  can force a type with `data-type="string|number|bool|date|formula"`,
  `data-value` and `data-formula="=SUM(B2:B9)"`; `WithNumberDetection(false)`
  (CLI `-no-numbers`) turns the detection off.
- `<a href>` becomes a cell hyperlink; PNG, JPEG and GIF `<img>` are embedded
  at their cell (sized by CSS width/height), SVG is reported as a warning.
- Document metadata (`<title>`, author, keywords, language, or `WithMetadata`)
  fills the workbook properties.

Text outside tables in a document that has tables is not written (a warning
counts the skipped blocks). Tests read the workbook back cell by cell and,
when LibreOffice is installed, round-trip it through `soffice` to compare the
displayed values, including an evaluated formula.

## CSS support

All author CSS is honoured through a real cascade: `<style>` blocks in the
head, `style=""` attributes on any tag, `<link rel="stylesheet">`, `@import`,
specificity, `!important`, inheritance, `var()`, `calc()/min()/max()/clamp()`,
`@media print`, `@page` size and margins, `@font-face` (TrueType), and
presentational attributes (`width`, `bgcolor`, `align`, …). The default
styles follow Chromium's user-agent sheet, including its quirks for line
metrics, monospace sizing, list markers and collapsed table borders.

Layout covers block flow with margin collapsing, inline text with wrapping,
justification, `white-space`, `letter-spacing`, `text-transform`,
decorations and super/subscripts, inline-block, images (PNG, JPEG, GIF),
lists, tables (auto and fixed layout, percentage column widths, colspan/rowspan, collapsed and separated borders,
repeated header rows), page breaks (`break-before/after/inside`,
orphans/widows), links, named destinations and a PDF outline from headings.
It also covers floats and `clear`, `position: relative/absolute/fixed` with
`z-index` (fixed boxes repeat on every page), flexbox, CSS grid (`fr`,
`minmax()`, `repeat()`, named areas, line placement), inline and file SVG,
`object-fit`, background images (`background-size/position/repeat`) and
`linear-gradient()`.

Not yet: radial gradients, SVG gradients and filters (gradients are drawn in
their average color), `grid-auto-flow: column`, WOFF fonts, `@page :first` and margin boxes, CSS counters beyond list
numbering. Unsupported input produces a warning (see `WithWarningHandler`),
never a failure.

## Security

Remote fetching is off by default; enable it with `WithHTTP(true)` or supply a
`ResourceLoader`. Local file access can be confined with `WithFS` or disabled
with `WithLocalFiles(false)`, and every resource is capped by
`WithMaxResourceSize` (50 MB by default).

## Robustness and resource use

- A conversion never panics into the caller: an internal bug comes back as an
  `*InternalError` (`errors.Is(err, ErrInternal)`) with the stack attached.
- Runaway documents are bounded: `WithMaxPages` (10,000 by default,
  `ErrTooManyPages`), images above 40 megapixels are skipped with a warning
  (decompression bombs), grid line numbers and spans are clamped to 1,000, and
  HTML nesting is limited to 512 levels by the parser.
- `go test -fuzz` targets cover the whole pipeline (`FuzzConvertHTML`,
  `FuzzConvertMarkdown`, `FuzzConvertCSS`) and the CSS parser (`css.FuzzParse`).
- `TestNoLeaks` converts every fixture 35 times and checks goroutines and
  retained heap do not grow; `TestConcurrentConversions` runs them in parallel
  under `-race`. `BenchmarkConvert` reports time, allocations and PDF size per
  fixture.
- Scanning the system fonts reads only each font's header tables; outlines are
  loaded on first use and shared between conversions.

## Accuracy gate

`accuracy/run.sh` renders every file in `testdata/` with proteus and with
headless Chromium (fonts pinned to Liberation via `accuracy/fonts.conf`), then
compares them two ways:

- `internal/tools/pixdiff` rasterises both at 96 dpi. CI fails when a page
  differs by more than 3% of its pixels, or by more than 0.5% once ink moved
  by at most 1px is forgiven (that second figure removes anti-aliasing noise
  and leaves missing, extra or misplaced content).
- `internal/tools/textdiff` extracts every word with its position
  (`pdftotext -bbox`), pairs the words of both PDFs and fails when fewer than
  99% of Chromium's words are found or fewer than 98% sit within 2pt of
  Chromium's position.

The corpus covers typography, lists, tables, boxes and borders, floats,
flexbox, grid, images, an invoice and two Markdown documents. Current results:
every word of every fixture is found and placed within 1.5pt of Chromium
(median 0.4pt); raw pixel mismatch is 0.2–3.0% per page and 0–0.31% beyond
1px. Chromium is used only as the reference in CI.
