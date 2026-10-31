// Command pixdiff compares two PDFs page by page and reports the share of
// pixels whose ink differs. It rasterises with pdftoppm (poppler) and is used
// by the accuracy gate to compare proteus output with Chromium's.
//
//	pixdiff [-dpi 96] [-max 3] [-diff out.png] reference.pdf candidate.pdf
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

func main() {
	dpi := flag.Int("dpi", 96, "rasterisation resolution")
	maxPct := flag.Float64("max", 3, "fail when any page mismatch exceeds this percentage")
	diffOut := flag.String("diff", "", "write an overlay of the first page (red: reference only, blue: candidate only)")
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	ref, err := raster(flag.Arg(0), *dpi)
	check(err)
	cand, err := raster(flag.Arg(1), *dpi)
	check(err)
	if len(ref) != len(cand) {
		fmt.Printf("page count differs: reference %d, candidate %d\n", len(ref), len(cand))
		os.Exit(1)
	}
	fail := false
	for i := range ref {
		pct, overlay := compare(ref[i], cand[i])
		fmt.Printf("page %d: %.3f%% mismatch\n", i+1, pct)
		if pct > *maxPct {
			fail = true
		}
		if i == 0 && *diffOut != "" {
			f, err := os.Create(*diffOut)
			check(err)
			check(png.Encode(f, overlay))
			check(f.Close())
		}
	}
	if fail {
		fmt.Printf("FAIL: mismatch above %.2f%%\n", *maxPct)
		os.Exit(1)
	}
}

func raster(pdf string, dpi int) ([]image.Image, error) {
	dir, err := os.MkdirTemp("", "pixdiff")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	cmd := exec.Command("pdftoppm", "-r", fmt.Sprint(dpi), "-png", pdf, filepath.Join(dir, "p"))
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm %s: %w", pdf, err)
	}
	names, _ := filepath.Glob(filepath.Join(dir, "p-*.png"))
	sort.Strings(names)
	var pages []image.Image
	for _, n := range names {
		f, err := os.Open(n)
		if err != nil {
			return nil, err
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			return nil, err
		}
		pages = append(pages, img)
	}
	return pages, nil
}

// inked treats a pixel as ink when its luma is clearly below white, so faint
// backgrounds and anti-aliasing fringes do not count.
func inked(c color.Color) bool {
	return color.GrayModel.Convert(c).(color.Gray).Y < 200
}

func compare(a, b image.Image) (float64, *image.RGBA) {
	r := a.Bounds().Intersect(b.Bounds())
	out := image.NewRGBA(r)
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ia, ib := inked(a.At(x, y)), inked(b.At(x, y))
			c := color.RGBA{255, 255, 255, 255}
			switch {
			case ia && ib:
				c = color.RGBA{0, 0, 0, 255}
			case ia:
				c = color.RGBA{255, 0, 0, 255}
				n++
			case ib:
				c = color.RGBA{0, 0, 255, 255}
				n++
			}
			out.Set(x, y, c)
		}
	}
	return 100 * float64(n) / float64(r.Dx()*r.Dy()), out
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "pixdiff:", err)
		os.Exit(2)
	}
}
