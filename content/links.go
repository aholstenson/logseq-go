package content

// HasLinkURL is used for nodes that are links, such as `Link` and `AutoLink`.
type HasLinkURL interface {
	Node

	// GetURL gets the target of this link.
	GetURL() string
}

// DestinationSourceAware is implemented by the nodes that are read from a
// link or an image with a destination in parentheses, such as Link and Image.
// They keep the text between the parentheses as it is in the file. A node
// whose text reads back as the same URL and title is written with this text,
// so that its quotes, angle brackets and backslashes do not change.
type DestinationSourceAware interface {
	// DestinationSource gets the text between the parentheses, such as
	// `http://a.com "title"`. It is empty for a node that was not read from
	// a file.
	DestinationSource() string

	// SetDestinationSource sets the text between the parentheses.
	SetDestinationSource(source string)
}

type destinationSourceImpl struct {
	destinationSource string
}

func (d *destinationSourceImpl) DestinationSource() string {
	return d.destinationSource
}

func (d *destinationSourceImpl) SetDestinationSource(source string) {
	d.destinationSource = source
}

type Link struct {
	baseNodeWithChildren
	destinationSourceImpl

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
var _ DestinationSourceAware = (*Link)(nil)

type AutoLink struct {
	baseNode

	// URL is where this link points to.
	URL string

	// Bracketed is whether the link is written in angle brackets, such as
	// `<http://a.com>`. A URL that Logseq does not find without the brackets
	// is always written in brackets.
	Bracketed bool
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

// WithBracketed sets if the link is written in angle brackets. See Bracketed.
func (l *AutoLink) WithBracketed(bracketed bool) *AutoLink {
	l.Bracketed = bracketed
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
	if l.Bracketed {
		p.Field("bracketed", "true")
	}
	p.EndType()
}

var _ InlineNode = (*AutoLink)(nil)
var _ HasLinkURL = (*AutoLink)(nil)

// PageRef is a reference to a page, such as a `PageLink`, `Hashtag` or
// `PageEmbed`.
type PageRef interface {
	Node

	isPageRef()

	// GetTo gets the title of the page this reference points to.
	GetTo() string

	// SetTo points this reference at another page.
	SetTo(target string)
}

// PageLink is a link to a page, written as `[[To]]`. A link with a label is
// written as `[label]([[To]])`, where the children of the link are the label.
// Logseq shows the label in place of the title of the page, and the link is
// still a reference to the page.
type PageLink struct {
	baseNodeWithChildren

	// To is the target of the link.
	To string
}

// NewPageLink creates a link to the page with the given title. The label is
// optional, without a label the link shows the title of the page.
func NewPageLink(target string, label ...Node) *PageLink {
	link := &PageLink{
		To: target,
	}
	link.self = link
	link.childValidator = allowOnlyInlineNodes
	link.AddChildren(label...)
	return link
}

func (l *PageLink) GetTo() string {
	return l.To
}

func (l *PageLink) SetTo(target string) {
	l.To = target
}

func (l *PageLink) WithTo(target string) *PageLink {
	l.To = target
	return l
}

// HasLabel checks if the link has a label to show in place of the title of
// the page.
func (l *PageLink) HasLabel() bool {
	return l.FirstChild() != nil
}

func (l *PageLink) isInline() {}

func (l *PageLink) debug(p *debugPrinter) {
	p.StartType("PageLink")
	p.Field("to", l.To)
	if l.HasLabel() {
		p.Children(l)
	}
	p.EndType()
}

func (l *PageLink) isPageRef() {}

var _ InlineNode = (*PageLink)(nil)
var _ PageRef = (*PageLink)(nil)

// PageRefText is a reference to a page written as plain text, without any
// markup around it. Logseq reads the values of the properties that hold a list
// of pages this way, so `tags:: Example, Other` refers to two pages even
// though neither is written as a link.
type PageRefText struct {
	baseNode

	// To is the target of the reference.
	To string
}

func NewPageRefText(target string) *PageRefText {
	return &PageRefText{
		To: target,
	}
}

func (l *PageRefText) GetTo() string {
	return l.To
}

func (l *PageRefText) SetTo(target string) {
	l.To = target
}

func (l *PageRefText) WithTo(target string) *PageRefText {
	l.To = target
	return l
}

func (l *PageRefText) isInline() {}

func (l *PageRefText) debug(p *debugPrinter) {
	p.StartType("PageRefText")
	p.Field("to", l.To)
	p.EndType()
}

func (l *PageRefText) isPageRef() {}

var _ InlineNode = (*PageRefText)(nil)
var _ PageRef = (*PageRefText)(nil)

// Hashtag is a tag, written as `#To` or as `#[[To]]`. Logseq reads both forms
// as a reference to the page.
type Hashtag struct {
	baseNode

	// To is the title of the page the tag points to.
	To string

	// Bracketed is true if the tag is written as `#[[To]]`. A tag that cannot
	// be written as `#To` and read back with the same title, such as a title
	// with a space or a comma, is always written as `#[[To]]`.
	Bracketed bool
}

func NewHashtag(target string) *Hashtag {
	return &Hashtag{
		To: target,
	}
}

func (l *Hashtag) GetTo() string {
	return l.To
}

func (l *Hashtag) SetTo(target string) {
	l.To = target
}

func (l *Hashtag) WithTo(target string) *Hashtag {
	l.To = target
	return l
}

// WithBracketed sets if the tag is written as `#[[To]]`.
func (l *Hashtag) WithBracketed(bracketed bool) *Hashtag {
	l.Bracketed = bracketed
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

// BlockRef is a reference to a block, written as `((ID))`. A reference with a
// label is written as `[label](((ID)))`, where the children of the reference
// are the label. Logseq shows the label in place of the content of the block.
type BlockRef struct {
	baseNodeWithChildren

	// ID is the id of the block to reference.
	ID string
}

// NewBlockRef creates a reference to the block with the given id. The label is
// optional, without a label the reference shows the content of the block.
func NewBlockRef(id string, label ...Node) *BlockRef {
	ref := &BlockRef{
		ID: id,
	}
	ref.self = ref
	ref.childValidator = allowOnlyInlineNodes
	ref.AddChildren(label...)
	return ref
}

func (l *BlockRef) WithID(id string) *BlockRef {
	l.ID = id
	return l
}

// HasLabel checks if the reference has a label to show in place of the
// content of the block.
func (l *BlockRef) HasLabel() bool {
	return l.FirstChild() != nil
}

func (b *BlockRef) debug(p *debugPrinter) {
	p.StartType("BlockRef")
	p.Field("id", b.ID)
	if b.HasLabel() {
		p.Children(b)
	}
	p.EndType()
}

func (l *BlockRef) isInline() {}

var _ InlineNode = (*BlockRef)(nil)
