// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package proteus_test

import (
	"bytes"
	"compress/zlib"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"regexp"
	"strings"
	"testing"

	proteus "github.com/gsoultan/proteus-converter"
)

const twoPages = `<!DOCTYPE html><html><body><p>First page</p><p style="page-break-before: always">Second page</p></body></html>`

// pdfStreams returns the inflated content of every stream in the PDF plus
// the uncompressed bytes between them, so operators and resource dicts
// can be searched as text.
func pdfStreams(t *testing.T, pdf []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	rest := pdf
	for {
		i := bytes.Index(rest, []byte("stream\n"))
		if i < 0 {
			out.Write(rest)
			return out.Bytes()
		}
		out.Write(rest[:i])
		rest = rest[i+len("stream\n"):]
		j := bytes.Index(rest, []byte("endstream"))
		if j < 0 {
			out.Write(rest)
			return out.Bytes()
		}
		data := rest[:j]
		if r, err := zlib.NewReader(bytes.NewReader(data)); err == nil {
			if b, err := io.ReadAll(r); err == nil {
				data = b
			}
		}
		out.Write(data)
		out.WriteByte('\n')
		rest = rest[j+len("endstream"):]
	}
}

// hasLetters reports whether every letter of word occurs in page. pdftotext
// splits rotated text into single letters on separate lines, so the word
// itself cannot be searched for.
func hasLetters(page, word string) bool {
	for _, r := range word {
		if !strings.ContainsRune(page, r) {
			return false
		}
	}
	return true
}

// pdfPages returns the text of each page (pdftotext puts a form feed between pages).
func pdfPages(t *testing.T, pdf []byte) []string {
	t.Helper()
	return strings.Split(strings.TrimRight(pdfText(t, pdf), "\f\n "), "\f")
}

func TestWatermarkTextOnEveryPage(t *testing.T) {
	pdf, err := proteus.ConvertString(context.Background(), proteus.FormatHTML, twoPages,
		proteus.WithWatermark(proteus.Watermark{Text: "DRAFT", Opacity: 0.2}))
	if err != nil {
		t.Fatal(err)
	}
	checkPDF(t, pdf)
	// Every page carries a translucent text run.
	if n := bytes.Count(pdfStreams(t, pdf), []byte("/TXGS1 gs")); n != 2 {
		t.Fatalf("translucent stamps = %d, want one per page", n)
	}
	if !regexp.MustCompile(`/ca 0\.2\b`).Match(pdfStreams(t, pdf)) {
		t.Fatal("opacity 0.2 not written as a constant alpha")
	}
	for i, page := range pdfPages(t, pdf) {
		if !hasLetters(page, "DRAFT") {
			t.Errorf("page %d has no DRAFT: %q", i+1, page)
		}
	}
}

func TestWatermarkPagesAndLayer(t *testing.T) {
	ctx := context.Background()
	first, err := proteus.ConvertString(ctx, proteus.FormatHTML, twoPages,
		proteus.WithWatermark(proteus.Watermark{Text: "COPY", Pages: proteus.WatermarkFirstPage, Layer: proteus.WatermarkOver}))
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(pdfStreams(t, first), []byte("/TXGS1 gs")); n != 1 {
		t.Fatalf("first-page stamps = %d, want 1", n)
	}
	rest, err := proteus.ConvertString(ctx, proteus.FormatHTML, twoPages,
		proteus.WithWatermark(proteus.Watermark{Text: "COPY", Pages: proteus.WatermarkAllButFirst}))
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(pdfStreams(t, rest), []byte("/TXGS1 gs")); n != 1 {
		t.Fatalf("all-but-first stamps = %d, want 1", n)
	}
	pages := pdfPages(t, rest)
	if len(pages) != 2 || strings.ContainsAny(pages[0], "COPY") || !hasLetters(pages[1], "COPY") {
		t.Fatalf("stamp on the wrong page: %q", pages)
	}
}

func TestWatermarkTiledAndEmpty(t *testing.T) {
	ctx := context.Background()
	tiled, err := proteus.ConvertString(ctx, proteus.FormatHTML, twoPages,
		proteus.WithWatermark(proteus.Watermark{Text: "CONFIDENTIAL", FontSize: 24, Placement: proteus.WatermarkTiled, Angle: 45}))
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(pdfStreams(t, tiled), []byte(" gs")); n < 10 {
		t.Fatalf("tiled stamps = %d, want many", n)
	}
	plain, err := proteus.ConvertString(ctx, proteus.FormatHTML, twoPages, proteus.WithWatermark(proteus.Watermark{Text: "   "}))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(pdfStreams(t, plain), []byte("TXGS")) {
		t.Fatal("blank watermark text painted something")
	}
}

func TestWatermarkImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	for i := range img.Pix {
		img.Pix[i] = 0x80
	}
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	var warnings []string
	pdf, err := proteus.ConvertString(context.Background(), proteus.FormatHTML, twoPages,
		proteus.WithWatermark(proteus.Watermark{Image: buf.Bytes(), Opacity: 0.3, ImageWidth: 200}),
		proteus.WithWarningHandler(func(m string) { warnings = append(warnings, m) }))
	if err != nil {
		t.Fatal(err)
	}
	checkPDF(t, pdf)
	if len(warnings) > 0 {
		t.Fatalf("warnings: %v", warnings)
	}
	// Two pages, each with one translucent image drawn through a graphics
	// state (the XObject is registered by an extra invisible run, so Do
	// appears twice per page).
	if n := bytes.Count(pdfStreams(t, pdf), []byte("/GS1 gs")); n != 2 {
		t.Fatalf("image stamps = %d, want 2", n)
	}
	if n := bytes.Count(pdfStreams(t, pdf), []byte("/Im1 Do")); n != 4 {
		t.Fatalf("image draws = %d, want 4", n)
	}
	if !regexp.MustCompile(`/ca 0\.3\b`).Match(pdfStreams(t, pdf)) {
		t.Fatal("image opacity not written")
	}

	warnings = nil
	_, err = proteus.ConvertString(context.Background(), proteus.FormatHTML, twoPages,
		proteus.WithWatermark(proteus.Watermark{Image: []byte("not an image")}),
		proteus.WithWarningHandler(func(m string) { warnings = append(warnings, m) }))
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) == 0 || !strings.Contains(warnings[0], "watermark") {
		t.Fatalf("bad image should warn, got %v", warnings)
	}
}
