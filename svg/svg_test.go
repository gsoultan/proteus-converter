// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package svg

import (
	"math"
	"testing"
)

func TestParseShapes(t *testing.T) {
	d, err := Parse([]byte(`<?xml version="1.0"?>
<svg xmlns="http://www.w3.org/2000/svg" width="40" height="20" viewBox="0 0 80 40">
<style>.a{fill:#f00}</style>
<defs><rect id="r" width="5" height="5"/></defs>
<g transform="translate(10 5)"><rect class="a" x="1" y="2" width="10" height="4"/></g>
<circle cx="40" cy="20" r="10" fill="none" stroke="blue" stroke-width="2"/>
<path d="M0 0h10v10H0z m20 0 a5 5 0 0 1 10 0"/>
<use href="#r" x="70"/>
<text x="5" y="30" font-size="12">Hi</text>
</svg>`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Width != 40 || d.Height != 20 || !d.HasViewBox || d.ViewBox[2] != 80 {
		t.Fatalf("size %v %v %v", d.Width, d.Height, d.ViewBox)
	}
	if len(d.Items) != 5 {
		t.Fatalf("items = %d", len(d.Items))
	}
	r := d.Items[0].Shape
	if r.Fill == nil || r.Fill.R != 1 || r.Transform[4] != 10 {
		t.Errorf("rect: %+v", r)
	}
	c := d.Items[1].Shape
	if c.Fill != nil || c.Stroke == nil || c.Stroke.B != 1 || c.StrokeWidth != 2 {
		t.Errorf("circle: %+v", c)
	}
	p := d.Items[2].Shape
	if p.Segs[len(p.Segs)-1].Op != 'C' || math.Abs(p.Segs[len(p.Segs)-1].Pts[4]-30) > 1e-9 {
		t.Errorf("path: %+v", p.Segs)
	}
	if u := d.Items[3].Shape; u.Transform[4] != 70 {
		t.Errorf("use: %+v", u.Transform)
	}
	if tx := d.Items[4].Text; tx == nil || tx.Text != "Hi" || tx.FontSize != 12 {
		t.Errorf("text: %+v", tx)
	}
	m := d.ViewportTransform(40, 20)
	if m[0] != 0.5 || m[3] != 0.5 {
		t.Errorf("viewport: %v", m)
	}
}
