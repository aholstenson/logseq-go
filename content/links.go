package content

// HasLinkURL is used for nodes that are links, such as `Link` and `AutoLink`.
type HasLinkURL interface {
	Node

	// GetURL gets the target of this link.
	GetURL() string
}

type Link struct {
	baseNodeWithChildren

	// URL is the target of the link.
	URL string
	// Title of this link as displayed on hover in a browser.
	Title string
}

func NewLink(url string, text ...Node) *Link {
	link := &Link{
		URL: url,
	}
	link.self = link
	link.childValidator = allowOnlyInlineNodes
	link.AddChildren(text...)
	return link
}

// GetURL returns the URL of this link.
func (l *Link) GetURL() string {
	return l.URL
}

// WithURL sets the URL of this link.
func (l *Link) WithURL(url string) *Link {
	l.URL = url
	return l
}

// WithTitle sets the title of this link. The title is the text displayed on
// hover in a browser.
func (l *Link) WithTitle(title string) *Link {
	l.Title = title
	return l
}

func (l *Link) isInline() {}

func (l *Link) debug(p *debugPrinter) {
	p.StartType("Link")
	p.Field("url", l.URL)
	p.Field("title", l.Title)
	p.Children(l)
	p.EndType()
}

var _ InlineNode = (*Link)(nil)
var _ HasLinkURL = (*Link)(nil)

type AutoLink struct {
	baseNode

	// URL is where this link points to.
	URL string
}

func NewAutoLink(url string) *AutoLink {
	return &AutoLink{
		URL: url,
	}
}

// GetURL returns the URL of this link.
func (l *AutoLink) GetURL() string {
	return l.URL
}

// WithURL sets the URL of this link.
func (l *AutoLink) WithURL(target string) *AutoLink {
	l.URL = target
	return l
}

// ReplaceWithLink replaces this node with a new `Link` node containing the
// given text.
func (l *AutoLink) ReplaceWithLink(text ...Node) *Link {
	newLink := NewLink(l.URL, text...)
	l.ReplaceWith(newLink)
	return newLink
}

func (l *AutoLink) isInline() {}

func (l *AutoLink) debug(p *debugPrinter) {
	p.StartType("AutoLink")
	p.Field("url", l.URL)
	p.EndType()
}

var _ InlineNode = (*AutoLink)(nil)
var _ HasLinkURL = (*AutoLink)(nil)

// PageRef is a reference to a page, such as a `PageLink` or `Hashtag`.
type PageRef interface {
	Node

	isPageRef()

	GetTo() string
}

type PageLink struct {
	baseNode

	// To is the target of the link.
	To string
}

func NewPageLink(target string) *PageLink {
	return &PageLink{
		To: target,
	}
}

func (l *PageLink) GetTo() string {
	return l.To
}

func (l *PageLink) WithTo(target string) *PageLink {
	l.To = target
	return l
}

func (l *PageLink) isInline() {}

func (l *PageLink) debug(p *debugPrinter) {
	p.StartType("PageLink")
	p.Field("to", l.To)
	p.EndType()
}

func (l *PageLink) isPageRef() {}

var _ InlineNode = (*PageLink)(nil)
var _ PageRef = (*PageLink)(nil)

type Hashtag struct {
	baseNode

	// Page is the target of the link.
	To string
}

func NewHashtag(target string) *Hashtag {
	return &Hashtag{
		To: target,
	}
}

func (l *Hashtag) GetTo() string {
	return l.To
}

func (l *Hashtag) WithTo(target string) *Hashtag {
	l.To = target
	return l
}

func (l *Hashtag) isInline() {}

func (l *Hashtag) debug(p *debugPrinter) {
	p.StartType("TagLink")
	p.Field("to", l.To)
	p.EndType()
}

func (l *Hashtag) isPageRef() {}

var _ InlineNode = (*Hashtag)(nil)
var _ PageRef = (*Hashtag)(nil)

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

func (l *BlockRef) WithID(id string) *BlockRef {
	l.ID = id
	return l
}

func (b *BlockRef) debug(p *debugPrinter) {
	p.StartType("BlockRef")
	p.Field("id", b.ID)
	p.EndType()
}

func (l *BlockRef) isInline() {}

var _ InlineNode = (*BlockRef)(nil)
