package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var footnoteRefKind = ast.NewNodeKind("FootnoteRef")

type footnoteRef struct {
	ast.BaseInline

	Label string
}

func (*footnoteRef) Kind() ast.NodeKind {
	return footnoteRefKind
}

func (n *footnoteRef) Dump(src []byte, level int) {
}

// footnoteRefParser parses the `[^label]` that points at a footnote from
// within the text.
type footnoteRefParser struct {
}

func (f *footnoteRefParser) Trigger() []byte {
	return []byte{'['}
}

func (f *footnoteRefParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()

	end := footnoteLabelEnd(line)
	if end < 0 {
		return nil
	}

	if end+1 < len(line) && (line[end+1] == '(' || line[end+1] == '[') {
		// A link destination or a link label follows, so this is the text of a
		// link rather than a reference to a footnote.
		return nil
	}

	block.Advance(end + 1)
	return &footnoteRef{
		Label: string(line[2:end]),
	}
}

var _ parser.InlineParser = (*footnoteRefParser)(nil)

// footnoteLabelEnd finds the closing bracket of the `[^label]` at the start of
// the given text, returning -1 when there is no label there. A label runs to
// the first closing bracket and holds no whitespace, which keeps the brackets
// of other syntax out of it.
func footnoteLabelEnd(line []byte) int {
	if len(line) < 4 || line[0] != '[' || line[1] != '^' {
		return -1
	}

	for i := 2; i < len(line); i++ {
		if line[i] == ']' {
			if i == 2 {
				// The label is empty.
				return -1
			}

			return i
		}

		if line[i] == '[' || util.IsSpace(line[i]) {
			return -1
		}
	}

	return -1
}

var footnoteDefinitionKind = ast.NewNodeKind("FootnoteDefinition")

type footnoteDefinition struct {
	ast.BaseBlock

	Label string
}

func (*footnoteDefinition) Kind() ast.NodeKind {
	return footnoteDefinitionKind
}

func (n *footnoteDefinition) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// footnoteDefinitionParser parses the `[^label]: content` that holds the
// content of a footnote.
//
// Goldmark has a parser for this, but it gathers every definition of a
// document into a list at the end of it. Logseq keeps a definition where it
// was written, so this parser leaves it in place.
type footnoteDefinitionParser struct {
}

func (f *footnoteDefinitionParser) Trigger() []byte {
	return []byte{'['}
}

func (f *footnoteDefinitionParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	end := footnoteLabelEnd(line[pos:])
	if end < 0 {
		return nil, parser.NoChildren
	}

	// The label of a definition is followed by the colon that separates it
	// from the content.
	end = pos + end + 1
	if end >= len(line) || line[end] != ':' {
		return nil, parser.NoChildren
	}

	node := &footnoteDefinition{
		Label: string(line[pos+2 : end-1]),
	}

	// The content of the footnote starts after the colon, and is parsed as the
	// children of the definition.
	padding := segment.Padding
	content := end + 1 - padding
	if content >= len(line) {
		reader.Advance(content)
		return node, parser.NoChildren
	}

	reader.AdvanceAndSetPadding(content, padding)
	return node, parser.HasChildren
}

func (f *footnoteDefinitionParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	if util.IsBlank(line) {
		return parser.Continue | parser.HasChildren
	}

	// Only a line that is indented under the definition carries the footnote
	// on, anything else starts something new.
	pos, padding := util.IndentPosition(line, reader.LineOffset(), 4)
	if pos < 0 {
		return parser.Close
	}

	reader.AdvanceAndSetPadding(pos, padding)
	return parser.Continue | parser.HasChildren
}

func (f *footnoteDefinitionParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (f *footnoteDefinitionParser) CanInterruptParagraph() bool {
	return true
}

func (f *footnoteDefinitionParser) CanAcceptIndentedLine() bool {
	return false
}

var _ parser.BlockParser = (*footnoteDefinitionParser)(nil)
