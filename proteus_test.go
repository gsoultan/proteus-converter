package proteus_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	proteus "github.com/gsoultan/proteus-converter"
)

// pdfText extracts the text of a PDF with pdftotext, skipping the check when
// poppler is not installed.
func pdfText(t *testing.T, pdf []byte) string {
	t.Helper()
	bin, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext not installed")
	}
	cmd := exec.Command(bin, "-layout", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("pdftotext: %v", err)
	}
	return string(out)
}

func checkPDF(t *testing.T, pdf []byte) {
	t.Helper()
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("output is not a PDF: %q", pdf[:min(len(pdf), 16)])
	}
	if !bytes.Contains(pdf[max(len(pdf)-1024, 0):], []byte("%%EOF")) {
		t.Fatal("PDF has no trailer")
	}
}

func TestConvertFileHTML(t *testing.T) {
	out := filepath.Join(t.TempDir(), "invoice.pdf")
	var warnings []string
	err := proteus.ConvertFile(context.Background(), "testdata/invoice.html", out,
		proteus.WithWarningHandler(func(m string) { warnings = append(warnings, m) }))
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	checkPDF(t, pdf)
	if len(warnings) > 0 {
		t.Errorf("unexpected warnings: %q", warnings)
	}
	text := pdfText(t, pdf)
	for _, want := range []string{"Invoice", "BILLED TO", "Acme Widgets Ltd.", "Total due", "$3,256.00", "Thank you for your business!"} {
		if !strings.Contains(text, want) {
			t.Errorf("text %q missing from output", want)
		}
	}
}

func TestConvertMarkdown(t *testing.T) {
	src, err := os.ReadFile("testdata/sample.md")
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := proteus.ConvertBytes(context.Background(), proteus.FormatMarkdown, src, proteus.WithTitle("Sample"))
	if err != nil {
		t.Fatal(err)
	}
	checkPDF(t, pdf)
	if !bytes.Contains(pdf, []byte("Sample")) {
		t.Error("title metadata missing")
	}
	pdfText(t, pdf) // must at least be parseable
}

func TestInlineAndHeadStyles(t *testing.T) {
	const doc = `<html><head><style>p { text-transform: uppercase } .x { text-transform: none !important }</style></head>
<body><p>shout</p><p class="x" style="text-transform: uppercase">quiet</p><p style="display:none">hidden</p></body></html>`
	pdf, err := proteus.ConvertString(context.Background(), proteus.FormatHTML, doc)
	if err != nil {
		t.Fatal(err)
	}
	text := pdfText(t, pdf)
	if !strings.Contains(text, "SHOUT") {
		t.Errorf("head style not applied: %q", text)
	}
	if !strings.Contains(text, "quiet") {
		t.Errorf("!important in head should beat the inline style: %q", text)
	}
	if strings.Contains(text, "hidden") {
		t.Errorf("display:none content rendered: %q", text)
	}
}

func TestRemoteResourcesDeniedByDefault(t *testing.T) {
	const doc = `<html><head><link rel="stylesheet" href="https://example.invalid/x.css"></head><body>ok</body></html>`
	var warnings []string
	_, err := proteus.ConvertString(context.Background(), proteus.FormatHTML, doc,
		proteus.WithWarningHandler(func(m string) { warnings = append(warnings, m) }))
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) == 0 || !strings.Contains(strings.Join(warnings, "\n"), "denied") {
		t.Errorf("expected a denied-resource warning, got %q", warnings)
	}
}

func TestRegistry(t *testing.T) {
	for path, want := range map[string]proteus.Format{
		"a.html": proteus.FormatHTML, "b.HTM": proteus.FormatHTML, "c.md": proteus.FormatMarkdown, "d.markdown": proteus.FormatMarkdown,
	} {
		got, err := proteus.FormatForPath(path)
		if err != nil || got != want {
			t.Errorf("FormatForPath(%q) = %q, %v; want %q", path, got, err, want)
		}
	}
	if _, err := proteus.Lookup(proteus.FormatDOCX); !errors.Is(err, proteus.ErrUnknownFormat) {
		t.Errorf("DOCX lookup: got %v, want ErrUnknownFormat", err)
	}
	if _, err := proteus.FormatForPath("x.txt"); !errors.Is(err, proteus.ErrUnknownFormat) {
		t.Errorf("txt: got %v, want ErrUnknownFormat", err)
	}

	r := proteus.NewRegistry()
	r.Register(proteus.ConverterFunc{Name: proteus.FormatDOCX, Exts: []string{".docx"},
		Fn: func(ctx context.Context, src io.Reader, dst io.Writer, opts ...proteus.Option) error { return nil }})
	if f, err := r.FormatForPath("x.docx"); err != nil || f != proteus.FormatDOCX {
		t.Errorf("custom converter not registered: %q %v", f, err)
	}
}

func TestContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := proteus.ConvertString(ctx, proteus.FormatHTML, "<p>x</p>"); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}
