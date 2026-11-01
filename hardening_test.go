// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package proteus_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"strings"
	"testing"

	proteus "github.com/gsoultan/proteus-converter"
	"github.com/gsoultan/proteus-converter/core"
)

func TestRunawayDocumentHitsPageLimit(t *testing.T) {
	src := `<div style="height:1e9px;background:red">x</div>`
	_, err := proteus.ConvertString(context.Background(), proteus.FormatHTML, src, proteus.WithMaxPages(50))
	if !errors.Is(err, proteus.ErrTooManyPages) {
		t.Fatalf("err = %v, want ErrTooManyPages", err)
	}
}

// pngHeader returns a PNG with a valid IHDR claiming w x h pixels and no
// image data: a decompression-bomb shaped file.
func pngHeader(w, h uint32) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6 // 8-bit RGBA
	chunk := append([]byte("IHDR"), ihdr...)
	binary.Write(&b, binary.BigEndian, uint32(len(ihdr)))
	b.Write(chunk)
	binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

func TestHugeImageIsRefusedWithWarning(t *testing.T) {
	src := `<img src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(pngHeader(60000, 60000)) + `">ok`
	var warnings []string
	pdf, err := proteus.ConvertString(context.Background(), proteus.FormatHTML, src,
		proteus.WithWarningHandler(func(m string) { warnings = append(warnings, m) }))
	if err != nil {
		t.Fatal(err)
	}
	checkPDF(t, pdf)
	if !strings.Contains(strings.Join(warnings, "\n"), "out of range") {
		t.Errorf("no warning about the oversized image: %q", warnings)
	}
}

func TestRecoverTurnsPanicIntoError(t *testing.T) {
	err := func() (err error) {
		defer core.Recover(&err)
		var m map[string]int
		m["x"] = 1 // panics
		return nil
	}()
	var ie *proteus.InternalError
	if !errors.As(err, &ie) || !errors.Is(err, proteus.ErrInternal) || len(ie.Stack) == 0 {
		t.Fatalf("err = %v, want *InternalError wrapping ErrInternal with a stack", err)
	}
}

func TestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := proteus.ConvertString(ctx, proteus.FormatMarkdown, "# hi")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
