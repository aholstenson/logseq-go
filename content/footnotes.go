package content

// FootnoteRef points at a footnote from within the text, written as
// `[^label]`. The label is what ties it to the definition that holds the
// content of the footnote.
type FootnoteRef struct {
	baseNode

	// Label identifies the footnote, and is the same as the label of the
	// definition it points at.
	Label string
}

func NewFootnoteRef(label string) *FootnoteRef {
	return &FootnoteRef{
		Label: label,
	}
}

// WithLabel sets the label of the footnote this points at.
func (f *FootnoteRef) WithLabel(label string) *FootnoteRef {
	f.Label = label
	return f
}

func (f *FootnoteRef) isInline() {}

func (f *FootnoteRef) debug(p *debugPrinter) {
	p.StartType("FootnoteRef")
	p.Field("label", f.Label)
	p.EndType()
}

var _ InlineNode = (*FootnoteRef)(nil)

// FootnoteDefinition holds the content of a footnote, written as
// `[^label]: content` on a line of its own.
type FootnoteDefinition struct {
	baseNodeWithChildren
	previousLineAwareImpl

	// Label identifies the footnote, and is what the references to it point
	// at.
	Label string
}

func NewFootnoteDefinition(label string, children ...Node) *FootnoteDefinition {
	f := &FootnoteDefinition{
		Label: label,
	}
	f.self = f
	f.childValidator = allowOnlyBlockNodes
	for _, child := range AddAutomaticParagraphs(children) {
		f.AddChild(child)
	}
	return f
}

// WithLabel sets the label of the footnote.
func (f *FootnoteDefinition) WithLabel(label string) *FootnoteDefinition {
	f.Label = label
	return f
}

func (f *FootnoteDefinition) WithPreviousLineType(lineType PreviousLineType) *FootnoteDefinition {
	f.previousLineType = lineType
	return f
}

func (f *FootnoteDefinition) debug(p *debugPrinter) {
	p.StartType("FootnoteDefinition")
	p.Field("label", f.Label)
	debugPreviousLineAware(p, f)
	p.Children(f)
	p.EndType()
}

func (f *FootnoteDefinition) isBlock() {}

var _ BlockNode = (*FootnoteDefinition)(nil)
