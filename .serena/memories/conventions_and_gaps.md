# Conventions and known gaps

- Errors are prefixed "proteus:"; non-fatal problems go through Config.Warnf, never fail.
- Remote fetching off by default (WithHTTP); local files on (WithLocalFiles, WithFS to confine).
- Tests: go test ./... (proteus_test.go uses pdftotext when installed); css, fonts, layout unit tests.
- Unsupported: floats, flex/grid, positioning, SVG, background images, WOFF/CFF, kerning in
  output, hierarchical outlines, synthetic bold/italic, text alpha, @page :first, counters.
