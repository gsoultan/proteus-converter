// Command textdiff compares the words of two PDFs and where they are placed.
// It extracts every word with its bounding box (pdftotext -bbox, poppler),
// aligns the two word sequences (longest common subsequence on the text), and
// reports how many reference words are present in the candidate and how far
// matched words moved. The accuracy gate uses it next to pixdiff, which only
// sees ink: textdiff catches missing, extra or reordered text and measures
// layout drift in points.
//
//	textdiff [-tol 2] [-min-match 99] [-min-within 95] reference.pdf candidate.pdf
package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
)

type word struct {
	page       int
	text       string
	x0, y0, y1 float64
}

func main() {
	tol := flag.Float64("tol", 2, "a matched word is in place when it moved at most this many points")
	minMatch := flag.Float64("min-match", 99, "fail when fewer than this percentage of reference words are found")
	minWithin := flag.Float64("min-within", 95, "fail when fewer than this percentage of matched words are within -tol")
	verbose := flag.Bool("v", false, "list unmatched words and the largest moves")
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	ref, err := words(flag.Arg(0))
	check(err)
	cand, err := words(flag.Arg(1))
	check(err)
	pairs := align(ref, cand)

	var devs []float64
	var sumDX, sumDY float64
	within := 0
	type move struct {
		w    word
		d    float64
		dx   float64
		dy   float64
		page int
	}
	var moves []move
	for _, p := range pairs {
		a, b := ref[p[0]], cand[p[1]]
		dx := b.x0 - a.x0
		dy := b.y1 - a.y1 // bottom edge tracks the baseline
		d := math.Hypot(dx, dy)
		if a.page != b.page {
			d = math.Inf(1)
		}
		devs = append(devs, d)
		sumDX += dx
		sumDY += dy
		if d <= *tol {
			within++
		}
		moves = append(moves, move{a, d, dx, dy, a.page})
	}
	matchPct := 100.0
	if len(ref) > 0 {
		matchPct = 100 * float64(len(pairs)) / float64(len(ref))
	}
	withinPct := 100.0
	if len(pairs) > 0 {
		withinPct = 100 * float64(within) / float64(len(pairs))
	}
	sort.Float64s(devs)
	pct := func(q float64) float64 {
		if len(devs) == 0 {
			return 0
		}
		return devs[min(len(devs)-1, int(q*float64(len(devs))))]
	}
	fmt.Printf("words: reference %d, candidate %d, matched %d (%.2f%%)\n", len(ref), len(cand), len(pairs), matchPct)
	fmt.Printf("position: %.2f%% within %.1fpt, median %.2fpt, p95 %.2fpt, max %.2fpt\n",
		withinPct, *tol, pct(0.5), pct(0.95), pct(1))
	if *verbose && len(pairs) > 0 {
		fmt.Printf("  mean shift (%.2f, %.2f)pt\n", sumDX/float64(len(pairs)), sumDY/float64(len(pairs)))
	}
	if *verbose {
		matched := map[int]bool{}
		for _, p := range pairs {
			matched[p[0]] = true
		}
		for i, w := range ref {
			if !matched[i] {
				fmt.Printf("  missing p%d %q at (%.1f, %.1f)\n", w.page, w.text, w.x0, w.y1)
			}
		}
		sort.Slice(moves, func(i, j int) bool { return moves[i].d > moves[j].d })
		for _, m := range moves[:min(len(moves), 15)] {
			if m.d <= *tol {
				break
			}
			fmt.Printf("  moved p%d %q by (%.2f, %.2f)\n", m.page, m.w.text, m.dx, m.dy)
		}
	}
	if matchPct < *minMatch || withinPct < *minWithin {
		fmt.Printf("FAIL: need >= %.1f%% words matched and >= %.1f%% within %.1fpt\n", *minMatch, *minWithin, *tol)
		os.Exit(1)
	}
}

// words runs pdftotext -bbox and returns the words in reading order.
func words(pdf string) ([]word, error) {
	out, err := exec.Command("pdftotext", "-bbox", pdf, "-").Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext %s: %w", pdf, err)
	}
	dec := xml.NewDecoder(bytes.NewReader(out))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity
	var ws []word
	page := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "page":
			page++
		case "word":
			w := word{page: page}
			for _, a := range se.Attr {
				var v float64
				fmt.Sscan(a.Value, &v)
				switch a.Name.Local {
				case "xMin":
					w.x0 = v
				case "yMin":
					w.y0 = v
				case "yMax":
					w.y1 = v
				}
			}
			var text string
			if err := dec.DecodeElement(&text, &se); err != nil {
				return nil, err
			}
			w.text = text
			ws = append(ws, w)
		}
	}
	return ws, nil
}

// align pairs reference and candidate words: first a longest common
// subsequence of the texts (reading order), then each remaining reference
// word with the nearest unpaired candidate word of the same text on the same
// page, since extraction order differs between producers in table cells and
// columns even when the layout is identical.
func align(a, b []word) [][2]int {
	pairs := lcs(a, b)
	usedA, usedB := map[int]bool{}, map[int]bool{}
	for _, p := range pairs {
		usedA[p[0]], usedB[p[1]] = true, true
	}
	for i, w := range a {
		if usedA[i] {
			continue
		}
		best, bestD := -1, math.Inf(1)
		for j, v := range b {
			if usedB[j] || v.text != w.text || v.page != w.page {
				continue
			}
			if d := math.Hypot(v.x0-w.x0, v.y1-w.y1); d < bestD {
				best, bestD = j, d
			}
		}
		if best >= 0 {
			usedB[best] = true
			pairs = append(pairs, [2]int{i, best})
		}
	}
	return pairs
}

// lcs returns index pairs of a longest common subsequence of the word texts.
func lcs(a, b []word) [][2]int {
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		return nil
	}
	// lengths[i][j] = LCS of a[i:], b[j:], stored as uint16 rows
	l := make([][]uint16, n+1)
	for i := range l {
		l[i] = make([]uint16, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i].text == b[j].text {
				l[i][j] = l[i+1][j+1] + 1
			} else {
				l[i][j] = max(l[i+1][j], l[i][j+1])
			}
		}
	}
	var out [][2]int
	for i, j := 0, 0; i < n && j < m; {
		switch {
		case a[i].text == b[j].text:
			out = append(out, [2]int{i, j})
			i++
			j++
		case l[i+1][j] >= l[i][j+1]:
			i++
		default:
			j++
		}
	}
	return out
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "textdiff:", err)
		os.Exit(2)
	}
}
