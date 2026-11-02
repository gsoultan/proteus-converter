// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package core

// WatermarkPlacement says where a watermark is stamped on a page.
type WatermarkPlacement int

const (
	// WatermarkCenter stamps one mark in the middle of the page.
	WatermarkCenter WatermarkPlacement = iota
	// WatermarkTiled repeats the mark in a staggered grid over the page.
	WatermarkTiled
)

// WatermarkLayer says whether a watermark is painted under or over the
// document.
type WatermarkLayer int

const (
	// WatermarkBehind paints the mark under the document's text. Page
	// backgrounds and images stay beneath it.
	WatermarkBehind WatermarkLayer = iota
	// WatermarkOver paints the mark over everything on the page.
	WatermarkOver
)

// WatermarkPages selects the pages a watermark is stamped on.
type WatermarkPages int

const (
	WatermarkAllPages WatermarkPages = iota
	WatermarkFirstPage
	WatermarkAllButFirst
)

// Watermark is a translucent stamp painted on the output pages, such as
// DRAFT, COPY, PAID, CONFIDENTIAL or a faint logo. Set Text or Image.
type Watermark struct {
	// Text is stamped in a bold sans-serif face unless FontFamily says
	// otherwise. Empty with no Image means no watermark.
	Text string
	// FontFamily is a CSS font-family list for Text; default "sans-serif".
	FontFamily string
	// FontSize in points; default 72.
	FontSize float64
	// Color is a CSS colour for Text; default a dark grey (#444).
	Color string

	// Image is PNG, JPEG, GIF or SVG data stamped instead of text.
	Image []byte
	// ImageWidth is the drawn width in points; default half the page width.
	// The height follows the image's aspect ratio.
	ImageWidth float64

	// Opacity from 0 (invisible) to 1 (solid); default 0.15.
	Opacity float64
	// Angle in degrees counter-clockwise; default 30. 0 keeps it level.
	Angle float64
	// AngleSet must be true for an Angle of 0 to be used.
	AngleSet bool

	Placement WatermarkPlacement
	Layer     WatermarkLayer
	Pages     WatermarkPages
}

// Empty reports whether the watermark has nothing to stamp.
func (w Watermark) Empty() bool { return w.Text == "" && len(w.Image) == 0 }

// WithWatermark stamps w on the output pages.
func WithWatermark(w Watermark) Option { return func(c *Config) { c.Watermark = &w } }
