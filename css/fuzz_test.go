// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package css

import "testing"

func FuzzParse(f *testing.F) {
	for _, s := range []string{
		UserAgentCSS,
		`a{color:red!important;margin:calc(1px + 2em) var(--x, 3px)}@media print{b{x:y}}`,
		`@import url("a.css") print;@font-face{src:url(x.ttf)}@page{size:A4;margin:1cm}`,
		`:root{--a:var(--b);--b:var(--a)}p{width:var(--a)}`,
		`div{grid-template-areas:"a b" "c";grid-template-columns:repeat(auto-fill,minmax(10px,1fr))}`,
		`/* unterminated`, `a{b:"unterminated`, `}}}{{{`, `@media (min-width:0){@supports(x){a{}}}`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		ss := Parse(src, func(string) bool { return true })
		ParseDeclarations(src)
		for _, r := range ss.Rules {
			for _, d := range r.Decls {
				for _, tok := range SplitValue(d.Value) {
					ParseLength(tok, LengthContext{FontSize: 12, RootFontSize: 12, ViewportW: 500, ViewportH: 700}, true)
					ParseColor(tok, Color{A: 1})
					Function(tok)
					URLValue(tok)
				}
				SplitCommas(d.Value)
			}
		}
	})
}
