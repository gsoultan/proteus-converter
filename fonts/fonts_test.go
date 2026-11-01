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
