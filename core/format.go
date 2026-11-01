// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package core

import (
	"path/filepath"
	"strings"
)

// Format identifies a source document format, e.g. "html" or "markdown".
type Format string

// Built-in formats. FormatDOCX is reserved for the upcoming Word converter.
const (
	FormatHTML     Format = "html"
	FormatMarkdown Format = "markdown"
	FormatDOCX     Format = "docx"
)

// String returns the format name.
func (f Format) String() string { return string(f) }

// normalizeExt lower-cases an extension and strips its leading dot.
func normalizeExt(ext string) string {
	return strings.TrimPrefix(strings.ToLower(ext), ".")
}

// extOf returns the normalized extension of a path.
func extOf(path string) string {
	return normalizeExt(filepath.Ext(path))
}
