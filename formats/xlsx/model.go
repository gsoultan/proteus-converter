// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"fmt"
	"strings"
)

// workbook is the in-memory spreadsheet that the extractor fills and the
// OOXML writer serialises.
type workbook struct {
	sheets []*sheet
	styles *styleTable
	sst    *sharedStrings
}

type sheet struct {
	name       string
	rows       []*row // rows[i] is row i+1; nil rows are empty
	cols       []colSpec
	merges     []cellRange
	freezeRows int
	links      []hyperlink
	images     []anchoredImage
	// src names where the sheet came from, for warnings.
	src string
}

type colSpec struct {
	width  float64 // in characters of the default font; 0 means default
	hidden bool
}

type row struct {
	cells    []*cell // cells[i] is column i+1; nil cells are empty
	heightPt float64 // 0 means default
	hidden   bool
}

type cellKind int

const (
	kindEmpty cellKind = iota
	kindString
	kindNumber
	kindBool
	kindFormula
)

type cell struct {
	kind    cellKind
	text    string // kindString: value; kindFormula: cached text, if any
	runs    []textRun
	num     float64
	boolean bool
	formula string
	xf      int // index into styleTable.xfs
}

// textRun is a piece of rich text with its own font.
type textRun struct {
	text string
	font font
}

type cellRange struct {
	r1, c1, r2, c2 int // 1-based, inclusive
}

func (r cellRange) String() string {
	return cellRef(r.r1, r.c1) + ":" + cellRef(r.r2, r.c2)
}

type hyperlink struct {
	r, c    int
	target  string
	tooltip string
}

type anchoredImage struct {
	r, c      int // 1-based anchor cell
	name      string
	ext       string // "png" or "jpeg"
	data      []byte
	wPx, hPx  float64
	offXEMU   int64
	offYEMU   int64
	rowHeight float64 // pt needed to show the image
}

// colName returns the column letters for a 1-based column number.
func colName(c int) string {
	var b []byte
	for c > 0 {
		c--
		b = append([]byte{byte('A' + c%26)}, b...)
		c /= 26
	}
	return string(b)
}

// cellRef returns the A1 reference of a 1-based row and column.
func cellRef(r, c int) string { return colName(c) + fmt.Sprint(r) }

func (s *sheet) rowAt(r int) *row {
	for len(s.rows) < r {
		s.rows = append(s.rows, nil)
	}
	if s.rows[r-1] == nil {
		s.rows[r-1] = &row{}
	}
	return s.rows[r-1]
}

func (s *sheet) cellAt(r, c int) *cell {
	rw := s.rowAt(r)
	for len(rw.cells) < c {
		rw.cells = append(rw.cells, nil)
	}
	if rw.cells[c-1] == nil {
		rw.cells[c-1] = &cell{}
	}
	return rw.cells[c-1]
}

func (s *sheet) colAt(c int) *colSpec {
	for len(s.cols) < c {
		s.cols = append(s.cols, colSpec{})
	}
	return &s.cols[c-1]
}

// Excel limits.
const (
	maxRows      = 1_048_576
	maxCols      = 16_384
	maxSheetName = 31
	maxCellChars = 32_767
)

// sheetName makes a name Excel accepts: no []:*?/\ characters, at most 31
// characters, not empty and not surrounded by apostrophes.
func sheetName(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '[', ']', ':', '*', '?', '/', '\\':
			return ' '
		}
		if r < 0x20 {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Trim(s, "'")
	if rs := []rune(s); len(rs) > maxSheetName {
		s = strings.TrimSpace(string(rs[:maxSheetName]))
	}
	return s
}

// uniqueSheetName returns name, or name with a numeric suffix, so that it
// differs (case-insensitively) from every name in used.
func uniqueSheetName(name string, used map[string]bool) string {
	if name == "" {
		name = "Sheet"
	}
	cand := name
	for i := 2; used[strings.ToLower(cand)]; i++ {
		suffix := fmt.Sprintf(" (%d)", i)
		base := []rune(name)
		if len(base)+len(suffix) > maxSheetName {
			base = base[:maxSheetName-len(suffix)]
		}
		cand = strings.TrimSpace(string(base)) + suffix
	}
	used[strings.ToLower(cand)] = true
	return cand
}
