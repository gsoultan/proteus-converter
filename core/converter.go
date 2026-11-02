// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

// Package core holds the contracts shared by every converter: the Converter
// strategy interface, the format registry, functional options and resource
// loading. Format implementations live in formats/*; most users should use the
// root package, which re-exports this API with the built-in formats registered.
package core

import (
	"context"
	"io"
)

// Converter converts one source format into PDF. Each format (HTML, Markdown,
// DOCX, ...) provides its own implementation; callers pick one through a
// Registry or construct it directly.
type Converter interface {
	// Format returns the source format this converter reads.
	Format() Format

	// Extensions returns the file extensions (without dot) handled by this
	// converter, used to detect the format from a file name.
	Extensions() []string

	// Convert reads a source document from src and writes a PDF to dst.
	Convert(ctx context.Context, src io.Reader, dst io.Writer, opts ...Option) error
}

// ConverterFunc adapts a plain function into a Converter, which is handy for
// tests and small custom formats.
type ConverterFunc struct {
	Name Format
	Exts []string
	Fn   func(ctx context.Context, src io.Reader, dst io.Writer, opts ...Option) error
}

// Format implements Converter.
func (c ConverterFunc) Format() Format { return c.Name }

// Extensions implements Converter.
func (c ConverterFunc) Extensions() []string { return c.Exts }

// Convert implements Converter.
func (c ConverterFunc) Convert(ctx context.Context, src io.Reader, dst io.Writer, opts ...Option) error {
	return c.Fn(ctx, src, dst, opts...)
}
