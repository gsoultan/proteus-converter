# Conventions and known gaps

- Errors are prefixed "proteus:"; non-fatal problems go through Config.Warnf, never fail.
- Remote fetching off by default (WithHTTP); local files on (WithLocalFiles, WithFS to confine).
- Tests: go test ./... (proteus_test.go uses pdftotext when installed); css, fonts, svg and layout
  unit tests. layout/layout_test.go runs cascade + build + layout + paginate on small HTML and
  checks item geometry (floats, positioning, flex, grid, gradients).
- Supported layout: block/inline, tables, lists, floats (left/right, clear, BFC avoidance),
  position relative/absolute/fixed (fixed repeats on every page), z-index ordering, flexbox
  (wrap, grow/shrink, justify/align, order, reverse, gaps), grid (fr, minmax, repeat,
  auto-fill/auto-fit, named areas, line placement, spans, implicit tracks, row auto-flow),
  inline and file SVG (svg package), <img> with object-fit/position, background-image with
  size/position/repeat/origin/clip, linear-gradient, rounded uniform borders.
- Unsupported / approximated: quirks-mode UA margins (use <!DOCTYPE html>), grid-auto-flow column
  and dense, radial/conic gradients, SVG gradients (average stop color, with a warning), SVG
  filters/masks/clipPath, overflow clipping of text, canvas/video, WOFF/CFF, kerning in output,
  hierarchical outlines, synthetic bold/italic, text alpha, @page :first, counters.
