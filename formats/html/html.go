// Package html converts HTML documents (with their CSS) to PDF.
package html

import (
	"context"
	"fmt"
	"io"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/pipeline"
	"golang.org/x/net/html"
)

// Converter converts HTML to PDF. The zero value is ready to use and safe for
// concurrent use.
type Converter struct{}

// New returns an HTML converter.
func New() *Converter { return &Converter{} }

// Format implements core.Converter.
func (*Converter) Format() core.Format { return core.FormatHTML }

// Extensions implements core.Converter.
func (*Converter) Extensions() []string { return []string{"html", "htm", "xhtml"} }

// Convert implements core.Converter.
func (*Converter) Convert(ctx context.Context, src io.Reader, dst io.Writer, opts ...core.Option) error {
	cfg := core.NewConfig(opts...)
	doc, err := html.Parse(src)
	if err != nil {
		return fmt.Errorf("proteus: parse HTML: %w", err)
	}
	return pipeline.Render(ctx, pipeline.Input{Doc: doc}, cfg, dst)
}
