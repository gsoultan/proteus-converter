package layout

import (
	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
	"github.com/gsoultan/proteus-converter/svg"
)

// Item is a positioned display item. Coordinates are in points; X is from
// the left edge of the page content area and Y grows downward from the top
// of the first page's content area (pages are stacked), until Paginate maps
// them onto pages.
type Item interface {
	translate(dx, dy float64)
}

// RectItem fills a rectangle.
type RectItem struct {
	X, Y, W, H float64
	Color      css.Color
	Radius     float64
}

// LineItem strokes a straight line.
type LineItem struct {
	X1, Y1, X2, Y2 float64
	Width          float64
	Color          css.Color
	Dash           []float64
}

// TextItem draws a run of text on a baseline.
type TextItem struct {
	X, Baseline   float64
	Text          string
	Face          *fonts.Face
	Size          float64
	Color         css.Color
	LetterSpacing float64
}

// Clip is a rounded-rectangle clip region; a zero width means no clipping.
type Clip struct {
	X, Y, W, H float64
	Radius     float64
}

// Active reports whether the clip is set.
func (c Clip) Active() bool { return c.W > 0 && c.H > 0 }

// ImageItem draws an image scaled to W x H with its top-left corner at X, Y,
// optionally clipped (rounded corners, background painting area, object-fit).
type ImageItem struct {
	X, Y, W, H float64
	Image      *Image
	Clip       Clip
}

// PathStyle is how a PathItem is painted.
type PathStyle struct {
	Fill        *css.Color
	EvenOdd     bool
	Stroke      *css.Color
	StrokeWidth float64 // in user units
	LineCap     int     // 0 butt, 1 round, 2 square
	LineJoin    int     // 0 miter, 1 round, 2 bevel
	MiterLimit  float64
	Dash        []float64
}

// PathItem fills and/or strokes a vector path given in user units; M maps
// user units to layout coordinates. Top is the y used to choose its page.
type PathItem struct {
	M     svg.Matrix
	Segs  []svg.Seg
	Style PathStyle
	Clip  Clip
	Top   float64
}

// EllipseItem fills (Stroke == 0) or strokes an ellipse inscribed in X, Y, W, H.
type EllipseItem struct {
	X, Y, W, H float64
	Color      css.Color
	Stroke     float64
}

// LinkItem is a clickable area linking to a URL or an in-document fragment.
type LinkItem struct {
	X, Y, W, H float64
	URL        string
	Dest       string // fragment name, without '#'
}

// AnchorItem marks a fragment target (an element id).
type AnchorItem struct {
	Name string
	Y    float64
}

// BookmarkItem is a heading for the document outline.
type BookmarkItem struct {
	Title string
	Level int
	Y     float64
}

func (r *RectItem) translate(dx, dy float64) { r.X += dx; r.Y += dy }
func (l *LineItem) translate(dx, dy float64) { l.X1 += dx; l.X2 += dx; l.Y1 += dy; l.Y2 += dy }
func (t *TextItem) translate(dx, dy float64) { t.X += dx; t.Baseline += dy }
func (i *ImageItem) translate(dx, dy float64) {
	i.X += dx
	i.Y += dy
	i.Clip.X += dx
	i.Clip.Y += dy
}
func (p *PathItem) translate(dx, dy float64) {
	p.M[4] += dx
	p.M[5] += dy
	p.Top += dy
	p.Clip.X += dx
	p.Clip.Y += dy
}
func (e *EllipseItem) translate(dx, dy float64) { e.X += dx; e.Y += dy }
func (l *LinkItem) translate(dx, dy float64)    { l.X += dx; l.Y += dy }
func (a *AnchorItem) translate(_, dy float64)   { a.Y += dy }
func (b *BookmarkItem) translate(_, dy float64) { b.Y += dy }
