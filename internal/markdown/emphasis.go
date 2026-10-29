package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// emphasisDelimiterProcessor pairs up the `*` and `_` markers of emphasis and
// strong text, in the same way as the emphasis parser of Goldmark.
type emphasisDelimiterProcessor struct {
}

func (p *emphasisDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == '*' || b == '_'
}

func (p *emphasisDelimiterProcessor) CanOpenCloser(opener *parser.Delimiter, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *emphasisDelimiterProcessor) OnMatch(consumes int) ast.Node {
	return ast.NewEmphasis(consumes)
}

var defaultEmphasisDelimiterProcessor = &emphasisDelimiterProcessor{}

// emphasisParser parses emphasis and strong text. It follows CommonMark, with
// one difference that Logseq also has: a `**` that comes after a space can
// close strong text, so `**text **` is strong. Such a `**` can only close when
// it cannot open, which keeps `**a** **b**` as two strong parts.
type emphasisParser struct {
}

func (s *emphasisParser) Trigger() []byte {
	return []byte{'*', '_'}
}

func (s *emphasisParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 1, defaultEmphasisDelimiterProcessor)
	if node == nil {
		return nil
	}

	if node.Char == '*' && node.Length >= 2 && !node.CanOpen && util.IsSpaceRune(before) {
		node.CanClose = true
	}

	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

var _ parser.InlineParser = (*emphasisParser)(nil)
