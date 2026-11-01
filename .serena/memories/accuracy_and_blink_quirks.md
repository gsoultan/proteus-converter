# Accuracy against Chromium

Gate: accuracy/run.sh renders testdata/* with proteus and Chromium (Playwright, print media,
A4, 1cm margins, FONTCONFIG_FILE=accuracy/fonts.conf pinning generics to Liberation), then
- internal/tools/pixdiff compares pages (ink = luma < 200): raw mismatch <= 3% (MAX_MISMATCH)
  and, with -near 1, mismatch ignoring ink displaced by <= 1px <= 0.5% (MAX_NEAR). The near
  figure is the real fidelity signal; the raw one is mostly anti-aliasing of small serif text.
- internal/tools/textdiff pairs words from pdftotext -bbox (LCS on text, then nearest same-text
  word on the same page) and needs >= 99% words found, >= 98% within 2pt (MIN_MATCH, MIN_WITHIN,
  TOL). -v lists missing words, the largest moves and the mean shift.
Corpus (11): boxes, flex, floats, grid, images, invoice, lists, tables, typography (HTML),
report, sample (Markdown). Current (2026-10-31): all words found and within 1.5pt (median
~0.4pt); raw 0.2-3.0% (typography p1 2.98% is the closest to the gate); near 0-0.31% (floats).
Debug loop: render one file with both (node accuracy/chromium.cjs in.html out.pdf; proteus -q),
then textdiff -v and a side-by-side pdftoppm image.
Fixtures must declare @page { size: A4; margin: ... }: without it Playwright's format A4 gives a
595.92pt page with integer-px margins, which shifts everything ~1-2px and costs 2-3%.
Fractional line heights (e.g. 1.45 x 13px) leave sub-pixel baseline differences; snapping
baselines to whole px was tried and made invoice worse, so it is not done.

Blink details reproduced on purpose:
- margins adjoining an unforced page break are truncated (layouter.afterMargin): a collapsed
  margin that crosses into a new page makes the content start at that page's top
- text widths include pair kerning (Face.Width = truetype TextWidth); gpdf draws kerning as TJ
- inside disc/circle/square markers: bullet 1px in, text after 1px + bullet + font-size
  (measured against Chromium, layouter.insideSymbolFrag)
- tables are border-box; table-layout: fixed uses first-row widths, free columns share the rest;
  percentage cells get pct x inner width first (distributePercent) and an auto-width table grows
  so they fit; rows are measured first (measureRows) so rowspan cells stretch the last row;
  cells align content by vertical-align against the row height even with an explicit height
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
- the pinned gpdf both measures (TextWidth) and draws (TJ) with pair kerning, so widths keep it
- link annotations lack /Border [0 0 0], so viewers draw a box (needs a gpdf change)
- outlines are flat; paint order is graphics, then text, then images. Images are registered as
  XObjects through an invisible 0.01pt ImageRun and painted from a custom GraphicRun
  (q clip cm /ImN Do Q) so they keep call order; this relies on gpdf naming them Im%d in order
- gpdf DecodePNGToRaw drops alpha, so render decodes images itself (NRGBA + soft mask)
- gpdf writes image streams uncompressed; %g number encoding means num() rounds values
- reported to the gpdf thread (2026-10-31), not fixed in proteus: font subsets keep post/name/
  cmap and full-size loca/hmtx (a few CJK glyphs embed ~224 KB; flate of these is ~50% of CPU),
  raw image streams, no /Border [0 0 0] on links, output not byte-reproducible (random subset
  tags, map-ordered dicts), no shading support (gradients are emitted as strips)
