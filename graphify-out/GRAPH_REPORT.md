# Graph Report - proteus-converter  (2026-11-14)

## Corpus Check
- 98 files · ~107,028 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 10 file(s) not represented in the graph (top: (none) 2, .ttf 2, .otf 2)

## Summary
- 1193 nodes · 4534 edges · 55 communities (45 shown, 10 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 454 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `759c13bb`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- GPDF
- options.go
- frag
- Resolver
- ParseLength
- Style
- reader_test.go
- parser
- steps
- layoutHTML
- Image
- testing.T
- Converter
- golang.org/x/net/html.Node
- pixdiff/main.go
- chromium.cjs
- Proteus sample
- run.sh
- github.com/gsoultan/proteus-converter
- layouter
- .layoutBlockInner
- extractor
- Render
- ConvertBytes
- Box
- Render
- parser
- .emit
- .layoutTable
- Paint
- sheet
- Matrix
- ConvertString
- .Paginate
- proteus.go
- ResourceLoader
- collapseBorders
- Seg
- gexcel.go
- Quarterly Engineering Report
- .placeFloat
- Stamp
- Color
- borderSideOf
- InternalError
- .absItems
- accuracy_and_blink_quirks.md
- conventions_and_gaps.md
- project_overview.md
- computeDoc
- MediaContext
- content
- Writer
- markerText

## God Nodes (most connected - your core abstractions)
1. `Style` - 126 edges
2. `Box` - 105 edges
3. `layouter` - 83 edges
4. `Painter` - 81 edges
5. `GPDF` - 80 edges
6. `Length` - 56 edges
7. `parser` - 55 edges
8. `Cascade` - 54 edges
9. `Engine` - 52 edges
10. `Resolver` - 45 edges

## Surprising Connections (you probably didn't know these)
- `Design` --references--> `Converter`  [INFERRED]
  README.md → core/converter.go
- `Robustness and resource use` --references--> `WithMaxPages()`  [INFERRED]
  README.md → core/options.go
- `Design` --references--> `Document`  [INFERRED]
  README.md → core/output.go
- `Design` --references--> `Writer`  [INFERRED]
  README.md → core/output.go
- `Design` --references--> `Registry`  [INFERRED]
  README.md → core/registry.go

## Import Cycles
- 2-file cycle: `fuzz_test.go -> watermark_test.go -> fuzz_test.go`
- 2-file cycle: `hardening_test.go -> watermark_test.go -> hardening_test.go`
- 2-file cycle: `leak_test.go -> watermark_test.go -> leak_test.go`
- 2-file cycle: `proteus_test.go -> watermark_test.go -> proteus_test.go`
- 2-file cycle: `fuzz_test.go -> leak_test.go -> fuzz_test.go`
- 2-file cycle: `hardening_test.go -> leak_test.go -> hardening_test.go`
- 2-file cycle: `leak_test.go -> proteus_test.go -> leak_test.go`
- 2-file cycle: `fuzz_test.go -> hardening_test.go -> fuzz_test.go`
- 2-file cycle: `hardening_test.go -> proteus_test.go -> hardening_test.go`
- 2-file cycle: `fuzz_test.go -> proteus_test.go -> fuzz_test.go`
- 2-file cycle: `layout/fidelity_test.go -> layout/hardening_test.go -> layout/fidelity_test.go`
- 2-file cycle: `layout/fidelity_test.go -> layout/layout_test.go -> layout/fidelity_test.go`
- 2-file cycle: `layout/hardening_test.go -> layout/layout_test.go -> layout/hardening_test.go`
- 3-file cycle: `fuzz_test.go -> hardening_test.go -> watermark_test.go -> fuzz_test.go`
- 3-file cycle: `fuzz_test.go -> leak_test.go -> watermark_test.go -> fuzz_test.go`
- 3-file cycle: `fuzz_test.go -> proteus_test.go -> watermark_test.go -> fuzz_test.go`
- 3-file cycle: `fuzz_test.go -> watermark_test.go -> hardening_test.go -> fuzz_test.go`
- 3-file cycle: `hardening_test.go -> leak_test.go -> watermark_test.go -> hardening_test.go`
- 3-file cycle: `hardening_test.go -> proteus_test.go -> watermark_test.go -> hardening_test.go`
- 3-file cycle: `fuzz_test.go -> watermark_test.go -> leak_test.go -> fuzz_test.go`

## Communities (55 total, 10 thin omitted)

### Community 0 - "GPDF"
Cohesion: 0.11
Nodes (22): Converter, extOf(), Format, normalizeExt(), Registry, Watermark, Cascade, Length (+14 more)

### Community 1 - "options.go"
Cohesion: 0.09
Nodes (35): FontSource, Option, Margins, PageSize, UniformMargins(), WithBaseURL(), WithFont(), WithFontDir() (+27 more)

### Community 2 - "frag"
Cohesion: 0.09
Nodes (20): fragIntrinsic(), relativeOffset(), ctxExtent, frag, fragBuilder, fragKind, m3(), firstUnplacedFloat() (+12 more)

### Community 3 - "Resolver"
Cohesion: 0.05
Nodes (37): chainKey, Face, FuzzDecodeWOFF2(), TestDecodeWOFF2(), TestStandardFallback(), TestSystemChain(), Metrics, classOf() (+29 more)

### Community 4 - "ParseLength"
Cohesion: 0.05
Nodes (77): calcParser, calcVal, ParseColor(), parseHex(), computeCtx, FuzzParse(), expandLayoutShorthand(), GridLine (+69 more)

### Community 5 - "Style"
Cohesion: 0.07
Nodes (19): BackgroundSize, blockify(), sortDecls(), substituteVars(), LineHeight, matchedDecl, propDef, isMonospaceOnly() (+11 more)

### Community 6 - "reader_test.go"
Cohesion: 0.13
Nodes (21): parseRef(), readBook(), readRels(), unesc(), detect(), detectDate(), detectRupiah(), explicit() (+13 more)

### Community 7 - "parser"
Cohesion: 0.15
Nodes (15): AtPage, rankOf(), compiledRule, FontFace, Import, Origin, Declaration, Stylesheet (+7 more)

### Community 8 - "steps"
Cohesion: 0.11
Nodes (24): runs-on, steps, runs-on, steps, jobs, accuracy, fuzz, test (+16 more)

### Community 9 - "layoutHTML"
Cohesion: 0.16
Nodes (27): TestAlignAttributeCentresNestedTable(), TestBoxShadowAndOutline(), TestColumnsBalanceAndRule(), TestFirstLetterFloat(), TestFirstLineStylesOnlyTheFirstLine(), TestHalfLeadingAboveTextIsFloored(), TestPercentageHeights(), TestRowspanExcessIsProportional() (+19 more)

### Community 10 - "Image"
Cohesion: 0.23
Nodes (6): Image, num(), op(), pathOps(), rasterize(), rawImage

### Community 11 - "testing.T"
Cohesion: 0.21
Nodes (25): WithSystemFonts(), New(), New(), New(), convert(), convertHTML(), TestBackgroundLayers(), TestCollapsedBordersInherit() (+17 more)

### Community 13 - "golang.org/x/net/html.Node"
Cohesion: 0.13
Nodes (22): ElementStyle, ancestorTable(), attr(), htmlColor(), htmlLength(), PresentationalHints(), Expand(), addClass() (+14 more)

### Community 14 - "pixdiff/main.go"
Cohesion: 0.16
Nodes (13): check(), compare(), compareNear(), inked(), inkMask(), main(), raster(), align() (+5 more)

### Community 15 - "chromium.cjs"
Cohesion: 0.29
Nodes (3): { chromium }, path, { pathToFileURL }

### Community 16 - "Proteus sample"
Cohesion: 0.50
Nodes (3): Lists, Proteus sample, Table

### Community 19 - "layouter"
Cohesion: 0.21
Nodes (5): hasContent(), hasFloatFrags(), inlineResult, layouter, pendingMarker

### Community 20 - ".layoutBlockInner"
Cohesion: 0.16
Nodes (12): Pt(), legacyAlign(), blockResult, edges, isForcedBreak(), translateItems(), alignOffset(), gridSelf() (+4 more)

### Community 21 - "extractor"
Cohesion: 0.15
Nodes (20): attr(), findBody(), firstNonEmpty(), isBlockText(), isLeafBlock(), spanAttr(), fontName(), fontOf() (+12 more)

### Community 22 - "Render"
Cohesion: 0.17
Nodes (8): ConverterFunc, Recover(), Cm(), NewConfig(), Document, TestRecoverTurnsPanicIntoError(), Converter, Convert()

### Community 23 - "ConvertBytes"
Cohesion: 0.24
Nodes (15): fuzzConvert(), FuzzConvertCSS(), FuzzConvertHTML(), FuzzConvertMarkdown(), fuzzSeeds(), BenchmarkConvert(), convertFixture(), fixtures() (+7 more)

### Community 24 - "Box"
Cohesion: 0.13
Nodes (20): absPart(), clampWidth(), isBFC(), textContent(), Box, contentsResult, allCollapsibleSpace(), anonymousBlock() (+12 more)

### Community 25 - "Render"
Cohesion: 0.12
Nodes (27): ResourceLoader, ResolveURL(), isIgnorableSelector(), NewCascade(), selectorKey(), Parse(), stripComments(), cached (+19 more)

### Community 26 - "parser"
Cohesion: 0.21
Nodes (12): ParseDeclarations(), Inherited, paint, parser, state, attr(), defaultState(), FromNodeInherit() (+4 more)

### Community 27 - ".emit"
Cohesion: 0.15
Nodes (10): boxAreas(), cornerRadii(), normCDF(), pickArea(), tilePositions(), ImageItem, Clip, pageClip() (+2 more)

### Community 28 - ".layoutTable"
Cohesion: 0.20
Nodes (9): collapseBorders(), toPx(), rowCell, columnPercents(), rowCells(), tableGrid(), tableSpacing(), tcell (+1 more)

### Community 30 - "sheet"
Cohesion: 0.15
Nodes (15): fitImages(), cellRef(), colName(), sheetName(), uniqueSheetName(), widthChars(), anchoredImage, cell (+7 more)

### Community 31 - "Matrix"
Cohesion: 0.14
Nodes (13): PathStyle, PathItem, parseTransform(), Item, Shape, FromNode(), Document, Matrix (+5 more)

### Community 32 - "ConvertString"
Cohesion: 0.17
Nodes (18): pngHeader(), TestCancelledContext(), TestHugeImageIsRefusedWithWarning(), TestRunawayDocumentHitsPageLimit(), ConvertString(), checkPDF(), pdfText(), TestContextCancelled() (+10 more)

### Community 33 - ".Paginate"
Cohesion: 0.24
Nodes (8): finite(), flattenItems(), Page, pathExtent(), spanPages(), cloneItem(), copyItems(), Document

### Community 34 - "proteus.go"
Cohesion: 0.08
Nodes (21): fail(), main(), parsePage(), OutputFormat, NewRegistry(), Theme(), main(), ConvertBytesTo() (+13 more)

### Community 35 - "ResourceLoader"
Cohesion: 0.20
Nodes (6): LoaderFunc, Config, decodeDataURL(), defaultLoad(), DefaultLoader(), readLimited()

### Community 36 - "collapseBorders"
Cohesion: 0.50
Nodes (4): cborder, resolve(), sideBorder(), TestCollapsedBorderConflicts()

### Community 37 - "Seg"
Cohesion: 0.26
Nodes (9): arc(), ellipse(), ParsePath(), parsePath(), quad(), roundRect(), scanner, Seg (+1 more)

### Community 38 - "gexcel.go"
Cohesion: 0.24
Nodes (5): Metadata, cleanXML(), fillSheet(), write(), mergeMeta()

### Community 39 - "Quarterly Engineering Report"
Cohesion: 0.29
Nodes (6): Appendix, Code, Follow-ups, Metrics, Quarterly Engineering Report, Summary

### Community 40 - ".placeFloat"
Cohesion: 0.21
Nodes (3): atomicBox, floatBox, floatCtx

### Community 41 - "Stamp"
Cohesion: 0.22
Nodes (5): WatermarkLayer, WatermarkPages, WatermarkPlacement, StampRun, Stamp

### Community 42 - "Color"
Cohesion: 0.10
Nodes (14): clamp01(), colorArgs(), Color, hslToRGB(), parseAlpha(), parseColorFunc(), parseComponent(), Shadow (+6 more)

### Community 44 - "borderSideOf"
Cohesion: 0.24
Nodes (8): pickSide(), sideWidth(), argb(), argbOver(), blend(), borderOf(), borderSideOf(), clamp()

### Community 46 - ".absItems"
Cohesion: 0.25
Nodes (3): absBox, groupItem, posCtx

### Community 50 - "computeDoc"
Cohesion: 0.48
Nodes (7): rgb255(), computeDoc(), find(), TestCascadeOrder(), TestLayoutProperties(), TestMonospaceQuirkAndHeadings(), TestVarsCalcShorthands()

### Community 51 - "MediaContext"
Cohesion: 0.38
Nodes (3): MediaContext, splitAnd(), MediaMatcher

### Community 52 - "content"
Cohesion: 0.38
Nodes (7): collapseSegments(), transform(), content, font, imageRef, segment, textRun

### Community 54 - "markerText"
Cohesion: 0.60
Nodes (5): alpha(), itoa(), markerText(), roman(), sscan()

## Knowledge Gaps
- **2 isolated node(s):** `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 133 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Style` connect `Style` to `GPDF`, `frag`, `ParseLength`, `collapseBorders`, `Color`, `borderSideOf`, `golang.org/x/net/html.Node`, `layouter`, `.layoutBlockInner`, `extractor`, `markerText`, `content`, `Box`, `.emit`, `.layoutTable`?**
  _High betweenness centrality (0.107) - this node is a cross-community bridge._
- **What connects `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter` to the rest of the system?**
  _2 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `GPDF` be split into smaller, more focused modules?**
  _Cohesion score 0.11066887783305694 - nodes in this community are weakly interconnected._
- **Why does `Box` connect `Box` to `GPDF`, `frag`, `Style`, `.placeFloat`, `Color`, `Image`, `golang.org/x/net/html.Node`, `.absItems`, `layouter`, `.layoutBlockInner`, `.emit`, `.layoutTable`?**
  _High betweenness centrality (0.099) - this node is a cross-community bridge._
- **Should `options.go` be split into smaller, more focused modules?**
  _Cohesion score 0.08658536585365853 - nodes in this community are weakly interconnected._
- **Why does `Color` connect `Color` to `GPDF`, `.Paginate`, `Resolver`, `ParseLength`, `Style`, `collapseBorders`, `layoutHTML`, `Stamp`, `borderSideOf`, `computeDoc`, `Box`, `parser`, `.emit`, `Paint`, `Matrix`?**
  _High betweenness centrality (0.051) - this node is a cross-community bridge._
- **Should `frag` be split into smaller, more focused modules?**
  _Cohesion score 0.09178743961352658 - nodes in this community are weakly interconnected._