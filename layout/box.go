// Package layout turns a styled HTML tree into positioned display items:
// box generation, block and inline formatting, tables, lists, images and
// pagination. It is format-agnostic: any front end that produces an HTML
// tree with computed styles can use it.
package layout

import (
	"github.com/gsoultan/proteus-converter/css"
	"golang.org/x/net/html"
)

// Kind is the type of a box.
type Kind int

const (
	KindBlock Kind = iota
	KindInline
	KindText
	KindImage
	KindBreak
	KindInlineBlock
	KindTable
	KindRowGroup
	KindRow
	KindCell
	KindCaption
	KindCheckbox
	KindWordBreak // <wbr>
)

// Box is a node of the box tree.
type Box struct {
	Kind     Kind
	Style    *css.Style
	Node     *html.Node // nil for anonymous and generated boxes
	Children []*Box

	Text  string // KindText
	Image *Image // KindImage
	Alt   string

	Marker *Box   // list-item marker content (inline box)
	Symbol string // "disc", "circle" or "square" for markers painted as shapes

	Href     string // links
	ID       string // fragment target
	Heading  int    // 1..6 for h1..h6
	Checked  bool   // KindCheckbox
	Disabled bool

	ColSpan, RowSpan int
	RowGroup         string // "header", "body", "footer" for KindRowGroup
	collapsed        bool   // table whose collapsed borders were resolved

	Decorations []Decoration // text decorations propagated from ancestors (including self)
}

// Decoration is a text-decoration applied by an ancestor.
type Decoration struct {
	Lines css.Decoration
	Color css.Color
	Style string
}

// Image is a decoded raster image ready for embedding.
type Image struct {
	Data   []byte // PNG or JPEG file bytes
	JPEG   bool
	Width  int // pixels
	Height int
}

// ImageLoader fetches and decodes an image referenced by src.
type ImageLoader func(src string) (*Image, error)

func (b *Box) isBlockLevel() bool {
	switch b.Kind {
	case KindBlock, KindTable, KindRowGroup, KindRow, KindCell, KindCaption:
		return true
	case KindImage, KindCheckbox:
		return b.Style.IsBlockLevel()
	}
	return false
}

func (b *Box) isInlineLevel() bool { return !b.isBlockLevel() }
