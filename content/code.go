package content

type CodeBlock struct {
	baseNode
	previousLineAwareImpl

	// Language is the programming language of this code block.
	Language string

	// Code is the raw value of the code block.
	Code string
}

func NewCodeBlock(code string) *CodeBlock {
	return &CodeBlock{
		Code: code,
	}
}

// WithLanguage sets the language of the code block.
func (c *CodeBlock) WithLanguage(language string) *CodeBlock {
	c.Language = language
	return c
}

// WithCode sets the code of the code block.
func (c *CodeBlock) WithCode(code string) *CodeBlock {
	c.Code = code
	return c
}

func (c *CodeBlock) WithPreviousLineType(previousLineType PreviousLineType) *CodeBlock {
	c.previousLineType = previousLineType
	return c
}

func (c *CodeBlock) debug(p *debugPrinter) {
	p.StartType("CodeBlock")
	p.Field("language", c.Language)
	p.Field("code", c.Code)
	p.EndType()
}

func (c *CodeBlock) isBlock() {}

var _ BlockNode = (*CodeBlock)(nil)
