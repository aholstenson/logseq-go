package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var highlightKind = ast.NewNodeKind("Highlight")

type highlight struct {
	ast.BaseInline
}

func (*highlight) Kind() ast.NodeKind {
	return highlightKind
}

func (n *highlight) Dump(src []byte, level int) {
}

// highlightDelimiterProcessor pairs up the `^^` of a highlight, in the same way
// Goldmark pairs up the markers of emphasis.
type highlightDelimiterProcessor struct {
}

func (p *highlightDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == '^'
}

func (p *highlightDelimiterProcessor) CanOpenCloser(opener *parser.Delimiter, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *highlightDelimiterProcessor) OnMatch(consumes int) ast.Node {
	return &highlight{}
}

var defaultHighlightDelimiterProcessor = &highlightDelimiterProcessor{}

// highlightParser parses the `^^text^^` highlight of Logseq. A single `^` is
// left to the other parsers, so that footnote references keep working.
type highlightParser struct {
}

func (s *highlightParser) Trigger() []byte {
	return []byte{'^'}
}

func (s *highlightParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, defaultHighlightDelimiterProcessor)
	if node == nil {
		return nil
	}

	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

var _ parser.InlineParser = (*highlightParser)(nil)
