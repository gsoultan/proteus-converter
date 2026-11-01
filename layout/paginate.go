package layout

import (
	"errors"
	"math"
)

// ErrTooManyPages is returned by Paginate when content would need more than
// Engine.MaxPages pages.
var ErrTooManyPages = errors.New("too many pages")

// Page holds the items of one page, in page coordinates (points, origin at
// the top-left of the page content area, Y downward).
type Page struct {
	Items []Item
}

// Paginate distributes flow items over pages of the engine's content height.
// Rectangles and vertical lines crossing a page boundary are split. Items
// with non-finite coordinates are dropped; content beyond Engine.MaxPages
// (when positive) makes Paginate fail with ErrTooManyPages.
func (e *Engine) Paginate(items []Item) ([]*Page, error) {
	h := e.PageHeight
	limit := e.MaxPages
	if limit <= 0 {
		limit = math.MaxInt32
	}
	var pages []*Page
	overflow := false
	// sink collects items for pages beyond the limit so callers need no checks.
	sink := &Page{}
	get := func(i int) *Page {
		if i >= limit {
			overflow = true
			sink.Items = sink.Items[:0]
			return sink
		}
		for len(pages) <= i {
			pages = append(pages, &Page{})
		}
		return pages[i]
	}
	idx := func(y float64) int {
		p := math.Floor((y + eps) / h)
		switch {
		case !(p > 0): // negative or NaN
			return 0
		case p >= float64(limit):
			return limit
		}
		return int(p)
	}
	var add func(it Item)
	add = func(it Item) {
		switch v := it.(type) {
		case *groupItem:
			for _, c := range v.items {
				add(c)
			}
		case *RectItem:
			if v.H <= 0 || v.W <= 0 {
				return
			}
			p0, p1 := idx(v.Y), idx(v.Y+v.H-eps)
			for p := p0; p <= p1; p++ {
				top := math.Max(v.Y, float64(p)*h)
				bot := math.Min(v.Y+v.H, float64(p+1)*h)
				if bot-top <= 0 {
					continue
				}
				c := *v
				c.Y, c.H = top-float64(p)*h, bot-top
				if p0 != p1 {
					c.Radius = 0
				}
				get(p).Items = append(get(p).Items, &c)
			}
		case *LineItem:
			y0, y1 := math.Min(v.Y1, v.Y2), math.Max(v.Y1, v.Y2)
			p0, p1 := idx(y0), idx(y1-eps)
			if p0 == p1 || v.X1 != v.X2 {
				c := *v
				c.Y1 -= float64(p0) * h
				c.Y2 -= float64(p0) * h
				get(p0).Items = append(get(p0).Items, &c)
				return
			}
			for p := p0; p <= p1; p++ {
				c := *v
				c.Y1 = math.Max(y0, float64(p)*h) - float64(p)*h
				c.Y2 = math.Min(y1, float64(p+1)*h) - float64(p)*h
				get(p).Items = append(get(p).Items, &c)
			}
		case *TextItem:
			c := *v
			// a line belongs to the page that holds its baseline's line box
			p := idx(v.Baseline - v.Size*0.8)
			c.Baseline -= float64(p) * h
			get(p).Items = append(get(p).Items, &c)
		case *ImageItem:
			y0, y1 := v.Y, v.Y+v.H
			if v.Clip.Active() {
				y0, y1 = math.Max(y0, v.Clip.Y), math.Min(y1, v.Clip.Y+v.Clip.H)
			}
			if y1 <= y0 {
				return
			}
			for _, p := range spanPages(y0, y1, idx) {
				c := *v
				c.Y -= float64(p) * h
				c.Clip = pageClip(v.Clip, p, h, y0, y1, idx)
				get(p).Items = append(get(p).Items, &c)
			}
		case *PathItem:
			y0, y1 := pathExtent(v)
			if v.Clip.Active() {
				y0, y1 = math.Max(y0, v.Clip.Y), math.Min(y1, v.Clip.Y+v.Clip.H)
			}
			if y1 < y0 {
				y0, y1 = v.Top, v.Top
			}
			for _, p := range spanPages(y0, y1, idx) {
				c := *v
				c.M[5] -= float64(p) * h
				c.Top -= float64(p) * h
				c.Clip = pageClip(v.Clip, p, h, y0, y1, idx)
				get(p).Items = append(get(p).Items, &c)
			}
		case *EllipseItem:
			c := *v
			p := idx(v.Y)
			c.Y -= float64(p) * h
			get(p).Items = append(get(p).Items, &c)
		case *LinkItem:
			c := *v
			p := idx(v.Y)
			c.Y -= float64(p) * h
			get(p).Items = append(get(p).Items, &c)
		case *AnchorItem:
			if v.Name == "" {
				return
			}
			c := *v
			p := idx(v.Y)
			c.Y -= float64(p) * h
			get(p).Items = append(get(p).Items, &c)
		case *BookmarkItem:
			if v.Title == "" {
				return
			}
			c := *v
			p := idx(v.Y)
			c.Y -= float64(p) * h
			get(p).Items = append(get(p).Items, &c)
		}
	}
	for _, it := range items {
		add(it)
		if overflow {
			return nil, ErrTooManyPages
		}
	}
	if len(pages) == 0 {
		pages = append(pages, &Page{})
	}
	// position: fixed boxes repeat on every page, on top of the flow
	if len(e.fixedItems) > 0 {
		for _, pg := range pages {
			flow := pg.Items
			tmp := &Page{}
			pages0 := pages
			pages = []*Page{tmp}
			for _, it := range copyItems(e.fixedItems, 0, 0) {
				add(it)
			}
			pages = pages0
			pg.Items = append(flow, tmp.Items...)
		}
	}
	return pages, nil
}

// spanPages returns the pages a vertical extent [y0, y1] touches.
func spanPages(y0, y1 float64, idx func(float64) int) []int {
	p0, p1 := idx(y0), idx(y1-eps)
	if p1 < p0 {
		p1 = p0
	}
	out := make([]int, 0, p1-p0+1)
	for p := p0; p <= p1; p++ {
		out = append(out, p)
	}
	return out
}

// pageClip moves clip onto page p and, when the content spans several pages,
// limits it to that page.
func pageClip(clip Clip, p int, h, y0, y1 float64, idx func(float64) int) Clip {
	off := float64(p) * h
	if clip.Active() {
		clip.Y -= off
	}
	if idx(y0) == idx(y1-eps) {
		return clip
	}
	page := Clip{X: -1e5, Y: 0, W: 2e5, H: h}
	if !clip.Active() {
		return page
	}
	top := math.Max(clip.Y, 0)
	bot := math.Min(clip.Y+clip.H, h)
	if bot <= top {
		return Clip{X: clip.X, Y: 0, W: clip.W, H: eps}
	}
	return Clip{X: clip.X, Y: top, W: clip.W, H: bot - top}
}

// pathExtent returns the vertical extent of a path in layout coordinates.
func pathExtent(p *PathItem) (float64, float64) {
	y0, y1 := math.Inf(1), math.Inf(-1)
	pad := 0.0
	if p.Style.Stroke != nil {
		pad = p.Style.StrokeWidth * p.M.Scale() / 2
	}
	for _, s := range p.Segs {
		n := 0
		switch s.Op {
		case 'M', 'L':
			n = 1
		case 'C':
			n = 3
		}
		for k := range n {
			_, y := p.M.Apply(s.Pts[2*k], s.Pts[2*k+1])
			y0 = math.Min(y0, y-pad)
			y1 = math.Max(y1, y+pad)
		}
	}
	return y0, y1
}
