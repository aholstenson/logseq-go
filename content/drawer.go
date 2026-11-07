package content

// Drawer is a drawer that is not a logbook or properties, such as:
//
//	:notes:
//	Some text
//	:end:
//
// Logseq does not read the content of such a drawer as Markdown, so the
// content is text, and links in it are not references to pages.
type Drawer struct {
	baseNode
	previousLineAwareImpl

	// Name is the name between the colons of the first line, as written,
	// such as `notes`.
	Name string

	// Content is the lines between the first and the last line of the
	// drawer, each one with its line break.
	Content string

	// EndLine is the last line of the drawer as written, such as `:END:` or
	// `:end:`, with the spaces before it that are more than the indentation of
	// the first line. It is empty for `:END:`. A line that does not end a
	// drawer is written as `:END:`.
	EndLine string
}

// NewDrawer creates a drawer with the name and the content.
func NewDrawer(name string, content string) *Drawer {
	return &Drawer{
		Name:    name,
		Content: content,
	}
}

// WithEndLine sets the last line of the drawer, see EndLine.
func (d *Drawer) WithEndLine(endLine string) *Drawer {
	d.EndLine = endLine
	return d
}

// WithPreviousLineType sets the type of the line before the drawer.
func (d *Drawer) WithPreviousLineType(previousLineType PreviousLineType) *Drawer {
	d.previousLineType = previousLineType
	return d
}

func (d *Drawer) debug(p *debugPrinter) {
	p.StartType("Drawer")
	p.Field("name", d.Name)
	p.Field("content", d.Content)
	if d.EndLine != "" {
		p.Field("endLine", d.EndLine)
	}
	p.EndType()
}

func (d *Drawer) isBlock() {}

var _ BlockNode = (*Drawer)(nil)
