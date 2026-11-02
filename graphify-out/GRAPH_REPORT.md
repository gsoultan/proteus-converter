# Graph Report - wt7  (2026-11-02)

## Corpus Check
- 86 files · ~91,035 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 11 file(s) not represented in the graph (top: (none) 3, .ttf 2, .otf 2)

## Summary
- 1004 nodes · 3714 edges · 49 communities (38 shown, 11 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 372 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `07dd116c`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- GPDF
- options.go
- frag
- Resolver
- ParseLength
- Style
- .computeStyle
- parser
- steps
- testing.T
- Image
- Format
- markdown.go
- svg.go
- pixdiff/main.go
- chromium.cjs
- Proteus sample
- run.sh
- github.com/gsoultan/proteus-converter
- .layoutContents
- .layoutAtomic
- Convert
- Render
- ConvertBytes
- Box
- Render
- LineItem
- .emit
- .layoutTable
- Paint
- rgb
- pageRules
- Converter
- .Paginate
- proteus_test.go
- ResourceLoader
- collapseBorders
- textdiff/main.go
- Decoration
- Quarterly Engineering Report
- .placeFloat
- Stamp
- Color
- Track
- InternalError
- accuracy_and_blink_quirks.md
- conventions_and_gaps.md
- project_overview.md

## God Nodes (most connected - your core abstractions)
1. `Style` - 108 edges
2. `Box` - 105 edges
3. `layouter` - 83 edges
4. `GPDF` - 80 edges
5. `Painter` - 80 edges
6. `Engine` - 52 edges
7. `Length` - 50 edges
8. `parser` - 50 edges
9. `Cascade` - 49 edges
10. `Resolver` - 45 edges

## Surprising Connections (you probably didn't know these)
- `Design` --references--> `Converter`  [INFERRED]
  README.md → core/converter.go
- `Robustness and resource use` --references--> `WithMaxPages()`  [INFERRED]
  README.md → core/options.go
- `Design` --references--> `Registry`  [INFERRED]
  README.md → core/registry.go
- `Security` --references--> `ResourceLoader`  [INFERRED]
  README.md → core/resource.go
- `Robustness and resource use` --references--> `FuzzParse()`  [INFERRED]
  README.md → css/fuzz_test.go

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

## Communities (49 total, 11 thin omitted)

### Community 0 - "GPDF"
Cohesion: 0.16
Nodes (17): BackgroundSize, Cascade, Length, init(), sideNames(), TrackKind, TrackSize, AnchorItem (+9 more)

### Community 1 - "options.go"
Cohesion: 0.10
Nodes (30): FontSource, Option, WithBaseURL(), WithFont(), WithFontDir(), WithFS(), WithHTTP(), WithLocalFiles() (+22 more)

### Community 2 - "frag"
Cohesion: 0.12
Nodes (16): fragIntrinsic(), frag, fragKind, m3(), firstUnplacedFloat(), hasContent(), hasFloatFrags(), canBreakAnywhere() (+8 more)

### Community 3 - "Resolver"
Cohesion: 0.06
Nodes (36): chainKey, Face, FuzzDecodeWOFF2(), TestDecodeWOFF2(), TestStandardFallback(), TestSystemChain(), Metrics, classOf() (+28 more)

### Community 4 - "ParseLength"
Cohesion: 0.07
Nodes (61): calcParser, calcVal, ParseColor(), computeCtx, FuzzParse(), expandLayoutShorthand(), GridLine, isInteger() (+53 more)

### Community 5 - "Style"
Cohesion: 0.11
Nodes (13): LineHeight, propDef, Style, relativeOffset(), textContent(), ctxExtent, fragBuilder, innerRadius() (+5 more)

### Community 6 - ".computeStyle"
Cohesion: 0.20
Nodes (6): blockify(), sortDecls(), substituteVars(), matchedDecl, isMonospaceOnly(), isGlobalKeyword()

### Community 7 - "parser"
Cohesion: 0.14
Nodes (24): AtPage, isIgnorableSelector(), NewCascade(), rankOf(), selectorKey(), compiledRule, FontFace, Import (+16 more)

### Community 8 - "steps"
Cohesion: 0.11
Nodes (24): runs-on, steps, runs-on, steps, jobs, accuracy, fuzz, test (+16 more)

### Community 9 - "testing.T"
Cohesion: 0.09
Nodes (54): rgb255(), computeDoc(), find(), TestCascadeOrder(), TestLayoutProperties(), TestMonospaceQuirkAndHeadings(), TestVarsCalcShorthands(), pngHeader() (+46 more)

### Community 10 - "Image"
Cohesion: 0.21
Nodes (6): Image, num(), op(), pathOps(), rasterize(), rawImage

### Community 11 - "Format"
Cohesion: 0.17
Nodes (5): extOf(), Format, normalizeExt(), New(), Converter

### Community 12 - "markdown.go"
Cohesion: 0.14
Nodes (6): addClass(), markTaskLists(), New(), Theme(), main(), Converter

### Community 13 - "svg.go"
Cohesion: 0.05
Nodes (55): ElementStyle, ancestorTable(), attr(), htmlColor(), htmlLength(), PresentationalHints(), Initial(), applyFirstLetter() (+47 more)

### Community 14 - "pixdiff/main.go"
Cohesion: 0.32
Nodes (7): check(), compare(), compareNear(), inked(), inkMask(), main(), raster()

### Community 15 - "chromium.cjs"
Cohesion: 0.29
Nodes (3): { chromium }, path, { pathToFileURL }

### Community 16 - "Proteus sample"
Cohesion: 0.50
Nodes (3): Lists, Proteus sample, Table

### Community 19 - ".layoutContents"
Cohesion: 0.21
Nodes (4): isBFC(), columnGeometry(), contentsResult, margins

### Community 20 - ".layoutAtomic"
Cohesion: 0.11
Nodes (18): absBox, absPart(), clampWidth(), legacyAlign(), blockResult, edges, isForcedBreak(), translateItems() (+10 more)

### Community 21 - "Convert"
Cohesion: 0.50
Nodes (3): Lookup(), Register(), TestRegistry()

### Community 22 - "Render"
Cohesion: 0.25
Nodes (8): ConverterFunc, Recover(), Cm(), NewConfig(), TestRecoverTurnsPanicIntoError(), Convert(), ConvertBytes(), ConvertFile()

### Community 23 - "ConvertBytes"
Cohesion: 0.17
Nodes (16): BenchmarkSystemScan(), fuzzConvert(), FuzzConvertCSS(), FuzzConvertHTML(), FuzzConvertMarkdown(), fuzzSeeds(), BenchmarkConvert(), convertFixture() (+8 more)

### Community 24 - "Box"
Cohesion: 0.28
Nodes (11): Box, allCollapsibleSpace(), anonymousBlock(), fixTable(), fixTree(), hasBlockChild(), nextInFlowIsInline(), normalizeBlockChildren() (+3 more)

### Community 25 - "Render"
Cohesion: 0.13
Nodes (23): ResourceLoader, ResolveURL(), cached, cachingLoader, imageLoader, decodeImage(), newImageLoader(), Input (+15 more)

### Community 27 - ".emit"
Cohesion: 0.15
Nodes (10): Pt(), boxAreas(), cornerRadii(), normCDF(), pickArea(), tilePositions(), ImageItem, Clip (+2 more)

### Community 28 - ".layoutTable"
Cohesion: 0.26
Nodes (8): rowCell, columnPercents(), distributePercent(), rowCells(), tableGrid(), tableSpacing(), tcell, trow

### Community 31 - "pageRules"
Cohesion: 0.40
Nodes (5): Margins, PageSize, UniformMargins(), pageRules(), pageSize()

### Community 32 - "Converter"
Cohesion: 0.17
Nodes (5): Converter, Registry, NewRegistry(), Watermark, Default()

### Community 33 - ".Paginate"
Cohesion: 0.24
Nodes (7): finite(), flattenItems(), Page, pageClip(), spanPages(), cloneItem(), copyItems()

### Community 34 - "proteus_test.go"
Cohesion: 0.19
Nodes (3): fail(), main(), parsePage()

### Community 35 - "ResourceLoader"
Cohesion: 0.22
Nodes (7): LoaderFunc, Config, Metadata, decodeDataURL(), defaultLoad(), DefaultLoader(), readLimited()

### Community 36 - "collapseBorders"
Cohesion: 0.38
Nodes (6): cborder, collapseBorders(), resolve(), sideBorder(), TestCollapsedBorderConflicts(), toPx()

### Community 37 - "textdiff/main.go"
Cohesion: 0.50
Nodes (6): align(), check(), lcs(), main(), words(), word

### Community 39 - "Quarterly Engineering Report"
Cohesion: 0.29
Nodes (6): Appendix, Code, Follow-ups, Metrics, Quarterly Engineering Report, Summary

### Community 40 - ".placeFloat"
Cohesion: 0.19
Nodes (3): atomicBox, floatBox, floatCtx

### Community 41 - "Stamp"
Cohesion: 0.20
Nodes (6): WatermarkLayer, WatermarkPages, WatermarkPlacement, Document, StampRun, Stamp

### Community 42 - "Color"
Cohesion: 0.15
Nodes (14): clamp01(), colorArgs(), Color, hslToRGB(), parseAlpha(), parseColorFunc(), parseComponent(), parseHex() (+6 more)

### Community 44 - "Track"
Cohesion: 0.31
Nodes (9): Track, TrackList, autoTrack(), expandTracks(), fixedTrackSize(), sizeTracks(), trackPositions(), visibleTracks() (+1 more)

## Knowledge Gaps
- **2 isolated node(s):** `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 123 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **11 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Box` connect `Box` to `GPDF`, `frag`, `collapseBorders`, `Style`, `Decoration`, `parser`, `.placeFloat`, `Color`, `Image`, `svg.go`, `.layoutContents`, `.layoutAtomic`, `.emit`, `.layoutTable`?**
  _High betweenness centrality (0.105) - this node is a cross-community bridge._
- **What connects `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter` to the rest of the system?**
  _2 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `options.go` be split into smaller, more focused modules?**
  _Cohesion score 0.10252100840336134 - nodes in this community are weakly interconnected._
- **Why does `Style` connect `Style` to `GPDF`, `frag`, `ParseLength`, `collapseBorders`, `.computeStyle`, `Decoration`, `Color`, `Track`, `svg.go`, `.layoutContents`, `.layoutAtomic`, `Box`, `.emit`, `.layoutTable`?**
  _High betweenness centrality (0.094) - this node is a cross-community bridge._
- **Should `frag` be split into smaller, more focused modules?**
  _Cohesion score 0.12012012012012012 - nodes in this community are weakly interconnected._
- **Why does `Painter` connect `GPDF` to `Converter`, `options.go`, `Resolver`, `Style`, `parser`, `Stamp`, `Color`, `Format`, `Image`, `svg.go`, `Box`, `Paint`, `rgb`?**
  _High betweenness centrality (0.077) - this node is a cross-community bridge._
- **Should `Resolver` be split into smaller, more focused modules?**
  _Cohesion score 0.05879692446856626 - nodes in this community are weakly interconnected._