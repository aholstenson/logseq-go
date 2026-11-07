package content

import (
	"strconv"
)

type Text struct {
	baseNode

	// Value is the text of this node.
	Value string

	HardLineBreak bool

	SoftLineBreak bool

	// TrailingSpace is the whitespace between the text and its line break,
	// as it is in the file. It is written back so that saving does not
	// change the line. For a hard line break it holds the spaces that make
	// the break. A hard line break is written with two spaces if it does not
	// end with two spaces, and BackslashLineBreak is false. It is not used
	// if the text has no line break.
	TrailingSpace string

	// BackslashLineBreak is whether the hard line break of the text is
	// written as a backslash at the end of the line, as it is in the file.
	// Logseq shows the backslash as text, so a new hard line break is
	// written with two spaces.
	BackslashLineBreak bool

	// LazyLine is whether the line after the line break of the text is in a
	// quote but is written without the `>` marker, as `b` is in "> a\nb", or
	// is in a footnote definition or a list item but is not indented under
	// it, as `b` is in "[^1]: a\nb" and in "* a\nb". Logseq and CommonMark
	// read such a line as a part of the quote, the footnote or the list item.
	// It is written back without the marker or the indentation, so that
	// saving does not change the line. It is not used if the text has no line
	// break.
	LazyLine bool

	// LazyLineMarkers is the number of `>` markers that a lazy line of a
	// nested quote still has, as the one marker before `b` in "> > a\n> b".
	// The markers of the outer quotes are written, and the others are left
	// out. If it is zero, the line is written without any marker. It is not
	// used if LazyLine is false.
	LazyLineMarkers int

	// NextLineIndentation is the spaces and tabs at the start of the line
	// after the line break of the text, as in the two spaces before `b` in
	// "a\n  b". It does not include the indentation of the block or quote
	// that holds the text. It is written back so that saving does not change
	// the line. It is not used if the text has no line break.
	NextLineIndentation string

	// LazyBlockLine is whether the line after the line break of the text is
	// indented less than the text of the block that holds it, as `b` is in
	// "- a\nb". Logseq reads such a line as a part of the block. It is
	// written without the indentation of the text of the block, and
	// NextLineIndentation then holds its indentation after the column of the
	// bullet of the block. It is not used if the text has no line break.
	LazyBlockLine bool

	source string
}

func NewText(text string) *Text {
	return &Text{
		Value: text,
	}
}

// WithValue sets the value of the text.
func (t *Text) WithValue(value string) *Text {
	t.Value = value
	return t
}

func (t *Text) WithHardLineBreak() *Text {
	t.HardLineBreak = true
	t.SoftLineBreak = false
	return t
}

func (t *Text) WithSoftLineBreak() *Text {
	t.SoftLineBreak = true
	t.HardLineBreak = false
	return t
}

// WithLazyLine marks the line after the line break of the text as a lazy
// line of a quote, which is written without the `>` marker. See LazyLine.
func (t *Text) WithLazyLine() *Text {
	t.LazyLine = true
	return t
}

func (t *Text) WithNoLineBreak() *Text {
	t.SoftLineBreak = false
	t.HardLineBreak = false
	return t
}

// TextSource gets the text as it is in the file, with the backslashes that
// escape punctuation, such as `a \* b` for the value `a * b`. It is empty for
// a text that was not read from a file, or that has no such backslashes and
// does not look like block syntax at the start of a line, such as `3. a`. A
// text whose value is still the value of this text is written with its
// backslashes, so that they do not change on save.
func (t *Text) TextSource() string {
	return t.source
}

// SetTextSource sets the text as it is in the file, see TextSource.
func (t *Text) SetTextSource(source string) {
	t.source = source
}

func (t *Text) isInline() {}

func (t *Text) debug(p *debugPrinter) {
	p.StartType("Text")
	p.Field("value", t.Value)
	if t.HardLineBreak {
		p.Field("lineBreak", "hard")
	} else if t.SoftLineBreak {
		p.Field("lineBreak", "soft")
	}
	if t.LazyLine {
		p.Field("lazyLine", "true")
	}
	p.EndType()
}

var _ InlineNode = (*Text)(nil)

// RawText is used to wrap a string of text that should not be processed as
// Markdown. This is useful for when you want to include pre-generated
// Markdown.
type RawText struct {
	baseNode

	// Value is the text of this node.
	Value string
}

func NewRawText(text string) *RawText {
	return &RawText{
		Value: text,
	}
}

func (t *RawText) isInline() {}

func (t *RawText) debug(p *debugPrinter) {
	p.StartType("RawText")
	p.Field("value", t.Value)
	p.EndType()
}

var _ InlineNode = (*RawText)(nil)

// CodeSpan is code inside a line of text, written as `code`.
type CodeSpan struct {
	baseNode

	// Value is the text of this node.
	Value string

	source string
}

func NewCodeSpan(text string) *CodeSpan {
	return &CodeSpan{
		Value: text,
	}
}

// CodeSpanSource gets the text of the code span as it is in the file, with
// its backticks and the spaces inside them, such as "“ a ` b “". It is
// empty for a code span that was not read from a file. A code span whose
// text reads back as its Value is written with this text, so that its
// backticks and spaces do not change.
func (c *CodeSpan) CodeSpanSource() string {
	return c.source
}

// SetCodeSpanSource sets the text of the code span as it is in the file, see
// CodeSpanSource.
func (c *CodeSpan) SetCodeSpanSource(source string) {
	c.source = source
}

func (c *CodeSpan) isInline() {}

func (c *CodeSpan) debug(p *debugPrinter) {
	p.StartType("Code")
	p.Field("value", c.Value)
	p.EndType()
}

var _ InlineNode = (*CodeSpan)(nil)

type Emphasis struct {
	baseNodeWithChildren

	// Underscore is whether the emphasis is written with `_` markers, such
	// as `_text_`, instead of `*` markers. The `*` markers are used where `_`
	// would not be read back as emphasis, such as directly after a letter.
	Underscore bool
}

func NewEmphasis(children ...Node) *Emphasis {
	e := &Emphasis{}
	e.self = e
	e.childValidator = allowOnlyInlineNodes
	e.AddChildren(children...)
	return e
}

// WithUnderscore sets if the emphasis is written with `_` markers. See
// Underscore.
func (e *Emphasis) WithUnderscore(underscore bool) *Emphasis {
	e.Underscore = underscore
	return e
}

func (e *Emphasis) isInline() {}

func (e *Emphasis) debug(p *debugPrinter) {
	p.StartType("Emphasis")
	if e.Underscore {
		p.Field("marker", "_")
	}
	p.Children(e)
	p.EndType()
}

var _ InlineNode = (*Emphasis)(nil)

type Strong struct {
	baseNodeWithChildren

	// Underscore is whether the strong text is written with `__` markers,
	// such as `__text__`, instead of `**` markers. The `**` markers are used
	// where `__` would not be read back as strong text, such as directly
	// after a letter.
	Underscore bool
}

func NewStrong(children ...Node) *Strong {
	s := &Strong{}
	s.self = s
	s.childValidator = allowOnlyInlineNodes
	s.AddChildren(children...)
	return s
}

// WithUnderscore sets if the strong text is written with `__` markers. See
// Underscore.
func (s *Strong) WithUnderscore(underscore bool) *Strong {
	s.Underscore = underscore
	return s
}

func (s *Strong) isInline() {}

func (s *Strong) debug(p *debugPrinter) {
	p.StartType("Strong")
	if s.Underscore {
		p.Field("marker", "__")
	}
	p.Children(s)
	p.EndType()
}

var _ InlineNode = (*Strong)(nil)

type Paragraph struct {
	baseNodeWithChildren
	previousLineAwareImpl

	// Indentation is the whitespace before each line of the paragraph, after
	// the indentation of the block or other node that holds it. It is empty
	// for most paragraphs. Logseq has no indented code blocks, so a paragraph
	// that starts on a line indented four or more columns keeps this
	// indentation as part of its text, and it is written back on save.
	Indentation string

	// TrailingSpace is the spaces and tabs at the end of the last line of the
	// paragraph, as it is in the file. It is written back so that saving does
	// not change the line.
	TrailingSpace string

	// FirstLineIndentation is the spaces and tabs at the start of the first
	// line of the paragraph, as it is in the file, such as the two spaces
	// before `a` in "# H\n  a". It does not include the indentation of the
	// block or quote that holds the paragraph. It is written back so that
	// saving does not change the line.
	FirstLineIndentation string
}

type Strikethrough struct {
	baseNodeWithChildren
}

func NewStrikethrough(children ...Node) *Strikethrough {
	s := &Strikethrough{}
	s.self = s
	s.childValidator = allowOnlyInlineNodes
	s.AddChildren(children...)
	return s
}

func (s *Strikethrough) isInline() {}

func (s *Strikethrough) debug(p *debugPrinter) {
	p.StartType("Strikethrough")
	p.Children(s)
	p.EndType()
}

var _ InlineNode = (*Strikethrough)(nil)

// Highlight is text that Logseq marks out, written as `^^text^^`.
type Highlight struct {
	baseNodeWithChildren

	// EqualSigns is whether the highlight is written with `==` markers, such
	// as `==text==`, instead of `^^` markers. Logseq reads both.
	EqualSigns bool
}

func NewHighlight(children ...Node) *Highlight {
	h := &Highlight{}
	h.self = h
	h.childValidator = allowOnlyInlineNodes
	h.AddChildren(children...)
	return h
}

// WithEqualSigns sets if the highlight is written with `==` markers. See
// EqualSigns.
func (h *Highlight) WithEqualSigns(equalSigns bool) *Highlight {
	h.EqualSigns = equalSigns
	return h
}

func (h *Highlight) isInline() {}

func (h *Highlight) debug(p *debugPrinter) {
	p.StartType("Highlight")
	if h.EqualSigns {
		p.Field("marker", "==")
	}
	p.Children(h)
	p.EndType()
}

var _ InlineNode = (*Highlight)(nil)

func NewParagraph(children ...Node) *Paragraph {
	p := &Paragraph{}
	p.self = p
	p.childValidator = allowInlineNodesAndProperties
	p.AddChildren(children...)
	return p
}

func (p *Paragraph) debug(pr *debugPrinter) {
	pr.StartType("Paragraph")
	debugPreviousLineAware(pr, p)
	if p.Indentation != "" {
		pr.Field("indentation", strconv.Quote(p.Indentation))
	}
	pr.Children(p)
	pr.EndType()
}

func (p *Paragraph) WithPreviousLineType(t PreviousLineType) *Paragraph {
	p.previousLineType = t
	return p
}

// WithIndentation sets the indentation of the paragraph, see Indentation.
func (p *Paragraph) WithIndentation(indentation string) *Paragraph {
	p.Indentation = indentation
	return p
}

func (p *Paragraph) isBlock() {}

var _ BlockNode = (*Paragraph)(nil)

type Blockquote struct {
	baseNodeWithChildren
	previousLineAwareImpl

	// NoSpaceAfterMarker is whether the `>` markers of the quote are written
	// without a space after them, as in ">quote", as they are in the file.
	NoSpaceAfterMarker bool

	// BlankFirstLine and BlankLastLine are whether the quote starts or ends
	// with a line that has only the `>` marker, as in ">\n> a\n>", as it is
	// in the file. Such lines are not a part of the content of the quote.
	BlankFirstLine bool
	BlankLastLine  bool
}

func NewBlockquote(children ...Node) *Blockquote {
	b := &Blockquote{}
	b.self = b
	b.childValidator = allowOnlyBlockNodes
	for _, child := range AddAutomaticParagraphs(children) {
		b.AddChild(child)
	}
	return b
}

func (b *Blockquote) debug(p *debugPrinter) {
	p.StartType("Blockquote")
	debugPreviousLineAware(p, b)
	p.Children(b)
	p.EndType()
}

func (b *Blockquote) WithPreviousLineType(t PreviousLineType) *Blockquote {
	b.previousLineType = t
	return b
}

func (b *Blockquote) isBlock() {}

var _ BlockNode = (*Blockquote)(nil)

type Heading struct {
	baseNodeWithChildren
	previousLineAwareImpl

	// Level is the level of this heading.
	Level int

	// Indentation is the spaces before the `#` marks of the heading, as it
	// is in the file, such as the space in " # Title". It does not include
	// the indentation of the block or quote that holds the heading.
	Indentation string

	// MarksSpace is the spaces and tabs after the `#` marks of the heading,
	// as it is in the file, such as the three spaces of "##   Title". If it
	// is empty, one space is written between the marks and the text.
	MarksSpace string

	// TrailingSpace is the spaces and tabs at the end of the line of the
	// heading, after its text, as it is in the file. It is written back so
	// that saving does not change the line.
	TrailingSpace string
}

func NewHeading(level int, children ...Node) *Heading {
	h := &Heading{
		Level: level,
	}
	h.self = h
	h.childValidator = allowOnlyInlineNodes
	h.AddChildren(children...)
	return h
}

func (h *Heading) debug(p *debugPrinter) {
	p.StartType("Heading")
	p.Field("Level", strconv.Itoa(h.Level))
	p.Children(h)
	p.EndType()
}

func (h *Heading) isBlock() {}

var _ BlockNode = (*Heading)(nil)

type ThematicBreak struct {
	baseNode
	previousLineAwareImpl

	// Marker is the text of the thematic break as it is in the file, such
	// as `***`, `___` or `* * *`. It is written back as it is. It is empty
	// for `---`. If it is empty or is not a thematic break, `---` is written.
	Marker string

	// Indentation is the spaces before the thematic break, after the
	// indentation of the block or quote that holds it, as it is in the file.
	Indentation string
}

func NewThematicBreak() *ThematicBreak {
	return &ThematicBreak{}
}

// WithPreviousLineType sets the type of the line before the thematic break
// and returns it.
func (t *ThematicBreak) WithPreviousLineType(lineType PreviousLineType) *ThematicBreak {
	t.previousLineType = lineType
	return t
}

// WithMarker sets the text of the thematic break, such as `***`. See Marker.
func (t *ThematicBreak) WithMarker(marker string) *ThematicBreak {
	t.Marker = marker
	return t
}

func (t *ThematicBreak) debug(p *debugPrinter) {
	p.StartType("ThematicBreak")
	if t.Marker != "" {
		p.Field("marker", t.Marker)
	}
	debugPreviousLineAware(p, t)
	p.EndType()
}

func (t *ThematicBreak) isBlock() {}

var _ BlockNode = (*ThematicBreak)(nil)
