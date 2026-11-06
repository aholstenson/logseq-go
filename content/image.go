package content

type Image struct {
	baseNodeWithChildren

	// URL is the URL of the image.
	URL string

	// Title is the title of the image.
	Title string

	// Metadata is the text in curly braces directly after the image, such as
	// `{:height 100, :width 200}`. Logseq writes it when an image is resized.
	// It includes the braces and is written back as it is. It is empty if the
	// image has no metadata.
	Metadata string
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

// WithMetadata sets the metadata of the image, such as
// `{:height 100, :width 200}`. The value must include the braces.
func (i *Image) WithMetadata(metadata string) *Image {
	i.Metadata = metadata
	return i
}

func (i *Image) isInline() {}

func (i *Image) debug(p *debugPrinter) {
	p.StartType("Image")
	p.Field("src", i.URL)
	p.Field("title", i.Title)
	if i.Metadata != "" {
		p.Field("metadata", i.Metadata)
	}
	p.Children(i)
	p.EndType()
}

var _ InlineNode = (*Image)(nil)
