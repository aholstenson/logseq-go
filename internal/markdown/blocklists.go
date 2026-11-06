package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// blockListParser parses lists in the same way as the list parser of
// Goldmark, with these differences that Logseq has:
//
//   - A list with "*", "+" or numbers in a page or a block ends at a line
//     that starts with "- ", as such a line is always a block.
//   - A bullet without text, such as "-", can interrupt a paragraph. Thus the
//     second line of "- a\n\t-" is an empty child block of "a". CommonMark
//     reads it as the text "-".
//   - A blank line after a block without text does not end the list, so the
//     child in "-\n\n\t- c" stays in the list, under the empty block.
type blockListParser struct {
	parser.BlockParser
}

func newBlockListParser() *blockListParser {
	return &blockListParser{BlockParser: parser.NewListParser()}
}

func (p *blockListParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	if _, ok := indentedUnderHeading(parent, reader, pc); ok {
		// The bullets are children of the heading, see indentedBlocksParser.
		return nil, parser.NoChildren
	}

	if interruptsWithEmptyBlock(parent, reader, pc) {
		// The list item parser of Goldmark adds the empty item to the list.
		return ast.NewList('-'), parser.HasChildren
	}

	return p.BlockParser.Open(parent, reader, pc)
}

func (p *blockListParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	if !isBlockList(node) && inBlockContent(node) && startsWithBlockMarker(reader) {
		// Logseq reads each line that starts with "- " as a block, at any
		// indentation. Thus a list with "*", "+" or numbers ends before it,
		// also when CommonMark would nest the bullet in the list.
		return parser.Close
	}

	state := p.BlockParser.Continue(node, reader, pc)
	if state&parser.Close == 0 || !isBlockList(node) {
		return state
	}

	if startsDividerBlock(reader, pc) {
		// CommonMark ends the list at a thematic break such as "- ---". In
		// Logseq the line is the next block of the list.
		return parser.Continue | parser.HasChildren
	}

	if indentedUnderLastItem(node, reader) {
		// CommonMark ends the list at a line after an empty item and a blank
		// line, also in the lists that hold this list. In Logseq the line
		// belongs to the last block.
		return parser.Continue | parser.HasChildren
	}

	return state
}

// inBlockContent returns true if the node is in the content of a page or of
// blocks, and not in other content such as a quote.
func inBlockContent(node ast.Node) bool {
	for p := node.Parent(); p != nil; p = p.Parent() {
		switch p.(type) {
		case *ast.Document:
			return true
		case *ast.List, *ast.ListItem, *indentedBlocks:
			continue
		default:
			return false
		}
	}

	return false
}

// startsWithBlockMarker returns true if the first character of the line that
// is not a space starts a bullet "- ".
func startsWithBlockMarker(reader text.Reader) bool {
	line, _ := reader.PeekLine()
	_, pos := util.IndentWidth(line, reader.LineOffset())
	return pos < len(line) && isBlockMarker(line[pos:])
}

// indentedUnderLastItem returns true if the line is not blank and is indented
// at least as far as the text of the last item of the list starts.
func indentedUnderLastItem(list ast.Node, reader text.Reader) bool {
	item, ok := list.LastChild().(*ast.ListItem)
	if !ok {
		return false
	}

	line, _ := reader.PeekLine()
	if util.IsBlank(line) {
		return false
	}

	indent, _ := util.IndentWidth(line, reader.LineOffset())
	return indent >= item.Offset
}

// interruptsWithEmptyBlock returns true if the line is a bullet without text
// that comes directly after a paragraph in content that holds blocks.
func interruptsWithEmptyBlock(parent ast.Node, reader text.Reader, pc parser.Context) bool {
	last := pc.LastOpenedBlock().Node
	if !ast.IsParagraph(last) || last.Parent() != parent || !holdsBlocks(parent) {
		return false
	}

	line, _ := reader.PeekLine()
	pos, indent := blockStart(line, reader, pc)
	if pos < 0 || indent >= 4 {
		return false
	}

	return isBlockMarker(line[pos:]) && util.IsBlank(line[pos+1:])
}

// blockStart gets the position and the width of the indentation of the first
// character of the line that is not a space. Returns -1 for a blank line.
//
// Goldmark finds this before it calls the block parsers, but it compares the
// width with the length of the line in bytes. A tab is one byte but up to four
// columns, so Goldmark takes a short line such as "\t\t-" in a block as blank.
// This finds the position again for such a line.
func blockStart(line []byte, reader text.Reader, pc parser.Context) (int, int) {
	if pos := pc.BlockOffset(); pos >= 0 {
		return pos, pc.BlockIndent()
	}

	if util.IsBlank(line) {
		return -1, -1
	}

	indent, pos := util.IndentWidth(line, reader.LineOffset())
	return pos, indent
}

// holdsBlocks returns true if the bullets in the node are blocks: the node is
// a page, a block, or bullets that are indented under content.
func holdsBlocks(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.Document, *indentedBlocks:
		return true
	case *ast.ListItem:
		return isBlockList(n.Parent())
	}

	return false
}

// isBlockList returns true if the node is a list of blocks, which uses "-" as
// its marker.
func isBlockList(node ast.Node) bool {
	list, ok := node.(*ast.List)
	return ok && list.Marker == '-'
}

// blockListItemParser parses list items in the same way as the list item
// parser of Goldmark, with one difference that Logseq has: a block without
// text keeps the lines that are indented at least as far as its text would
// start. Thus "c" is a child of the empty block in
//
//	-
//	  - c
//
// CommonMark ends an empty item at a line with a list marker, so it reads "c"
// as the next item of the list.
type blockListItemParser struct {
	parser.BlockParser
}

func newBlockListItemParser() *blockListItemParser {
	return &blockListItemParser{BlockParser: parser.NewListItemParser()}
}

func (p *blockListItemParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	item, ok := node.(*ast.ListItem)
	if !ok || !keepsLine(item, reader) {
		return p.BlockParser.Continue(node, reader, pc)
	}

	line, _ := reader.PeekLine()
	pos, padding := util.IndentPosition(line, reader.LineOffset(), item.Offset)
	reader.AdvanceAndSetPadding(pos, padding)
	return parser.Continue | parser.HasChildren
}

// keepsLine returns true if the item is a block without text, and the line is
// indented under it. Such a line belongs to the block.
func keepsLine(item *ast.ListItem, reader text.Reader) bool {
	if item.ChildCount() > 0 || !isBlockList(item.Parent()) {
		return false
	}

	return indentedUnderLastItem(item.Parent(), reader)
}

// blockThematicBreakParser parses thematic breaks in the same way as the
// thematic break parser of Goldmark, with one difference that Logseq has: in
// content that holds blocks, "- ---" is a block that holds a thematic break.
// CommonMark reads the whole line as one thematic break.
type blockThematicBreakParser struct {
	parser.BlockParser
}

func newBlockThematicBreakParser() *blockThematicBreakParser {
	return &blockThematicBreakParser{BlockParser: parser.NewThematicBreakParser()}
}

func (p *blockThematicBreakParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	if (holdsBlocks(parent) || isBlockList(parent)) && startsDividerBlock(reader, pc) {
		// The list parsers read the line as a block.
		return nil, parser.NoChildren
	}

	return p.BlockParser.Open(parent, reader, pc)
}

// startsDividerBlock returns true if the line is a bullet followed by a
// thematic break, such as "- ---" or "- ***".
func startsDividerBlock(reader text.Reader, pc parser.Context) bool {
	line, _ := reader.PeekLine()
	pos, indent := blockStart(line, reader, pc)
	if pos < 0 {
		indent, pos = util.IndentWidth(line, reader.LineOffset())
	}

	if pos >= len(line) || indent >= 4 || !isBlockMarker(line[pos:]) || pos+1 >= len(line) {
		return false
	}

	return isThematicBreak(line[pos+1:])
}

// isThematicBreak returns true if the line is a thematic break: three or more
// of the same character out of "-", "*" and "_", with only spaces and tabs
// between them.
func isThematicBreak(line []byte) bool {
	var marker byte
	count := 0
	for _, c := range line {
		switch c {
		case ' ', '\t', '\n', '\r':
			continue
		case '-', '*', '_':
			if marker != 0 && c != marker {
				return false
			}

			marker = c
			count++
		default:
			return false
		}
	}

	return count >= 3
}

var _ parser.BlockParser = (*blockListParser)(nil)
var _ parser.BlockParser = (*blockThematicBreakParser)(nil)
var _ parser.BlockParser = (*blockListItemParser)(nil)
