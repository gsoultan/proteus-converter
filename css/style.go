package css

// Side indexes for four-sided properties.
const (
	Top = iota
	Right
	Bottom
	Left
)

// LineHeight is a computed line-height: 'normal', a unitless factor
// (inherited as a factor) or an absolute length in points.
type LineHeight struct {
	Normal bool
	Factor float64
	Pt     float64
}

// Style holds the computed values of one element (or pseudo-element).
// Lengths are in points; percentages stay unresolved in Length values.
type Style struct {
	Display  string
	Position string
	Float    string
	Clear    string

	Color           Color
	BackgroundColor Color
	BackgroundImage string // url target, if any
	Opacity         float64
	Visibility      string

	FontFamily  []string
	FontSize    float64
	FontWeight  int
	FontStyle   string
	FontVariant string
	keywordSize float64 // unscaled size when font-size came from a keyword (monospace quirk)

	LineHeight     LineHeight
	TextAlign      string
	TextAlignLast  string
	TextIndent     Length
	TextTransform  string
	WhiteSpace     string
	LetterSpacing  float64
	WordSpacing    float64
	VerticalAlign  string
	VerticalLength Length // when VerticalAlign == "length"
	OverflowWrap   string
	WordBreak      string
	Hyphens        string
	Direction      string

	TextDecorationLine  Decoration
	TextDecorationColor *Color // nil means currentColor
	TextDecorationStyle string

	Margin       [4]Length
	Padding      [4]Length
	BorderWidth  [4]float64
	BorderStyle  [4]string
	BorderColor  [4]Color
	BorderRadius [4]Length // top-left, top-right, bottom-right, bottom-left

	Width, Height       Length
	MinWidth, MinHeight Length
	MaxWidth, MaxHeight Length // Auto means none
	BoxSizing           string
	Overflow            string

	ListStyleType     string
	ListStylePosition string
	ListStyleImage    string

	BreakBefore string
	BreakAfter  string
	BreakInside string
	Orphans     int
	Widows      int

	BorderCollapse string
	BorderSpacingH float64
	BorderSpacingV float64
	TableLayout    string
	CaptionSide    string
	EmptyCells     string

	Content          string // generated content (::before/::after/::marker), already resolved
	ContentNone      bool
	CounterReset     string
	CounterIncrement string

	PrintColorAdjust string

	// Custom holds custom properties (--*), already var()-substituted.
	Custom map[string]string
}

// Decoration is a set of text-decoration lines.
type Decoration struct {
	Underline, Overline, LineThrough bool
}

// Any reports whether any line is set.
func (d Decoration) Any() bool { return d.Underline || d.Overline || d.LineThrough }

// Initial returns the initial style used for the root element.
func Initial() *Style {
	s := &Style{}
	s.resetNonInherited()
	s.Color = Black
	s.FontFamily = []string{"serif"}
	s.FontSize = 12 // 16px
	s.keywordSize = 12
	s.FontWeight = 400
	s.FontStyle = "normal"
	s.FontVariant = "normal"
	s.LineHeight = LineHeight{Normal: true}
	s.TextAlign = "start"
	s.TextAlignLast = "auto"
	s.TextTransform = "none"
	s.WhiteSpace = "normal"
	s.Visibility = "visible"
	s.OverflowWrap = "normal"
	s.WordBreak = "normal"
	s.Hyphens = "manual"
	s.Direction = "ltr"
	s.ListStyleType = "disc"
	s.ListStylePosition = "outside"
	s.Orphans = 2
	s.Widows = 2
	s.BorderCollapse = "separate"
	s.CaptionSide = "top"
	s.EmptyCells = "show"
	s.PrintColorAdjust = "economy"
	return s
}

// resetNonInherited sets every non-inherited property to its initial value.
func (s *Style) resetNonInherited() {
	s.Display = "inline"
	s.Position = "static"
	s.Float = "none"
	s.Clear = "none"
	s.BackgroundColor = Transparent
	s.BackgroundImage = ""
	s.Opacity = 1
	s.VerticalAlign = "baseline"
	s.VerticalLength = Length{}
	s.TextDecorationLine = Decoration{}
	s.TextDecorationColor = nil
	s.TextDecorationStyle = "solid"
	s.Margin = [4]Length{}
	s.Padding = [4]Length{}
	s.BorderWidth = [4]float64{2.25, 2.25, 2.25, 2.25} // medium = 3px
	s.BorderStyle = [4]string{"none", "none", "none", "none"}
	s.BorderColor = [4]Color{}
	s.BorderRadius = [4]Length{}
	s.Width, s.Height = Auto, Auto
	s.MinWidth, s.MinHeight = Length{}, Length{}
	s.MaxWidth, s.MaxHeight = Auto, Auto
	s.BoxSizing = "content-box"
	s.Overflow = "visible"
	s.BreakBefore, s.BreakAfter, s.BreakInside = "auto", "auto", "auto"
	s.TableLayout = "auto"
	s.Content = ""
	s.ContentNone = false
	s.CounterReset, s.CounterIncrement = "", ""
}

// Inherit returns a child style: inherited properties copied from s,
// non-inherited properties at their initial values.
func (s *Style) Inherit() *Style {
	c := *s
	c.resetNonInherited()
	return &c
}

// BorderUsed returns the used border width of a side (0 when the style is none/hidden).
func (s *Style) BorderUsed(side int) float64 {
	switch s.BorderStyle[side] {
	case "none", "hidden", "":
		return 0
	}
	return s.BorderWidth[side]
}

// LineHeightPt resolves the used line-height for a font with the given
// normal line height (ascent + descent + gap, in points).
func (s *Style) LineHeightPt(normal float64) float64 {
	switch {
	case s.LineHeight.Normal:
		return normal
	case s.LineHeight.Factor != 0:
		return s.LineHeight.Factor * s.FontSize
	default:
		return s.LineHeight.Pt
	}
}

// IsBlockLevel reports whether the display value creates a block-level box.
func (s *Style) IsBlockLevel() bool {
	switch s.Display {
	case "block", "list-item", "table", "flex", "grid", "flow-root", "table-caption",
		"table-row-group", "table-header-group", "table-footer-group", "table-row", "table-cell",
		"table-column", "table-column-group":
		return true
	}
	return false
}

// DecorationColor returns the used text-decoration color.
func (s *Style) DecorationColor() Color {
	if s.TextDecorationColor != nil {
		return *s.TextDecorationColor
	}
	return s.Color
}
