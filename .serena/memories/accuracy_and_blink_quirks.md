# Accuracy against Chromium

Gate: accuracy/run.sh renders testdata/* with proteus and Chromium (Playwright, print media,
A4, 1cm margins, FONTCONFIG_FILE=accuracy/fonts.conf pinning generics to Liberation), then
internal/tools/pixdiff compares pages (ink = luma < 200). CI fails above 3% (MAX_MISMATCH).
Current: invoice.html ~1.35%, sample.md ~1.2%.

Blink details reproduced on purpose:
- line metrics: ascent/descent/lineGap each rounded to px (fonts.Face.LineMetrics)
- monospace default 13px quirk; border widths snapped to whole px
- list symbol markers: offset=asc*2/3, bullet=(offset+1)/2 px, marker padding 7px
- collapsed table borders (layout/collapse.go): CSS 2.1 conflict resolution, each grid line
  W px split floor(W/2) to the cell above/left, the rest below/right; table keeps outer halves
- canvas background propagates from html/body

gpdf quirks worked around:
- Tc (letter spacing) persists across text runs and gpdf only writes it when non-zero, so
  render/gpdf.go always passes >= 1e-4 (%g encoding forbids smaller values)
- TextWidth includes kerning but drawing does not: widths subtract kerning
- link annotations lack /Border [0 0 0], so viewers draw a box (needs a gpdf change)
- outlines are flat; paint order is graphics, then text, then images
