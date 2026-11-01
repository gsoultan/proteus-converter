# Conventions and known gaps

- Errors are prefixed "proteus:"; non-fatal problems go through Config.Warnf, never fail.
- Hardening: format Convert methods and pipeline.Render defer core.Recover(&err), turning panics
  into *core.InternalError (ErrInternal, with stack). Limits: Config.MaxPages (default 10000,
  ErrTooManyPages via layout.Engine.MaxPages / Paginate error), images > 40 MP or > 32768 px a
  side refused with a warning (pipeline/images.go), css.MaxGridTracks = 1000 clamps grid lines,
  spans and repeat(); x/net/html refuses nesting > 512. layoutBlockInner only restarts a block on a
  later page when that moves it forward (restartAt > y): a negative top margin used to loop forever
  (fuzz seed testdata/fuzz/FuzzConvertHTML/a676d47999945e8f). HTML fuzzing runs only a few execs/s
  (already so on main 2026-11-01). CI must install every system font a fixture names (Carlito). Paginate drops non-finite coordinates.
- Tests: fuzz_test.go (FuzzConvertHTML/Markdown/CSS; 10s watchdog dumps stacks to
  $PROTEUS_FUZZ_HANG and panics so the fuzzer keeps the input), css/fuzz_test.go (FuzzParse),
  leak_test.go (TestNoLeaks, TestConcurrentConversions compares pdftotext -bbox output because
  gpdf bytes differ per run, BenchmarkConvert with KiB/pdf), hardening_test.go, layout/
  hardening_test.go (tables, page-break margins, MaxPages). fonts BenchmarkSystemScan.
- Perf (2026-11-01, 4 vCPU, gpdf 393b425 pools zlib writers): 4-27 ms per fixture, 1.7-12 MB allocated;
  webfonts 55 ms / 14 MB (whole OpenType-CFF fonts are embedded, no CFF subsetting -> 449 KB PDF). Font directory scan reads only header tables
  (parseSfntInfoAt over io.ReaderAt): 123 MB -> 0.6 MB allocated, ~3 ms. rasterize() reads
  NRGBA/Paletted/Gray pixels directly.
- Remote fetching off by default (WithHTTP); local files on (WithLocalFiles, WithFS to confine).
- Tests: go test ./... (proteus_test.go and leak_test.go use pdftotext when installed); css,
  fonts, svg and layout unit tests. layout/layout_test.go runs cascade + build + layout + paginate on small HTML and
  checks item geometry (floats, positioning, flex, grid, gradients).
- Supported layout: block/inline, tables, lists, floats (left/right, clear, BFC avoidance),
  position relative/absolute/fixed (fixed repeats on every page), z-index ordering, flexbox
  (wrap, grow/shrink, justify/align, order, reverse, gaps), grid (fr, minmax, repeat,
  auto-fill/auto-fit, named areas, line placement, spans, implicit tracks, row auto-flow),
  inline and file SVG (svg package), <img> with object-fit/position, background-image with
  size/position/repeat/origin/clip, linear-gradient, rounded borders (per-corner radii for uniform
  borders and backgrounds), box-shadow, outline, ::first-letter, percentage heights, small-caps,
  synthetic bold/italic, @font-face TrueType/OpenType-CFF/WOFF 1.0, GPOS kerning (gpdf).
- Unsupported / approximated: quirks-mode UA margins (use <!DOCTYPE html>), grid-auto-flow column
  and dense, radial/conic gradients, SVG gradients (average stop color, with a warning), SVG
  filters/masks/clipPath, overflow clipping of text, canvas/video, WOFF2, contextual alternates
  (calt, e.g. JetBrains Mono "!=" ligatures), inset box-shadow, ::first-line, CSS columns,
  hierarchical outlines, text alpha, @page :first, counters, % heights inside tables/grid items.
  Mitered corners for differently coloured border sides, inset/outset shading and Blink's
  double-border line widths are approximated. Trailing blank space (e.g. a tall empty div at
  the end) does not create extra pages the way Chromium does.
