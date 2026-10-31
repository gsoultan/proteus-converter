# Graph Report - gpdf-converter  (2026-10-31)

## Corpus Check
- 55 files · ~39,152 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 3 file(s) not represented in the graph (top: (none) 1, .conf 1, .css 1)

## Summary
- 606 nodes · 1881 edges · 19 communities (17 shown, 2 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 145 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- GPDF
- options.go
- layouter
- Resolver
- Style
- Cascade
- Box
- parser
- steps
- proteus.go
- images.go
- Format
- markdown.go
- proteus_test.go
- pixdiff/main.go
- chromium.cjs
- Proteus sample
- run.sh
- github.com/gsoultan/proteus-converter

## God Nodes (most connected - your core abstractions)
1. `Box` - 69 edges
2. `Style` - 62 edges
3. `GPDF` - 58 edges
4. `Painter` - 49 edges
5. `layouter` - 41 edges
6. `parser` - 40 edges
7. `Cascade` - 39 edges
8. `Resolver` - 38 edges
9. `Engine` - 37 edges
10. `Length` - 36 edges

## Surprising Connections (you probably didn't know these)
- `Design` --references--> `Converter`  [INFERRED]
  README.md → core/converter.go
- `Design` --references--> `Registry`  [INFERRED]
  README.md → core/registry.go
- `Design` --references--> `GPDF`  [INFERRED]
  README.md → render/gpdf.go
- `Design` --references--> `Painter`  [INFERRED]
  README.md → render/painter.go
- `Design` --references--> `Option`  [INFERRED]
  README.md → core/options.go

## Import Cycles
- None detected.

## Pending regeneration (manual note, 2026-10-31)
graphify was not available when floats, positioning, flexbox, grid, SVG and images landed, so
graph.json and the communities above predate them. Run `graphify update .` to refresh. New nodes
to expect:
- package `svg` (svg.go, geometry.go): Document, Shape, Seg, Matrix, Parse, FromNode, ParsePath
- layout/float.go: floatCtx, floatBox, posCtx, absBox, placeFloat, layoutAbsolute, absItems
- layout/flex.go: layoutFlex, flexItem, resolveFlexible, flexMainPositions
- layout/grid.go: layoutGrid, gridItem, gridTrack, sizeTracks, resolveLines, gridIntrinsic
- layout/background.go, layout/images.go: emitBackground, emitGradient, emitReplaced, emitSVG
- layout/items.go: PathItem, PathStyle, Clip; render/gpdf_ops.go: GPDF.Path, GPDF.Image
- css/layoutprops.go: TrackList, GridLine, BackgroundSize, expandLayoutShorthand
Expected edges: layouter -> floatCtx/posCtx/layoutFlex/layoutGrid; Painter -> svg.Matrix;
Box -> svg.Document (Image.SVG) and BgImage.

## Communities (19 total, 2 thin omitted)

### Community 0 - "GPDF"
Cohesion: 0.06
Nodes (35): clamp01(), colorArgs(), Color, hslToRGB(), parseAlpha(), parseColorFunc(), parseComponent(), Length (+27 more)

### Community 1 - "options.go"
Cohesion: 0.05
Nodes (66): ConverterFunc, FontSource, LoaderFunc, Option, Cm(), Config, Margins, Metadata (+58 more)

### Community 2 - "layouter"
Cohesion: 0.06
Nodes (34): fragIntrinsic(), atomicBox, textContent(), blockResult, contentsResult, ctxExtent, edges, isForcedBreak() (+26 more)

### Community 3 - "Resolver"
Cohesion: 0.08
Nodes (22): chainKey, Face, TestStandardFallback(), TestSystemChain(), Metrics, classOf(), Resolver, Run (+14 more)

### Community 4 - "Style"
Cohesion: 0.09
Nodes (33): calcParser, calcVal, ParseColor(), parseHex(), computeCtx, htmlColor(), evalCalc(), evalMinMax() (+25 more)

### Community 5 - "Cascade"
Cohesion: 0.08
Nodes (32): Cascade, blockify(), ElementStyle, isIgnorableSelector(), NewCascade(), rankOf(), selectorKey(), sortDecls() (+24 more)

### Community 6 - "Box"
Cohesion: 0.12
Nodes (27): ancestorTable(), attr(), htmlLength(), PresentationalHints(), addClass(), markTaskLists(), isBFC(), Box (+19 more)

### Community 7 - "parser"
Cohesion: 0.24
Nodes (16): AtPage, Declaration, FontFace, Import, MediaMatcher, parser, Stylesheet, isIdentByte() (+8 more)

### Community 8 - "steps"
Cohesion: 0.11
Nodes (21): runs-on, steps, jobs, accuracy, test, name, on, pull_request (+13 more)

### Community 9 - "proteus.go"
Cohesion: 0.16
Nodes (5): fail(), main(), parsePage(), Lookup(), Register()

### Community 10 - "images.go"
Cohesion: 0.17
Nodes (3): Converter, New(), Converter

### Community 11 - "Format"
Cohesion: 0.20
Nodes (6): extOf(), Format, normalizeExt(), Registry, NewRegistry(), Default()

### Community 12 - "markdown.go"
Cohesion: 0.16
Nodes (4): New(), Theme(), main(), Converter

### Community 13 - "proteus_test.go"
Cohesion: 0.32
Nodes (12): ConvertBytes(), ConvertFile(), ConvertString(), FormatForPath(), checkPDF(), pdfText(), TestContextCancelled(), TestConvertFileHTML() (+4 more)

### Community 14 - "pixdiff/main.go"
Cohesion: 0.22
Nodes (5): check(), compare(), inked(), main(), raster()

### Community 15 - "chromium.cjs"
Cohesion: 0.29
Nodes (3): { chromium }, path, { pathToFileURL }

### Community 16 - "Proteus sample"
Cohesion: 0.50
Nodes (3): Lists, Proteus sample, Table

## Knowledge Gaps
- **2 isolated node(s):** `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter`
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 88 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Box` connect `Box` to `GPDF`, `options.go`, `layouter`, `Style`, `Cascade`, `parser`, `images.go`?**
  _High betweenness centrality (0.119) - this node is a cross-community bridge._
- **What connects `FONTCONFIG_FILE`, `github.com/gsoultan/proteus-converter` to the rest of the system?**
  _2 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `GPDF` be split into smaller, more focused modules?**
  _Cohesion score 0.06173558532323821 - nodes in this community are weakly interconnected._
- **Why does `Style` connect `Style` to `GPDF`, `options.go`, `layouter`, `Cascade`, `Box`?**
  _High betweenness centrality (0.092) - this node is a cross-community bridge._
- **Should `options.go` be split into smaller, more focused modules?**
  _Cohesion score 0.05030643513789581 - nodes in this community are weakly interconnected._
- **Why does `GPDF` connect `GPDF` to `options.go`, `Resolver`, `Style`, `Cascade`, `Box`, `parser`, `proteus.go`, `images.go`, `pixdiff/main.go`?**
  _High betweenness centrality (0.089) - this node is a cross-community bridge._
- **Should `layouter` be split into smaller, more focused modules?**
  _Cohesion score 0.06414565826330532 - nodes in this community are weakly interconnected._