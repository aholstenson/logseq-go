package markdown

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// pipelessDelimiterRowRegexp matches the row that sets the alignment of the
// columns of a table when it is written without any pipes, which is a run of
// dashes with the colons of an alignment around it.
var pipelessDelimiterRowRegexp = regexp.MustCompile(`^[ \t]*:?-+:?[ \t]*$`)

// tableParagraphTransformer turns a paragraph into a table, in the GitHub
// Flavored Markdown syntax.
//
// A paragraph that has a line of only dashes in it is left alone, even though
// that is a row a table can set the alignment of a single column with. Logseq
// writes a block without content as a lone dash, so reading it as a table
// would take the block away.
type tableParagraphTransformer struct {
	transformer parser.ParagraphTransformer
}

func newTableParagraphTransformer(transformer parser.ParagraphTransformer) *tableParagraphTransformer {
	return &tableParagraphTransformer{
		transformer: transformer,
	}
}

func (t *tableParagraphTransformer) Transform(node *ast.Paragraph, reader text.Reader, pc parser.Context) {
	lines := node.Lines()

	// The first line of a paragraph is the header of a table rather than the
	// row that sets the alignment, so it is not looked at.
	for i := 1; i < lines.Len(); i++ {
		line := lines.At(i)
		if isDelimiterRowWithoutPipes(line.Value(reader.Source())) {
			return
		}
	}

	parent := node.Parent()
	previous := node.PreviousSibling()
	next := node.NextSibling()
	blankPreviousLines := node.HasBlankPreviousLines()

	// Logseq writes the properties of a block after its content, so a table
	// can be followed by property lines. They are taken off before the table
	// is made, as every line after the header would be a row of the table.
	properties := trailingPropertyLines(node, reader.Source())
	if len(properties) > 0 {
		lines.SetSliced(0, lines.Len()-len(properties))
	}

	t.transformer.Transform(node, reader, pc)
	addPropertyLinesAfterTable(parent, previous, next, node, properties)

	// A table is put next to the paragraph it was found in without taking over
	// whether there was a blank line before that paragraph. When the whole
	// paragraph turned into the table it starts where the paragraph did, so
	// the blank line is carried across to keep it in the output.
	var first ast.Node
	if previous == nil {
		first = parent.FirstChild()
	} else {
		first = previous.NextSibling()
	}

	if table, ok := first.(*east.Table); ok {
		table.SetBlankPreviousLines(blankPreviousLines)
	}
}

// trailingPropertyLines finds the property lines at the end of a paragraph
// that has other lines before them.
func trailingPropertyLines(node *ast.Paragraph, source []byte) []text.Segment {
	lines := node.Lines()
	start := lines.Len()
	for start > 1 {
		line := lines.At(start - 1)
		if !isPropertyLine(line.Value(source)) {
			break
		}

		start--
	}

	properties := make([]text.Segment, 0, lines.Len()-start)
	for i := start; i < lines.Len(); i++ {
		properties = append(properties, lines.At(i))
	}

	return properties
}

// addPropertyLinesAfterTable puts the property lines that were taken off a
// paragraph back. If the paragraph became a table they are put in a paragraph
// of their own after it, otherwise they are put back in the paragraph.
func addPropertyLinesAfterTable(parent ast.Node, previous ast.Node, next ast.Node, node *ast.Paragraph, properties []text.Segment) {
	if len(properties) == 0 {
		return
	}

	// The nodes that the paragraph turned into are between the nodes that
	// were before and after it.
	start := parent.FirstChild()
	if previous != nil {
		start = previous.NextSibling()
	}

	var table ast.Node
	for c := start; c != nil && c != next; c = c.NextSibling() {
		if _, ok := c.(*east.Table); ok {
			table = c
		}
	}

	if table == nil {
		for _, line := range properties {
			node.Lines().Append(line)
		}

		return
	}

	paragraph := ast.NewParagraph()
	for _, line := range properties {
		paragraph.Lines().Append(line)
	}

	parent.InsertAfter(parent, table, paragraph)
}

// isDelimiterRowWithoutPipes checks if a line is read as the row that sets the
// alignment of the columns of a table, but is written without the pipes that
// separate the columns.
func isDelimiterRowWithoutPipes(line []byte) bool {
	if bytes.ContainsRune(line, '|') {
		return false
	}

	return pipelessDelimiterRowRegexp.Match(bytes.TrimRight(line, "\r\n"))
}

var _ parser.ParagraphTransformer = (*tableParagraphTransformer)(nil)
