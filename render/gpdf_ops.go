package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // GIF decoding
	_ "image/jpeg" // JPEG decoding
	_ "image/png"  // PNG decoding
	"math"

	"github.com/gsoultan/gpdf/content"
	"github.com/gsoultan/gpdf/doc"
	"github.com/gsoultan/gpdf/doc/builder"
	"github.com/gsoultan/gpdf/model"
	"github.com/gsoultan/proteus-converter/layout"
	"github.com/gsoultan/proteus-converter/svg"
)

// gpdf paints all vector graphics, then all text, then all images of a page,
// whatever the call order. Images (including CSS backgrounds) must instead be
// painted in document order with the other graphics, under the text. The
// adapter therefore registers each image as an XObject through an invisible
// image run and paints it with its own graphic run ("q cm /ImN Do Q"), which
// keeps call order and allows clipping. This relies on gpdf naming the image
// XObjects of a page Im1, Im2... in run order; see review notes in Serena.

// rawImage is a decoded raster image with an optional alpha mask.
type rawImage struct {
	rgb, mask  []byte
	gray       bool
	w, h       int
	jpeg       bool
	colorSpace string
}

// num rounds a coordinate so gpdf's %g encoder never writes exponent notation.
func num(v float64) model.Real {
	if math.Abs(v) < 1e-4 {
		return 0
	}
	return model.Real(math.Round(v*1e4) / 1e4)
}

func op(name string, args ...float64) content.Op {
	a := make([]model.Object, len(args))
	for i, v := range args {
		a[i] = num(v)
	}
	return content.Op{Name: name, Args: a}
}

// addOps appends a graphic run to the current page, keeping call order.
func (g *GPDF) addOps(ops []content.Op, gs map[model.Name]model.Dict) {
	pg := g.b.PageAt(g.page)
	if pg == nil {
		return
	}
	all := make([]content.Op, 0, len(ops)+2)
	all = append(all, content.Op{Name: "q"})
	all = append(all, ops...)
	all = append(all, content.Op{Name: "Q"})
	pg.GraphicRuns = append(pg.GraphicRuns, builder.GraphicRun{Ops: all, ExtGStates: gs})
}

// extGState returns a fresh graphics state name for the current page.
func (g *GPDF) extGState(fill, stroke float64) (model.Name, model.Dict) {
	pg := g.b.PageAt(g.page)
	pg.NextGSIndex++
	name := model.Name(fmt.Sprintf("GS%d", pg.NextGSIndex))
	d := model.Dict{model.Name("Type"): model.Name("ExtGState")}
	if fill < 1 {
		d[model.Name("ca")] = num(fill)
	}
	if stroke < 1 {
		d[model.Name("CA")] = num(stroke)
	}
	return name, d
}

// clipOps returns ops that clip to a rounded rectangle given in page coordinates (y down).
func (g *GPDF) clipOps(c layout.Clip) []content.Op {
	if !c.Active() {
		return nil
	}
	r := math.Min(c.Radius, math.Min(c.W, c.H)/2)
	m := svg.Matrix{1, 0, 0, -1, 0, g.pageH}
	ops := pathOps(m, svg.RoundRect(c.X, c.Y, c.W, c.H, r, r))
	return append(ops, content.Op{Name: "W"}, content.Op{Name: "n"})
}

// pathOps converts segments to PDF path construction ops through m.
func pathOps(m svg.Matrix, segs []svg.Seg) []content.Op {
	ops := make([]content.Op, 0, len(segs))
	for _, s := range segs {
		switch s.Op {
		case 'M':
			x, y := m.Apply(s.Pts[0], s.Pts[1])
			ops = append(ops, op("m", x, y))
		case 'L':
			x, y := m.Apply(s.Pts[0], s.Pts[1])
			ops = append(ops, op("l", x, y))
		case 'C':
			x1, y1 := m.Apply(s.Pts[0], s.Pts[1])
			x2, y2 := m.Apply(s.Pts[2], s.Pts[3])
			x3, y3 := m.Apply(s.Pts[4], s.Pts[5])
			ops = append(ops, op("c", x1, y1, x2, y2, x3, y3))
		case 'Z':
			ops = append(ops, content.Op{Name: "h"})
		}
	}
	return ops
}

// Path implements Painter.
func (g *GPDF) Path(m svg.Matrix, segs []svg.Seg, st layout.PathStyle, clip layout.Clip) {
	fill, stroke := st.Fill, st.Stroke
	if fill != nil && fill.A <= 0 {
		fill = nil
	}
	if stroke != nil && (stroke.A <= 0 || st.StrokeWidth <= 0) {
		stroke = nil
	}
	if (fill == nil && stroke == nil) || len(segs) == 0 {
		return
	}
	ops := g.clipOps(clip)
	var gs map[model.Name]model.Dict
	fa, sa := 1.0, 1.0
	if fill != nil {
		fa = fill.A
	}
	if stroke != nil {
		sa = stroke.A
	}
	if fa < 1 || sa < 1 {
		name, d := g.extGState(fa, sa)
		gs = map[model.Name]model.Dict{name: d}
		ops = append(ops, content.Op{Name: "gs", Args: []model.Object{name}})
	}
	// user space -> page (y down) -> PDF (y up), applied with cm so strokes scale like SVG
	pdf := svg.Matrix{1, 0, 0, -1, 0, g.pageH}.Mul(m)
	ops = append(ops, op("cm", pdf[0], pdf[1], pdf[2], pdf[3], pdf[4], pdf[5]))
	if fill != nil {
		ops = append(ops, op("rg", fill.R, fill.G, fill.B))
	}
	if stroke != nil {
		ops = append(ops, op("RG", stroke.R, stroke.G, stroke.B), op("w", st.StrokeWidth),
			op("J", float64(st.LineCap)), op("j", float64(st.LineJoin)))
		if st.MiterLimit >= 1 {
			ops = append(ops, op("M", st.MiterLimit))
		}
		if len(st.Dash) > 0 {
			arr := make(model.Array, len(st.Dash))
			for i, d := range st.Dash {
				arr[i] = num(d)
			}
			ops = append(ops, content.Op{Name: "d", Args: []model.Object{arr, model.Integer(0)}})
		}
	}
	ops = append(ops, pathOps(svg.Identity, segs)...)
	paint := "f"
	switch {
	case fill != nil && stroke != nil:
		paint = "B"
	case stroke != nil:
		paint = "S"
	}
	if fill != nil && st.EvenOdd {
		paint += "*"
	}
	ops = append(ops, content.Op{Name: paint})
	g.addOps(ops, gs)
}

// Image implements Painter.
func (g *GPDF) Image(x, y, w, h float64, img *layout.Image, clip layout.Clip) {
	if img == nil || w <= 0 || h <= 0 || img.SVG != nil {
		return
	}
	name, ok := g.xobject(img)
	if !ok {
		return
	}
	ops := g.clipOps(clip)
	ops = append(ops, op("cm", w, 0, 0, h, x, g.pageH-y-h), content.Op{Name: "Do", Args: []model.Object{model.Name(name)}})
	g.addOps(ops, nil)
}

// xobject registers img on the current page and returns its XObject name.
func (g *GPDF) xobject(img *layout.Image) (string, bool) {
	if name, ok := g.xobjects[img]; ok {
		return name, true
	}
	raw, err := g.decode(img)
	if err != nil {
		g.warnf("render: image: %v", err)
		return "", false
	}
	pg := g.b.PageAt(g.page)
	if pg == nil {
		return "", false
	}
	before := len(pg.ImageRuns)
	// an invisible run (tiny and off the page) only creates the XObject
	opts := doc.ImageOptions{PageIndex: g.page, X: -10, Y: -10, Width: 0.01, Height: 0.01,
		Data: raw.rgb, PixelWidth: raw.w, PixelHeight: raw.h, BitsPerComponent: 8, ColorSpace: raw.colorSpace,
		IsJPEG: raw.jpeg}
	if raw.mask != nil {
		opts.HasMask, opts.Mask, opts.MaskWidth, opts.MaskHeight = true, raw.mask, raw.w, raw.h
	}
	g.b.DrawImageWith(opts)
	if len(pg.ImageRuns) != before+1 {
		g.warnf("render: image could not be embedded")
		return "", false
	}
	name := fmt.Sprintf("Im%d", before+1)
	g.xobjects[img] = name
	return name, true
}

// decode turns PNG/GIF data into raw RGB (or gray) samples plus an alpha
// mask; JPEG data is embedded as is.
func (g *GPDF) decode(img *layout.Image) (*rawImage, error) {
	if r, ok := g.decoded[img]; ok {
		return r, nil
	}
	var r *rawImage
	if img.JPEG {
		cs := "DeviceRGB"
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(img.Data)); err == nil {
			switch cfg.ColorModel {
			case color.GrayModel, color.Gray16Model:
				cs = "DeviceGray"
			case color.CMYKModel:
				cs = "DeviceCMYK"
			}
		}
		r = &rawImage{rgb: img.Data, w: img.Width, h: img.Height, jpeg: true, colorSpace: cs}
	} else {
		m, _, err := image.Decode(bytes.NewReader(img.Data))
		if err != nil {
			return nil, err
		}
		r = rasterize(m)
	}
	g.decoded[img] = r
	return r, nil
}

func rasterize(m image.Image) *rawImage {
	b := m.Bounds()
	w, h := b.Dx(), b.Dy()
	r := &rawImage{w: w, h: h, colorSpace: "DeviceRGB"}
	gray := true
	opaque := true
	rgb := make([]byte, w*h*3)
	mask := make([]byte, w*h)
	for y := range h {
		for x := range w {
			c := color.NRGBAModel.Convert(m.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			i := y*w + x
			rgb[i*3], rgb[i*3+1], rgb[i*3+2] = c.R, c.G, c.B
			mask[i] = c.A
			if c.A != 255 {
				opaque = false
			}
			if c.R != c.G || c.G != c.B {
				gray = false
			}
		}
	}
	if gray {
		g := make([]byte, w*h)
		for i := range g {
			g[i] = rgb[i*3]
		}
		r.rgb, r.colorSpace, r.gray = g, "DeviceGray", true
	} else {
		r.rgb = rgb
	}
	if !opaque {
		r.mask = mask
	}
	return r
}
