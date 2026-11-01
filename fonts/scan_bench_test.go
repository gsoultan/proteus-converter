package fonts

import "testing"

// BenchmarkSystemScan measures scanning the OS font directories, the cost
// paid by the first conversion in a process.
func BenchmarkSystemScan(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		r := NewResolver()
		for _, d := range systemFontDirs() {
			_ = r.AddDir(d)
		}
	}
}
