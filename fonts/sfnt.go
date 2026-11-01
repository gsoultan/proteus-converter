package fonts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"unicode/utf16"
)

// sfntInfo is the subset of an sfnt font's tables needed for matching and line metrics.
type sfntInfo struct {
	family       string
	altFamily    string // name ID 1 when a typographic family (ID 16) exists
	subfamily    string
	psName       string
	weight       int
	italic       bool
	unitsPerEm   int
	ascent       int // hhea, or OS/2 typo when USE_TYPO_METRICS
	descent      int // positive
	lineGap      int
	underlinePos int // negative below baseline
	underlineThk int
	strikePos    int
	strikeThk    int
	xHeight      int
	capHeight    int
	hasGlyf      bool
}

var errNotSfnt = errors.New("fonts: not a TrueType font")

// readAt reads n bytes at off, or returns nil when they are not all there.
func readAt(r io.ReaderAt, size int64, off, n int64) []byte {
	if off < 0 || n < 0 || off+n > size {
		return nil
	}
	b := make([]byte, n)
	if _, err := r.ReadAt(b, off); err != nil && !(errors.Is(err, io.EOF) && off+n == size) {
		return nil
	}
	return b
}

// collectionOffsets returns the font offsets inside data (one entry for a plain font).
func collectionOffsets(data []byte) ([]uint32, error) {
	return collectionOffsetsAt(bytes.NewReader(data), int64(len(data)))
}

func collectionOffsetsAt(r io.ReaderAt, size int64) ([]uint32, error) {
	hdr := readAt(r, size, 0, 12)
	if hdr == nil {
		return nil, errNotSfnt
	}
	if string(hdr[:4]) == "ttcf" {
		n := binary.BigEndian.Uint32(hdr[8:12])
		if n > 256 {
			return nil, errNotSfnt
		}
		dir := readAt(r, size, 12, int64(n)*4)
		if dir == nil {
			return nil, errNotSfnt
		}
		out := make([]uint32, n)
		for i := range out {
			out[i] = binary.BigEndian.Uint32(dir[i*4:])
		}
		return out, nil
	}
	return []uint32{0}, nil
}

func parseSfntInfo(data []byte, off uint32) (*sfntInfo, error) {
	return parseSfntInfoAt(bytes.NewReader(data), int64(len(data)), off)
}

// parseSfntInfoAt reads only the table directory and the small tables it
// needs (head, hhea, OS/2, post header, name), so scanning a font directory
// does not load every glyph outline into memory.
func parseSfntInfoAt(r io.ReaderAt, size int64, off uint32) (*sfntInfo, error) {
	hdr := readAt(r, size, int64(off), 12)
	if hdr == nil {
		return nil, errNotSfnt
	}
	tag := binary.BigEndian.Uint32(hdr)
	if tag != 0x00010000 && tag != 0x74727565 && tag != 0x4F54544F { // 1.0, 'true', 'OTTO'
		return nil, errNotSfnt
	}
	numTables := int64(binary.BigEndian.Uint16(hdr[4:]))
	recs := readAt(r, size, int64(off)+12, numTables*16)
	if recs == nil {
		return nil, errNotSfnt
	}
	// at most this many bytes are read from each table
	want := map[string]int64{"head": 54, "hhea": 36, "OS/2": 96, "post": 32, "name": 1 << 20}
	tables := map[string][]byte{}
	hasGlyf := false
	for i := range numTables {
		rec := recs[i*16:]
		t := string(rec[:4])
		o := int64(binary.BigEndian.Uint32(rec[8:]))
		l := int64(binary.BigEndian.Uint32(rec[12:]))
		if o+l > size {
			continue
		}
		if t == "glyf" || t == "CFF " {
			hasGlyf = true // outlines gpdf can embed (TrueType or OpenType-CFF)
		}
		if limit, ok := want[t]; ok {
			if b := readAt(r, size, o, min(l, limit)); b != nil {
				tables[t] = b
			}
		}
	}
	info := &sfntInfo{weight: 400, hasGlyf: hasGlyf}
	u16 := func(b []byte, i int) int {
		if i+2 > len(b) {
			return 0
		}
		return int(binary.BigEndian.Uint16(b[i:]))
	}
	i16 := func(b []byte, i int) int {
		if i+2 > len(b) {
			return 0
		}
		return int(int16(binary.BigEndian.Uint16(b[i:])))
	}
	head := tables["head"]
	info.unitsPerEm = u16(head, 18)
	if info.unitsPerEm == 0 {
		return nil, errNotSfnt
	}
	macStyle := u16(head, 44)
	if hhea := tables["hhea"]; len(hhea) >= 10 {
		info.ascent = i16(hhea, 4)
		info.descent = -i16(hhea, 6)
		info.lineGap = i16(hhea, 8)
	}
	if os2 := tables["OS/2"]; len(os2) >= 78 {
		info.weight = u16(os2, 4)
		fsSel := u16(os2, 62)
		info.italic = fsSel&1 != 0
		info.strikeThk = i16(os2, 26)
		info.strikePos = i16(os2, 28)
		typoA, typoD, typoG := i16(os2, 68), -i16(os2, 70), i16(os2, 72)
		if fsSel&(1<<7) != 0 || (info.ascent == 0 && info.descent == 0) {
			info.ascent, info.descent, info.lineGap = typoA, typoD, typoG
		}
		if len(os2) >= 90 {
			info.xHeight = i16(os2, 86)
			info.capHeight = i16(os2, 88)
		}
	} else {
		info.italic = macStyle&2 != 0
		if macStyle&1 != 0 {
			info.weight = 700
		}
	}
	if post := tables["post"]; len(post) >= 12 {
		info.underlinePos = i16(post, 8)
		info.underlineThk = i16(post, 10)
	}
	parseNames(tables["name"], info)
	if info.family == "" {
		return nil, errNotSfnt
	}
	return info, nil
}

func parseNames(b []byte, info *sfntInfo) {
	if len(b) < 6 {
		return
	}
	count := int(binary.BigEndian.Uint16(b[2:]))
	strOff := int(binary.BigEndian.Uint16(b[4:]))
	names := map[int]string{}
	prio := map[int]int{}
	for i := range count {
		r := 6 + i*12
		if r+12 > len(b) {
			break
		}
		platform := binary.BigEndian.Uint16(b[r:])
		lang := binary.BigEndian.Uint16(b[r+4:])
		id := int(binary.BigEndian.Uint16(b[r+6:]))
		l := int(binary.BigEndian.Uint16(b[r+8:]))
		o := strOff + int(binary.BigEndian.Uint16(b[r+10:]))
		if o+l > len(b) || (id != 1 && id != 2 && id != 4 && id != 6 && id != 16 && id != 17) {
			continue
		}
		raw := b[o : o+l]
		var s string
		p := 0
		switch platform {
		case 3, 0:
			u := make([]uint16, len(raw)/2)
			for j := range u {
				u[j] = binary.BigEndian.Uint16(raw[j*2:])
			}
			s = string(utf16.Decode(u))
			p = 2
			if platform == 3 && lang == 0x409 {
				p = 3
			}
		case 1:
			s = string(raw)
			p = 1
		default:
			continue
		}
		if p > prio[id] {
			names[id] = s
			prio[id] = p
		}
	}
	info.family = names[1]
	if names[16] != "" && names[16] != names[1] {
		info.family, info.altFamily = names[16], names[1]
	}
	info.subfamily = names[2]
	if names[17] != "" {
		info.subfamily = names[17]
	}
	info.psName = names[6]
}
