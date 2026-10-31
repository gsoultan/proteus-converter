// Command mdhtml writes the HTML that the Markdown converter lays out, with
// the theme inlined, so it can be rendered by a reference browser.
package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/gsoultan/proteus-converter/formats/markdown"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: mdhtml in.md out.html")
		os.Exit(2)
	}
	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	page, err := markdown.New().ToHTML(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	page = bytes.Replace(page, []byte("<head>"), []byte("<head><style>\n"+markdown.Theme()+"\n</style>"), 1)
	if err := os.WriteFile(os.Args[2], page, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
