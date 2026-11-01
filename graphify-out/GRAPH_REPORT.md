# Graph Report - proteus-converter  (2026-11-01)

## Corpus Check
- 80 files · ~79,640 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 9 file(s) not represented in the graph (top: .ttf 3, .otf 2, (none) 1)

## Summary
- 934 nodes · 3336 edges · 47 communities (35 shown, 12 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 334 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `6294ca73`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- GPDF
- options.go
- frag
- Resolver
- ParseLength
- testing.T
- golang.org/x/net/html.Node
- parser
- steps
- hardening_test.go
- Resolver
- Format
- markdown.go
- svg.go
- pixdiff/main.go
- chromium.cjs
- Proteus sample
- run.sh
- github.com/gsoultan/proteus-converter
- layouter
- .layoutAtomic
- .absItems
- Render
- ConvertBytes
- Box
- collectSheets
- .placeFloat
- .emit
- .layoutTable
- Paint
- rgb
- Config
- Converter
- .Paginate
- proteus/main.go
- loadFontFaces
- collapseBorders
- textdiff/main.go
- Decoration
- Quarterly Engineering Report
- EllipseItem
- RectItem
- Color
- accuracy_and_blink_quirks.md
- conventions_and_gaps.md
- project_overview.md

## God Nodes (most connected - your core abstractions)
1. `Box` - 102 edges
2. `Style` - 99 edges
3. `layouter` - 80 edges
4. `GPDF` - 74 edges
5. `Painter` - 66 edges
6. `Engine` - 49 edges
7. `Length` - 47 edges
8. `parser` - 47 edges
9. `Cascade` - 46 edges
10. `Resolver` - 42 edges

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

## Communities (47 total, 12 thin omitted)

### Community 0 - "GPDF"
Cohesion: 0.16
Nodes (14): Cascade, Length, init(), sideNames(), AnchorItem, BookmarkItem, alpha(), boolInt() (+6 more)

### Community 1 - "options.go"
Cohesion: 0.10
Nodes (28): FontSource, Option, WithBaseURL(), WithFont(), WithFontDir(), WithFS(), WithHTTP(), WithLocalFiles() (+20 more)

### Community 2 - "frag"
Cohesion: 0.12
Nodes (16): fragIntrinsic(), atomicBox, frag, fragKind, m3(), firstUnplacedFloat(), hasContent(), hasFloatFrags() (+8 more)

### Community 3 - "Resolver"
Cohesion: 0.12
Nodes (14): Face, Metrics, Run, collectionOffsets(), collectionOffsetsAt(), parseNames(), parseSfntInfo(), parseSfntInfoAt() (+6 more)

### Community 4 - "ParseLength"
Cohesion: 0.06
Nodes (74): BackgroundSize, calcParser, calcVal, ParseColor(), parseHex(), computeCtx, FuzzParse(), expandLayoutShorthand() (+66 more)

### Community 5 - "testing.T"
Cohesion: 0.12
Nodes (36): rgb255(), computeDoc(), find(), TestCascadeOrder(), TestLayoutProperties(), TestMonospaceQuirkAndHeadings(), TestVarsCalcShorthands(), TestDecodeWOFF() (+28 more)

### Community 6 - "golang.org/x/net/html.Node"
Cohesion: 0.06
Nodes (37): blockify(), ElementStyle, sortDecls(), substituteVars(), ancestorTable(), attr(), htmlColor(), htmlLength() (+29 more)

### Community 7 - "parser"
Cohesion: 0.11
Nodes (26): AtPage, isIgnorableSelector(), NewCascade(), rankOf(), selectorKey(), compiledRule, FontFace, Import (+18 more)

### Community 8 - "steps"
Cohesion: 0.11
Nodes (24): runs-on, steps, runs-on, steps, jobs, accuracy, fuzz, test (+16 more)

### Community 9 - "hardening_test.go"
Cohesion: 0.25
Nodes (12): pngHeader(), TestCancelledContext(), TestHugeImageIsRefusedWithWarning(), TestRunawayDocumentHitsPageLimit(), ConvertString(), checkPDF(), pdfText(), TestContextCancelled() (+4 more)

### Community 10 - "Resolver"
Cohesion: 0.16
Nodes (10): chainKey, TestSystemChain(), classOf(), Resolver, matchFace(), System(), systemFontDirs(), weightScore() (+2 more)

### Community 11 - "Format"
Cohesion: 0.17
Nodes (13): extOf(), Format, normalizeExt(), Registry, NewRegistry(), Convert(), ConvertBytes(), ConvertFile() (+5 more)

### Community 12 - "markdown.go"
Cohesion: 0.10
Nodes (5): InternalError, New(), Theme(), main(), Converter

### Community 13 - "svg.go"
Cohesion: 0.07
Nodes (37): PathStyle, pathExtent(), PathItem, num(), op(), pathOps(), arc(), ellipse() (+29 more)

### Community 14 - "pixdiff/main.go"
Cohesion: 0.28
Nodes (7): check(), compare(), compareNear(), inked(), inkMask(), main(), raster()

### Community 15 - "chromium.cjs"
Cohesion: 0.29
Nodes (3): { chromium }, path, { pathToFileURL }

### Community 16 - "Proteus sample"
Cohesion: 0.50
Nodes (3): Lists, Proteus sample, Table

### Community 19 - "layouter"
Cohesion: 0.21
Nodes (7): Pt(), isBFC(), blockResult, contentsResult, isForcedBreak(), margins, distributePercent()

### Community 20 - ".layoutAtomic"
Cohesion: 0.18
Nodes (10): absPart(), clampWidth(), legacyAlign(), edges, resolveAlign(), resolveFlexible(), flexItem, flexLine (+2 more)

### Community 21 - ".absItems"
Cohesion: 0.29
Nodes (4): absBox, translateItems(), groupItem, posCtx

### Community 22 - "Render"
Cohesion: 0.18
Nodes (11): ConverterFunc, Recover(), LoaderFunc, Cm(), NewConfig(), decodeDataURL(), defaultLoad(), DefaultLoader() (+3 more)

### Community 23 - "ConvertBytes"
Cohesion: 0.24
Nodes (14): fuzzConvert(), FuzzConvertCSS(), FuzzConvertHTML(), FuzzConvertMarkdown(), fuzzSeeds(), BenchmarkConvert(), convertFixture(), fixtures() (+6 more)

### Community 24 - "Box"
Cohesion: 0.24
Nodes (12): textContent(), Box, allCollapsibleSpace(), anonymousBlock(), fixTable(), fixTree(), hasBlockChild(), nextInFlowIsInline() (+4 more)

### Community 25 - "collectSheets"
Cohesion: 0.25
Nodes (11): Metadata, attr(), documentMetadata(), findBase(), mergeMeta(), textOf(), walk(), sourceSheet (+3 more)

### Community 27 - ".emit"
Cohesion: 0.18
Nodes (7): boxAreas(), cornerRadii(), normCDF(), pickArea(), tilePositions(), ImageItem, Clip

### Community 28 - ".layoutTable"
Cohesion: 0.24
Nodes (7): rowCell, columnPercents(), rowCells(), tableGrid(), tableSpacing(), tcell, trow

### Community 31 - "Config"
Cohesion: 0.32
Nodes (6): Config, Margins, PageSize, UniformMargins(), pageRules(), pageSize()

### Community 32 - "Converter"
Cohesion: 0.18
Nodes (3): Converter, New(), Converter

### Community 33 - ".Paginate"
Cohesion: 0.24
Nodes (6): Page, pageClip(), spanPages(), cloneItem(), copyItems(), Document

### Community 34 - "proteus/main.go"
Cohesion: 0.38
Nodes (3): fail(), main(), parsePage()

### Community 35 - "loadFontFaces"
Cohesion: 0.14
Nodes (15): ResourceLoader, ResolveURL(), IsWOFF2(), Image, cached, cachingLoader, imageLoader, decodeImage() (+7 more)

### Community 36 - "collapseBorders"
Cohesion: 0.38
Nodes (6): cborder, collapseBorders(), resolve(), sideBorder(), TestCollapsedBorderConflicts(), toPx()

### Community 37 - "textdiff/main.go"
Cohesion: 0.31
Nodes (6): align(), check(), lcs(), main(), words(), word

### Community 39 - "Quarterly Engineering Report"
Cohesion: 0.29
Nodes (6): Appendix, Code, Follow-ups, Metrics, Quarterly Engineering Report, Summary

### Community 42 - "Color"
Cohesion: 0.22
Nodes (10): clamp01(), colorArgs(), Color, hslToRGB(), parseAlpha(), parseColorFunc(), parseComponent(), Shadow (+2 more)

## Knowledge Gaps
- **2 isolated node(s):** `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 117 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **12 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Box` connect `Box` to `GPDF`, `frag`, `loadFontFaces`, `ParseLength`, `collapseBorders`, `golang.org/x/net/html.Node`, `parser`, `Decoration`, `Color`, `svg.go`, `layouter`, `.layoutAtomic`, `.absItems`, `.placeFloat`, `.emit`, `.layoutTable`?**
  _High betweenness centrality (0.125) - this node is a cross-community bridge._
- **What connects `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter` to the rest of the system?**
  _2 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `options.go` be split into smaller, more focused modules?**
  _Cohesion score 0.10252100840336134 - nodes in this community are weakly interconnected._
- **Why does `Style` connect `golang.org/x/net/html.Node` to `GPDF`, `frag`, `ParseLength`, `collapseBorders`, `Decoration`, `Color`, `svg.go`, `layouter`, `.layoutAtomic`, `Box`, `.emit`, `.layoutTable`?**
  _High betweenness centrality (0.098) - this node is a cross-community bridge._
- **Should `frag` be split into smaller, more focused modules?**
  _Cohesion score 0.11746031746031746 - nodes in this community are weakly interconnected._
- **Why does `GPDF` connect `GPDF` to `Converter`, `options.go`, `Resolver`, `ParseLength`, `loadFontFaces`, `golang.org/x/net/html.Node`, `parser`, `Color`, `Resolver`, `markdown.go`, `svg.go`, `Box`, `collectSheets`, `rgb`?**
  _High betweenness centrality (0.065) - this node is a cross-community bridge._
- **Should `Resolver` be split into smaller, more focused modules?**
  _Cohesion score 0.11895161290322581 - nodes in this community are weakly interconnected._