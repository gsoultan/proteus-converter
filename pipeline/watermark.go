// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package pipeline

import (
	"fmt"
	"strings"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
	"github.com/gsoultan/proteus-converter/render"
)

// Watermark limits: a stamp is one line, so very long text or a huge size
// would only paint off the page.
const (
	maxWatermarkRunes = 200
	maxWatermarkSize  = 1000
)

// resolveWatermark turns the option into a render.Stamp: fonts are picked
// from the conversion's resolver, the colour parsed as CSS, the image
// decoded. Problems fall back to defaults with a warning; an unusable image
// yields nil.
func resolveWatermark(w core.Watermark, res *fonts.Resolver, warn func(string)) *render.Stamp {
	st := &render.Stamp{Opacity: w.Opacity, Angle: w.Angle, Placement: w.Placement, Layer: w.Layer, Pages: w.Pages}
	if st.Opacity <= 0 {
		st.Opacity = 0.15
	}
	st.Opacity = min(st.Opacity, 1)
	if w.Angle == 0 && !w.AngleSet {
		st.Angle = 30
	}
	if len(w.Image) > 0 {
		img, err := decodeImage(w.Image)
		if err != nil {
			warn(fmt.Sprintf("watermark: image: %v", err))
			return nil
		}
		if img.SVG != nil {
			warn("watermark: SVG images are not supported; use PNG or JPEG")
			return nil
		}
		st.Image, st.ImageWidth = img, w.ImageWidth
		return st
	}
	text := strings.TrimSpace(w.Text)
	if r := []rune(text); len(r) > maxWatermarkRunes {
		text = string(r[:maxWatermarkRunes])
		warn("watermark: text shortened to 200 characters")
	}
	if text == "" {
		return nil
	}
	st.Size = w.FontSize
	if st.Size <= 0 {
		st.Size = 72
	}
	st.Size = min(st.Size, maxWatermarkSize)
	st.Color = css.Color{R: 0x44 / 255.0, G: 0x44 / 255.0, B: 0x44 / 255.0, A: 1}
	if w.Color != "" {
		if c, ok := css.ParseColor(w.Color, st.Color); ok {
			st.Color = c
		} else {
			warn(fmt.Sprintf("watermark: unknown colour %q", w.Color))
		}
	}
	families := []string{"sans-serif"}
	if f := strings.TrimSpace(w.FontFamily); f != "" {
		families = nil
		for _, name := range strings.Split(f, ",") {
			if name = strings.Trim(strings.TrimSpace(name), `"'`); name != "" {
				families = append(families, name)
			}
		}
	}
	chain := res.Chain(families, 700, false)
	for _, run := range res.Split(text, chain) {
		st.Runs = append(st.Runs, render.StampRun{Text: run.Text, Face: run.Face})
	}
	return st
}
