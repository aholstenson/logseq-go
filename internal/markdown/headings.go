package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// atxHeadingParser parses headings in the same way as the ATX heading parser
// of Goldmark, with these differences that Logseq has:
//
//   - A heading can have any number of `#` marks. CommonMark reads seven or
//     more marks, as in "####### TODO x", as text, but Logseq reads a
//     heading, and thus also the task marker after the marks.
//   - `#` marks at the end of the line are text of the heading. CommonMark
//     removes them as a closing sequence, so "## x ##" is the heading "x".
//     Logseq reads the heading "x ##", and the text must not change when it
//     is written.
type atxHeadingParser struct {
	parser.BlockParser
}

func newATXHeadingParser() *atxHeadingParser {
	return &atxHeadingParser{BlockParser: parser.NewATXHeadingParser()}
}

func (p *atxHeadingParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	i := pos
	for i < len(line) && line[i] == '#' {
		i++
	}

	if i == pos {
		return nil, parser.NoChildren
	}

	node := ast.NewHeading(i - pos)
	_, inFootnote := parent.(*footnoteDefinition)
	if pos > 0 && !(inFootnote && parent.FirstChild() == nil) {
		// On the line of the label of a footnote, the space before the
		// heading belongs to the label, see footnoteDefinition.LabelSpace.
		node.SetAttributeString(headingIndentationAttribute, string(line[:pos]))
	}

	if i == len(line) {
		// The marks end the file, as in "##".
		return node, parser.NoChildren
	}

	if !util.IsSpace(line[i]) {
		// Text directly after the marks, as in "##x", is not a heading.
		return nil, parser.NoChildren
	}

	start := i + util.TrimLeftSpaceLength(line[i:])
	stop := len(line) - util.TrimRightSpaceLength(line)
	if stop > start {
		node.Lines().Append(text.NewSegment(segment.Start+start-segment.Padding, segment.Start+stop-segment.Padding))
		node.SetAttributeString(headingMarksSpaceAttribute, string(spacesAndTabs(line[i:])))
		node.SetAttributeString(headingTrailingSpaceAttribute, string(spacesAndTabs(line[stop:])))
	} else {
		// A heading without text, as in "## ", keeps the spaces after its
		// marks.
		node.SetAttributeString(headingMarksSpaceAttribute, string(spacesAndTabs(line[i:])))
	}

	return node, parser.NoChildren
}

// headingMarksSpaceAttribute and headingTrailingSpaceAttribute are the
// attributes of a heading that hold the spaces and tabs after its `#` marks
// and at the end of its line, see content.Heading.
const (
	headingMarksSpaceAttribute    = "logseq-marks-space"
	headingTrailingSpaceAttribute = "logseq-trailing-space"
)

// headingIndentationAttribute is the attribute of a heading that holds the
// spaces before its `#` marks, see content.Heading.Indentation.
const headingIndentationAttribute = "logseq-indentation"

// spacesAndTabs gets the spaces and tabs at the start of the text.
func spacesAndTabs(text []byte) []byte {
	i := 0
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}

	return text[:i]
}

var _ parser.BlockParser = (*atxHeadingParser)(nil)
