# proteus-converter overview

Pure-Go HTML and Markdown to PDF converter (module github.com/gsoultan/proteus-converter,
repo formerly gpdf-converter), plus Excel (XLSX) output. PDF backend: github.com/gsoultan/gpdf.
DOCX input is planned. proteus-server exposes conversion over gRPC; it is a separate repo.

Pipeline: parse (x/net/html, goldmark for Markdown) -> collect stylesheets (pipeline/styles.go)
-> cascade (css/cascade.go, layout props in css/layoutprops.go) -> box tree (layout/build.go,
fixup.go) -> layout (layout/block.go, inline*.go, float.go, flex.go, grid.go, table.go,
atomic.go, images.go, background.go) -> paginate (layout/paginate.go) -> paint through
render.Painter (render/gpdf.go, render/gpdf_ops.go).

Packages: svg (public) parses SVG into shapes/paths (svg.Parse, svg.FromNode, Matrix, Seg);
layout turns them into PathItem/TextItem. Painter has Image(x,y,w,h,img,clip) and
Path(matrix, segs, style, clip). Floats live in a floatCtx per block formatting context;
absolute boxes collect in a posCtx on their containing block (layoutBlockW).

Public API lives in proteus.go (aliases of core): Convert, ConvertBytes, ConvertString,
ConvertFile (output format from the output extension), ConvertTo/ConvertBytesTo (named
OutputFormat), Register, Lookup, FormatForPath, RegisterWriter, LookupWriter, Outputs,
OutputForPath, With* options. Patterns: Converter strategy, Registry factory, functional
options, Painter adapter.

Output formats (core/output.go): OutputPDF (every Converter) and writers (core.Writer:
Output/Extension/MIMEType/Write(ctx, *core.Document, *Config, w)). A Converter that also
implements core.Parser (Parse -> *core.Document{Root, BaseURL, ThemeCSS}; html and markdown
do) can feed any writer; Registry.ConvertTo wires them. pipeline.Style(ctx, Input, cfg) runs
stylesheet collection + cascade only (no layout) and returns *Styled{Styles, BaseURL,
Metadata, Load}, which is what non-paged writers consume.

formats/xlsx (no third-party deps; writes OOXML with archive/zip): each top-level <table> ->
one sheet (name: caption, data-sheet, aria-label, title, id, previous heading, "Table N";
sanitised and unique), no tables -> one "Document" sheet with a row per text block. HTML
table model placement (rowspan/colspan incl. rowspan=0) -> mergeCells; thead rows frozen +
print titles; display:none skipped. Text: CSS white-space, <br>, nested blocks as lines, list
markers, text-transform, ::before/::after; mixed fonts -> rich-text runs (sharedStrings).
Styles from computed CSS: font (generic families mapped, -apple-system etc. skipped), fill =
table/col/rowgroup/row/cell backgrounds composited over white, borders (collapsing model:
widest of cell/row/group/table wins), alignment, indent from padding-left, column widths from
<col>/cell widths else auto-fit, row heights for multi-line text and images. value.go detects
numbers, %, currency ($ € £ ¥ ₹ USD EUR GBP, Rp/IDR with dot thousands), accounting negatives,
ISO dates -> typed cells with a number format that shows the original text; header cells and
identifiers (007) stay text; data-type/data-value/data-formula override; Config.DetectNumbers
(WithNumberDetection, CLI -no-numbers). <a href> -> hyperlink rels; PNG/JPEG/GIF <img> ->
drawing oneCellAnchor (GIF re-encoded PNG, SVG warned). Metadata -> docProps/core.xml.
Tests: reader_test.go re-parses the package independently; xlsx_test.go checks cells, styles,
merges, names, images, limits (32767 chars, control chars), and TestLibreOfficeRoundTrip
(skips without soffice) compares LibreOffice's CSV export, formula evaluated.

Accuracy tooling: accuracy/run.sh with internal/tools/pixdiff (raw and -near ink diff),
internal/tools/textdiff (word positions vs Chromium) and internal/tools/mdhtml (Markdown to the
HTML Chromium renders). See accuracy_and_blink_quirks for thresholds and current numbers.

Toolchain: go 1.26 (GOTOOLCHAIN=auto). Deps: gpdf, goldmark, x/net, cascadia.
