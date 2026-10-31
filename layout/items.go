package layout

import (
	"github.com/gsoultan/proteus-converter/css"
	"github.com/gsoultan/proteus-converter/fonts"
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

// ImageItem draws an image scaled to W x H with its top-left corner at X, Y.
type ImageItem struct {
	X, Y, W, H float64
	Image      *Image
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

func (r *RectItem) translate(dx, dy float64)    { r.X += dx; r.Y += dy }
func (l *LineItem) translate(dx, dy float64)    { l.X1 += dx; l.X2 += dx; l.Y1 += dy; l.Y2 += dy }
func (t *TextItem) translate(dx, dy float64)    { t.X += dx; t.Baseline += dy }
func (i *ImageItem) translate(dx, dy float64)   { i.X += dx; i.Y += dy }
func (e *EllipseItem) translate(dx, dy float64) { e.X += dx; e.Y += dy }
func (l *LinkItem) translate(dx, dy float64)    { l.X += dx; l.Y += dy }
func (a *AnchorItem) translate(_, dy float64)   { a.Y += dy }
func (b *BookmarkItem) translate(_, dy float64) { b.Y += dy }
