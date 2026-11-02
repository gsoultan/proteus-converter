# Graph Report - proteus-converter  (2026-11-02)

## Corpus Check
- 82 files · ~87,958 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 10 file(s) not represented in the graph (top: (none) 2, .ttf 2, .otf 2)

## Summary
- 971 nodes · 3501 edges · 45 communities (35 shown, 10 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 364 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `aa68927e`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- GPDF
- options.go
- frag
- Resolver
- ParseLength
- Style
- golang.org/x/net/html.Node
- parser
- steps
- hardening_test.go
- proteus-converter
- Format
- markdown.go
- svg.go
- pixdiff/main.go
- chromium.cjs
- Proteus sample
- run.sh
- github.com/gsoultan/proteus-converter
- .layoutBlockInner
- .layoutAtomic
- Convert
- Render
- ConvertBytes
- Box
- collectSheets
- TextItem
- .emit
- .layoutTable
- Paint
- rgb
- styles.go
- Converter
- .Paginate
- proteus/main.go
- ResourceLoader
- collapseBorders
- textdiff/main.go
- Decoration
- Quarterly Engineering Report
- Color
- accuracy_and_blink_quirks.md
- conventions_and_gaps.md
- project_overview.md

## God Nodes (most connected - your core abstractions)
1. `Style` - 106 edges
2. `Box` - 104 edges
3. `layouter` - 83 edges
4. `GPDF` - 76 edges
5. `Painter` - 68 edges
6. `Engine` - 51 edges
7. `Length` - 48 edges
8. `parser` - 48 edges
9. `Cascade` - 47 edges
10. `Resolver` - 42 edges

## Surprising Connections (you probably didn't know these)
- `Design` --references--> `Converter`  [INFERRED]
  README.md → core/converter.go
- `Security` --references--> `WithMaxResourceSize()`  [INFERRED]
  README.md → core/options.go
- `Robustness and resource use` --references--> `WithMaxPages()`  [INFERRED]
  README.md → core/options.go
- `CSS support` --references--> `WithWarningHandler()`  [INFERRED]
  README.md → core/options.go
- `Design` --references--> `Registry`  [INFERRED]
  README.md → core/registry.go

## Import Cycles
- 2-file cycle: `fuzz_test.go -> leak_test.go -> fuzz_test.go`
- 2-file cycle: `hardening_test.go -> leak_test.go -> hardening_test.go`
- 2-file cycle: `leak_test.go -> proteus_test.go -> leak_test.go`
- 2-file cycle: `fuzz_test.go -> hardening_test.go -> fuzz_test.go`
- 2-file cycle: `hardening_test.go -> proteus_test.go -> hardening_test.go`
- 2-file cycle: `fuzz_test.go -> proteus_test.go -> fuzz_test.go`
- 2-file cycle: `layout/fidelity_test.go -> layout/hardening_test.go -> layout/fidelity_test.go`
- 2-file cycle: `layout/fidelity_test.go -> layout/layout_test.go -> layout/fidelity_test.go`
- 2-file cycle: `layout/hardening_test.go -> layout/layout_test.go -> layout/hardening_test.go`
- 3-file cycle: `fuzz_test.go -> hardening_test.go -> leak_test.go -> fuzz_test.go`
- 3-file cycle: `fuzz_test.go -> proteus_test.go -> leak_test.go -> fuzz_test.go`
- 3-file cycle: `fuzz_test.go -> leak_test.go -> hardening_test.go -> fuzz_test.go`
- 3-file cycle: `hardening_test.go -> proteus_test.go -> leak_test.go -> hardening_test.go`
- 3-file cycle: `fuzz_test.go -> leak_test.go -> proteus_test.go -> fuzz_test.go`
- 3-file cycle: `hardening_test.go -> leak_test.go -> proteus_test.go -> hardening_test.go`
- 3-file cycle: `fuzz_test.go -> proteus_test.go -> hardening_test.go -> fuzz_test.go`
- 3-file cycle: `fuzz_test.go -> hardening_test.go -> proteus_test.go -> fuzz_test.go`
- 3-file cycle: `layout/fidelity_test.go -> layout/hardening_test.go -> layout/layout_test.go -> layout/fidelity_test.go`
- 3-file cycle: `layout/fidelity_test.go -> layout/layout_test.go -> layout/hardening_test.go -> layout/fidelity_test.go`
- 4-file cycle: `fuzz_test.go -> hardening_test.go -> proteus_test.go -> leak_test.go -> fuzz_test.go`

## Communities (45 total, 10 thin omitted)

### Community 0 - "GPDF"
Cohesion: 0.10
Nodes (46): Cascade, Length, init(), sideNames(), TestStandardFallback(), TestSystemChain(), AnchorItem, BookmarkItem (+38 more)

### Community 1 - "options.go"
Cohesion: 0.12
Nodes (27): FontSource, Option, Cm(), Margins, NewConfig(), UniformMargins(), WithBaseURL(), WithFont() (+19 more)

### Community 2 - "frag"
Cohesion: 0.11
Nodes (20): fragIntrinsic(), atomicBox, ctxExtent, frag, fragBuilder, fragKind, firstUnplacedFloat(), hasContent() (+12 more)

### Community 3 - "Resolver"
Cohesion: 0.06
Nodes (34): chainKey, Face, FuzzDecodeWOFF2(), TestDecodeWOFF2(), Metrics, classOf(), Resolver, Run (+26 more)

### Community 4 - "ParseLength"
Cohesion: 0.06
Nodes (70): calcParser, calcVal, ParseColor(), parseHex(), computeCtx, FuzzParse(), expandLayoutShorthand(), GridLine (+62 more)

### Community 5 - "Style"
Cohesion: 0.10
Nodes (12): BackgroundSize, LineHeight, propDef, Style, clampWidth(), legacyAlign(), relativeOffset(), columnGeometry() (+4 more)

### Community 6 - "golang.org/x/net/html.Node"
Cohesion: 0.08
Nodes (28): blockify(), ElementStyle, sortDecls(), substituteVars(), ancestorTable(), attr(), htmlColor(), htmlLength() (+20 more)

### Community 7 - "parser"
Cohesion: 0.09
Nodes (32): AtPage, isIgnorableSelector(), NewCascade(), rankOf(), selectorKey(), rgb255(), compiledRule, computeDoc() (+24 more)

### Community 8 - "steps"
Cohesion: 0.11
Nodes (24): runs-on, steps, runs-on, steps, jobs, accuracy, fuzz, test (+16 more)

### Community 9 - "hardening_test.go"
Cohesion: 0.16
Nodes (13): InternalError, pngHeader(), TestCancelledContext(), TestHugeImageIsRefusedWithWarning(), TestRunawayDocumentHitsPageLimit(), ConvertString(), checkPDF(), pdfText() (+5 more)

### Community 10 - "proteus-converter"
Cohesion: 0.29
Nodes (5): WithFS(), Accuracy gate, CSS support, proteus-converter, Security

### Community 11 - "Format"
Cohesion: 0.27
Nodes (6): extOf(), Format, normalizeExt(), Registry, NewRegistry(), Default()

### Community 12 - "markdown.go"
Cohesion: 0.19
Nodes (3): New(), Theme(), main()

### Community 13 - "svg.go"
Cohesion: 0.06
Nodes (43): Image, PathStyle, pathExtent(), PathItem, decodeImage(), num(), op(), pathOps() (+35 more)

### Community 14 - "pixdiff/main.go"
Cohesion: 0.28
Nodes (7): check(), compare(), compareNear(), inked(), inkMask(), main(), raster()

### Community 15 - "chromium.cjs"
Cohesion: 0.29
Nodes (3): { chromium }, path, { pathToFileURL }

### Community 16 - "Proteus sample"
Cohesion: 0.50
Nodes (3): Lists, Proteus sample, Table

### Community 19 - ".layoutBlockInner"
Cohesion: 0.14
Nodes (5): contentsResult, isForcedBreak(), floatBox, floatCtx, margins

### Community 20 - ".layoutAtomic"
Cohesion: 0.11
Nodes (14): absBox, absPart(), blockResult, translateItems(), resolveAlign(), resolveFlexible(), flexItem, flexLine (+6 more)

### Community 21 - "Convert"
Cohesion: 0.33
Nodes (6): Convert(), ConvertFile(), FormatForPath(), Lookup(), Register(), TestRegistry()

### Community 22 - "Render"
Cohesion: 0.20
Nodes (11): ConverterFunc, Recover(), ResolveURL(), TestDecodeWOFF(), DecodeWOFF(), IsWOFF(), TestRecoverTurnsPanicIntoError(), parsedSheet (+3 more)

### Community 23 - "ConvertBytes"
Cohesion: 0.22
Nodes (15): fuzzConvert(), FuzzConvertCSS(), FuzzConvertHTML(), FuzzConvertMarkdown(), fuzzSeeds(), BenchmarkConvert(), convertFixture(), fixtures() (+7 more)

### Community 24 - "Box"
Cohesion: 0.19
Nodes (13): isBFC(), textContent(), Box, allCollapsibleSpace(), anonymousBlock(), fixTable(), fixTree(), hasBlockChild() (+5 more)

### Community 25 - "collectSheets"
Cohesion: 0.25
Nodes (11): Metadata, attr(), documentMetadata(), findBase(), mergeMeta(), textOf(), walk(), sourceSheet (+3 more)

### Community 27 - ".emit"
Cohesion: 0.13
Nodes (12): Pt(), boxAreas(), cornerRadii(), normCDF(), pickArea(), tilePositions(), ImageItem, m3() (+4 more)

### Community 28 - ".layoutTable"
Cohesion: 0.26
Nodes (8): rowCell, columnPercents(), distributePercent(), rowCells(), tableGrid(), tableSpacing(), tcell, trow

### Community 31 - "styles.go"
Cohesion: 0.33
Nodes (6): PageSize, cached, cachingLoader, newCachingLoader(), pageRules(), pageSize()

### Community 32 - "Converter"
Cohesion: 0.21
Nodes (3): Converter, New(), Converter

### Community 33 - ".Paginate"
Cohesion: 0.24
Nodes (6): Page, pageClip(), spanPages(), cloneItem(), copyItems(), Document

### Community 34 - "proteus/main.go"
Cohesion: 0.38
Nodes (3): fail(), main(), parsePage()

### Community 35 - "ResourceLoader"
Cohesion: 0.21
Nodes (9): LoaderFunc, Config, decodeDataURL(), defaultLoad(), DefaultLoader(), ResourceLoader, readLimited(), imageLoader (+1 more)

### Community 36 - "collapseBorders"
Cohesion: 0.47
Nodes (5): cborder, collapseBorders(), resolve(), sideBorder(), toPx()

### Community 37 - "textdiff/main.go"
Cohesion: 0.42
Nodes (6): align(), check(), lcs(), main(), words(), word

### Community 39 - "Quarterly Engineering Report"
Cohesion: 0.29
Nodes (6): Appendix, Code, Follow-ups, Metrics, Quarterly Engineering Report, Summary

### Community 42 - "Color"
Cohesion: 0.22
Nodes (10): clamp01(), colorArgs(), Color, hslToRGB(), parseAlpha(), parseColorFunc(), parseComponent(), Shadow (+2 more)

## Knowledge Gaps
- **2 isolated node(s):** `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 120 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Box` connect `Box` to `GPDF`, `frag`, `collapseBorders`, `Style`, `golang.org/x/net/html.Node`, `parser`, `Decoration`, `Color`, `svg.go`, `.layoutBlockInner`, `.layoutAtomic`, `.emit`, `.layoutTable`?**
  _High betweenness centrality (0.124) - this node is a cross-community bridge._
- **What connects `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter` to the rest of the system?**
  _2 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `GPDF` be split into smaller, more focused modules?**
  _Cohesion score 0.09807330146313196 - nodes in this community are weakly interconnected._
- **Why does `Style` connect `Style` to `GPDF`, `frag`, `ParseLength`, `collapseBorders`, `Decoration`, `golang.org/x/net/html.Node`, `Color`, `svg.go`, `.layoutBlockInner`, `.layoutAtomic`, `Box`, `.emit`, `.layoutTable`, `styles.go`?**
  _High betweenness centrality (0.102) - this node is a cross-community bridge._
- **Should `options.go` be split into smaller, more focused modules?**
  _Cohesion score 0.12258064516129032 - nodes in this community are weakly interconnected._
- **Why does `GPDF` connect `GPDF` to `Converter`, `options.go`, `Resolver`, `Style`, `parser`, `Color`, `svg.go`, `Box`, `collectSheets`, `rgb`?**
  _High betweenness centrality (0.065) - this node is a cross-community bridge._
- **Should `frag` be split into smaller, more focused modules?**
  _Cohesion score 0.10685249709639953 - nodes in this community are weakly interconnected._