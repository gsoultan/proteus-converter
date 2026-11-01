// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package core

import (
	"fmt"
	"slices"
	"sync"
)

// Registry maps formats and file extensions to converters. It is safe for
// concurrent use. The zero value is not usable; call NewRegistry.
type Registry struct {
	mu     sync.RWMutex
	byName map[Format]Converter
	byExt  map[string]Format
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byName: map[Format]Converter{}, byExt: map[string]Format{}}
}

// Register adds or replaces the converter for c.Format() and maps its extensions to it.
func (r *Registry) Register(c Converter) {
	if c == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byName[c.Format()] = c
	for _, ext := range c.Extensions() {
		r.byExt[normalizeExt(ext)] = c.Format()
	}
}

// Lookup returns the converter registered for format f.
func (r *Registry) Lookup(f Format) (Converter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byName[f]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownFormat, f)
	}
	return c, nil
}

// FormatForPath detects the format of a file from its extension.
func (r *Registry) FormatForPath(path string) (Format, error) {
	ext := extOf(path)
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.byExt[ext]
	if !ok {
		return "", fmt.Errorf("%w: extension %q", ErrUnknownFormat, ext)
	}
	return f, nil
}

// Formats lists the registered formats in sorted order.
func (r *Registry) Formats() []Format {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Format, 0, len(r.byName))
	for f := range r.byName {
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}
