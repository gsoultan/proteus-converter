# proteus-converter overview

Pure-Go HTML and Markdown to PDF converter (module github.com/gsoultan/proteus-converter,
repo formerly gpdf-converter). PDF backend: github.com/gsoultan/gpdf. DOCX input is planned.
A future proteus-server will expose conversion over the network; it is not in this repo.

Pipeline: parse (x/net/html, goldmark for Markdown) -> collect stylesheets (pipeline/styles.go)
-> cascade (css/cascade.go) -> box tree (layout/build.go, fixup.go) -> layout (layout/*.go)
-> paginate (layout/paginate.go) -> paint through render.Painter (render/gpdf.go).

Public API lives in proteus.go (aliases of core): Convert, ConvertBytes, ConvertString,
ConvertFile, Register, Lookup, FormatForPath, With* options. Patterns: Converter strategy,
Registry factory, functional options, Painter adapter.

Toolchain: go 1.26 (GOTOOLCHAIN=auto). Deps: gpdf, goldmark, x/net, cascadia.
