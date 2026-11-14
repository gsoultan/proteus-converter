// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package render

import (
	"math"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
	"github.com/gsoultan/proteus-converter/layout"
)

// StampRun is one piece of watermark text in a single face.
type StampRun struct {
	Text string
	Face *fonts.Face
}

// Stamp is a resolved watermark: the text runs or the image to paint, with
// the geometry and the page selection from core.Watermark.
type Stamp struct {
	Runs  []StampRun // text watermark; empty for an image
	Size  float64    // font size in points
	Color css.Color

	Image      *layout.Image // image watermark; nil for text
	ImageWidth float64       // drawn width in points; 0 means half the page width

	Opacity   float64
	Angle     float64 // degrees counter-clockwise
	Placement core.WatermarkPlacement
	Layer     core.WatermarkLayer
	Pages     core.WatermarkPages
}

// applies reports whether the stamp goes on page index i (0-based).
func (s *Stamp) applies(i int) bool {
	switch s.Pages {
	case core.WatermarkFirstPage:
		return i == 0
	case core.WatermarkAllButFirst:
		return i > 0
	}
	return true
}

// width returns the drawn width and height of one mark.
func (s *Stamp) extent(pageW float64) (w, h float64) {
	if s.Image != nil {
		w = s.ImageWidth
		if w <= 0 {
			w = pageW / 2
		}
		if s.Image.Width > 0 && s.Image.Height > 0 {
			h = w * float64(s.Image.Height) / float64(s.Image.Width)
		} else {
			h = w
		}
		return w, h
	}
	for _, r := range s.Runs {
		w += r.Face.Width(r.Text, s.Size)
	}
	return w, s.Size
}

// paint stamps one page.
func (s *Stamp) paint(p Painter, pageW, pageH float64) {
	w, h := s.extent(pageW)
	if w <= 0 || h <= 0 {
		return
	}
	if s.Placement != core.WatermarkTiled {
		s.mark(p, pageW/2, pageH/2, w, h)
		return
	}
	// A staggered grid: every other row shifted by half a cell. The cell is
	// the mark plus a gap, so neighbouring marks never touch even rotated.
	rad := s.Angle * math.Pi / 180
	cw := w*math.Abs(math.Cos(rad)) + h*math.Abs(math.Sin(rad))
	ch := w*math.Abs(math.Sin(rad)) + h*math.Abs(math.Cos(rad))
	stepX, stepY := cw+h, ch+h
	rows := int(math.Ceil(pageH/stepY)) + 1
	cols := int(math.Ceil(pageW/stepX)) + 1
	if rows*cols > 400 {
		s.mark(p, pageW/2, pageH/2, w, h)
		return
	}
	for r := 0; r < rows; r++ {
		cy := stepY/2 + float64(r)*stepY
		offset := 0.0
		if r%2 == 1 {
			offset = stepX / 2
		}
		for c := -1; c < cols; c++ {
			cx := stepX/2 + float64(c)*stepX + offset
			if cx+cw/2 < 0 || cx-cw/2 > pageW || cy-ch/2 > pageH {
				continue
			}
			s.mark(p, cx, cy, w, h)
		}
	}
}

// mark paints one mark centred on (cx, cy).
func (s *Stamp) mark(p Painter, cx, cy, w, h float64) {
	rad := s.Angle * math.Pi / 180
	sin, cos := math.Sin(rad), math.Cos(rad)
	if s.Image != nil {
		p.StampImage(cx-w/2, cy-h/2, w, h, s.Image, s.Angle, s.Opacity)
		return
	}
	// Baseline direction u and the direction towards the descenders d, in
	// page coordinates (y down). The em box is centred on (cx, cy).
	ux, uy := cos, -sin
	dx, dy := sin, cos
	var asc, desc float64 = 0.8, 0.2
	if len(s.Runs) > 0 {
		m := s.Runs[0].Face.Metrics()
		if m.Ascent > 0 {
			asc, desc = m.Ascent, m.Descent
		}
	}
	shift := (asc - desc) / 2 * s.Size
	x := cx - ux*w/2 + dx*shift
	y := cy - uy*w/2 + dy*shift
	for _, r := range s.Runs {
		p.StampText(x, y, r.Text, r.Face, s.Size, s.Color, s.Angle, s.Opacity)
		adv := r.Face.Width(r.Text, s.Size)
		x += ux * adv
		y += uy * adv
	}
}
