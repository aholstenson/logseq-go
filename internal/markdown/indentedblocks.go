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

	// width is how many columns of indentation are taken off each line.
	width int
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
//
// Bullets after a heading line of a page are children of the heading at any
// indentation, so there one to three spaces are enough:
//
//	# Heading
//	  - A child of the heading
type indentedBlocksParser struct {
}

func (p *indentedBlocksParser) Trigger() []byte {
	return []byte{'-'}
}

func (p *indentedBlocksParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	width, ok := indentedUnderHeading(parent, reader, pc)
	if !ok {
		pos, indent := blockStart(line, reader, pc)
		if pos < 0 || indent < 4 || !isBlockMarker(line[pos:]) {
			return nil, parser.NoChildren
		}

		width = 4
	}

	if !advanceIndent(line, reader, width) {
		return nil, parser.NoChildren
	}

	return &indentedBlocks{width: width}, parser.HasChildren
}

func (p *indentedBlocksParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	if util.IsBlank(line) {
		return parser.Continue | parser.HasChildren
	}

	if !advanceIndent(line, reader, node.(*indentedBlocks).width) {
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

// advanceIndent moves the reader past the given number of columns of
// indentation. A tab is four columns. It returns false if the line is not
// indented that far.
func advanceIndent(line []byte, reader text.Reader, width int) bool {
	pos, padding := util.IndentPosition(line, reader.LineOffset(), width)
	if pos < 0 {
		return false
	}

	reader.AdvanceAndSetPadding(pos, padding)
	return true
}

// indentedUnderHeading checks if the line is a bullet that is indented by one
// to three spaces directly after a heading line of a page, and returns the
// width of the indentation. Logseq reads such a bullet as a child of the
// heading. A bullet with more indentation is found by the usual rule of
// indentedBlocksParser.
func indentedUnderHeading(parent ast.Node, reader text.Reader, pc parser.Context) (int, bool) {
	if _, ok := parent.(*ast.Document); !ok {
		return 0, false
	}

	// Only a page has heading blocks, see AsPage.
	if options, ok := pc.Get(parseOptionsKey).(*parseOptions); !ok || !options.page {
		return 0, false
	}

	if _, ok := parent.LastChild().(*ast.Heading); !ok {
		return 0, false
	}

	line, _ := reader.PeekLine()
	pos, indent := blockStart(line, reader, pc)
	if pos < 0 || indent < 1 || indent >= 4 || !isBlockMarker(line[pos:]) {
		return 0, false
	}

	return indent, true
}

// isBlockMarker checks if a line starts with the marker of a block, which is a
// dash followed by a space or by the end of the line. A dash directly followed
// by the `#` marks of a heading is also a block marker, see isHeadingBullet.
func isBlockMarker(line []byte) bool {
	if len(line) == 0 || line[0] != '-' {
		return false
	}

	return len(line) == 1 || util.IsSpace(line[1]) || isHeadingBullet(line)
}

// isHeadingBullet checks if a line starts with a dash directly followed by
// the `#` marks of a heading and then a space or the end of the line, as in
// "-## Title". Logseq reads such a line as a bullet with a heading, the same
// as "- ## Title". CommonMark needs a space after the dash of a bullet, so the
// list parsers of Goldmark do not read it.
func isHeadingBullet(line []byte) bool {
	if len(line) < 2 || line[0] != '-' || line[1] != '#' {
		return false
	}

	i := 1
	for i < len(line) && line[i] == '#' {
		i++
	}

	return i == len(line) || util.IsSpace(line[i])
}
