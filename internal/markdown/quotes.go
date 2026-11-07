package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// quoteNoSpaceAttribute is the name of the attribute that is set on a quote
// whose `>` marker has no space or tab after it, as in ">q".
const quoteNoSpaceAttribute = "logseq-quote-no-space"

// quoteParser is the quote parser of Goldmark, changed so that it records if
// the `>` marker of the quote has no space after it, see
// content.Blockquote.NoSpaceAfterMarker.
type quoteParser struct {
	parser.BlockParser
}

func newQuoteParser() *quoteParser {
	return &quoteParser{BlockParser: parser.NewBlockquoteParser()}
}

func (p *quoteParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	node, state := p.BlockParser.Open(parent, reader, pc)
	if node != nil && pos >= 0 && pos+1 < len(line) && line[pos] == '>' {
		if c := line[pos+1]; c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			node.SetAttributeString(quoteNoSpaceAttribute, true)
		}
	}

	return node, state
}

var _ parser.BlockParser = (*quoteParser)(nil)
