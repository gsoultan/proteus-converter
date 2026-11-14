// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package xlsx

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"  // GIF decoding, re-encoded as PNG
	_ "image/jpeg" // JPEG decoding
	"image/png"
	"math"
	"path"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/css"
)

// loadedImage is an image ready to embed: PNG or JPEG bytes and pixel size.
type loadedImage struct {
	data   []byte
	ext    string
	w, h   int
	err    error
	warned bool
}

// imageStore loads and caches the images a document references.
type imageStore struct {
	cache map[string]*loadedImage
}

// Decoded images cost about 8 bytes per pixel; refuse decompression bombs.
const (
	maxImagePixels = 40_000_000
	maxImageSide   = 1 << 15
)

func (e *extractor) loadImage(src string) *loadedImage {
	u := core.ResolveURL(e.styled.BaseURL, src)
	if li, ok := e.images.cache[u]; ok {
		return li
	}
	li := &loadedImage{}
	e.images.cache[u] = li
	data, err := e.styled.Load(e.ctx, src)
	if err != nil {
		li.err = err
		return li
	}
	trim := bytes.TrimSpace(data)
	if bytes.HasPrefix(trim, []byte("<svg")) || bytes.HasPrefix(trim, []byte("<?xml")) || bytes.HasPrefix(trim, []byte("<!DOCTYPE svg")) {
		li.err = fmt.Errorf("SVG images are not embedded in spreadsheets")
		return li
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		li.err = err
		return li
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImageSide || cfg.Height > maxImageSide || cfg.Width*cfg.Height > maxImagePixels {
		li.err = fmt.Errorf("image dimensions %dx%d out of range", cfg.Width, cfg.Height)
		return li
	}
	li.w, li.h = cfg.Width, cfg.Height
	switch format {
	case "png":
		li.data, li.ext = data, "png"
	case "jpeg":
		li.data, li.ext = data, "jpeg"
	default:
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			li.err = err
			return li
		}
		var out bytes.Buffer
		if err := png.Encode(&out, img); err != nil {
			li.err = err
			return li
		}
		li.data, li.ext = out.Bytes(), "png"
	}
	return li
}

// placeImage anchors an image at a cell, sized by the <img> CSS width and
// height (falling back to the intrinsic pixel size), and grows the row and
// column so that it is visible.
func (e *extractor) placeImage(sh *sheet, ref imageRef, r, c int, cellStyle *css.Style) {
	li := e.loadImage(ref.src)
	if li.err != nil {
		if !li.warned {
			li.warned = true
			e.warn(fmt.Sprintf("xlsx: image %s: %v", ref.src, li.err))
		}
		return
	}
	w, h := float64(li.w), float64(li.h)
	if ref.style != nil {
		cw, ch := ref.style.Width, ref.style.Height
		switch {
		case !cw.Auto && cw.Pt > 0 && cw.Pct == 0 && !ch.Auto && ch.Pt > 0 && ch.Pct == 0:
			w, h = cw.Pt/0.75, ch.Pt/0.75
		case !cw.Auto && cw.Pt > 0 && cw.Pct == 0:
			h = h * (cw.Pt / 0.75) / w
			w = cw.Pt / 0.75
		case !ch.Auto && ch.Pt > 0 && ch.Pct == 0:
			w = w * (ch.Pt / 0.75) / h
			h = ch.Pt / 0.75
		}
	}
	w, h = math.Round(w), math.Round(h)
	if w < 1 || h < 1 {
		return
	}
	name := path.Base(ref.src)
	if ref.alt != "" {
		name = ref.alt
	}
	// Images in a cell with text sit after it: offset by one line so the text stays readable.
	var offY int64
	if cl := sh.cellAt(r, c); cl.kind != kindEmpty {
		lh := fontOf(cellStyle).sizePt * 1.3 / 0.75
		offY = int64(lh * emuPerPx)
	}
	sh.images = append(sh.images, anchoredImage{r: r, c: c, name: name, ext: li.ext, data: li.data,
		wPx: w, hPx: h, offYEMU: offY, rowHeight: roundPt(h*0.75) + roundPt(float64(offY)/emuPerPx*0.75)})
}

// fitImages grows rows and columns to the images anchored in them. Column
// widths are in characters (about 7 px each at the default font); row heights
// in points.
func fitImages(sh *sheet, digitW float64) {
	for _, img := range sh.images {
		if rw := sh.rowAt(img.r); img.rowHeight > rw.heightPt {
			rw.heightPt = img.rowHeight
		}
		if col := sh.colAt(img.c); widthChars(img.wPx, digitW)+1 > col.width {
			col.width = widthChars(img.wPx, digitW) + 1
		}
	}
}
