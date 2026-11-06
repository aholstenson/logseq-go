package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var highlightKind = ast.NewNodeKind("Highlight")

type highlight struct {
	ast.BaseInline

	// EqualSigns is whether the highlight is written with `==` markers
	// instead of `^^` markers.
	EqualSigns bool
}

func (*highlight) Kind() ast.NodeKind {
	return highlightKind
}

func (n *highlight) Dump(src []byte, level int) {
}

// highlightDelimiterProcessor pairs up the `^^` or the `==` of a highlight,
// in the same way Goldmark pairs up the markers of emphasis. Each marker has
// its own processor, so that the highlight that is made knows which marker it
// was written with.
type highlightDelimiterProcessor struct {
	char byte
}

func (p *highlightDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == p.char
}

func (p *highlightDelimiterProcessor) CanOpenCloser(opener *parser.Delimiter, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *highlightDelimiterProcessor) OnMatch(consumes int) ast.Node {
	return &highlight{EqualSigns: p.char == '='}
}

var caretHighlightDelimiterProcessor = &highlightDelimiterProcessor{char: '^'}
var equalSignsHighlightDelimiterProcessor = &highlightDelimiterProcessor{char: '='}

// highlightDelimiterProcessorFor gets the processor for the marker, which is
// `^` or `=`.
func highlightDelimiterProcessorFor(c byte) *highlightDelimiterProcessor {
	if c == '=' {
		return equalSignsHighlightDelimiterProcessor
	}

	return caretHighlightDelimiterProcessor
}

// highlightParser parses the `^^text^^` and `==text==` highlights of Logseq.
// A single `^` or `=` is left to the other parsers, so that footnote
// references keep working.
type highlightParser struct {
}

func (s *highlightParser) Trigger() []byte {
	return []byte{'^', '='}
}

func (s *highlightParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	if len(line) == 0 {
		return nil
	}

	node := scanDelimiter(line, before, 2, highlightDelimiterProcessorFor(line[0]))
	if node == nil {
		return nil
	}

	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

var _ parser.InlineParser = (*highlightParser)(nil)
