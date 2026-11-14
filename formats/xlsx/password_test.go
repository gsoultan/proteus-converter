// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gsoultan/gexcel"
	"github.com/gsoultan/gexcel/reader"
	"github.com/gsoultan/proteus-converter/core"
	htmlfmt "github.com/gsoultan/proteus-converter/formats/html"
)

// TestPassword checks that WithPassword produces an encrypted workbook that
// opens only with the password and still holds the table.
func TestPassword(t *testing.T) {
	cfg := core.NewConfig(core.WithSystemFonts(false), core.WithPassword("hunter2!", ""))
	doc, err := htmlfmt.New().Parse(context.Background(), strings.NewReader(`<table><tr><th>Name</th><th>Amount</th></tr><tr><td>Ann</td><td>1,250.50</td></tr></table>`), cfg)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := New().Write(context.Background(), doc, cfg, &out); err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(out.Bytes(), []byte("PK")) {
		t.Fatal("output is a plain zip, not encrypted")
	}
	if _, err := gexcel.OpenBytes(out.Bytes()); !errors.Is(err, reader.ErrEncrypted) {
		t.Fatalf("open without password: %v", err)
	}
	if _, err := gexcel.OpenBytes(out.Bytes(), gexcel.Options{Password: "wrong"}); !errors.Is(err, reader.ErrWrongPassword) {
		t.Fatalf("open with wrong password: %v", err)
	}
	wb, err := gexcel.OpenBytes(out.Bytes(), gexcel.Options{Password: "hunter2!"})
	if err != nil {
		t.Fatal(err)
	}
	sh := wb.SheetAt(0)
	if got := sh.Cell("A2").String(); got != "Ann" {
		t.Errorf("A2 = %q", got)
	}
	if v, ok := sh.Cell("B2").Float(); !ok || v != 1250.5 {
		t.Errorf("B2 = %v %v", v, ok)
	}
}
