package content

type Image struct {
	baseNodeWithChildren

	// Src is the URL of the image.
	Src string

	// Title is the title of the image.
	Title string
}

func NewImage(src string, nodes ...Node) *Image {
	i := &Image{
		Src: src,
	}
	i.self = i
	i.childValidator = allowOnlyInlineNodes
	i.AddChildren(nodes...)
	return i
}

func (i *Image) WithTitle(title string) *Image {
	i.Title = title
	return i
}

func (i *Image) isInline() {}

func (i *Image) debug(p *debugPrinter) {
	p.StartType("Image")
	p.Field("src", i.Src)
	p.Field("title", i.Title)
	p.Children(i)
	p.EndType()
}

var _ InlineNode = (*Image)(nil)
