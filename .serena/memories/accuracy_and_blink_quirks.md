# Accuracy against Chromium

Gate: accuracy/run.sh renders testdata/* with proteus and Chromium (Playwright, print media,
A4, 1cm margins, FONTCONFIG_FILE=accuracy/fonts.conf pinning generics to Liberation), then
internal/tools/pixdiff compares pages (ink = luma < 200). CI fails above 3% (MAX_MISMATCH).
Current: invoice 1.35%, sample 1.2%, floats 2.7%, flex 0.3%, grid 0.2%, images 0.3%.
Fixtures must declare @page { size: A4; margin: ... }: without it Playwright's format A4 gives a
595.92pt page with integer-px margins, which shifts everything ~1-2px and costs 2-3%.
Fractional line heights (e.g. 1.45 x 13px) leave sub-pixel baseline differences; snapping
baselines to whole px was tried and made invoice worse, so it is not done.

Blink details reproduced on purpose:
- line metrics: ascent/descent/lineGap each rounded to px (fonts.Face.LineMetrics)
- monospace default 13px quirk; border widths snapped to whole px
- list symbol markers: offset=asc*2/3, bullet=(offset+1)/2 px, marker padding 7px
- collapsed table borders (layout/collapse.go): CSS 2.1 conflict resolution, each grid line
  W px split floor(W/2) to the cell above/left, the rest below/right; table keeps outer halves
- canvas background propagates from html/body
- float placement: a float is placed once its block offset is known. At the start of a container
  whose top margin may still collapse through, it waits for the next in-flow sibling's margin;
  otherwise it goes at the current position plus pending margins (layoutContents deferOOF)
- children of flex/grid containers are blockified; absolutely positioned boxes get float:none

gpdf quirks worked around:
- Tc (letter spacing) persists across text runs and gpdf only writes it when non-zero, so
  render/gpdf.go always passes >= 1e-4 (%g encoding forbids smaller values)
- TextWidth includes kerning but drawing does not: widths subtract kerning
- link annotations lack /Border [0 0 0], so viewers draw a box (needs a gpdf change)
- outlines are flat; paint order is graphics, then text, then images. Images are registered as
  XObjects through an invisible 0.01pt ImageRun and painted from a custom GraphicRun
  (q clip cm /ImN Do Q) so they keep call order; this relies on gpdf naming them Im%d in order
- gpdf DecodePNGToRaw drops alpha, so render decodes images itself (NRGBA + soft mask)
- gpdf writes image streams uncompressed; %g number encoding means num() rounds values
