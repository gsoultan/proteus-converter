package fonts

import (
	"encoding/binary"
	"errors"
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

// collectionOffsets returns the font offsets inside data (one entry for a plain font).
func collectionOffsets(data []byte) ([]uint32, error) {
	if len(data) < 12 {
		return nil, errNotSfnt
	}
	if string(data[:4]) == "ttcf" {
		n := binary.BigEndian.Uint32(data[8:12])
		if n > 256 || len(data) < 12+int(n)*4 {
			return nil, errNotSfnt
		}
		out := make([]uint32, n)
		for i := range out {
			out[i] = binary.BigEndian.Uint32(data[12+i*4:])
		}
		return out, nil
	}
	return []uint32{0}, nil
}

func parseSfntInfo(data []byte, off uint32) (*sfntInfo, error) {
	if int(off)+12 > len(data) {
		return nil, errNotSfnt
	}
	tag := binary.BigEndian.Uint32(data[off:])
	if tag != 0x00010000 && tag != 0x74727565 && tag != 0x4F54544F { // 1.0, 'true', 'OTTO'
		return nil, errNotSfnt
	}
	numTables := int(binary.BigEndian.Uint16(data[off+4:]))
	tables := map[string][]byte{}
	for i := range numTables {
		rec := int(off) + 12 + i*16
		if rec+16 > len(data) {
			return nil, errNotSfnt
		}
		t := string(data[rec : rec+4])
		o := binary.BigEndian.Uint32(data[rec+8:])
		l := binary.BigEndian.Uint32(data[rec+12:])
		if uint64(o)+uint64(l) > uint64(len(data)) {
			continue
		}
		tables[t] = data[o : o+l]
	}
	info := &sfntInfo{weight: 400}
	_, info.hasGlyf = tables["glyf"]
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
