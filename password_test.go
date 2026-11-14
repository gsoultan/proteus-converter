// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package proteus_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/gsoultan/gpdf/doc"
	proteus "github.com/gsoultan/proteus-converter"
)

// TestPDFPassword checks that WithPassword writes an AES-256 encrypted PDF
// that opens with the user password and keeps its text.
func TestPDFPassword(t *testing.T) {
	src := `<!DOCTYPE html><html><body><h1>Confidential</h1><p>Only for the holder of the password.</p></body></html>`
	pdf, err := proteus.ConvertBytesTo(context.Background(), proteus.FormatHTML, proteus.OutputPDF, []byte(src),
		proteus.WithSystemFonts(false), proteus.WithPassword("open-me", "owner-key"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(pdf, []byte("/Encrypt")) {
		t.Fatal("no /Encrypt dictionary in the output")
	}
	if bytes.Contains(pdf, []byte("Confidential")) {
		t.Fatal("plain text visible in the encrypted PDF")
	}
	if _, err := doc.OpenReader(bytes.NewReader(pdf), int64(len(pdf))); err == nil {
		// Some readers open encrypted files lazily; the text must still be unreadable.
		d, _ := doc.OpenReader(bytes.NewReader(pdf), int64(len(pdf)))
		if txt, err := d.ReadContent(); err == nil && strings.Contains(txt, "Confidential") {
			t.Fatal("content readable without the password")
		}
	}
	for _, pw := range []string{"open-me", "owner-key"} {
		d, err := doc.OpenReaderWithPassword(bytes.NewReader(pdf), int64(len(pdf)), pw)
		if err != nil {
			t.Fatalf("open with %q: %v", pw, err)
		}
		txt, err := d.ReadContent()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(txt, "Confidential") || !strings.Contains(txt, "holder of the password") {
			t.Fatalf("open with %q: text not found in %q", pw, txt)
		}
	}
	if _, err := doc.OpenReaderWithPassword(bytes.NewReader(pdf), int64(len(pdf)), "wrong"); err == nil {
		t.Fatal("wrong password accepted")
	}
}
