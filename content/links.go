package content

type Link struct {
	baseNodeWithChildren

	// Target gets the target of this link.
	Target string

	// Title gets the title of this link.
	Title string
}

func NewLink(target string, text ...Node) *Link {
	link := &Link{
		Target: target,
	}
	link.self = link
	link.childValidator = allowOnlyInlineNodes
	link.AddChildren(text...)
	return link
}

func (l *Link) WithTitle(title string) *Link {
	l.Title = title
	return l
}

func (l *Link) isInline() {}

func (l *Link) debug(p *debugPrinter) {
	p.StartType("Link")
	p.Field("target", l.Target)
	p.Field("title", l.Title)
	p.Children(l)
	p.EndType()
}

var _ InlineNode = (*Link)(nil)

type AutoLink struct {
	baseNode

	// Target gets the target of this link.
	Target string
}

func NewAutoLink(target string) *AutoLink {
	return &AutoLink{
		Target: target,
	}
}

func (l *AutoLink) isInline() {}

func (l *AutoLink) debug(p *debugPrinter) {
	p.StartType("AutoLink")
	p.Field("target", l.Target)
	p.EndType()
}

var _ InlineNode = (*AutoLink)(nil)

// PageRef is a reference to a page, such as a `PageLink` or `Hashtag`.
type PageRef interface {
	Node

	isPageRef()

	To() string
}

type PageLink struct {
	baseNode

	// To is the target of the link.
	to string
}

func NewPageLink(target string) *PageLink {
	return &PageLink{
		to: target,
	}
}

func (l *PageLink) isInline() {}

func (l *PageLink) debug(p *debugPrinter) {
	p.StartType("PageLink")
	p.Field("to", l.to)
	p.EndType()
}

func (l *PageLink) isPageRef() {}

func (l *PageLink) To() string {
	return l.to
}

var _ InlineNode = (*PageLink)(nil)

type Hashtag struct {
	baseNode

	// Page is the target of the link.
	to string
}

func NewHashtag(target string) *Hashtag {
	return &Hashtag{
		to: target,
	}
}

func (l *Hashtag) isInline() {}

func (l *Hashtag) debug(p *debugPrinter) {
	p.StartType("TagLink")
	p.Field("to", l.to)
	p.EndType()
}

func (l *Hashtag) isPageRef() {}

func (l *Hashtag) To() string {
	return l.to
}

var _ InlineNode = (*Hashtag)(nil)

type BlockRef struct {
	baseNode

	// ID is the id of the block to reference.
	ID string
}

func NewBlockRef(id string) *BlockRef {
	return &BlockRef{
		ID: id,
	}
}

func (b *BlockRef) debug(p *debugPrinter) {
	p.StartType("BlockRef")
	p.Field("id", b.ID)
	p.EndType()
}

func (l *BlockRef) isInline() {}

var _ InlineNode = (*BlockRef)(nil)
