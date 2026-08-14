package content

// Math is a LaTeX formula that sits within a line, written as `$formula$`.
// Logseq renders it with KaTeX.
type Math struct {
	baseNode

	// Value is the LaTeX of the formula.
	Value string

	// Displayed is whether the formula is set apart from the text around it,
	// which is what `$$formula$$` does, instead of being rendered within the
	// line it is on.
	Displayed bool
}

func NewMath(value string) *Math {
	return &Math{
		Value: value,
	}
}

// NewDisplayedMath creates a formula that is set apart from the text around
// it, which is written as `$$formula$$`.
func NewDisplayedMath(value string) *Math {
	return &Math{
		Value:     value,
		Displayed: true,
	}
}

// WithValue sets the LaTeX of the formula.
func (m *Math) WithValue(value string) *Math {
	m.Value = value
	return m
}

// WithDisplayed sets whether the formula is set apart from the text around it.
func (m *Math) WithDisplayed(displayed bool) *Math {
	m.Displayed = displayed
	return m
}

func (m *Math) isInline() {}

func (m *Math) debug(p *debugPrinter) {
	p.StartType("Math")
	p.Field("value", m.Value)
	if m.Displayed {
		p.Field("displayed", "true")
	}
	p.EndType()
}

var _ InlineNode = (*Math)(nil)

// MathBlock is a LaTeX formula written on lines of its own, which is what the
// math block of Logseq is:
//
//	$$
//	E = mc^2
//	$$
type MathBlock struct {
	baseNode
	previousLineAwareImpl

	// Value is the LaTeX of the formula, which is everything between the lines
	// that mark the block out.
	Value string
}

func NewMathBlock(value string) *MathBlock {
	return &MathBlock{
		Value: value,
	}
}

// WithValue sets the LaTeX of the formula.
func (m *MathBlock) WithValue(value string) *MathBlock {
	m.Value = value
	return m
}

func (m *MathBlock) WithPreviousLineType(lineType PreviousLineType) *MathBlock {
	m.previousLineType = lineType
	return m
}

func (m *MathBlock) debug(p *debugPrinter) {
	p.StartType("MathBlock")
	p.Field("value", m.Value)
	debugPreviousLineAware(p, m)
	p.EndType()
}

func (m *MathBlock) isBlock() {}

var _ BlockNode = (*MathBlock)(nil)
