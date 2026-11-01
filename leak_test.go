// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package proteus_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	proteus "github.com/gsoultan/proteus-converter"
)

type fixture struct {
	name   string
	format proteus.Format
	src    []byte
	base   string
}

func fixtures(tb testing.TB) []fixture {
	tb.Helper()
	names, _ := filepath.Glob("testdata/*.*")
	var out []fixture
	for _, n := range names {
		f, err := proteus.FormatForPath(n)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(n)
		if err != nil {
			tb.Fatal(err)
		}
		out = append(out, fixture{strings.TrimSuffix(filepath.Base(n), filepath.Ext(n)), f, data, n})
	}
	if len(out) == 0 {
		tb.Fatal("no fixtures")
	}
	return out
}

func convertFixture(tb testing.TB, fx fixture) []byte {
	pdf, err := proteus.ConvertBytes(context.Background(), fx.format, fx.src, proteus.WithBaseURL(fx.base))
	if err != nil {
		tb.Fatalf("%s: %v", fx.name, err)
	}
	return pdf
}

func heapInuse() uint64 {
	var ms runtime.MemStats
	for range 3 {
		runtime.GC()
	}
	runtime.ReadMemStats(&ms)
	return ms.HeapInuse
}

// TestNoLeaks converts every fixture many times and checks that neither
// goroutines nor retained heap grow with the number of conversions.
func TestNoLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	fxs := fixtures(t)
	round := func() {
		for _, fx := range fxs {
			convertFixture(t, fx)
		}
	}
	round() // warm caches (system fonts, UA stylesheet)
	g0 := runtime.NumGoroutine()
	for range 5 {
		round()
	}
	h0 := heapInuse()
	for range 30 {
		round()
	}
	h1 := heapInuse()
	// give finished goroutines a moment to exit before counting
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > g0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if g := runtime.NumGoroutine(); g > g0 {
		buf := make([]byte, 1<<16)
		t.Errorf("goroutines grew from %d to %d:\n%s", g0, g, buf[:runtime.Stack(buf, true)])
	}
	t.Logf("retained heap after %d conversions: %d KiB, after %d: %d KiB",
		5*len(fxs), h0>>10, 35*len(fxs), h1>>10)
	if h1 > h0+4<<20 {
		t.Errorf("retained heap grew by %d KiB over %d conversions", (h1-h0)>>10, 30*len(fxs))
	}
}

// pdfLayout returns every word of a PDF with its position (pdftotext -bbox),
// a stable fingerprint of the rendered result. gpdf output is not byte-for-byte
// reproducible (random subset tags, map-ordered dictionaries), so outputs are
// compared through it.
func pdfLayout(t testing.TB, pdf []byte) string {
	t.Helper()
	bin, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext not installed")
	}
	cmd := exec.Command(bin, "-bbox", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("pdftotext: %v", err)
	}
	s := string(out)
	return s[strings.Index(s, "<body>"):]
}

// TestConcurrentConversions runs conversions in parallel (go test -race
// catches shared-state races) and checks every run lays out the same way.
func TestConcurrentConversions(t *testing.T) {
	fxs := fixtures(t)
	want := map[string]string{}
	for _, fx := range fxs {
		want[fx.name] = pdfLayout(t, convertFixture(t, fx))
	}
	errs := make(chan string, 4*len(fxs))
	done := make(chan struct{})
	for range 4 {
		go func() {
			defer func() { done <- struct{}{} }()
			for _, fx := range fxs {
				pdf, err := proteus.ConvertBytes(context.Background(), fx.format, fx.src, proteus.WithBaseURL(fx.base))
				if err != nil {
					errs <- fx.name + ": " + err.Error()
				} else if pdfLayout(t, pdf) != want[fx.name] {
					errs <- fx.name + ": output differs between runs"
				}
			}
		}()
	}
	for range 4 {
		<-done
	}
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

func BenchmarkConvert(b *testing.B) {
	for _, fx := range fixtures(b) {
		b.Run(fx.name, func(b *testing.B) {
			convertFixture(b, fx)
			b.ReportAllocs()
			var size int
			for b.Loop() {
				size = len(convertFixture(b, fx))
			}
			b.ReportMetric(float64(size)/1024, "KiB/pdf")
		})
	}
}
