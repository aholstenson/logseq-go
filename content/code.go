package content

type CodeBlock struct {
	baseNode

	// Language gets the language of this code block.
	Language string

	// Code gets the code of this code block.
	Code string
}

func NewCodeBlock(code string) *CodeBlock {
	return &CodeBlock{
		Code: code,
	}
}

func (c *CodeBlock) WithLanguage(language string) *CodeBlock {
	c.Language = language
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
