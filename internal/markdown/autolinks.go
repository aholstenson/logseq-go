package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// bracketedAttribute is the name of the attribute that is set on an autolink
// that is written in angle brackets, such as `<http://a.com>`.
const bracketedAttribute = "logseq-bracketed"

// bracketedAutoLinkParser is the autolink parser of Goldmark, which reads
// `<http://a.com>`, changed so that the link it makes is marked as written in
// angle brackets. A URL without brackets is read by the linkify parser, and
// makes the same kind of node.
type bracketedAutoLinkParser struct {
	parser.InlineParser
}

func newBracketedAutoLinkParser() *bracketedAutoLinkParser {
	return &bracketedAutoLinkParser{InlineParser: parser.NewAutoLinkParser()}
}

func (p *bracketedAutoLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	result := p.InlineParser.Parse(parent, block, pc)
	if link, ok := result.(*ast.AutoLink); ok {
		link.SetAttributeString(bracketedAttribute, true)
	}

	return result
}

var _ parser.InlineParser = (*bracketedAutoLinkParser)(nil)
