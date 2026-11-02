// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

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
func (c *Converter) Convert(ctx context.Context, src io.Reader, dst io.Writer, opts ...core.Option) (err error) {
	defer core.Recover(&err)
	cfg := core.NewConfig(opts...)
	doc, err := c.Parse(ctx, src, cfg)
	if err != nil {
		return err
	}
	return pipeline.Render(ctx, pipeline.Input{Doc: doc.Root}, cfg, dst)
}

// Parse implements core.Parser, so HTML can be written to every output format.
func (*Converter) Parse(_ context.Context, src io.Reader, _ *core.Config) (*core.Document, error) {
	doc, err := html.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("proteus: parse HTML: %w", err)
	}
	return &core.Document{Root: doc}, nil
}
