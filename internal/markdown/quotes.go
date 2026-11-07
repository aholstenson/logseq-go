package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
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

// quoteBlankFirstLineAttribute is the name of the attribute that is set on a
// quote whose first line has nothing after the `>` marker.
const quoteBlankFirstLineAttribute = "logseq-quote-blank-first-line"

// quoteBlankLastLineAttribute is the name of the attribute that holds
// whether the last line of a quote has nothing after the `>` marker.
const quoteBlankLastLineAttribute = "logseq-quote-blank-last-line"

func (p *quoteParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	node, state := p.BlockParser.Open(parent, reader, pc)
	if node != nil && pos >= 0 && pos+1 < len(line) && line[pos] == '>' {
		if c := line[pos+1]; c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			node.SetAttributeString(quoteNoSpaceAttribute, true)
		}
	}

	if node != nil {
		rest, _ := reader.PeekLine()
		blank := util.IsBlank(rest)
		if blank {
			node.SetAttributeString(quoteBlankFirstLineAttribute, true)
		}
		node.SetAttributeString(quoteBlankLastLineAttribute, blank)
	}

	return node, state
}

func (p *quoteParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	state := p.BlockParser.Continue(node, reader, pc)
	if state != parser.Close {
		rest, _ := reader.PeekLine()
		node.SetAttributeString(quoteBlankLastLineAttribute, util.IsBlank(rest))
	}

	return state
}

var _ parser.BlockParser = (*quoteParser)(nil)
