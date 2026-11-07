package content

type CodeBlock struct {
	baseNode
	previousLineAwareImpl

	// Language is the programming language of this code block.
	Language string

	// Options is the text after the language on the line of the opening
	// fence, such as `{:title "x"}` in "```clojure {:title "x"}". It is only
	// written when the code block has a language.
	Options string

	// Fence is the characters of the opening and closing fence, such as
	// `~~~` or "````". It is empty for "```". A fence that is not three or more
	// of "`" or "~" is written as "```". When a line of the code would close
	// the fence, a longer fence of the same character is written.
	Fence string

	// Code is the raw value of the code block.
	Code string

	// infoSource is the text after the opening fence as it is in the file.
	infoSource string
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

// WithOptions sets the text after the language on the line of the opening
// fence. See Options.
func (c *CodeBlock) WithOptions(options string) *CodeBlock {
	c.Options = options
	return c
}

// WithFence sets the characters of the fence, such as `~~~`. See Fence.
func (c *CodeBlock) WithFence(fence string) *CodeBlock {
	c.Fence = fence
	return c
}

// InfoSource gets the text after the opening fence as it is in the file, such
// as ` js  ` in "``` js  ". It is empty for a code block that was not read
// from a file. The text is written back when it still gives the same Language
// and Options, so that its spaces do not change.
func (c *CodeBlock) InfoSource() string {
	return c.infoSource
}

// SetInfoSource sets the text after the opening fence. See InfoSource.
func (c *CodeBlock) SetInfoSource(source string) {
	c.infoSource = source
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
	if c.Options != "" {
		p.Field("options", c.Options)
	}
	if c.Fence != "" {
		p.Field("fence", c.Fence)
	}
	p.Field("code", c.Code)
	p.EndType()
}

func (c *CodeBlock) isBlock() {}

var _ BlockNode = (*CodeBlock)(nil)
