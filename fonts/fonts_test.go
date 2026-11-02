// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package fonts

import (
	"os"
	"testing"
)

func TestSystemChain(t *testing.T) {
	r := System()
	ch := r.Chain([]string{"serif"}, 400, false)
	t.Logf("serif -> %s (%s)", ch[0].Family, ch[0].PDFName())
	ch = r.Chain([]string{"Arial", "sans-serif"}, 700, false)
	t.Logf("Arial bold -> %s w%d (%s)", ch[0].Family, ch[0].Weight, ch[0].PDFName())
	a, d, n := ch[0].LineMetrics(12)
	t.Logf("metrics 16px: %v %v %v width(Hello)=%v", a/0.75, d/0.75, n/0.75, ch[0].Width("Hello", 12))
	runs := r.Split("Hello 你好", ch)
	for _, run := range runs {
		t.Logf("run %q -> %s", run.Text, run.Face.Family)
	}
}

func TestStandardFallback(t *testing.T) {
	r := NewResolver()
	ch := r.Chain([]string{"monospace"}, 700, true)
	if len(ch) != 1 || ch[0].PDFName() != "Courier-BoldOblique" {
		t.Fatalf("got %+v", ch[0])
	}
	if w := ch[0].Width("abc", 10); w != 18 {
		t.Fatalf("courier width %v", w)
	}
}

func TestDecodeWOFF(t *testing.T) {
	data, err := os.ReadFile("../testdata/assets/fonts/Caladea-Italic.woff")
	if err != nil {
		t.Fatal(err)
	}
	sfnt, err := DecodeWOFF(data)
	if err != nil {
		t.Fatal(err)
	}
	r := NewResolver()
	if err := r.AddData(sfnt, "", 0, false); err != nil {
		t.Fatal(err)
	}
	if !r.HasFamily("Caladea") {
		t.Fatal("decoded WOFF does not register the Caladea family")
	}
	if _, err := DecodeWOFF(data[:60]); err == nil {
		t.Fatal("truncated WOFF accepted")
	}
}

// WOFF2 with the glyf, loca and hmtx transforms decodes to a usable
// TrueType font; the accuracy gate checks its rendering against Chromium.
func TestDecodeWOFF2(t *testing.T) {
	data, err := os.ReadFile("../testdata/assets/fonts/Caladea-Bold.woff2")
	if err != nil {
		t.Fatal(err)
	}
	sfnt, err := DecodeWOFF2(data)
	if err != nil {
		t.Fatal(err)
	}
	r := NewResolver()
	if err := r.AddData(sfnt, "", 0, false); err != nil {
		t.Fatal(err)
	}
	if !r.HasFamily("Caladea") {
		t.Fatal("decoded WOFF2 does not register the Caladea family")
	}
	f := r.Chain([]string{"Caladea"}, 700, false)[0]
	if f.Weight != 700 {
		t.Fatalf("decoded face weight %d, want 700", f.Weight)
	}
	if w := f.Width("Hello", 10); w < 20 || w > 30 {
		t.Errorf("width of Hello at 10pt = %.2f, want about 24", w)
	}
	for _, n := range []int{10, 60, 200} {
		if _, err := DecodeWOFF2(data[:n]); err == nil {
			t.Errorf("WOFF2 truncated to %d bytes accepted", n)
		}
	}
}

// FuzzDecodeWOFF2 checks that hostile WOFF2 input never panics.
func FuzzDecodeWOFF2(f *testing.F) {
	if data, err := os.ReadFile("../testdata/assets/fonts/Caladea-Bold.woff2"); err == nil {
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if sfnt, err := DecodeWOFF2(data); err == nil {
			_ = NewResolver().AddData(sfnt, "", 0, false)
		}
	})
}
