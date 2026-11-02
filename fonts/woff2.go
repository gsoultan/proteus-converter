// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package fonts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sort"

	"github.com/andybalholm/brotli"
)

var errBadWOFF2 = errors.New("fonts: malformed WOFF2")

// woff2Tags are the known table tags of the WOFF2 table directory, by index.
var woff2Tags = [63]string{"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post", "cvt ", "fpgm", "glyf",
	"loca", "prep", "CFF ", "VORG", "EBDT", "EBLC", "gasp", "hdmx", "kern", "LTSH", "PCLT", "VDMX", "vhea", "vmtx",
	"BASE", "GDEF", "GPOS", "GSUB", "EBSC", "JSTF", "MATH", "CBDT", "CBLC", "COLR", "CPAL", "SVG ", "sbix", "acnt",
	"avar", "bdat", "bloc", "bsln", "cvar", "fdsc", "feat", "fmtx", "fvar", "gvar", "hsty", "just", "lcar", "mort",
	"morx", "opbd", "prop", "trak", "Zapf", "Silf", "Glat", "Gloc", "Feat", "Sill"}

// woffReader reads the variable-length integers of WOFF2.
type woffReader struct {
	b   []byte
	err bool
}

func (r *woffReader) u8() int {
	if len(r.b) < 1 {
		r.err = true
		return 0
	}
	v := r.b[0]
	r.b = r.b[1:]
	return int(v)
}

func (r *woffReader) u16() int {
	if len(r.b) < 2 {
		r.err = true
		return 0
	}
	v := binary.BigEndian.Uint16(r.b)
	r.b = r.b[2:]
	return int(v)
}

func (r *woffReader) u32() int {
	if len(r.b) < 4 {
		r.err = true
		return 0
	}
	v := binary.BigEndian.Uint32(r.b)
	r.b = r.b[4:]
	return int(v)
}

func (r *woffReader) take(n int) []byte {
	if n < 0 || len(r.b) < n {
		r.err = true
		return nil
	}
	v := r.b[:n]
	r.b = r.b[n:]
	return v
}

// base128 reads a UIntBase128.
func (r *woffReader) base128() int {
	v := 0
	for i := 0; i < 5; i++ {
		c := r.u8()
		if r.err || (i == 0 && c == 0x80) || v&0xFE000000 != 0 {
			r.err = true
			return 0
		}
		v = v<<7 | c&0x7f
		if c&0x80 == 0 {
			return v
		}
	}
	r.err = true
	return 0
}

// u255 reads a 255UInt16.
func (r *woffReader) u255() int {
	switch c := r.u8(); c {
	case 253:
		return r.u16()
	case 255:
		return r.u8() + 253
	case 254:
		return r.u8() + 506
	default:
		return c
	}
}

// DecodeWOFF2 converts a WOFF 2.0 font into the plain sfnt font it wraps,
// reversing the glyf/loca and hmtx transforms. Font collections are not
// supported.
func DecodeWOFF2(data []byte) ([]byte, error) {
	if !IsWOFF2(data) || len(data) < 48 {
		return nil, errBadWOFF2
	}
	r := &woffReader{b: data[4:]}
	flavor := r.u32()
	r.u32() // length
	n := r.u16()
	r.u16()
	r.u32() // totalSfntSize
	compSize := r.u32()
	r.take(24) // versions, metadata and private blocks
	if n == 0 || flavor == 0x74746366 /* ttcf */ {
		return nil, errBadWOFF2
	}
	type table struct {
		tag       string
		origLen   int
		len       int // length in the decompressed stream
		transform bool
		data      []byte
	}
	tables := make([]*table, n)
	total := 0
	for i := range tables {
		flags := r.u8()
		t := &table{}
		if idx := flags & 0x3f; idx == 63 {
			t.tag = string(r.take(4))
		} else {
			t.tag = woff2Tags[idx]
		}
		version := flags >> 6
		t.origLen = r.base128()
		t.len = t.origLen
		if t.tag == "glyf" || t.tag == "loca" {
			t.transform = version == 0
		} else {
			t.transform = version != 0
		}
		if t.transform {
			t.len = r.base128()
		}
		if r.err || t.origLen > maxWOFFSize || t.len > maxWOFFSize {
			return nil, errBadWOFF2
		}
		total += t.len
		tables[i] = t
	}
	if r.err || compSize > len(r.b) || total > maxWOFFSize {
		return nil, errBadWOFF2
	}
	stream, err := io.ReadAll(io.LimitReader(brotli.NewReader(bytes.NewReader(r.b[:compSize])), int64(total)+1))
	if err != nil || len(stream) < total {
		return nil, errBadWOFF2
	}
	byTag := map[string]*table{}
	off := 0
	for _, t := range tables {
		t.data = stream[off : off+t.len]
		off += t.len
		byTag[t.tag] = t
	}

	var xMins []int16
	if g, l := byTag["glyf"], byTag["loca"]; g != nil && g.transform {
		if l == nil {
			return nil, errBadWOFF2
		}
		glyf, loca, mins, err := decodeGlyf(g.data)
		if err != nil {
			return nil, err
		}
		g.data, l.data, xMins = glyf, loca, mins
	}
	if h := byTag["hmtx"]; h != nil && h.transform {
		hhea, maxp := byTag["hhea"], byTag["maxp"]
		if hhea == nil || maxp == nil || len(hhea.data) < 36 || len(maxp.data) < 6 || xMins == nil {
			return nil, errBadWOFF2
		}
		numH := int(binary.BigEndian.Uint16(hhea.data[34:]))
		numG := int(binary.BigEndian.Uint16(maxp.data[4:]))
		hm, err := decodeHmtx(h.data, numH, numG, xMins)
		if err != nil {
			return nil, err
		}
		h.data = hm
	}

	sort.Slice(tables, func(a, b int) bool { return tables[a].tag < tables[b].tag })
	be := binary.BigEndian
	size := 12 + 16*n
	for _, t := range tables {
		size += (len(t.data) + 3) &^ 3
	}
	if size > maxWOFFSize {
		return nil, errBadWOFF2
	}
	out := make([]byte, 12+16*n, size)
	be.PutUint32(out, uint32(flavor))
	be.PutUint16(out[4:], uint16(n))
	sr, es := 1, 0
	for sr*2 <= n {
		sr *= 2
		es++
	}
	be.PutUint16(out[6:], uint16(sr*16))
	be.PutUint16(out[8:], uint16(es))
	be.PutUint16(out[10:], uint16(n*16-sr*16))
	for i, t := range tables {
		rec := out[12+i*16:]
		copy(rec, t.tag)
		be.PutUint32(rec[4:], checksum(t.data))
		be.PutUint32(rec[8:], uint32(len(out)))
		be.PutUint32(rec[12:], uint32(len(t.data)))
		out = append(out, t.data...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	return out, nil
}

func checksum(b []byte) uint32 {
	var s uint32
	for i := 0; i < len(b); i += 4 {
		var w [4]byte
		copy(w[:], b[i:])
		s += binary.BigEndian.Uint32(w[:])
	}
	return s
}

// decodeGlyf rebuilds the glyf and loca tables from the transformed glyf
// table (WOFF2 section 5.1) and returns each glyph's xMin for hmtx.
func decodeGlyf(b []byte) (glyf, loca []byte, xMins []int16, err error) {
	h := &woffReader{b: b}
	h.u16()
	opts := h.u16()
	numGlyphs := h.u16()
	indexFormat := h.u16()
	var sizes [7]int
	for i := range sizes {
		sizes[i] = h.u32()
	}
	if h.err {
		return nil, nil, nil, errBadWOFF2
	}
	var st [7]*woffReader
	for i, sz := range sizes {
		st[i] = &woffReader{b: h.take(sz)}
	}
	var overlap []byte
	if opts&1 != 0 {
		overlap = h.take((numGlyphs + 7) / 8)
	}
	if h.err {
		return nil, nil, nil, errBadWOFF2
	}
	nContour, nPoints, flagS, glyphS, compS, bboxS, instrS := st[0], st[1], st[2], st[3], st[4], st[5], st[6]
	bboxBitmap := bboxS.take(4 * ((numGlyphs + 31) / 32))
	if bboxS.err {
		return nil, nil, nil, errBadWOFF2
	}
	be := binary.BigEndian
	offsets := make([]int, numGlyphs+1)
	xMins = make([]int16, numGlyphs)
	var out []byte
	put16 := func(v int) { out = be.AppendUint16(out, uint16(v)) }
	for i := 0; i < numGlyphs; i++ {
		offsets[i] = len(out)
		nc := int(int16(nContour.u16()))
		hasBox := bboxBitmap[i>>3]&(0x80>>(i&7)) != 0
		var box [4]int
		if hasBox {
			for k := range box {
				box[k] = int(int16(bboxS.u16()))
			}
		}
		switch {
		case nc == 0:
			if hasBox {
				return nil, nil, nil, errBadWOFF2
			}
		case nc < 0:
			if !hasBox {
				return nil, nil, nil, errBadWOFF2
			}
			start := compS.b
			size, instr := 0, false
			for more := true; more; {
				fl := compS.u16()
				compS.u16()
				n := 2
				if fl&0x0001 != 0 {
					n = 4
				}
				switch {
				case fl&0x0008 != 0:
					n += 2
				case fl&0x0040 != 0:
					n += 4
				case fl&0x0080 != 0:
					n += 8
				}
				compS.take(n)
				size += 4 + n
				more = fl&0x0020 != 0
				instr = instr || fl&0x0100 != 0
				if compS.err {
					return nil, nil, nil, errBadWOFF2
				}
			}
			put16(-1)
			for _, v := range box {
				put16(v)
			}
			out = append(out, start[:size]...)
			if instr {
				il := glyphS.u255()
				put16(il)
				out = append(out, instrS.take(il)...)
			}
		default:
			ends := make([]int, nc)
			total := 0
			for c := range ends {
				total += nPoints.u255()
				ends[c] = total - 1
			}
			if nPoints.err || total > 1<<16 {
				return nil, nil, nil, errBadWOFF2
			}
			xs, ys := make([]int, total), make([]int, total)
			on := make([]bool, total)
			x, y := 0, 0
			for p := 0; p < total; p++ {
				f := flagS.u8()
				on[p] = f&0x80 == 0
				f &= 0x7f
				dx, dy := tripletDelta(f, glyphS)
				x += dx
				y += dy
				xs[p], ys[p] = x, y
			}
			if flagS.err || glyphS.err {
				return nil, nil, nil, errBadWOFF2
			}
			il := glyphS.u255()
			ins := instrS.take(il)
			if !hasBox && total > 0 {
				box = [4]int{xs[0], ys[0], xs[0], ys[0]}
				for p := range xs {
					box[0], box[2] = min(box[0], xs[p]), max(box[2], xs[p])
					box[1], box[3] = min(box[1], ys[p]), max(box[3], ys[p])
				}
			}
			put16(nc)
			for _, v := range box {
				put16(v)
			}
			for _, e := range ends {
				put16(e)
			}
			put16(il)
			out = append(out, ins...)
			ov := overlap != nil && overlap[i>>3]&(0x80>>(i&7)) != 0
			flagAt := len(out)
			out = append(out, make([]byte, total)...)
			var coords []byte
			px := 0
			for p := range xs {
				f := byte(0)
				if on[p] {
					f |= 0x01
				}
				if p == 0 && ov {
					f |= 0x40
				}
				f |= coordFlag(xs[p]-px, 0x02, 0x10)
				out[flagAt+p] = f
				px = xs[p]
			}
			px = 0
			for p := range xs {
				coords = appendCoord(coords, xs[p]-px)
				px = xs[p]
			}
			py := 0
			for p := range ys {
				d := ys[p] - py
				out[flagAt+p] |= coordFlag(d, 0x04, 0x20)
				coords = appendCoord(coords, d)
				py = ys[p]
			}
			out = append(out, coords...)
		}
		xMins[i] = int16(box[0])
		if instrS.err || glyphS.err || compS.err || bboxS.err {
			return nil, nil, nil, errBadWOFF2
		}
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
		if len(out) > maxWOFFSize {
			return nil, nil, nil, errBadWOFF2
		}
	}
	offsets[numGlyphs] = len(out)
	if indexFormat == 0 {
		if len(out) > 0x1fffe {
			return nil, nil, nil, errBadWOFF2
		}
		for _, o := range offsets {
			loca = be.AppendUint16(loca, uint16(o/2))
		}
	} else {
		for _, o := range offsets {
			loca = be.AppendUint32(loca, uint32(o))
		}
	}
	return out, loca, xMins, nil
}

// coordFlag returns the glyf flag bits for a coordinate delta: short (one
// byte) with the sign in the same-bit, or same (zero delta).
func coordFlag(d int, short, same byte) byte {
	switch {
	case d == 0:
		return same
	case d > 0 && d < 256:
		return short | same
	case d < 0 && d > -256:
		return short
	}
	return 0
}

func appendCoord(b []byte, d int) []byte {
	switch {
	case d == 0:
		return b
	case d > -256 && d < 256:
		if d < 0 {
			d = -d
		}
		return append(b, byte(d))
	}
	return binary.BigEndian.AppendUint16(b, uint16(int16(d)))
}

// tripletDelta decodes one point delta of the WOFF2 glyph stream.
func tripletDelta(f int, r *woffReader) (dx, dy int) {
	sign := func(fl, v int) int {
		if fl&1 != 0 {
			return v
		}
		return -v
	}
	switch {
	case f < 10:
		dy = sign(f, (f&14)<<7+r.u8())
	case f < 20:
		dx = sign(f, ((f-10)&14)<<7+r.u8())
	case f < 84:
		b0, b1 := f-20, r.u8()
		dx = sign(f, 1+(b0&0x30)+b1>>4)
		dy = sign(f>>1, 1+(b0&0x0c)<<2+b1&0x0f)
	case f < 120:
		b0 := f - 84
		dx = sign(f, 1+(b0/12)<<8+r.u8())
		dy = sign(f>>1, 1+((b0%12)>>2)<<8+r.u8())
	case f < 124:
		b0, b1, b2 := r.u8(), r.u8(), r.u8()
		dx = sign(f, b0<<4+b1>>4)
		dy = sign(f>>1, (b1&0x0f)<<8+b2)
	default:
		a, b, c, d := r.u8(), r.u8(), r.u8(), r.u8()
		dx = sign(f, a<<8+b)
		dy = sign(f>>1, c<<8+d)
	}
	return dx, dy
}

// decodeHmtx rebuilds an hmtx table from its transformed form (WOFF2
// section 5.4): omitted side bearings equal the glyphs' xMin.
func decodeHmtx(b []byte, numH, numG int, xMins []int16) ([]byte, error) {
	r := &woffReader{b: b}
	flags := r.u8()
	if numH < 1 || numH > numG || len(xMins) < numG {
		return nil, errBadWOFF2
	}
	adv := make([]int, numH)
	for i := range adv {
		adv[i] = r.u16()
	}
	lsb := make([]int, numG)
	for i := range numG {
		switch {
		case i < numH && flags&1 == 0, i >= numH && flags&2 == 0:
			lsb[i] = int(int16(r.u16()))
		default:
			lsb[i] = int(xMins[i])
		}
	}
	if r.err {
		return nil, errBadWOFF2
	}
	var out []byte
	for i := range numG {
		if i < numH {
			out = binary.BigEndian.AppendUint16(out, uint16(adv[i]))
		}
		out = binary.BigEndian.AppendUint16(out, uint16(int16(lsb[i])))
	}
	return out, nil
}
