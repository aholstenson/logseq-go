package content

// FrontMatter is YAML front matter at the start of a page, which is the text
// between a `---` line on the first line of the file and the next `---` line.
// Pages that come from other tools, such as Obsidian, Jekyll or Hugo, often
// start with it:
//
//	---
//	title: My Page
//	alias: Other
//	---
//
// Logseq reads the front matter as the properties of the page when each of its
// lines is a `name: value` pair. Front matter with other YAML in it, such as a
// list over several lines, gives no properties, and its text is not shown.
//
// The text is kept as it was read and is written back without a change, unless
// the properties are changed. Then the front matter is written again from the
// properties, with one `name: value` line for each property.
type FrontMatter struct {
	baseNodeWithChildren

	// Raw is the text between the two `---` lines, which includes the line
	// break at the end of its last line.
	Raw string
}

// NewFrontMatter creates front matter with the given text, which is the text
// between the two `---` lines. Give the properties that Logseq reads from the
// text, or nil if Logseq reads no properties from it.
func NewFrontMatter(raw string, properties *Properties) *FrontMatter {
	f := &FrontMatter{Raw: raw}
	f.self = f
	f.childValidator = allowOnlyPropertiesNode
	if properties != nil {
		f.AddChild(properties)
	}
	return f
}

// FindProperties gets the properties that are read from the front matter. It
// returns nil if the front matter has no properties that Logseq reads.
func (f *FrontMatter) FindProperties() *Properties {
	if properties, ok := f.FirstChild().(*Properties); ok {
		return properties
	}

	return nil
}

func (f *FrontMatter) debug(p *debugPrinter) {
	p.StartType("FrontMatter")
	p.Field("Raw", f.Raw)
	p.Children(f)
	p.EndType()
}

func (f *FrontMatter) isBlock() {}

var _ BlockNode = (*FrontMatter)(nil)
var _ HasChildren = (*FrontMatter)(nil)

func allowOnlyPropertiesNode(node Node) bool {
	_, ok := node.(*Properties)
	return ok
}
