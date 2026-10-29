package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var indentedBlocksKind = ast.NewNodeKind("IndentedBlocks")

// indentedBlocks holds bullets that are indented one level or more further
// than the content before them. Its children are the bullets with one level of
// indentation taken off.
type indentedBlocks struct {
	ast.BaseBlock
}

func (*indentedBlocks) Kind() ast.NodeKind {
	return indentedBlocksKind
}

func (n *indentedBlocks) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// indentedBlocksParser parses bullets that are indented further than the
// content they follow, such as bullets indented with a tab under the content
// before the first bullet of a page:
//
//	Some text before the first bullet
//
//		- A child of that text
//
// CommonMark reads these lines as an indented code block, or as more text of
// the paragraph before them. Logseq reads them as child blocks, so this parser
// takes them before the code block and paragraph parsers do.
type indentedBlocksParser struct {
}

func (p *indentedBlocksParser) Trigger() []byte {
	return []byte{'-'}
}

func (p *indentedBlocksParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || pc.BlockIndent() < 4 || !isBlockMarker(line[pos:]) {
		return nil, parser.NoChildren
	}

	if !advanceIndentLevel(line, reader) {
		return nil, parser.NoChildren
	}

	return &indentedBlocks{}, parser.HasChildren
}

func (p *indentedBlocksParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	if util.IsBlank(line) {
		return parser.Continue | parser.HasChildren
	}

	if !advanceIndentLevel(line, reader) {
		return parser.Close
	}

	return parser.Continue | parser.HasChildren
}

func (p *indentedBlocksParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (p *indentedBlocksParser) CanInterruptParagraph() bool {
	return true
}

func (p *indentedBlocksParser) CanAcceptIndentedLine() bool {
	return true
}

// advanceIndentLevel moves the reader past one level of indentation, which is
// a tab or four spaces. It returns false if the line is not indented that far.
func advanceIndentLevel(line []byte, reader text.Reader) bool {
	pos, padding := util.IndentPosition(line, reader.LineOffset(), 4)
	if pos < 0 {
		return false
	}

	reader.AdvanceAndSetPadding(pos, padding)
	return true
}

// isBlockMarker checks if a line starts with the marker of a block, which is a
// dash followed by a space or by the end of the line.
func isBlockMarker(line []byte) bool {
	if len(line) == 0 || line[0] != '-' {
		return false
	}

	return len(line) == 1 || util.IsSpace(line[1])
}
