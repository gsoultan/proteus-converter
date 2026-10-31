// Package render paints paginated layout items through a Painter. The gpdf
// adapter is the default backend; other backends (an image rasterizer for
// tests, a different PDF library) only need to implement Painter.
package render

import (
	"io"

	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
	"github.com/gsoultan/proteus-converter/layout"
)

// Painter receives drawing commands in page coordinates: points, origin at
// the top-left corner of the page, Y growing downward.
type Painter interface {
	BeginPage(width, height float64)
	FillRect(x, y, w, h, radius float64, c css.Color)
	Line(x1, y1, x2, y2, width float64, c css.Color, dash []float64)
	Ellipse(x, y, w, h float64, c css.Color, stroke float64)
	Text(x, baseline float64, text string, face *fonts.Face, size float64, c css.Color, letterSpacing float64)
	Image(x, y, w, h float64, img *layout.Image)
	LinkURL(x, y, w, h float64, url string)
	LinkDest(x, y, w, h float64, dest string)
	NamedDest(name string, y float64)
	Bookmark(title string, level int, y float64)
	Finish(w io.Writer) error
}

// Document describes the output pages.
type Document struct {
	PageWidth, PageHeight float64
	MarginTop, MarginLeft float64
	Background            *css.Color // canvas background, painted over the whole page
	Pages                 []*layout.Page
}

// Paint draws every page of d with p and writes the result to w.
func Paint(d *Document, p Painter, w io.Writer) error {
	for _, pg := range d.Pages {
		p.BeginPage(d.PageWidth, d.PageHeight)
		if d.Background != nil {
			p.FillRect(0, 0, d.PageWidth, d.PageHeight, 0, *d.Background)
		}
		ox, oy := d.MarginLeft, d.MarginTop
		for _, it := range pg.Items {
			switch v := it.(type) {
			case *layout.RectItem:
				p.FillRect(ox+v.X, oy+v.Y, v.W, v.H, v.Radius, v.Color)
			case *layout.LineItem:
				p.Line(ox+v.X1, oy+v.Y1, ox+v.X2, oy+v.Y2, v.Width, v.Color, v.Dash)
			case *layout.EllipseItem:
				p.Ellipse(ox+v.X, oy+v.Y, v.W, v.H, v.Color, v.Stroke)
			case *layout.TextItem:
				p.Text(ox+v.X, oy+v.Baseline, v.Text, v.Face, v.Size, v.Color, v.LetterSpacing)
			case *layout.ImageItem:
				p.Image(ox+v.X, oy+v.Y, v.W, v.H, v.Image)
			case *layout.LinkItem:
				if v.URL != "" {
					p.LinkURL(ox+v.X, oy+v.Y, v.W, v.H, v.URL)
				} else {
					p.LinkDest(ox+v.X, oy+v.Y, v.W, v.H, v.Dest)
				}
			case *layout.AnchorItem:
				p.NamedDest(v.Name, oy+v.Y)
			case *layout.BookmarkItem:
				p.Bookmark(v.Title, v.Level, oy+v.Y)
			}
		}
	}
	return p.Finish(w)
}
