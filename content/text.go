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
	// the break. If it is empty, a hard line break is written as a
	// backslash. It is not used if the text has no line break.
	TrailingSpace string

	// LazyLine is whether the line after the line break of the text is in a
	// quote but is written without the `>` marker, as `b` is in "> a\nb".
	// Logseq and CommonMark read such a line as a part of the quote. It is
	// written back without the marker, so that saving does not change the
	// line. It is not used if the text has no line break.
	LazyLine bool
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

type CodeSpan struct {
	baseNode

	// Value is the text of this node.
	Value string
}

func NewCodeSpan(text string) *CodeSpan {
	return &CodeSpan{
		Value: text,
	}
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
}

func NewEmphasis(children ...Node) *Emphasis {
	e := &Emphasis{}
	e.self = e
	e.childValidator = allowOnlyInlineNodes
	e.AddChildren(children...)
	return e
}

func (e *Emphasis) isInline() {}

func (e *Emphasis) debug(p *debugPrinter) {
	p.StartType("Emphasis")
	p.Children(e)
	p.EndType()
}

var _ InlineNode = (*Emphasis)(nil)

type Strong struct {
	baseNodeWithChildren
}

func NewStrong(children ...Node) *Strong {
	s := &Strong{}
	s.self = s
	s.childValidator = allowOnlyInlineNodes
	s.AddChildren(children...)
	return s
}

func (s *Strong) isInline() {}

func (s *Strong) debug(p *debugPrinter) {
	p.StartType("Strong")
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
}

func NewHighlight(children ...Node) *Highlight {
	h := &Highlight{}
	h.self = h
	h.childValidator = allowOnlyInlineNodes
	h.AddChildren(children...)
	return h
}

func (h *Highlight) isInline() {}

func (h *Highlight) debug(p *debugPrinter) {
	p.StartType("Highlight")
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

	// Level is the level of this heading.
	Level int
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

func (t *ThematicBreak) debug(p *debugPrinter) {
	p.StartType("ThematicBreak")
	debugPreviousLineAware(p, t)
	p.EndType()
}

func (t *ThematicBreak) isBlock() {}

var _ BlockNode = (*ThematicBreak)(nil)
