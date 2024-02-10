package content

type RawHTML struct {
	baseNode

	// HTML is the HTML of this node.
	HTML string
}

func NewRawHTML(html string) *RawHTML {
	return &RawHTML{
		HTML: html,
	}
}

// WithHTML sets the HTML.
func (h *RawHTML) WithHTML(html string) *RawHTML {
	h.HTML = html
	return h
}

func (h *RawHTML) isInline() {}

func (n *RawHTML) debug(p *debugPrinter) {
	p.StartType("RawHTML")
	p.Field("HTML", n.HTML)
	p.EndType()
}

var _ InlineNode = (*RawHTML)(nil)

type RawHTMLBlock struct {
	baseNode

	// HTML is the HTML of this node.
	HTML string
}

func NewRawHTMLBlock(html string) *RawHTMLBlock {
	return &RawHTMLBlock{
		HTML: html,
	}
}

// WithHTML sets the HTML.
func (h *RawHTMLBlock) WithHTML(html string) *RawHTMLBlock {
	h.HTML = html
	return h
}

func (h *RawHTMLBlock) debug(p *debugPrinter) {
	p.StartType("RawHTMLBlock")
	p.Field("HTML", h.HTML)
	p.EndType()
}

func (h *RawHTMLBlock) isBlock() {}

var _ BlockNode = (*RawHTMLBlock)(nil)
