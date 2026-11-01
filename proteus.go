// Package proteus converts documents to PDF in pure Go, using gpdf as the PDF
// backend. HTML and Markdown are supported today; Microsoft Word (DOCX) is
// planned. Every format implements the same Converter interface and is
// registered in a Registry, so new formats plug in without API changes.
//
// Quick start:
//
//	err := proteus.ConvertFile(ctx, "README.md", "README.pdf",
//		proteus.WithPageSize(proteus.A4),
//		proteus.WithMargins(proteus.UniformMargins(proteus.Mm(20))))
package proteus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gsoultan/proteus-converter/core"
	htmlfmt "github.com/gsoultan/proteus-converter/formats/html"
	mdfmt "github.com/gsoultan/proteus-converter/formats/markdown"
)

// Re-exported core types so callers only import this package.
type (
	Converter      = core.Converter
	ConverterFunc  = core.ConverterFunc
	Format         = core.Format
	Option         = core.Option
	Config         = core.Config
	Registry       = core.Registry
	PageSize       = core.PageSize
	Margins        = core.Margins
	FontSource     = core.FontSource
	Metadata       = core.Metadata
	ResourceLoader = core.ResourceLoader
	LoaderFunc     = core.LoaderFunc
)

// Formats.
const (
	FormatHTML     = core.FormatHTML
	FormatMarkdown = core.FormatMarkdown
	FormatDOCX     = core.FormatDOCX
)

// Page sizes.
var (
	A3     = core.A3
	A4     = core.A4
	A5     = core.A5
	Letter = core.Letter
	Legal  = core.Legal
)

// Errors.
var (
	ErrUnknownFormat    = core.ErrUnknownFormat
	ErrResourceDenied   = core.ErrResourceDenied
	ErrResourceTooLarge = core.ErrResourceTooLarge
	ErrTooManyPages     = core.ErrTooManyPages
	ErrInternal         = core.ErrInternal
)

// InternalError is returned (wrapping ErrInternal) when a conversion hits a
// bug in proteus; the panic is recovered and never reaches the caller.
type InternalError = core.InternalError

// Options (see core for documentation).
var (
	WithPageSize           = core.WithPageSize
	WithMargins            = core.WithMargins
	WithBaseURL            = core.WithBaseURL
	WithFS                 = core.WithFS
	WithResourceLoader     = core.WithResourceLoader
	WithHTTP               = core.WithHTTP
	WithLocalFiles         = core.WithLocalFiles
	WithMaxResourceSize    = core.WithMaxResourceSize
	WithMaxPages           = core.WithMaxPages
	WithStylesheet         = core.WithStylesheet
	WithMediaType          = core.WithMediaType
	WithFont               = core.WithFont
	WithFontDir            = core.WithFontDir
	WithSystemFonts        = core.WithSystemFonts
	WithMetadata           = core.WithMetadata
	WithTitle              = core.WithTitle
	WithOutline            = core.WithOutline
	WithTagged             = core.WithTagged
	WithMarkdownStylesheet = core.WithMarkdownStylesheet
	WithoutMarkdownTheme   = core.WithoutMarkdownTheme
	WithWarningHandler     = core.WithWarningHandler
)

// Unit helpers and constructors.
var (
	Mm             = core.Mm
	Cm             = core.Cm
	Inch           = core.Inch
	Px             = core.Px
	UniformMargins = core.UniformMargins
	NewRegistry    = core.NewRegistry
)

var defaultRegistry = func() *core.Registry {
	r := core.NewRegistry()
	r.Register(htmlfmt.New())
	r.Register(mdfmt.New())
	return r
}()

// Default returns the process-wide registry with the built-in formats.
func Default() *Registry { return defaultRegistry }

// Register adds a converter to the default registry (e.g. a custom format).
func Register(c Converter) { defaultRegistry.Register(c) }

// Lookup returns the converter for a format from the default registry.
func Lookup(f Format) (Converter, error) { return defaultRegistry.Lookup(f) }

// FormatForPath detects a format from a file name.
func FormatForPath(path string) (Format, error) { return defaultRegistry.FormatForPath(path) }

// Convert reads a document of the given format from src and writes a PDF to dst.
func Convert(ctx context.Context, f Format, src io.Reader, dst io.Writer, opts ...Option) error {
	c, err := Lookup(f)
	if err != nil {
		return err
	}
	return c.Convert(ctx, src, dst, opts...)
}

// ConvertBytes converts an in-memory document and returns the PDF bytes.
func ConvertBytes(ctx context.Context, f Format, src []byte, opts ...Option) ([]byte, error) {
	var out bytes.Buffer
	if err := Convert(ctx, f, bytes.NewReader(src), &out, opts...); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// ConvertString converts a document held in a string and returns the PDF bytes.
func ConvertString(ctx context.Context, f Format, src string, opts ...Option) ([]byte, error) {
	return ConvertBytes(ctx, f, []byte(src), opts...)
}

// ConvertFile converts the file at in and writes the PDF to out. The format is
// detected from the input extension and relative resources resolve against
// the input's directory unless WithBaseURL is given.
func ConvertFile(ctx context.Context, in, out string, opts ...Option) error {
	f, err := FormatForPath(in)
	if err != nil {
		return err
	}
	src, err := os.Open(in)
	if err != nil {
		return err
	}
	defer src.Close()
	abs, err := filepath.Abs(in)
	if err != nil {
		return err
	}
	opts = append([]Option{WithBaseURL(filepath.Dir(abs) + string(filepath.Separator))}, opts...)
	tmp, err := os.CreateTemp(filepath.Dir(out), "."+strings.TrimSuffix(filepath.Base(out), filepath.Ext(out))+"-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := Convert(ctx, f, src, tmp, opts...); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), out); err != nil {
		return fmt.Errorf("proteus: write %s: %w", out, err)
	}
	return nil
}
