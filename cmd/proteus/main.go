// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Command proteus converts HTML and Markdown files to PDF.
//
// Usage:
//
//	proteus [flags] input.(html|md) [output.pdf]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/gsoultan/proteus-converter"
)

func main() {
	page := flag.String("page", "", "page size: A3, A4, A5, Letter, Legal (append -landscape to rotate); default from CSS @page or A4")
	margin := flag.String("margin", "", "page margin in mm for all sides; default from CSS @page or 10mm")
	css := flag.String("css", "", "extra stylesheet file applied after the document's styles")
	allowHTTP := flag.Bool("http", false, "allow fetching http(s) resources")
	noSystemFonts := flag.Bool("no-system-fonts", false, "do not scan OS font directories")
	fontDir := flag.String("font-dir", "", "additional directory with .ttf fonts")
	title := flag.String("title", "", "document title")
	quiet := flag.Bool("q", false, "do not print warnings")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: proteus [flags] input.(html|md) [output.pdf]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 || flag.NArg() > 2 {
		flag.Usage()
		os.Exit(2)
	}
	in := flag.Arg(0)
	out := strings.TrimSuffix(in, filepath.Ext(in)) + ".pdf"
	if flag.NArg() == 2 {
		out = flag.Arg(1)
	}
	var opts []proteus.Option
	if *page != "" {
		size, err := parsePage(*page)
		if err != nil {
			fail(err)
		}
		opts = append(opts, proteus.WithPageSize(size))
	}
	if *margin != "" {
		var mm float64
		if _, err := fmt.Sscanf(*margin, "%g", &mm); err != nil {
			fail(fmt.Errorf("bad -margin %q", *margin))
		}
		opts = append(opts, proteus.WithMargins(proteus.UniformMargins(proteus.Mm(mm))))
	}
	if *css != "" {
		data, err := os.ReadFile(*css)
		if err != nil {
			fail(err)
		}
		opts = append(opts, proteus.WithStylesheet(string(data)))
	}
	if *fontDir != "" {
		opts = append(opts, proteus.WithFontDir(*fontDir))
	}
	if *title != "" {
		opts = append(opts, proteus.WithTitle(*title))
	}
	opts = append(opts, proteus.WithHTTP(*allowHTTP), proteus.WithSystemFonts(!*noSystemFonts))
	if !*quiet {
		opts = append(opts, proteus.WithWarningHandler(func(msg string) { fmt.Fprintln(os.Stderr, "warning:", msg) }))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := proteus.ConvertFile(ctx, in, out, opts...); err != nil {
		fail(err)
	}
}

func parsePage(s string) (proteus.PageSize, error) {
	land := strings.HasSuffix(strings.ToLower(s), "-landscape")
	name := strings.ToLower(strings.TrimSuffix(strings.ToLower(s), "-landscape"))
	sizes := map[string]proteus.PageSize{"a3": proteus.A3, "a4": proteus.A4, "a5": proteus.A5, "letter": proteus.Letter, "legal": proteus.Legal}
	size, ok := sizes[name]
	if !ok {
		return proteus.PageSize{}, fmt.Errorf("unknown page size %q", s)
	}
	if land {
		size = size.Landscape()
	}
	return size, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "proteus:", err)
	os.Exit(1)
}
