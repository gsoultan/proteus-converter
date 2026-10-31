package layout

import (
	"math"
)

// Page holds the items of one page, in page coordinates (points, origin at
// the top-left of the page content area, Y downward).
type Page struct {
	Items []Item
}

// Paginate distributes flow items over pages of the engine's content height.
// Rectangles and vertical lines crossing a page boundary are split.
func (e *Engine) Paginate(items []Item) []*Page {
	h := e.PageHeight
	var pages []*Page
	get := func(i int) *Page {
		for len(pages) <= i {
			pages = append(pages, &Page{})
		}
		return pages[i]
	}
	idx := func(y float64) int { return max(0, int(math.Floor((y+eps)/h))) }
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
			c := *v
			p := idx(v.Y)
			c.Y -= float64(p) * h
			get(p).Items = append(get(p).Items, &c)
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
	}
	if len(pages) == 0 {
		pages = append(pages, &Page{})
	}
	return pages
}
