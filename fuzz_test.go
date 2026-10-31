package proteus_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	proteus "github.com/gsoultan/proteus-converter"
)

// fuzzSeeds adds every fixture plus small hand-picked edge cases.
func fuzzSeeds(f *testing.F, glob string, extra ...string) {
	names, _ := filepath.Glob(glob)
	for _, n := range names {
		if data, err := os.ReadFile(n); err == nil {
			f.Add(data)
		}
	}
	for _, s := range extra {
		f.Add([]byte(s))
	}
}

// fuzzConvert converts src and only requires that the converter returns
// (a PDF or an error) without panicking or hanging.
func fuzzConvert(t *testing.T, format proteus.Format, src []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// A conversion that is still running after 10s counts as a hang: dump
	// every goroutine (PROTEUS_FUZZ_HANG=file) and crash so the fuzzer keeps
	// the input.
	watchdog := time.AfterFunc(10*time.Second, func() {
		if path := os.Getenv("PROTEUS_FUZZ_HANG"); path != "" {
			buf := make([]byte, 1<<20)
			_ = os.WriteFile(path, append(append(src, "\n----\n"...), buf[:runtime.Stack(buf, true)]...), 0o644)
		}
		panic("conversion hung for 10s")
	})
	defer watchdog.Stop()
	pdf, err := proteus.ConvertBytes(ctx, format, src,
		proteus.WithLocalFiles(false), proteus.WithWarningHandler(func(string) {}))
	var ie *proteus.InternalError
	if errors.As(err, &ie) {
		t.Fatalf("panic: %v\n%s", ie.Value, ie.Stack)
	}
	if err == nil && !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("no error but output is not a PDF")
	}
}

func FuzzConvertHTML(f *testing.F) {
	fuzzSeeds(f, "testdata/*.html",
		`<table><tr><td colspan=1000 rowspan=65535>x</td></tr></table>`,
		`<div style="display:grid;grid-template-columns:repeat(10000,1fr)">a</div>`,
		`<div style="width:1e308px;height:-1e308px;font-size:1e30px">x</div>`,
		`<ol start=-2147483648><li>a</ol><ul style="list-style:upper-roman"><li value=99999999>b</ul>`,
		`<svg viewBox="0 0 0 0"><path d="M0 0 A 0 0 0 1 1 NaN 1e400 Z"/></svg>`,
		`<p style="letter-spacing:-1e9px;word-spacing:1e9px;line-height:0">a b c</p>`,
		`<div style="display:flex;flex-wrap:wrap"><div style="flex:1e30 1e30 -5px">a</div></div>`,
		`<style>@page{size:1px 1px;margin:0}</style>x`,
		`<img src="data:image/png;base64,iVBORw0KGgo=" width=0 height=-1>`,
	)
	f.Fuzz(func(t *testing.T, src []byte) {
		fuzzConvert(t, proteus.FormatHTML, src)
	})
}

func FuzzConvertMarkdown(f *testing.F) {
	fuzzSeeds(f, "testdata/*.md",
		"| a | b |\n|---|---|\n| 1 |\n",
		"[^1]\n\n[^1]: note",
		"- [ ] task\n  - [x] nested\n",
		"```go\nfunc(){}\n```\n> > > quote\n",
		"Term\n: definition\n",
	)
	f.Fuzz(func(t *testing.T, src []byte) {
		fuzzConvert(t, proteus.FormatMarkdown, src)
	})
}

// FuzzConvertCSS styles a document exercising most layout modes with a fuzzed
// stylesheet, so property values reach the cascade and every layouter.
func FuzzConvertCSS(f *testing.F) {
	for _, s := range []string{
		`*{display:flex;flex-wrap:wrap;gap:1e9px}`,
		`*{display:grid;grid-template-columns:repeat(auto-fill,minmax(0,1fr));grid-row:-1/span 99999}`,
		`*{float:left;position:absolute;inset:-1e30px;width:calc(100% / 0)}`,
		`*{display:table-cell;border:1e9px double red;border-collapse:collapse}`,
		`p{font:italic 900 0px/0 serif;letter-spacing:1e30px;text-indent:-1e30px;columns:3}`,
		`*{background:linear-gradient(1e30deg,red -1e9%,blue);background-size:0 0;background-repeat:repeat}`,
		`:root{--a:var(--b);--b:var(--a)}*{width:var(--a);margin:var(--c,var(--c,var(--c)))}`,
		`@page{size:0 0;margin:-5cm}*{break-before:page;orphans:0;widows:-1}`,
		`li{list-style:upper-roman inside;counter-increment:x 1e9}`,
		`img{object-fit:cover;object-position:1e9% -1e9%;transform:rotate(45deg)}`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, sheet string) {
		fuzzConvert(t, proteus.FormatHTML, []byte("<!DOCTYPE html><style>"+sheet+"</style>"+fuzzBody))
	})
}

const fuzzBody = `<body><h1 id=t>Title</h1><p>Some <b>bold</b> <a href="#t">link</a> <code>code</code>
text<sup>2</sup> that wraps across several lines of the page to test inline layout.</p>
<ul><li>one<li>two<ol><li>nested</ol></ul>
<table border=1><thead><tr><th>A<th colspan=2>B</thead><tr><td rowspan=2>1<td>2<td>3<tr><td>4<td>5</table>
<div style="display:flex"><div>f1</div><div style="flex:1">f2</div></div>
<div style="display:grid;grid-template-columns:1fr 2fr"><span>g1</span><span>g2</span><span>g3</span></div>
<div style="float:left;width:50px">float</div><p style="position:relative">after float</p>
<svg width=40 height=40 viewBox="0 0 10 10"><circle cx=5 cy=5 r=4 fill=red /><text x=1 y=8>s</text></svg>
<img alt=x width=20 height=20 src="data:image/gif;base64,R0lGODlhAQABAAAAACw=">
<pre>pre   formatted
  text</pre><blockquote>quote</blockquote><hr><input value=x><button>b</button></body>`
