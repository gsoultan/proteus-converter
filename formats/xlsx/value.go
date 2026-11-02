// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// typed is a detected cell value: how the text should be stored so that a
// spreadsheet shows exactly what the document showed and can calculate with it.
type typed struct {
	kind    cellKind
	num     float64
	boolean bool
	numFmt  string
}

var (
	// 1,234.56 or 1234.56 or (1,234.56) with an optional sign, % and a
	// currency symbol before or after.
	reNumber = regexp.MustCompile(`^(?P<open>\()?(?P<sign>[-+−])?(?P<cur>[$€£¥₹]|USD|EUR|GBP)?\s?(?P<int>\d{1,3}(?:,\d{3})+|\d+)(?P<frac>\.\d+)?(?P<pct>\s?%)?(?P<cur2>\s?(?:[$€£¥₹]|USD|EUR|GBP))?(?P<close>\))?$`)
	// Rupiah amounts use a dot as thousands separator and a comma for decimals.
	reRupiah  = regexp.MustCompile(`^(?P<sign>[-+−])?(?P<cur>Rp|IDR)\.?\s?(?P<int>\d{1,3}(?:\.\d{3})+|\d+)(?P<frac>,\d+)?$`)
	reISODate = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})(?:[T ](\d{2}):(\d{2})(?::(\d{2}))?)?$`)
)

// excelEpoch is day 0 of the 1900 date system (with its leap-year bug, day 1
// is 1900-01-01 and 1900-03-01 is day 61).
var excelEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

// detect classifies trimmed cell text. Plain text returns kindString.
func detect(text string) typed {
	t := strings.TrimSpace(text)
	if t == "" || len(t) > 64 || strings.ContainsAny(t, "\n\t") {
		return typed{kind: kindString}
	}
	if m := reISODate.FindStringSubmatch(t); m != nil {
		return detectDate(m)
	}
	if m := reRupiah.FindStringSubmatch(t); m != nil {
		return detectRupiah(m)
	}
	m := reNumber.FindStringSubmatch(t)
	if m == nil {
		return typed{kind: kindString}
	}
	g := func(name string) string { return m[reNumber.SubexpIndex(name)] }
	open, closeP := g("open") != "", g("close") != ""
	if open != closeP {
		return typed{kind: kindString}
	}
	cur := g("cur")
	curAfter := strings.TrimSpace(g("cur2"))
	if cur != "" && curAfter != "" {
		return typed{kind: kindString}
	}
	if open && (g("sign") != "") {
		return typed{kind: kindString}
	}
	intPart, frac := g("int"), g("frac")
	// A leading zero like "007" or "0123" is an identifier, not a number.
	if len(intPart) > 1 && intPart[0] == '0' && !strings.Contains(intPart, ",") {
		return typed{kind: kindString}
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(intPart, ",", "")+frac, 64)
	if err != nil || math.IsInf(v, 0) || math.IsNaN(v) {
		return typed{kind: kindString}
	}
	neg := open || g("sign") == "-" || g("sign") == "−"
	if neg {
		v = -v
	}
	grouped := strings.Contains(intPart, ",")
	decimals := 0
	if frac != "" {
		decimals = len(frac) - 1
	}
	fmtCode := numberFormat(grouped, decimals)
	switch {
	case g("pct") != "":
		v /= 100
		fmtCode = numberFormat(false, decimals)
		if fmtCode == "" {
			fmtCode = "0"
		}
		fmtCode += "%"
	case cur != "" || curAfter != "":
		if fmtCode == "" {
			fmtCode = "0"
		}
		if cur != "" {
			fmtCode = `"` + cur + `"` + fmtCode
		} else {
			fmtCode += `"` + g("cur2") + `"`
		}
	}
	if open {
		if fmtCode == "" {
			fmtCode = "0"
		}
		// Accounting negatives: show positives plainly and negatives in parentheses.
		fmtCode = fmtCode + ";(" + fmtCode + ")"
	}
	return typed{kind: kindNumber, num: v, numFmt: fmtCode}
}

func detectRupiah(m []string) typed {
	g := func(name string) string { return m[reRupiah.SubexpIndex(name)] }
	intPart, frac := g("int"), g("frac")
	if len(intPart) > 1 && intPart[0] == '0' && !strings.Contains(intPart, ".") {
		return typed{kind: kindString}
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(intPart, ".", "")+strings.Replace(frac, ",", ".", 1), 64)
	if err != nil || math.IsInf(v, 0) || math.IsNaN(v) {
		return typed{kind: kindString}
	}
	if g("sign") == "-" || g("sign") == "−" {
		v = -v
	}
	decimals := 0
	if frac != "" {
		decimals = len(frac) - 1
	}
	code := numberFormat(strings.Contains(intPart, "."), decimals)
	if code == "" {
		code = "0"
	}
	return typed{kind: kindNumber, num: v, numFmt: `"` + g("cur") + `"` + code}
}

func numberFormat(grouped bool, decimals int) string {
	if !grouped && decimals == 0 {
		return "" // General shows plain integers as typed
	}
	var b strings.Builder
	if grouped {
		b.WriteString("#,##0")
	} else {
		b.WriteString("0")
	}
	if decimals > 0 {
		b.WriteByte('.')
		b.WriteString(strings.Repeat("0", decimals))
	}
	return b.String()
}

func detectDate(m []string) typed {
	y, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	d, _ := strconv.Atoi(m[3])
	if y < 1900 || mo < 1 || mo > 12 || d < 1 || d > 31 {
		return typed{kind: kindString}
	}
	t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
	if t.Day() != d || t.Month() != time.Month(mo) {
		return typed{kind: kindString}
	}
	code := "yyyy-mm-dd"
	if m[4] != "" {
		h, _ := strconv.Atoi(m[4])
		mi, _ := strconv.Atoi(m[5])
		s, _ := strconv.Atoi(m[6])
		if h > 23 || mi > 59 || s > 59 {
			return typed{kind: kindString}
		}
		t = t.Add(time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute + time.Duration(s)*time.Second)
		code = "yyyy-mm-dd hh:mm"
		if m[6] != "" {
			code += ":ss"
		}
	}
	return typed{kind: kindNumber, num: serial(t), numFmt: code}
}

// serial returns the Excel serial number of t (1900 date system).
func serial(t time.Time) float64 {
	days := t.Sub(excelEpoch).Hours() / 24
	if t.Before(time.Date(1900, 3, 1, 0, 0, 0, 0, time.UTC)) {
		days-- // Excel thinks 1900-02-29 exists
	}
	return days
}

// explicit parses the data-type / data-value attributes an author can put
// on a cell to override detection. ok is false when there is no override.
func explicit(typ, value, text string) (typed, bool) {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if value == "" {
		value = strings.TrimSpace(text)
	}
	switch typ {
	case "string", "text":
		return typed{kind: kindString}, true
	case "number", "numeric":
		v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(value), ",", ""), 64)
		if err != nil || math.IsInf(v, 0) || math.IsNaN(v) {
			return typed{kind: kindString}, true
		}
		return typed{kind: kindNumber, num: v}, true
	case "bool", "boolean":
		b, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(value)))
		if err != nil {
			return typed{kind: kindString}, true
		}
		return typed{kind: kindBool, boolean: b}, true
	case "date", "datetime":
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
			if t, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
				code := "yyyy-mm-dd"
				if layout != "2006-01-02" {
					code = "yyyy-mm-dd hh:mm:ss"
				}
				return typed{kind: kindNumber, num: serial(t.UTC()), numFmt: code}, true
			}
		}
		return typed{kind: kindString}, true
	case "formula":
		return typed{kind: kindFormula}, true
	}
	if typ == "" && value != text {
		// data-value without data-type: a number if it parses, else text.
		if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && !math.IsInf(v, 0) && !math.IsNaN(v) {
			return typed{kind: kindNumber, num: v}, true
		}
	}
	return typed{}, false
}
