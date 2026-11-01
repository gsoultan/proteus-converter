package core

import "io/fs"

// PageSize is a page size in PDF points (1/72 inch).
type PageSize struct {
	Width, Height float64
}

// Landscape returns the size with width and height swapped so that width >= height.
func (s PageSize) Landscape() PageSize {
	if s.Width < s.Height {
		return PageSize{Width: s.Height, Height: s.Width}
	}
	return s
}

// Common page sizes.
var (
	A3     = PageSize{Width: 841.89, Height: 1190.55}
	A4     = PageSize{Width: 595.28, Height: 841.89}
	A5     = PageSize{Width: 419.53, Height: 595.28}
	Letter = PageSize{Width: 612, Height: 792}
	Legal  = PageSize{Width: 612, Height: 1008}
)

// Margins are page margins in points.
type Margins struct {
	Top, Right, Bottom, Left float64
}

// UniformMargins returns margins with the same value on every side.
func UniformMargins(v float64) Margins { return Margins{v, v, v, v} }

// Unit conversions to points.
func Mm(v float64) float64   { return v * 72 / 25.4 }
func Cm(v float64) float64   { return v * 72 / 2.54 }
func Inch(v float64) float64 { return v * 72 }
func Px(v float64) float64   { return v * 0.75 }

// FontSource is a font file supplied by the caller, registered under a CSS family name.
type FontSource struct {
	Family string // CSS font-family name, e.g. "Inter"
	Weight int    // 100..900; 0 means 400
	Italic bool
	Data   []byte // TrueType (.ttf) font program
}

// Metadata is written to the PDF document information dictionary.
// Empty fields are filled from the source document where possible (e.g. HTML <title>).
type Metadata struct {
	Title, Author, Subject, Keywords, Creator, Producer, Language string
}

// Config is the resolved set of options passed to a Converter. Build it with
// NewConfig and functional options rather than filling it directly.
type Config struct {
	PageSize    PageSize
	PageSizeSet bool // true when the caller chose a size; CSS @page size is then ignored
	Margins     Margins
	MarginsSet  bool // true when the caller chose margins; CSS @page margin is then ignored

	BaseURL string         // base for relative URLs (file path, directory or URL)
	FS      fs.FS          // optional file system for relative resources; overrides disk access
	Loader  ResourceLoader // custom loader; when nil a default loader is built from the fields below

	AllowHTTP       bool  // allow http(s) resources (stylesheets, images, fonts); off by default
	AllowFile       bool  // allow local file resources; on by default
	MaxResourceSize int64 // per-resource byte limit
	MaxPages        int   // fail with ErrTooManyPages beyond this many pages; <= 0 means no limit

	Stylesheets []string // extra author CSS applied after the document's own styles
	MediaType   string   // CSS media type used for @media evaluation; "print" by default

	Fonts       []FontSource
	FontDirs    []string
	SystemFonts bool // scan the OS font directories for families; on by default

	Metadata Metadata
	Outline  bool // build PDF bookmarks from headings; on by default
	Tagged   bool // mark the PDF as tagged

	// Markdown-specific.
	MarkdownCSS        string // replaces the built-in Markdown theme when non-empty
	NoMarkdownDefaults bool   // skip the built-in Markdown theme entirely

	Warn func(msg string) // receives non-fatal problems (missing images, unsupported CSS, ...)
}

// Option configures a conversion.
type Option func(*Config)

// NewConfig returns the default configuration with opts applied.
func NewConfig(opts ...Option) *Config {
	c := &Config{
		PageSize:        A4,
		Margins:         UniformMargins(Cm(1)),
		AllowFile:       true,
		MaxResourceSize: 50 << 20,
		MaxPages:        DefaultMaxPages,
		MediaType:       "print",
		SystemFonts:     true,
		Outline:         true,
	}
	for _, o := range opts {
		if o != nil {
			o(c)
		}
	}
	return c
}

// Warnf reports a non-fatal problem to the configured warning handler.
func (c *Config) Warnf(msg string) {
	if c.Warn != nil {
		c.Warn(msg)
	}
}

// WithPageSize sets the page size and makes it take precedence over CSS @page size.
func WithPageSize(s PageSize) Option {
	return func(c *Config) { c.PageSize, c.PageSizeSet = s, true }
}

// WithMargins sets page margins and makes them take precedence over CSS @page margins.
func WithMargins(m Margins) Option {
	return func(c *Config) { c.Margins, c.MarginsSet = m, true }
}

// WithBaseURL sets the base used to resolve relative URLs (a directory, file path or http URL).
func WithBaseURL(base string) Option { return func(c *Config) { c.BaseURL = base } }

// WithFS resolves relative resources from fsys instead of the local disk.
func WithFS(fsys fs.FS) Option { return func(c *Config) { c.FS = fsys } }

// WithResourceLoader replaces the default resource loader.
func WithResourceLoader(l ResourceLoader) Option { return func(c *Config) { c.Loader = l } }

// WithHTTP allows (or forbids) fetching http and https resources.
func WithHTTP(allow bool) Option { return func(c *Config) { c.AllowHTTP = allow } }

// WithLocalFiles allows (or forbids) reading local files referenced by the document.
func WithLocalFiles(allow bool) Option { return func(c *Config) { c.AllowFile = allow } }

// WithMaxResourceSize limits the size of each fetched resource in bytes.
func WithMaxResourceSize(n int64) Option { return func(c *Config) { c.MaxResourceSize = n } }

// DefaultMaxPages is the default page limit. It stops runaway documents (for
// example a 1e9px margin) from exhausting memory.
const DefaultMaxPages = 10000

// WithMaxPages limits the number of pages a conversion may produce; n <= 0
// removes the limit.
func WithMaxPages(n int) Option { return func(c *Config) { c.MaxPages = n } }

// WithStylesheet adds author CSS applied after the document's own stylesheets.
func WithStylesheet(css string) Option {
	return func(c *Config) { c.Stylesheets = append(c.Stylesheets, css) }
}

// WithMediaType sets the CSS media type ("print" or "screen").
func WithMediaType(t string) Option { return func(c *Config) { c.MediaType = t } }

// WithFont registers a TrueType font under a CSS family name.
func WithFont(f FontSource) Option { return func(c *Config) { c.Fonts = append(c.Fonts, f) } }

// WithFontDir adds a directory that is scanned for .ttf fonts.
func WithFontDir(dir string) Option { return func(c *Config) { c.FontDirs = append(c.FontDirs, dir) } }

// WithSystemFonts enables or disables scanning the OS font directories.
// With it disabled and no other fonts, the PDF standard 14 fonts are used.
func WithSystemFonts(enabled bool) Option { return func(c *Config) { c.SystemFonts = enabled } }

// WithMetadata sets document metadata; non-empty fields override values from the source.
func WithMetadata(m Metadata) Option { return func(c *Config) { c.Metadata = m } }

// WithTitle sets the document title.
func WithTitle(t string) Option { return func(c *Config) { c.Metadata.Title = t } }

// WithOutline enables or disables bookmarks generated from headings.
func WithOutline(enabled bool) Option { return func(c *Config) { c.Outline = enabled } }

// WithTagged marks the output as a tagged PDF.
func WithTagged(enabled bool) Option { return func(c *Config) { c.Tagged = enabled } }

// WithMarkdownStylesheet replaces the built-in Markdown theme with css.
func WithMarkdownStylesheet(css string) Option { return func(c *Config) { c.MarkdownCSS = css } }

// WithoutMarkdownTheme renders Markdown with browser default styles only.
func WithoutMarkdownTheme() Option { return func(c *Config) { c.NoMarkdownDefaults = true } }

// WithWarningHandler receives non-fatal conversion warnings.
func WithWarningHandler(fn func(msg string)) Option { return func(c *Config) { c.Warn = fn } }
