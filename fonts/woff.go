package fonts

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"sort"
)

// maxWOFFSize bounds the decoded size of a WOFF font (hostile inputs).
const maxWOFFSize = 64 << 20

var errBadWOFF = errors.New("fonts: malformed WOFF")

// IsWOFF reports whether data is a WOFF 1.0 font; IsWOFF2 a WOFF 2.0 font.
func IsWOFF(data []byte) bool  { return len(data) >= 4 && string(data[:4]) == "wOFF" }
func IsWOFF2(data []byte) bool { return len(data) >= 4 && string(data[:4]) == "wOF2" }

// DecodeWOFF converts a WOFF 1.0 font into the plain sfnt (TrueType or
// OpenType) font it wraps.
func DecodeWOFF(data []byte) ([]byte, error) {
	if !IsWOFF(data) || len(data) < 44 {
		return nil, errBadWOFF
	}
	be := binary.BigEndian
	flavor := be.Uint32(data[4:])
	n := int(be.Uint16(data[12:]))
	if n == 0 || 44+n*20 > len(data) {
		return nil, errBadWOFF
	}
	type table struct {
		tag           []byte
		off, comp, ln int
		sum           uint32
	}
	tables := make([]table, n)
	total := 12 + 16*n
	for i := range n {
		e := data[44+i*20:]
		t := table{tag: e[:4], off: int(be.Uint32(e[4:])), comp: int(be.Uint32(e[8:])), ln: int(be.Uint32(e[12:])), sum: be.Uint32(e[16:])}
		if t.off < 0 || t.comp < 0 || t.ln < 0 || t.off+t.comp > len(data) || t.comp > t.ln {
			return nil, errBadWOFF
		}
		total += (t.ln + 3) &^ 3
		if total > maxWOFFSize {
			return nil, errBadWOFF
		}
		tables[i] = t
	}
	sort.Slice(tables, func(a, b int) bool { return bytes.Compare(tables[a].tag, tables[b].tag) < 0 })
	out := make([]byte, 12+16*n, total)
	be.PutUint32(out, flavor)
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
		raw := data[t.off : t.off+t.comp]
		if t.comp < t.ln {
			zr, err := zlib.NewReader(bytes.NewReader(raw))
			if err != nil {
				return nil, errBadWOFF
			}
			dec, err := io.ReadAll(io.LimitReader(zr, int64(t.ln)+1))
			zr.Close()
			if err != nil || len(dec) != t.ln {
				return nil, errBadWOFF
			}
			raw = dec
		}
		rec := out[12+i*16:]
		copy(rec, t.tag)
		be.PutUint32(rec[4:], t.sum)
		be.PutUint32(rec[8:], uint32(len(out)))
		be.PutUint32(rec[12:], uint32(t.ln))
		out = append(out, raw...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	return out, nil
}
