package markdown

import (
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// scanDelimiter finds a run of markers, such as the `**` of strong text, in
// the same way as Goldmark. It has one difference that Logseq also has: a run
// that comes after a space and ends the line can close, so `**text **` is
// strong. In the middle of a line the rules of CommonMark apply.
func scanDelimiter(line []byte, before rune, min int, processor parser.DelimiterProcessor) *parser.Delimiter {
	node := parser.ScanDelimiter(line, before, min, processor)
	if node == nil {
		return nil
	}

	if util.IsSpaceRune(before) && util.IsBlank(line[node.OriginalLength:]) {
		node.CanClose = true
	}

	return node
}

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

// emphasisParser parses emphasis and strong text, with the markers that
// scanDelimiter accepts.
type emphasisParser struct {
}

func (s *emphasisParser) Trigger() []byte {
	return []byte{'*', '_'}
}

func (s *emphasisParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := scanDelimiter(line, before, 1, defaultEmphasisDelimiterProcessor)
	if node == nil {
		return nil
	}

	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

var _ parser.InlineParser = (*emphasisParser)(nil)

// strikethroughDelimiterProcessor pairs up the `~~` of strikethrough text, in
// the same way as the strikethrough extension of Goldmark.
type strikethroughDelimiterProcessor struct {
}

func (p *strikethroughDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == '~'
}

func (p *strikethroughDelimiterProcessor) CanOpenCloser(opener *parser.Delimiter, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *strikethroughDelimiterProcessor) OnMatch(consumes int) ast.Node {
	return east.NewStrikethrough()
}

var defaultStrikethroughDelimiterProcessor = &strikethroughDelimiterProcessor{}

// strikethroughParser parses `~~text~~`, with the markers that scanDelimiter
// accepts.
type strikethroughParser struct {
}

func (s *strikethroughParser) Trigger() []byte {
	return []byte{'~'}
}

func (s *strikethroughParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := scanDelimiter(line, before, 2, defaultStrikethroughDelimiterProcessor)
	if node == nil {
		return nil
	}

	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

var _ parser.InlineParser = (*strikethroughParser)(nil)
