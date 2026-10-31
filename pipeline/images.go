package pipeline

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif" // GIF decoding
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/layout"
)

type imageLoader struct {
	ctx    context.Context
	loader core.ResourceLoader
	base   string
	cache  map[string]*layout.Image
}

func newImageLoader(ctx context.Context, l core.ResourceLoader, base string) *imageLoader {
	return &imageLoader{ctx: ctx, loader: l, base: base, cache: map[string]*layout.Image{}}
}

var errSVG = errors.New("SVG images are not supported yet")

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
	il.cache[u] = img
	return img, nil
}

func decodeImage(data []byte) (*layout.Image, error) {
	trim := bytes.TrimSpace(data)
	if bytes.HasPrefix(trim, []byte("<svg")) || bytes.HasPrefix(trim, []byte("<?xml")) {
		return nil, errSVG
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > 100_000_000 {
		return nil, errors.New("image dimensions out of range")
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
