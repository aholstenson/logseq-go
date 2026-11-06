package content

// Math is a LaTeX formula that sits within a line, written as `$formula$`
// or `\(formula\)`. Logseq renders it with KaTeX.
type Math struct {
	baseNode

	// Value is the LaTeX of the formula.
	Value string

	// Displayed is whether the formula is set apart from the text around it,
	// which is what `$$formula$$` does, instead of being rendered within the
	// line it is on. With backslashes it is written as `\[formula\]`.
	Displayed bool

	// Backslashes is whether the formula is written between backslash
	// delimiters, `\(formula\)` or `\[formula\]`, instead of dollar signs.
	Backslashes bool
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

// WithBackslashes sets whether the formula is written between backslash
// delimiters instead of dollar signs.
func (m *Math) WithBackslashes(backslashes bool) *Math {
	m.Backslashes = backslashes
	return m
}

func (m *Math) isInline() {}

func (m *Math) debug(p *debugPrinter) {
	p.StartType("Math")
	p.Field("value", m.Value)
	if m.Displayed {
		p.Field("displayed", "true")
	}
	if m.Backslashes {
		p.Field("backslashes", "true")
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
//
// It can also be a LaTeX environment, which Logseq renders in the same way:
//
//	\begin{equation}
//	E = mc^2
//	\end{equation}
type MathBlock struct {
	baseNode
	previousLineAwareImpl

	// Value is the LaTeX of the formula, which is everything between the lines
	// that mark the block out.
	Value string

	// Environment is the name of the LaTeX environment of the formula, such
	// as `equation` for `\begin{equation}`. It is empty for a block between
	// two lines of `$$`.
	Environment string
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

// WithEnvironment sets the name of the LaTeX environment of the formula.
func (m *MathBlock) WithEnvironment(environment string) *MathBlock {
	m.Environment = environment
	return m
}

func (m *MathBlock) WithPreviousLineType(lineType PreviousLineType) *MathBlock {
	m.previousLineType = lineType
	return m
}

func (m *MathBlock) debug(p *debugPrinter) {
	p.StartType("MathBlock")
	p.Field("value", m.Value)
	if m.Environment != "" {
		p.Field("environment", m.Environment)
	}
	debugPreviousLineAware(p, m)
	p.EndType()
}

func (m *MathBlock) isBlock() {}

var _ BlockNode = (*MathBlock)(nil)
