# proteus-converter

Pure-Go conversion of **HTML** and **Markdown** to **PDF**, built on
[gpdf](https://github.com/gsoultan/gpdf). No browser, LibreOffice or cgo at
runtime. Microsoft Word (DOCX) input is planned; the API is shaped so new
formats plug in without changes for callers.

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
```

CLI: `go run ./cmd/proteus [-page A4|Letter[-landscape]] [-margin mm] [-css extra.css] [-http] in.html|in.md [out.pdf]`

## Design

| Pattern | Where | Why |
| --- | --- | --- |
| Strategy | `core.Converter` (`Format`, `Extensions`, `Convert`) | one implementation per input format; DOCX becomes another strategy |
| Registry / factory | `core.Registry`, `proteus.Register`, `proteus.Lookup`, `FormatForPath` | pick a converter by format or file extension; callers can register their own |
| Functional options | `core.Option` (`WithPageSize`, `WithStylesheet`, …) | defaults that work, settings that do not break when new ones are added |
| Adapter | `render.Painter`, `render.GPDF` | layout never touches gpdf directly, so the PDF backend can change |
| Pipeline | parse → cascade → box tree → layout → paginate → paint | each stage is a package with its own tests |

Packages: `core` (API types), `css` (parser, cascade, values), `fonts`
(TrueType discovery and metrics), `layout` (box tree, block/inline/table
layout, pagination), `render` (painter), `pipeline` (glue), `formats/html`,
`formats/markdown`.

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
lists, tables (auto layout, colspan/rowspan, collapsed and separated borders,
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

## Accuracy gate

`accuracy/run.sh` renders every file in `testdata/` with proteus and with
headless Chromium (fonts pinned to Liberation via `accuracy/fonts.conf`), then
compares the rasterised pages with `internal/tools/pixdiff`. CI fails when a
page differs by more than 3% of its pixels; the current fixtures sit at about
1.2–1.4%. Chromium is used only as the reference in CI.
