// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif" // GIF decoding
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/layout"
	"github.com/gsoultan/proteus-converter/svg"
)

type imageLoader struct {
	ctx    context.Context
	loader core.ResourceLoader
	base   string
	cache  map[string]*layout.Image
	warn   func(string)
}

func newImageLoader(ctx context.Context, l core.ResourceLoader, base string, warn func(string)) *imageLoader {
	return &imageLoader{ctx: ctx, loader: l, base: base, cache: map[string]*layout.Image{}, warn: warn}
}

func (il *imageLoader) load(src string) (*layout.Image, error) {
	u := core.ResolveURL(il.base, src)
	if img, ok := il.cache[u]; ok {
		return img, nil
	}
	data, err := il.loader.Load(il.ctx, u)
	if err != nil {
		return nil, err
	}
	img, err := decodeImage(data)
	if err != nil {
		return nil, err
	}
	if img.SVG != nil && il.warn != nil {
		for _, w := range img.SVG.Warnings {
			il.warn("layout: " + w)
		}
	}
	il.cache[u] = img
	return img, nil
}

// Decoded images cost about 8 bytes per pixel while rendering, so a small
// compressed file with huge dimensions (a decompression bomb) is refused.
const (
	maxImagePixels = 40_000_000 // e.g. 8000x5000
	maxImageSide   = 1 << 15
)

func decodeImage(data []byte) (*layout.Image, error) {
	trim := bytes.TrimSpace(data)
	if bytes.HasPrefix(trim, []byte("<svg")) || bytes.HasPrefix(trim, []byte("<?xml")) || bytes.HasPrefix(trim, []byte("<!--")) ||
		bytes.HasPrefix(trim, []byte("<!DOCTYPE svg")) {
		d, err := svg.Parse(trim)
		if err != nil {
			return nil, err
		}
		return &layout.Image{SVG: d, Width: int(d.Width), Height: int(d.Height)}, nil
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImageSide || cfg.Height > maxImageSide ||
		cfg.Width*cfg.Height > maxImagePixels {
		return nil, fmt.Errorf("image dimensions %dx%d out of range (max %d pixels)", cfg.Width, cfg.Height, maxImagePixels)
	}
	switch strings.ToLower(format) {
	case "png":
		return &layout.Image{Data: data, Width: cfg.Width, Height: cfg.Height}, nil
	case "jpeg":
		if _, err := jpeg.DecodeConfig(bytes.NewReader(data)); err != nil {
			return nil, err
		}
		return &layout.Image{Data: data, JPEG: true, Width: cfg.Width, Height: cfg.Height}, nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return &layout.Image{Data: buf.Bytes(), Width: cfg.Width, Height: cfg.Height}, nil
}
