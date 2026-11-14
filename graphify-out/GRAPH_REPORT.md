# Graph Report - proteus-converter  (2026-11-14)

## Corpus Check
- 99 files · ~109,576 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 10 file(s) not represented in the graph (top: (none) 2, .ttf 2, .otf 2)

## Summary
- 1213 nodes · 4629 edges · 56 communities (47 shown, 9 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 462 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `1f58920b`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- GPDF
- options.go
- frag
- Resolver
- ParseLength
- Style
- xlsx/value.go
- .computeStyle
- steps
- testing.T
- Image
- loadFontFaces
- Converter
- golang.org/x/net/html.Node
- hardening_test.go
- chromium.cjs
- Proteus sample
- run.sh
- github.com/gsoultan/proteus-converter
- LengthContext
- layouter
- extractor
- Render
- ParseColor
- Box
- Render
- parser
- .emit
- .layoutTable
- Track
- sheet
- Matrix
- parseLinearGradient
- .Paginate
- proteus.go
- Document
- collapseBorders
- Seg
- gexcel.go
- Quarterly Engineering Report
- .placeFloat
- Stamp
- parseColorFunc
- xlsx/styles.go
- InternalError
- .layoutAtomic
- accuracy_and_blink_quirks.md
- conventions_and_gaps.md
- project_overview.md
- GridLine
- MediaContext
- content
- Writer
- markerText
- MarkerText

## God Nodes (most connected - your core abstractions)
1. `Style` - 127 edges
2. `Box` - 105 edges
3. `layouter` - 83 edges
4. `Painter` - 81 edges
5. `GPDF` - 80 edges
6. `Length` - 57 edges
7. `parser` - 56 edges
8. `Cascade` - 55 edges
9. `Engine` - 52 edges
10. `Resolver` - 45 edges

## Surprising Connections (you probably didn't know these)
- `Design` --references--> `Converter`  [INFERRED]
  README.md → core/converter.go
- `Design` --references--> `Option`  [INFERRED]
  README.md → core/options.go
- `Design` --references--> `WithPageSize()`  [INFERRED]
  README.md → core/options.go
- `Robustness and resource use` --references--> `WithMaxPages()`  [INFERRED]
  README.md → core/options.go
- `Design` --references--> `WithStylesheet()`  [INFERRED]
  README.md → core/options.go

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

## Communities (56 total, 9 thin omitted)

### Community 0 - "GPDF"
Cohesion: 0.09
Nodes (24): Converter, Watermark, Cascade, Color, Length, parser, init(), sideNames() (+16 more)

### Community 1 - "options.go"
Cohesion: 0.08
Nodes (38): FontSource, Option, Cm(), Margins, Metadata, PageSize, UniformMargins(), WithBaseURL() (+30 more)

### Community 2 - "frag"
Cohesion: 0.12
Nodes (16): fragIntrinsic(), frag, fragKind, m3(), firstUnplacedFloat(), hasContent(), hasFloatFrags(), canBreakAnywhere() (+8 more)

### Community 3 - "Resolver"
Cohesion: 0.05
Nodes (37): chainKey, Face, FuzzDecodeWOFF2(), TestDecodeWOFF2(), TestStandardFallback(), TestSystemChain(), Metrics, classOf() (+29 more)

### Community 4 - "ParseLength"
Cohesion: 0.21
Nodes (20): BackgroundSize, computeCtx, parseBackgroundSize(), parseGap(), parsePosition(), parseRepeat(), registerLayoutProps(), ParseLength() (+12 more)

### Community 5 - "Style"
Cohesion: 0.10
Nodes (13): LineHeight, Shadow, Style, textContent(), ctxExtent, fragBuilder, innerRadius(), applyTransform() (+5 more)

### Community 6 - "xlsx/value.go"
Cohesion: 0.42
Nodes (7): detect(), detectDate(), detectRupiah(), explicit(), numberFormat(), serial(), typed

### Community 7 - ".computeStyle"
Cohesion: 0.06
Nodes (31): AtPage, blockify(), isIgnorableSelector(), NewCascade(), rankOf(), selectorKey(), sortDecls(), substituteVars() (+23 more)

### Community 8 - "steps"
Cohesion: 0.11
Nodes (24): runs-on, steps, runs-on, steps, jobs, accuracy, fuzz, test (+16 more)

### Community 9 - "testing.T"
Cohesion: 0.06
Nodes (81): WithSystemFonts(), rgb255(), computeDoc(), find(), TestCascadeOrder(), TestLayoutProperties(), TestMonospaceQuirkAndHeadings(), TestVarsCalcShorthands() (+73 more)

### Community 10 - "Image"
Cohesion: 0.23
Nodes (6): Image, num(), op(), pathOps(), rasterize(), rawImage

### Community 11 - "loadFontFaces"
Cohesion: 0.23
Nodes (16): FuzzParse(), parseAreas(), parseTrack(), parseTrackList(), splitTrackTokens(), parseImport(), Function(), isHex() (+8 more)

### Community 13 - "golang.org/x/net/html.Node"
Cohesion: 0.17
Nodes (17): ElementStyle, Initial(), addClass(), markTaskLists(), prevCell(), applyFirstLetter(), attr(), countListItems() (+9 more)

### Community 14 - "hardening_test.go"
Cohesion: 0.06
Nodes (53): fail(), main(), parsePage(), Theme(), fuzzConvert(), FuzzConvertCSS(), FuzzConvertHTML(), FuzzConvertMarkdown() (+45 more)

### Community 15 - "chromium.cjs"
Cohesion: 0.29
Nodes (3): { chromium }, path, { pathToFileURL }

### Community 16 - "Proteus sample"
Cohesion: 0.50
Nodes (3): Lists, Proteus sample, Table

### Community 19 - "LengthContext"
Cohesion: 0.40
Nodes (6): calcParser, calcVal, evalCalc(), evalMinMax(), LengthContext, isExpSign()

### Community 20 - "layouter"
Cohesion: 0.10
Nodes (18): Pt(), absBox, isBFC(), relativeOffset(), blockResult, columnGeometry(), contentsResult, isForcedBreak() (+10 more)

### Community 21 - "extractor"
Cohesion: 0.15
Nodes (17): attr(), findBody(), isBlockText(), isLeafBlock(), spanAttr(), widthChars(), collapseText(), hasAttr() (+9 more)

### Community 22 - "Render"
Cohesion: 0.17
Nodes (10): ConverterFunc, Recover(), Config, NewConfig(), Document, decodeDataURL(), defaultLoad(), readLimited() (+2 more)

### Community 23 - "ParseColor"
Cohesion: 0.31
Nodes (10): ParseColor(), parseHex(), expandLayoutShorthand(), isInteger(), parseBoxShadow(), expandBackground(), expandShorthand(), fourSides() (+2 more)

### Community 24 - "Box"
Cohesion: 0.22
Nodes (13): Decoration, Box, Decoration, allCollapsibleSpace(), anonymousBlock(), fixTable(), fixTree(), hasBlockChild() (+5 more)

### Community 25 - "Render"
Cohesion: 0.14
Nodes (24): LoaderFunc, DefaultLoader(), ResourceLoader, ResolveURL(), Parse(), cached, cachingLoader, imageLoader (+16 more)

### Community 26 - "parser"
Cohesion: 0.20
Nodes (12): ParseDeclarations(), stripComments(), paint, parser, state, attr(), defaultState(), FromNodeInherit() (+4 more)

### Community 27 - ".emit"
Cohesion: 0.15
Nodes (10): boxAreas(), cornerRadii(), normCDF(), pickArea(), tilePositions(), ImageItem, Clip, pageClip() (+2 more)

### Community 28 - ".layoutTable"
Cohesion: 0.26
Nodes (8): rowCell, columnPercents(), distributePercent(), rowCells(), tableGrid(), tableSpacing(), tcell, trow

### Community 29 - "Track"
Cohesion: 0.31
Nodes (9): Track, TrackList, autoTrack(), expandTracks(), fixedTrackSize(), sizeTracks(), trackPositions(), visibleTracks() (+1 more)

### Community 30 - "sheet"
Cohesion: 0.19
Nodes (13): fitImages(), cellRef(), colName(), sheetName(), uniqueSheetName(), anchoredImage, cell, cellKind (+5 more)

### Community 31 - "Matrix"
Cohesion: 0.17
Nodes (9): PathStyle, pathExtent(), PathItem, parseTransform(), Item, Shape, Matrix, ScaleXY() (+1 more)

### Community 32 - "parseLinearGradient"
Cohesion: 0.25
Nodes (9): parseTrackSize(), ParseNumber(), expandFont(), isWeightNumber(), TrackKind, TrackSize, isAngle(), parseAngle() (+1 more)

### Community 33 - ".Paginate"
Cohesion: 0.27
Nodes (7): finite(), flattenItems(), Page, spanPages(), cloneItem(), copyItems(), Document

### Community 34 - "proteus.go"
Cohesion: 0.09
Nodes (21): extOf(), Format, normalizeExt(), OutputFormat, Parser, Registry, NewRegistry(), Writer (+13 more)

### Community 35 - "Document"
Cohesion: 0.33
Nodes (6): decodeImage(), resolveWatermark(), FromNode(), Document, Parse(), TestParseShapes()

### Community 36 - "collapseBorders"
Cohesion: 0.38
Nodes (6): cborder, collapseBorders(), resolve(), sideBorder(), TestCollapsedBorderConflicts(), toPx()

### Community 37 - "Seg"
Cohesion: 0.26
Nodes (10): arc(), ellipse(), ParsePath(), parsePath(), quad(), RoundRect4(), roundRect(), scanner (+2 more)

### Community 38 - "gexcel.go"
Cohesion: 0.31
Nodes (5): cacheFormulas(), cleanXML(), fillSheet(), write(), workbook

### Community 39 - "Quarterly Engineering Report"
Cohesion: 0.29
Nodes (6): Appendix, Code, Follow-ups, Metrics, Quarterly Engineering Report, Summary

### Community 40 - ".placeFloat"
Cohesion: 0.23
Nodes (3): atomicBox, floatBox, floatCtx

### Community 41 - "Stamp"
Cohesion: 0.22
Nodes (5): WatermarkLayer, WatermarkPages, WatermarkPlacement, StampRun, Stamp

### Community 42 - "parseColorFunc"
Cohesion: 0.47
Nodes (6): clamp01(), colorArgs(), hslToRGB(), parseAlpha(), parseColorFunc(), parseComponent()

### Community 44 - "xlsx/styles.go"
Cohesion: 0.14
Nodes (18): firstNonEmpty(), pickSide(), sideWidth(), argb(), argbOver(), blend(), borderOf(), borderSideOf() (+10 more)

### Community 46 - ".layoutAtomic"
Cohesion: 0.19
Nodes (8): absPart(), clampWidth(), legacyAlign(), edges, resolveAlign(), resolveFlexible(), flexItem, flexLine

### Community 50 - "GridLine"
Cohesion: 0.40
Nodes (4): GridLine, parseGridLine(), findArea(), resolveLines()

### Community 51 - "MediaContext"
Cohesion: 0.38
Nodes (3): MediaContext, splitAnd(), MediaMatcher

### Community 52 - "content"
Cohesion: 0.28
Nodes (8): collapseSegments(), runFont(), transform(), content, font, imageRef, segment, textRun

### Community 54 - "markerText"
Cohesion: 0.60
Nodes (5): alpha(), itoa(), markerText(), roman(), sscan()

### Community 55 - "MarkerText"
Cohesion: 0.50
Nodes (4): alpha(), boolInt(), MarkerText(), roman()

## Knowledge Gaps
- **2 isolated node(s):** `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 134 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **9 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Style` connect `Style` to `GPDF`, `frag`, `ParseLength`, `collapseBorders`, `.computeStyle`, `xlsx/styles.go`, `golang.org/x/net/html.Node`, `.layoutAtomic`, `GridLine`, `content`, `layouter`, `markerText`, `extractor`, `Box`, `.emit`, `.layoutTable`, `Track`?**
  _High betweenness centrality (0.146) - this node is a cross-community bridge._
- **What connects `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter` to the rest of the system?**
  _2 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `GPDF` be split into smaller, more focused modules?**
  _Cohesion score 0.09324384787472036 - nodes in this community are weakly interconnected._
- **Why does `Box` connect `Box` to `GPDF`, `frag`, `collapseBorders`, `Style`, `Seg`, `.placeFloat`, `Image`, `golang.org/x/net/html.Node`, `.layoutAtomic`, `layouter`, `.emit`, `.layoutTable`?**
  _High betweenness centrality (0.075) - this node is a cross-community bridge._
- **Should `options.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07676767676767676 - nodes in this community are weakly interconnected._
- **Why does `Painter` connect `GPDF` to `options.go`, `proteus.go`, `Resolver`, `.Paginate`, `Style`, `Seg`, `Stamp`, `Box`, `Matrix`?**
  _High betweenness centrality (0.054) - this node is a cross-community bridge._
- **Should `frag` be split into smaller, more focused modules?**
  _Cohesion score 0.12012012012012012 - nodes in this community are weakly interconnected._