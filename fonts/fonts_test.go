package fonts

import "testing"

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
