// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package core

import (
	"context"
	"fmt"
	"io"
	"slices"

	"golang.org/x/net/html"
)

// OutputFormat identifies the format a document is converted to.
type OutputFormat string

// Built-in output formats. PDF is what every Converter writes; the others are
// produced by a Writer from a parsed document.
const (
	OutputPDF  OutputFormat = "pdf"
	OutputXLSX OutputFormat = "xlsx"
)

// String returns the output format name.
func (o OutputFormat) String() string { return string(o) }

// Document is a source document parsed into an HTML tree, the shared model
// every output writer reads.
type Document struct {
	Root     *html.Node
	BaseURL  string   // overrides Config.BaseURL when set
	ThemeCSS []string // format-specific author CSS applied before the document's own styles
}

// Parser is implemented by converters whose input becomes an HTML tree
// (HTML, Markdown). Converters that implement it can feed every Writer.
type Parser interface {
	Parse(ctx context.Context, src io.Reader, cfg *Config) (*Document, error)
}

// Writer writes a parsed document in one output format other than PDF.
type Writer interface {
	// Output returns the format this writer produces.
	Output() OutputFormat
	// Extension returns the file extension of the output, without dot.
	Extension() string
	// MIMEType returns the media type of the output.
	MIMEType() string
	// Write renders doc to dst.
	Write(ctx context.Context, doc *Document, cfg *Config, dst io.Writer) error
}

// RegisterWriter adds or replaces the writer for w.Output().
func (r *Registry) RegisterWriter(w Writer) {
	if w == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.writers == nil {
		r.writers = map[OutputFormat]Writer{}
	}
	r.writers[w.Output()] = w
}

// LookupWriter returns the writer registered for output format o.
func (r *Registry) LookupWriter(o OutputFormat) (Writer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.writers[o]
	if !ok {
		return nil, fmt.Errorf("%w: output %q", ErrUnknownFormat, o)
	}
	return w, nil
}

// Outputs lists the output formats: PDF first, then each registered writer.
func (r *Registry) Outputs() []OutputFormat {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []OutputFormat{OutputPDF}
	for o := range r.writers {
		if o != OutputPDF {
			out = append(out, o)
		}
	}
	slices.Sort(out[1:])
	return out
}

// OutputForPath detects the output format from a file name: a registered
// writer's extension, else PDF.
func (r *Registry) OutputForPath(path string) OutputFormat {
	ext := extOf(path)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for o, w := range r.writers {
		if normalizeExt(w.Extension()) == ext {
			return o
		}
	}
	return OutputPDF
}

// ConvertTo converts src of format f to output format o. PDF goes through
// the format's Converter; other outputs need a Converter that implements
// Parser and a registered Writer.
func (r *Registry) ConvertTo(ctx context.Context, f Format, o OutputFormat, src io.Reader, dst io.Writer, opts ...Option) (err error) {
	c, err := r.Lookup(f)
	if err != nil {
		return err
	}
	if o == "" || o == OutputPDF {
		return c.Convert(ctx, src, dst, opts...)
	}
	w, err := r.LookupWriter(o)
	if err != nil {
		return err
	}
	p, ok := c.(Parser)
	if !ok {
		return fmt.Errorf("%w: %s cannot be converted to %s", ErrUnknownFormat, f, o)
	}
	defer Recover(&err)
	cfg := NewConfig(opts...)
	doc, err := p.Parse(ctx, src, cfg)
	if err != nil {
		return err
	}
	return w.Write(ctx, doc, cfg, dst)
}
