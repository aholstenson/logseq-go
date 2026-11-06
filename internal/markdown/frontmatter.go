package markdown

import (
	"bytes"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var frontMatterKind = ast.NewNodeKind("FrontMatter")

// frontMatter is YAML front matter at the start of a page. Its lines are the
// lines between the two `---` lines.
type frontMatter struct {
	ast.BaseBlock

	// options are the parse options of the page, which are used to read the
	// values of the properties in the front matter.
	options *parseOptions
}

func (n *frontMatter) Kind() ast.NodeKind {
	return frontMatterKind
}

func (n *frontMatter) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// frontMatterParser reads YAML front matter, the same way as Logseq. Front
// matter starts with a line that is only `---` on the first line of the file,
// and ends at the next line that is only `---`. Without that second line, the
// first line is a thematic break.
//
// Logseq ends the front matter at the first `---` it finds, also in the middle
// of a line. Here the front matter ends only at a line that is `---`, which is
// how other tools read front matter.
type frontMatterParser struct {
}

func (p *frontMatterParser) Trigger() []byte {
	return []byte{'-'}
}

func (p *frontMatterParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	if _, ok := parent.(*ast.Document); !ok || parent.HasChildren() {
		return nil, parser.NoChildren
	}

	line, segment := reader.PeekLine()
	if segment.Start != 0 || segment.Padding != 0 || !isFrontMatterFence(line) || !hasEOL(line) {
		return nil, parser.NoChildren
	}

	if !hasFrontMatterEnd(reader.Source()[segment.Stop:]) {
		return nil, parser.NoChildren
	}

	options, _ := pc.Get(parseOptionsKey).(*parseOptions)
	reader.Advance(segment.Len() - 1)
	return &frontMatter{options: options}, parser.NoChildren
}

func (p *frontMatterParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if line == nil {
		return parser.Close
	}

	if isFrontMatterFence(line) {
		newline := 0
		if hasEOL(line) {
			newline = 1
		}
		reader.Advance(segment.Len() - newline)
		return parser.Close
	}

	node.Lines().Append(segment)
	reader.Advance(segment.Len() - 1)
	return parser.Continue | parser.NoChildren
}

func (p *frontMatterParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (p *frontMatterParser) CanInterruptParagraph() bool {
	return false
}

func (p *frontMatterParser) CanAcceptIndentedLine() bool {
	return false
}

var _ parser.BlockParser = (*frontMatterParser)(nil)

// isFrontMatterFence checks if a line is `---` and nothing else.
func isFrontMatterFence(line []byte) bool {
	return string(bytes.TrimRight(line, "\r\n")) == "---"
}

// hasEOL checks if a line ends with a line break.
func hasEOL(line []byte) bool {
	return bytes.HasSuffix(line, []byte("\n"))
}

// hasFrontMatterEnd checks if the text after the first line of front matter
// has a line that ends the front matter.
func hasFrontMatterEnd(rest []byte) bool {
	for len(rest) > 0 {
		end := bytes.IndexByte(rest, '\n')
		line := rest
		if end >= 0 {
			line = rest[:end+1]
			rest = rest[end+1:]
		} else {
			rest = nil
		}

		if isFrontMatterFence(line) {
			return true
		}
	}

	return false
}

// convertFrontMatter turns front matter into a node that keeps its text and
// holds the properties that Logseq reads from it.
func convertFrontMatter(src []byte, node *frontMatter) (content.Node, error) {
	var raw strings.Builder
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		raw.Write(line.Value(src))
	}

	options := node.options
	if options == nil {
		defaults := defaultParseOptions()
		options = &defaults
	}

	properties, err := readFrontMatterProperties(raw.String(), options)
	if err != nil {
		return nil, err
	}

	return content.NewFrontMatter(raw.String(), properties), nil
}

// readFrontMatterProperties reads the properties in the text of front matter.
// Logseq reads the properties only if each line is a `name: value` pair, so it
// returns nil for front matter with other YAML in it, such as lists over
// several lines.
//
// The values are read in the same way as the values of `name:: value`
// properties, as Logseq does.
func readFrontMatterProperties(raw string, options *parseOptions) (*content.Properties, error) {
	if raw == "" {
		return nil, nil
	}

	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	properties := content.NewProperties()
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		name, value, ok := strings.Cut(line, ":")
		if !ok || name == "" {
			return nil, nil
		}

		name = strings.TrimSpace(name)
		value = strings.TrimLeft(value, " \t")

		property, err := readFrontMatterProperty(name, value, options)
		if err != nil {
			return nil, err
		}

		if property != nil {
			properties.AddChild(property)
		}
	}

	return properties, nil
}

// readFrontMatterProperty reads one `name: value` line of front matter as the
// `name:: value` property that Logseq reads it as. It returns nil if the name
// is not one that Logseq keeps.
func readFrontMatterProperty(name string, value string, options *parseOptions) (*content.Property, error) {
	found, _, ok := findPropertyName(name + ":: ")
	if !ok || found != name {
		return nil, nil
	}

	block, err := parseWithOptions([]byte(name+":: "+value), options)
	if err != nil {
		return nil, err
	}

	properties, ok := block.FirstChild().(*content.Properties)
	if !ok || properties.NextSibling() != nil {
		return nil, nil
	}

	property, ok := properties.FirstChild().(*content.Property)
	if !ok || property.NextSibling() != nil {
		return nil, nil
	}

	properties.RemoveChild(property)
	return property, nil
}
