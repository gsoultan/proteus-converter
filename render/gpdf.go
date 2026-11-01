package render

import (
	"fmt"
	"io"

	"github.com/gsoultan/gpdf/doc"
	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
	"github.com/gsoultan/proteus-converter/layout"
)

// Metadata is written to the PDF information dictionary.
type Metadata struct {
	Title, Author, Subject, Keywords, Creator, Producer, Language string
}

// GPDF is a Painter backed by github.com/gsoultan/gpdf.
type GPDF struct {
	b          *doc.DocumentBuilder
	page       int
	pageH      float64
	registered map[string]bool
	dests      map[string]bool
	warn       func(string)
	err        error
	outline    bool
	xobjects   map[*layout.Image]string // image XObjects registered on the current page
	decoded    map[*layout.Image]*rawImage
}

// NewGPDF returns a gpdf painter. outline enables bookmarks from headings.
func NewGPDF(meta Metadata, outline bool, warn func(string)) *GPDF {
	b := doc.New()
	if meta.Producer == "" {
		meta.Producer = "proteus-converter (gpdf)"
	}
	b.Title(meta.Title).Author(meta.Author).Subject(meta.Subject).Keywords(meta.Keywords).
		Creator(meta.Creator).Producer(meta.Producer)
	if meta.Language != "" {
		b.SetLanguage(meta.Language)
	}
	return &GPDF{b: b, page: -1, registered: map[string]bool{}, dests: map[string]bool{}, warn: warn, outline: outline,
		decoded: map[*layout.Image]*rawImage{}}
}

func (g *GPDF) warnf(format string, args ...any) {
	if g.warn != nil {
		g.warn(fmt.Sprintf(format, args...))
	}
}

func rgb(c css.Color) doc.Color { return doc.Color{R: c.R, G: c.G, B: c.B} }

// BeginPage implements Painter.
func (g *GPDF) BeginPage(width, height float64) {
	g.b.PageSize(width, height).AddPage()
	g.page++
	g.pageH = height
	g.xobjects = map[*layout.Image]string{}
}

// FillRect implements Painter.
func (g *GPDF) FillRect(x, y, w, h, radius float64, c css.Color) {
	if c.A <= 0 || w <= 0 || h <= 0 {
		return
	}
	py := g.pageH - y - h
	state := doc.GraphicsState{}
	if c.A < 1 {
		state.FillOpacity = c.A
	}
	if radius > 0 {
		radius = min(radius, w/2, h/2)
		if state.IsDefault() {
			g.b.FillRoundedRect(g.page, x, py, w, h, radius, rgb(c))
		} else {
			g.b.FillRoundedRectWithState(g.page, x, py, w, h, radius, rgb(c), state)
		}
		return
	}
	if state.IsDefault() {
		g.b.FillRect(g.page, x, py, w, h, rgb(c))
	} else {
		g.b.FillRectWithState(g.page, x, py, w, h, rgb(c), state)
	}
}

// Line implements Painter.
func (g *GPDF) Line(x1, y1, x2, y2, width float64, c css.Color, dash []float64) {
	if c.A <= 0 || width <= 0 {
		return
	}
	ls := doc.LineStyle{Width: width, Color: rgb(c), DashArray: dash}
	state := doc.GraphicsState{}
	if c.A < 1 {
		state.StrokeOpacity = c.A
	}
	g.b.DrawLineWithState(g.page, x1, g.pageH-y1, x2, g.pageH-y2, ls, state)
}

// Ellipse implements Painter.
func (g *GPDF) Ellipse(x, y, w, h float64, c css.Color, stroke float64) {
	if c.A <= 0 || w <= 0 || h <= 0 {
		return
	}
	cx, cy := x+w/2, g.pageH-(y+h/2)
	if stroke > 0 {
		g.b.DrawEllipse(g.page, cx, cy, w/2, h/2, doc.LineStyle{Width: stroke, Color: rgb(c)})
		return
	}
	g.b.FillEllipse(g.page, cx, cy, w/2, h/2, rgb(c))
}

// Text implements Painter.
func (g *GPDF) Text(x, baseline float64, text string, face *fonts.Face, size float64, c css.Color, letterSpacing float64, synth Synthesis) {
	if text == "" || c.A <= 0 {
		return
	}
	name := face.PDFName()
	if !face.IsStandard() && !g.registered[name] {
		ttf, err := face.TrueType()
		if err != nil || ttf == nil {
			g.warnf("render: font %s unusable: %v", face.Family, err)
			return
		}
		g.b.RegisterFont(ttf)
		g.registered[name] = true
	}
	if synth.Bold || synth.Italic {
		g.b.DrawTextSynthetic(text, x, g.pageH-baseline, name, size, rgb(c), letterSpacing, 0, synth.Bold, synth.Italic)
		return
	}
	g.b.DrawTextWithSpacing(text, x, g.pageH-baseline, name, size, rgb(c), letterSpacing, 0)
}

// LinkURL implements Painter.
func (g *GPDF) LinkURL(x, y, w, h float64, url string) {
	g.b.AddLinkToURL(g.page, x, g.pageH-y-h, x+w, g.pageH-y, url)
}

// LinkDest implements Painter.
func (g *GPDF) LinkDest(x, y, w, h float64, dest string) {
	g.b.AddLinkToDest(g.page, x, g.pageH-y-h, x+w, g.pageH-y, dest)
}

// NamedDest implements Painter.
func (g *GPDF) NamedDest(name string, _ float64) {
	if g.dests[name] {
		return
	}
	g.dests[name] = true
	g.b.AddNamedDest(name, g.page)
}

// Bookmark implements Painter. gpdf outlines are flat, so level is not used yet.
func (g *GPDF) Bookmark(title string, _ int, _ float64) {
	if g.outline {
		g.b.AddOutline(title, g.page)
	}
}

// Finish implements Painter.
func (g *GPDF) Finish(w io.Writer) error {
	if g.page < 0 {
		g.BeginPage(595.28, 841.89)
	}
	d, err := g.b.Build()
	if err != nil {
		return fmt.Errorf("render: build PDF: %w", err)
	}
	defer d.Close()
	return d.Save(w)
}
