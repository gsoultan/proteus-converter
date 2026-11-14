// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package proteus_test

import (
	"context"
	"testing"

	proteus "github.com/gsoultan/proteus-converter"
)

// BenchmarkConvertTo converts every testdata fixture to PDF and, for the ones
// with tables, to XLSX. The out-bytes metric is the size of the output file.
func BenchmarkConvertTo(b *testing.B) {
	for _, o := range []proteus.OutputFormat{proteus.OutputPDF, proteus.OutputXLSX} {
		for _, fx := range fixtures(b) {
			b.Run(string(o)+"/"+fx.name, func(b *testing.B) {
				b.ReportAllocs()
				var out []byte
				for b.Loop() {
					var err error
					out, err = proteus.ConvertBytesTo(context.Background(), fx.format, o, fx.src, proteus.WithBaseURL(fx.base))
					if err != nil {
						b.Fatal(err)
					}
				}
				b.ReportMetric(float64(len(out)), "out-bytes")
				b.ReportMetric(float64(len(fx.src)), "in-bytes")
			})
		}
	}
}
