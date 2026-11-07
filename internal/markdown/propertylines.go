package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// isPropertyLine checks if a line, without its indentation, is a `name::`
// property line that Logseq reads as a property of the block.
func isPropertyLine(line []byte) bool {
	_, _, ok := findPropertyName(string(util.TrimLeftSpace(line)))
	return ok
}

// propertyLineParser starts a new paragraph at a property line that would
// otherwise be read as a lazy continuation line of a paragraph in a quote or
// in a list, such as the `id::` line in "- > quoted\n  id:: 65e5a0b2-...".
//
// Logseq writes the properties of a block after its content, and reads them
// as properties of the block. In CommonMark the line is a part of the quote.
//
// A property line that continues a paragraph in the same container is left
// to the paragraph, so that properties after text are read as before.
type propertyLineParser struct {
	paragraph parser.BlockParser
}

func newPropertyLineParser() *propertyLineParser {
	return &propertyLineParser{
		paragraph: newParagraphParser(),
	}
}

func (p *propertyLineParser) Trigger() []byte {
	return nil
}

func (p *propertyLineParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	last := pc.LastOpenedBlock().Node
	paragraph, ok := last.(*ast.Paragraph)
	if !ok || paragraph.Parent() == parent {
		return nil, parser.NoChildren
	}

	line, _ := reader.PeekLine()
	if !isPropertyLine(line) {
		return nil, parser.NoChildren
	}

	// A property line after a paragraph of only property lines is a lazy
	// line of the same group of properties, as in "- a:: b\nc:: d".
	if onlyPropertyLines(paragraph, reader.Source()) {
		return nil, parser.NoChildren
	}

	return p.paragraph.Open(parent, reader, pc)
}

// onlyPropertyLines checks if each line of the paragraph is a property line.
func onlyPropertyLines(paragraph *ast.Paragraph, source []byte) bool {
	lines := paragraph.Lines()
	if lines.Len() == 0 {
		return false
	}

	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		if !isPropertyLine(line.Value(source)) {
			return false
		}
	}

	return true
}

func (p *propertyLineParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return p.paragraph.Continue(node, reader, pc)
}

func (p *propertyLineParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	p.paragraph.Close(node, reader, pc)
}

func (p *propertyLineParser) CanInterruptParagraph() bool {
	return true
}

func (p *propertyLineParser) CanAcceptIndentedLine() bool {
	return false
}

var _ parser.BlockParser = (*propertyLineParser)(nil)

// htmlBlockParser is the HTML block parser of Goldmark, changed so that an
// HTML block that ends at a blank line also ends at a property line or at the
// bullet of a block. Logseq reads the property lines after HTML as properties
// of the block, as in "- <div>x</div>\n  id:: 65e5a0b2-...", and a bullet
// line as the start of a block, as in "- x\n<div>\n- y".
//
// HTML blocks that end with a closing marker, such as `</pre>` or `-->`, keep
// all lines up to that marker.
type htmlBlockParser struct {
	parser.BlockParser
}

func newHTMLBlockParser() *htmlBlockParser {
	return &htmlBlockParser{
		BlockParser: parser.NewHTMLBlockParser(),
	}
}

func (p *htmlBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	if block, ok := node.(*ast.HTMLBlock); ok &&
		(block.HTMLBlockType == ast.HTMLBlockType6 || block.HTMLBlockType == ast.HTMLBlockType7) {
		line, _ := reader.PeekLine()
		if isPropertyLine(line) || isBulletLine(util.TrimLeftSpace(line)) {
			return parser.Close
		}
	}

	return p.BlockParser.Continue(node, reader, pc)
}

var _ parser.BlockParser = (*htmlBlockParser)(nil)
