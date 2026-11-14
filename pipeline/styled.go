// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package pipeline

import (
	"context"

	"github.com/gsoultan/proteus-converter/core"
	"github.com/gsoultan/proteus-converter/css"
	"golang.org/x/net/html"
)

// Styled is a document with its computed CSS, for output writers that need
// styles but not page layout (spreadsheets, word processors).
type Styled struct {
	Styles   map[*html.Node]*css.ElementStyle
	BaseURL  string
	Metadata core.Metadata // from the document head, overridden by Config.Metadata
	loader   core.ResourceLoader
}

// Style collects the document's stylesheets and runs the cascade, the same
// way Render does, against a viewport of the configured page content box.
func Style(ctx context.Context, in Input, cfg *core.Config) (*Styled, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	loader := newCachingLoader(core.DefaultLoader(cfg))
	base := cfg.BaseURL
	if in.BaseURL != "" {
		base = in.BaseURL
	}
	if b := findBase(in.Doc); b != "" {
		base = core.ResolveURL(base, b)
	}
	warn := cfg.Warnf
	var author []sourceSheet
	for _, t := range in.ThemeCSS {
		author = append(author, sourceSheet{css: t, base: base})
	}
	collectSheets(ctx, in.Doc, base, loader, warn, &author)
	for _, s := range cfg.Stylesheets {
		author = append(author, sourceSheet{css: s, base: base})
	}
	cw := cfg.PageSize.Width - cfg.Margins.Left - cfg.Margins.Right
	ch := cfg.PageSize.Height - cfg.Margins.Top - cfg.Margins.Bottom
	mc := css.MediaContext{Type: cfg.MediaType, Width: cw, Height: ch}
	sheets := []css.Sheet{{Stylesheet: css.Parse(css.UserAgentCSS, mc.Match), Origin: css.OriginUserAgent}}
	for _, s := range author {
		if s.media != "" && !mc.Match(s.media) {
			continue
		}
		for _, ps := range expandImports(ctx, s.css, s.base, mc, loader, warn, 0) {
			sheets = append(sheets, css.Sheet{Stylesheet: ps.sheet, Origin: css.OriginAuthor})
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	styles := css.NewCascade(sheets, cw, ch, warn).Compute(in.Doc)
	m := documentMetadata(in.Doc)
	meta := core.Metadata{Title: m.Title, Author: m.Author, Subject: m.Subject, Keywords: m.Keywords,
		Creator: m.Creator, Producer: m.Producer, Language: m.Language}
	o := cfg.Metadata
	for _, f := range []struct {
		dst *string
		v   string
	}{
		{&meta.Title, o.Title}, {&meta.Author, o.Author}, {&meta.Subject, o.Subject},
		{&meta.Keywords, o.Keywords}, {&meta.Creator, o.Creator}, {&meta.Producer, o.Producer},
		{&meta.Language, o.Language},
	} {
		if f.v != "" {
			*f.dst = f.v
		}
	}
	return &Styled{Styles: styles, BaseURL: base, Metadata: meta, loader: loader}, nil
}

// Load fetches a resource referenced by the document (for example an image
// src), resolved against the document base, through the configured loader.
func (s *Styled) Load(ctx context.Context, ref string) ([]byte, error) {
	return s.loader.Load(ctx, core.ResolveURL(s.BaseURL, ref))
}
