package content

type Image struct {
	baseNodeWithChildren

	// URL is the URL of the image.
	URL string

	// Title is the title of the image.
	Title string
}

func NewImage(url string, nodes ...Node) *Image {
	i := &Image{
		URL: url,
	}
	i.self = i
	i.childValidator = allowOnlyInlineNodes
	i.AddChildren(nodes...)
	return i
}

// WithURL sets the source of the image.
func (i *Image) WithURL(src string) *Image {
	i.URL = src
	return i
}

// WithTitle sets the title of the image.
func (i *Image) WithTitle(title string) *Image {
	i.Title = title
	return i
}

func (i *Image) isInline() {}

func (i *Image) debug(p *debugPrinter) {
	p.StartType("Image")
	p.Field("src", i.URL)
	p.Field("title", i.Title)
	p.Children(i)
	p.EndType()
}

var _ InlineNode = (*Image)(nil)
