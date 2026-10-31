package fonts

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"unicode"
)

// Resolver maps CSS family lists to faces. It is safe for concurrent use.
type Resolver struct {
	mu       sync.RWMutex
	families map[string][]*Face // lower-case family name -> faces
	all      []*Face
	runeFace map[rune]*Face // fallback cache for runes missing from the requested chain
	chains   map[chainKey][]*Face
}

type chainKey struct {
	families string
	weight   int
	italic   bool
}

// NewResolver returns an empty resolver (standard-14 fonts only).
func NewResolver() *Resolver {
	return &Resolver{families: map[string][]*Face{}, runeFace: map[rune]*Face{}, chains: map[chainKey][]*Face{}}
}

var (
	systemOnce sync.Once
	systemRes  *Resolver
)

// System returns a shared resolver populated from the OS font directories.
// The scan runs once per process.
func System() *Resolver {
	systemOnce.Do(func() {
		systemRes = NewResolver()
		for _, d := range systemFontDirs() {
			_ = systemRes.AddDir(d)
		}
	})
	return systemRes
}

func systemFontDirs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		dirs := []string{filepath.Join(os.Getenv("WINDIR"), "Fonts")}
		if la := os.Getenv("LOCALAPPDATA"); la != "" {
			dirs = append(dirs, filepath.Join(la, "Microsoft", "Windows", "Fonts"))
		}
		return dirs
	case "darwin":
		return []string{"/System/Library/Fonts", "/Library/Fonts", filepath.Join(home, "Library", "Fonts")}
	}
	return []string{"/usr/share/fonts", "/usr/local/share/fonts", filepath.Join(home, ".fonts"), filepath.Join(home, ".local", "share", "fonts")}
}

// Clone returns a resolver with the same faces that can be extended
// independently (for per-conversion fonts).
func (r *Resolver) Clone() *Resolver {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c := NewResolver()
	for k, v := range r.families {
		c.families[k] = slices.Clone(v)
	}
	c.all = slices.Clone(r.all)
	return c
}

// AddDir scans dir recursively for TrueType fonts and collections.
func (r *Resolver) AddDir(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".ttf", ".ttc", ".otf", ".otc":
			_ = r.AddFile(path)
		}
		return nil
	})
}

// AddFile registers every face in a font file, under the families in its name table.
func (r *Resolver) AddFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return r.add(data, path, "", 0, false, false)
}

// AddData registers a font program. A non-empty family overrides the name
// table; weight/italic override the font's own values when family is set.
func (r *Resolver) AddData(data []byte, family string, weight int, italic bool) error {
	return r.add(data, "", family, weight, italic, true)
}

func (r *Resolver) add(data []byte, path, family string, weight int, italic, keep bool) error {
	offs, err := collectionOffsets(data)
	if err != nil {
		return err
	}
	added := 0
	for i, off := range offs {
		info, err := parseSfntInfo(data, off)
		if err != nil || !info.hasGlyf {
			continue
		}
		f := &Face{Family: info.family, Weight: info.weight, Italic: info.italic, path: path, offset: off, index: i, info: info}
		if len(offs) == 1 {
			f.index = 0
			f.offset = 0
		}
		if keep {
			f.data = data
		}
		names := []string{info.family}
		if info.altFamily != "" {
			names = append(names, info.altFamily)
		}
		if family != "" {
			names = []string{family}
			f.Family = family
			if weight > 0 {
				f.Weight = weight
			}
			f.Italic = italic
		}
		r.mu.Lock()
		for _, n := range names {
			k := strings.ToLower(n)
			r.families[k] = append(r.families[k], f)
		}
		r.all = append(r.all, f)
		r.chains = map[chainKey][]*Face{}
		r.mu.Unlock()
		added++
	}
	if added == 0 {
		return errNotSfnt
	}
	return nil
}

// HasFamily reports whether any face is registered for the family.
func (r *Resolver) HasFamily(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.families[strings.ToLower(name)]) > 0
}

// Generic families and metric-compatible aliases, following fontconfig on Linux.
var genericFamilies = map[string][]string{
	"serif":      {"Times New Roman", "Liberation Serif", "Tinos", "DejaVu Serif", "Noto Serif", "FreeSerif", "Times"},
	"sans-serif": {"Arial", "Liberation Sans", "Arimo", "Helvetica", "DejaVu Sans", "Noto Sans", "FreeSans"},
	"monospace":  {"Courier New", "Liberation Mono", "Cousine", "DejaVu Sans Mono", "Noto Sans Mono", "FreeMono", "Courier"},
}

var genericAliases = map[string]string{
	"system-ui": "sans-serif", "-apple-system": "sans-serif", "blinkmacsystemfont": "sans-serif",
	"ui-sans-serif": "sans-serif", "ui-serif": "serif", "ui-monospace": "monospace", "ui-rounded": "sans-serif",
	"cursive": "serif", "fantasy": "serif", "math": "serif", "emoji": "sans-serif", "fangsong": "serif",
}

var metricAliases = map[string][]string{
	"arial":           {"Liberation Sans", "Arimo"},
	"helvetica":       {"Liberation Sans", "Arimo"},
	"helvetica neue":  {"Liberation Sans", "Arimo"},
	"times new roman": {"Liberation Serif", "Tinos"},
	"times":           {"Liberation Serif", "Tinos"},
	"courier new":     {"Liberation Mono", "Cousine"},
	"courier":         {"Liberation Mono", "Cousine"},
	"calibri":         {"Carlito"},
	"cambria":         {"Caladea"},
}

// fallbackFamilies are tried for runes the requested families lack.
var fallbackFamilies = []string{"DejaVu Sans", "Noto Sans", "Noto Sans CJK SC", "Noto Sans CJK JP", "WenQuanYi Zen Hei",
	"Noto Sans Symbols", "Noto Sans Symbols2", "FreeSans", "FreeSerif"}

// Chain returns the ordered faces to try for a family list, weight and style.
// The last element is always a standard-14 face, so the chain is never empty.
func (r *Resolver) Chain(families []string, weight int, italic bool) []*Face {
	key := chainKey{strings.Join(families, "\x00"), weight, italic}
	r.mu.RLock()
	if c, ok := r.chains[key]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()

	var out []*Face
	seen := map[*Face]bool{}
	push := func(name string) bool {
		r.mu.RLock()
		faces := r.families[strings.ToLower(name)]
		r.mu.RUnlock()
		if f := matchFace(faces, weight, italic); f != nil {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
			return true
		}
		return false
	}
	class := "serif"
	classSet := false
	for _, fam := range families {
		lf := strings.ToLower(fam)
		if g, ok := genericAliases[lf]; ok {
			lf = g
		}
		if list, ok := genericFamilies[lf]; ok {
			if !classSet {
				class, classSet = lf, true
			}
			for _, n := range list {
				if push(n) {
					break
				}
			}
			continue
		}
		if push(fam) {
			continue
		}
		for _, alias := range metricAliases[lf] {
			if push(alias) {
				if !classSet {
					class, classSet = classOf(lf), true
				}
				break
			}
		}
	}
	if !classSet || len(out) == 0 {
		for _, n := range genericFamilies["serif"] {
			if push(n) {
				break
			}
		}
	}
	for _, n := range fallbackFamilies {
		push(n)
	}
	out = append(out, &Face{Family: class, Weight: weight, Italic: italic, Standard: standardFace(class, weight >= 600, italic)})
	r.mu.Lock()
	r.chains[key] = out
	r.mu.Unlock()
	return out
}

func classOf(lowerFamily string) string {
	switch lowerFamily {
	case "arial", "helvetica", "helvetica neue", "calibri":
		return "sans-serif"
	case "courier", "courier new":
		return "monospace"
	}
	return "serif"
}

// matchFace implements the CSS Fonts style and weight matching steps.
func matchFace(faces []*Face, weight int, italic bool) *Face {
	if len(faces) == 0 {
		return nil
	}
	var pool []*Face
	for _, f := range faces {
		if f.Italic == italic {
			pool = append(pool, f)
		}
	}
	if len(pool) == 0 {
		pool = faces
	}
	best := pool[0]
	bestScore := weightScore(best.Weight, weight)
	for _, f := range pool[1:] {
		if s := weightScore(f.Weight, weight); s < bestScore {
			best, bestScore = f, s
		}
	}
	return best
}

// weightScore ranks a face weight against the desired weight (lower is better).
func weightScore(have, want int) int {
	d := have - want
	switch {
	case d == 0:
		return 0
	case want >= 400 && want <= 500:
		if d > 0 && have <= 500 {
			return d
		}
		if d < 0 {
			return 1000 - d
		}
		return 2000 + d
	case want < 400:
		if d < 0 {
			return -d
		}
		return 1000 + d
	default:
		if d > 0 {
			return d
		}
		return 1000 - d
	}
}

// Run is a piece of text drawn with one face.
type Run struct {
	Text string
	Face *Face
}

// Split breaks text into runs, choosing per rune the first face in chain that
// has a glyph, then any registered face that has it.
func (r *Resolver) Split(text string, chain []*Face) []Run {
	var runs []Run
	var cur *Face
	var b strings.Builder
	for _, ch := range text {
		var f *Face
		if cur != nil && (unicode.IsSpace(ch) || unicode.Is(unicode.Mn, ch) || cur.Has(ch)) {
			f = cur
		} else {
			f = r.faceFor(ch, chain)
		}
		if f != cur && b.Len() > 0 {
			runs = append(runs, Run{Text: b.String(), Face: cur})
			b.Reset()
		}
		cur = f
		b.WriteRune(ch)
	}
	if b.Len() > 0 {
		runs = append(runs, Run{Text: b.String(), Face: cur})
	}
	return runs
}

func (r *Resolver) faceFor(ch rune, chain []*Face) *Face {
	for _, f := range chain {
		if f.Has(ch) {
			return f
		}
	}
	r.mu.RLock()
	f, ok := r.runeFace[ch]
	r.mu.RUnlock()
	if ok {
		if f != nil {
			return f
		}
		return chain[0]
	}
	r.mu.RLock()
	all := r.all
	r.mu.RUnlock()
	for _, cand := range all {
		if cand.Has(ch) {
			f = cand
			break
		}
	}
	r.mu.Lock()
	r.runeFace[ch] = f
	r.mu.Unlock()
	if f == nil {
		return chain[0]
	}
	return f
}
