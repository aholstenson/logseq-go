package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// paragraphIndentationAttribute is the name of the attribute that holds the
// indentation of a paragraph that indentedParagraphParser opens.
var paragraphIndentationAttribute = []byte("logseqIndentation")

// indentedParagraphParser parses text that is indented four or more columns
// further than the node that holds it, such as the last line of
// "- A block\n\n      [[A page]]".
//
// CommonMark reads such lines as an indented code block. Logseq has no
// indented code blocks, and reads them as a paragraph, so page links and
// other markup in them are read as usual. The paragraph keeps its
// indentation so that it is written back as it was.
//
// Text with less indentation is parsed by the paragraph parser of goldmark,
// which does not accept lines indented four or more columns.
type indentedParagraphParser struct {
}

func (p *indentedParagraphParser) Trigger() []byte {
	return nil
}

func (p *indentedParagraphParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	if util.IsBlank(line) {
		return nil, parser.NoChildren
	}

	width, pos := util.IndentWidth(line, reader.LineOffset())
	if width < 4 {
		return nil, parser.NoChildren
	}

	indentation := strings.Repeat(" ", segment.Padding) + string(line[:pos])

	node := ast.NewParagraph()
	node.SetAttribute(paragraphIndentationAttribute, indentation)

	segment = segment.TrimLeftSpace(reader.Source())
	node.Lines().Append(segment)
	reader.Advance(segment.Len() - 1)
	return node, parser.NoChildren
}

func (p *indentedParagraphParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.NewParagraphParser().Continue(node, reader, pc)
}

func (p *indentedParagraphParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	// The paragraph indentation is written for each line, so only the
	// indentation after it is recorded.
	recordLineIndentation(node, reader, pc, paragraphIndentation(node.(*ast.Paragraph)))
	parser.NewParagraphParser().Close(node, reader, pc)
}

func (p *indentedParagraphParser) CanInterruptParagraph() bool {
	// An indented line after a line of text is more text of that paragraph,
	// as in CommonMark.
	return false
}

func (p *indentedParagraphParser) CanAcceptIndentedLine() bool {
	return true
}

// paragraphIndentation returns the indentation that indentedParagraphParser
// recorded for a paragraph, or an empty string.
func paragraphIndentation(node *ast.Paragraph) string {
	value, ok := node.Attribute(paragraphIndentationAttribute)
	if !ok {
		return ""
	}

	indentation, _ := value.(string)
	return indentation
}
