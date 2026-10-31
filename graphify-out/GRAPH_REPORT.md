# Graph Report - proteus-converter  (2026-10-31)

## Corpus Check
- 77 files · ~72,343 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 3 file(s) not represented in the graph (top: (none) 1, .conf 1, .css 1)

## Summary
- 902 nodes · 3167 edges · 41 communities (35 shown, 6 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 302 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `0e50754e`
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
- proteus.go
- Clip
- Format
- markdown.go
- .element
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
- Seg
- .placeFloat
- layouter
- .layoutTable
- Paint
- Config
- Matrix
- Converter
- .Paginate
- Image
- textdiff/main.go
- Quarterly Engineering Report
- Color
- accuracy_and_blink_quirks.md
- conventions_and_gaps.md
- project_overview.md

## God Nodes (most connected - your core abstractions)
1. `Box` - 99 edges
2. `Style` - 87 edges
3. `layouter` - 76 edges
4. `GPDF` - 72 edges
5. `Painter` - 63 edges
6. `Length` - 47 edges
7. `parser` - 47 edges
8. `Cascade` - 46 edges
9. `Engine` - 45 edges
10. `Resolver` - 42 edges

## Surprising Connections (you probably didn't know these)
- `Design` --references--> `Converter`  [INFERRED]
  README.md → core/converter.go
- `Robustness and resource use` --references--> `WithMaxPages()`  [INFERRED]
  README.md → core/options.go
- `Design` --references--> `Registry`  [INFERRED]
  README.md → core/registry.go
- `Robustness and resource use` --references--> `FuzzParse()`  [INFERRED]
  README.md → css/fuzz_test.go
- `CSS support` --references--> `align()`  [INFERRED]
  README.md → internal/tools/textdiff/main.go

## Import Cycles
- 2-file cycle: `fuzz_test.go -> leak_test.go -> fuzz_test.go`
- 2-file cycle: `hardening_test.go -> leak_test.go -> hardening_test.go`
- 2-file cycle: `leak_test.go -> proteus_test.go -> leak_test.go`
- 2-file cycle: `fuzz_test.go -> hardening_test.go -> fuzz_test.go`
- 2-file cycle: `hardening_test.go -> proteus_test.go -> hardening_test.go`
- 2-file cycle: `fuzz_test.go -> proteus_test.go -> fuzz_test.go`
- 2-file cycle: `layout/hardening_test.go -> layout/layout_test.go -> layout/hardening_test.go`
- 3-file cycle: `fuzz_test.go -> hardening_test.go -> leak_test.go -> fuzz_test.go`
- 3-file cycle: `fuzz_test.go -> proteus_test.go -> leak_test.go -> fuzz_test.go`
- 3-file cycle: `fuzz_test.go -> leak_test.go -> hardening_test.go -> fuzz_test.go`
- 3-file cycle: `hardening_test.go -> proteus_test.go -> leak_test.go -> hardening_test.go`
- 3-file cycle: `fuzz_test.go -> leak_test.go -> proteus_test.go -> fuzz_test.go`
- 3-file cycle: `hardening_test.go -> leak_test.go -> proteus_test.go -> hardening_test.go`
- 3-file cycle: `fuzz_test.go -> proteus_test.go -> hardening_test.go -> fuzz_test.go`
- 3-file cycle: `fuzz_test.go -> hardening_test.go -> proteus_test.go -> fuzz_test.go`
- 4-file cycle: `fuzz_test.go -> hardening_test.go -> proteus_test.go -> leak_test.go -> fuzz_test.go`
- 4-file cycle: `fuzz_test.go -> proteus_test.go -> hardening_test.go -> leak_test.go -> fuzz_test.go`
- 4-file cycle: `fuzz_test.go -> proteus_test.go -> leak_test.go -> hardening_test.go -> fuzz_test.go`
- 4-file cycle: `fuzz_test.go -> leak_test.go -> hardening_test.go -> proteus_test.go -> fuzz_test.go`
- 4-file cycle: `fuzz_test.go -> hardening_test.go -> leak_test.go -> proteus_test.go -> fuzz_test.go`

## Communities (41 total, 6 thin omitted)

### Community 0 - "GPDF"
Cohesion: 0.16
Nodes (15): BackgroundSize, Cascade, Length, init(), sideNames(), AnchorItem, BookmarkItem, alpha() (+7 more)

### Community 1 - "options.go"
Cohesion: 0.07
Nodes (41): FontSource, Option, Cm(), Config, Margins, Metadata, PageSize, NewConfig() (+33 more)

### Community 2 - "frag"
Cohesion: 0.07
Nodes (25): LineHeight, Decoration, Style, fragIntrinsic(), relativeOffset(), textContent(), ctxExtent, Decoration (+17 more)

### Community 3 - "Resolver"
Cohesion: 0.07
Nodes (26): chainKey, Face, TestStandardFallback(), TestSystemChain(), Metrics, classOf(), Resolver, Run (+18 more)

### Community 4 - "ParseLength"
Cohesion: 0.06
Nodes (66): calcParser, calcVal, ParseColor(), parseHex(), computeCtx, FuzzParse(), expandLayoutShorthand(), GridLine (+58 more)

### Community 5 - "testing.T"
Cohesion: 0.11
Nodes (35): isIgnorableSelector(), NewCascade(), selectorKey(), rgb255(), computeDoc(), find(), TestCascadeOrder(), TestLayoutProperties() (+27 more)

### Community 6 - "golang.org/x/net/html.Node"
Cohesion: 0.09
Nodes (30): ResolveURL(), blockify(), ElementStyle, sortDecls(), substituteVars(), ancestorTable(), attr(), htmlColor() (+22 more)

### Community 7 - "parser"
Cohesion: 0.12
Nodes (25): AtPage, rankOf(), compiledRule, FontFace, Import, MediaContext, splitAnd(), MediaMatcher (+17 more)

### Community 8 - "steps"
Cohesion: 0.11
Nodes (24): runs-on, steps, runs-on, steps, jobs, accuracy, fuzz, test (+16 more)

### Community 9 - "proteus.go"
Cohesion: 0.16
Nodes (15): fail(), main(), parsePage(), pngHeader(), TestCancelledContext(), TestHugeImageIsRefusedWithWarning(), TestRunawayDocumentHitsPageLimit(), ConvertString() (+7 more)

### Community 10 - "Clip"
Cohesion: 0.17
Nodes (9): boxAreas(), pickArea(), Clip, PathStyle, pageClip(), PathItem, num(), op() (+1 more)

### Community 11 - "Format"
Cohesion: 0.16
Nodes (10): extOf(), Format, normalizeExt(), Registry, NewRegistry(), Default(), FormatForPath(), Lookup() (+2 more)

### Community 12 - "markdown.go"
Cohesion: 0.11
Nodes (7): InternalError, addClass(), markTaskLists(), New(), Theme(), main(), Converter

### Community 13 - ".element"
Cohesion: 0.25
Nodes (10): paint, parser, state, attr(), defaultState(), FromNode(), isAncestor(), length() (+2 more)

### Community 14 - "pixdiff/main.go"
Cohesion: 0.23
Nodes (7): check(), compare(), compareNear(), inked(), inkMask(), main(), raster()

### Community 15 - "chromium.cjs"
Cohesion: 0.29
Nodes (3): { chromium }, path, { pathToFileURL }

### Community 16 - "Proteus sample"
Cohesion: 0.50
Nodes (3): Lists, Proteus sample, Table

### Community 19 - "layouter"
Cohesion: 0.25
Nodes (6): isBFC(), blockResult, contentsResult, isForcedBreak(), margins, distributePercent()

### Community 20 - ".layoutAtomic"
Cohesion: 0.14
Nodes (13): absPart(), clampWidth(), edges, resolveAlign(), resolveFlexible(), flexItem, flexLine, alignOffset() (+5 more)

### Community 21 - ".absItems"
Cohesion: 0.27
Nodes (4): absBox, translateItems(), groupItem, posCtx

### Community 22 - "Render"
Cohesion: 0.22
Nodes (11): ConverterFunc, Recover(), TestRecoverTurnsPanicIntoError(), newImageLoader(), Input, mergeMeta(), Render(), Convert() (+3 more)

### Community 23 - "ConvertBytes"
Cohesion: 0.18
Nodes (16): BenchmarkSystemScan(), fuzzConvert(), FuzzConvertCSS(), FuzzConvertHTML(), FuzzConvertMarkdown(), fuzzSeeds(), BenchmarkConvert(), convertFixture() (+8 more)

### Community 24 - "Box"
Cohesion: 0.28
Nodes (10): Box, allCollapsibleSpace(), anonymousBlock(), fixTable(), fixTree(), hasBlockChild(), nextInFlowIsInline(), normalizeBlockChildren() (+2 more)

### Community 25 - "Seg"
Cohesion: 0.24
Nodes (10): arc(), ellipse(), ParsePath(), parsePath(), quad(), RoundRect(), roundRect(), scanner (+2 more)

### Community 26 - ".placeFloat"
Cohesion: 0.16
Nodes (4): atomicBox, floatBox, floatCtx, ImageItem

### Community 27 - "layouter"
Cohesion: 0.18
Nodes (3): tilePositions(), layouter, pendingMarker

### Community 28 - ".layoutTable"
Cohesion: 0.24
Nodes (7): rowCell, columnPercents(), rowCells(), tableGrid(), tableSpacing(), tcell, trow

### Community 30 - "Config"
Cohesion: 0.25
Nodes (5): LoaderFunc, decodeDataURL(), defaultLoad(), DefaultLoader(), readLimited()

### Community 31 - "Matrix"
Cohesion: 0.25
Nodes (7): parseTransform(), Item, Shape, Matrix, ScaleXY(), Translate(), Text

### Community 32 - "Converter"
Cohesion: 0.28
Nodes (3): Converter, New(), Converter

### Community 33 - ".Paginate"
Cohesion: 0.28
Nodes (6): Page, pathExtent(), spanPages(), cloneItem(), copyItems(), Document

### Community 35 - "Image"
Cohesion: 0.18
Nodes (8): Image, imageLoader, decodeImage(), rasterize(), rawImage, Document, Parse(), TestParseShapes()

### Community 37 - "textdiff/main.go"
Cohesion: 0.50
Nodes (6): align(), check(), lcs(), main(), words(), word

### Community 39 - "Quarterly Engineering Report"
Cohesion: 0.29
Nodes (6): Appendix, Code, Follow-ups, Metrics, Quarterly Engineering Report, Summary

### Community 42 - "Color"
Cohesion: 0.15
Nodes (13): clamp01(), colorArgs(), Color, hslToRGB(), parseAlpha(), parseColorFunc(), parseComponent(), colorAt() (+5 more)

## Knowledge Gaps
- **2 isolated node(s):** `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 115 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **6 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Box` connect `Box` to `GPDF`, `frag`, `Image`, `testing.T`, `golang.org/x/net/html.Node`, `parser`, `Color`, `layouter`, `.layoutAtomic`, `.absItems`, `.placeFloat`, `layouter`, `.layoutTable`?**
  _High betweenness centrality (0.129) - this node is a cross-community bridge._
- **What connects `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter` to the rest of the system?**
  _2 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `options.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07372549019607844 - nodes in this community are weakly interconnected._
- **Why does `Style` connect `frag` to `GPDF`, `ParseLength`, `testing.T`, `golang.org/x/net/html.Node`, `Color`, `Clip`, `layouter`, `.layoutAtomic`, `Box`, `layouter`, `.layoutTable`?**
  _High betweenness centrality (0.089) - this node is a cross-community bridge._
- **Should `frag` be split into smaller, more focused modules?**
  _Cohesion score 0.07450980392156863 - nodes in this community are weakly interconnected._
- **Why does `GPDF` connect `GPDF` to `Converter`, `options.go`, `frag`, `Resolver`, `Image`, `parser`, `Color`, `Clip`, `markdown.go`, `Render`, `Box`?**
  _High betweenness centrality (0.067) - this node is a cross-community bridge._
- **Should `Resolver` be split into smaller, more focused modules?**
  _Cohesion score 0.06734006734006734 - nodes in this community are weakly interconnected._