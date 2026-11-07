package markdown

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
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

	// The header must have as many cells as the delimiter row has columns,
	// otherwise the lines are text. Goldmark fills a short header with empty
	// cells before it checks this, so the check is made here.
	delimiter := -1
	for i := 1; i < lines.Len(); i++ {
		line := lines.At(i)
		if columns := tableDelimiterColumns(line.Value(reader.Source())); columns >= 0 {
			header := lines.At(i - 1)
			if len(tableRowCells(header.Value(reader.Source()))) != columns {
				return
			}

			delimiter = i
			break
		}
	}

	if delimiter < 0 {
		return
	}

	// A line that starts in a tab that is split by the column of a list or
	// a quote has padding: spaces for the part of the tab after the column.
	// Goldmark finds the cells in the text with the padding, but uses those
	// positions in the source, which has no padding, so the cells would get
	// the wrong text. The space at the start of a row has no meaning in a
	// table, so the padding is dropped.
	for i := 0; i < lines.Len(); i++ {
		if line := lines.At(i); line.Padding > 0 {
			line.Padding = 0
			lines.Set(i, line)
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

	// The lines of the table are kept before the paragraph is changed.
	tableLines := make([]text.Segment, 0, lines.Len()-delimiter+1)
	for i := delimiter - 1; i < lines.Len(); i++ {
		tableLines = append(tableLines, lines.At(i))
	}

	t.transformer.Transform(node, reader, pc)
	if table := tableBetween(parent, previous, next); table != nil {
		addExtraTableCells(table, tableLines[2:], reader.Source())
		source := tableSource(tableLines, reader.Source())

		// The paragraph parser takes the indentation off the first line,
		// and keeps it for the line, see paragraphParser.
		if len(tableLines) > 0 && tableLines[0].Padding == 0 {
			lineStart := bytes.LastIndexByte(reader.Source()[:tableLines[0].Start], '\n') + 1
			if indentation, _ := pc.Get(lineIndentationKey).(lineIndentation); indentation != nil {
				source = append([]byte(indentation[lineStart]), source...)
			}
		}

		table.SetAttributeString(tableSourceAttribute, source)
		if markers := quoteMarkersOfLines(table, tableLines, reader.Source()); markers != nil {
			table.SetAttribute(tableLineMarkersAttribute, markers)
		}
	}
	addPropertyLinesAfterTable(parent, previous, next, node, properties, reader.Source())

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

// tableLineMarkersAttribute is the name of the attribute that holds the
// number of `>` markers of each line of a table in a quote, see
// content.Table.SourceLineMarkers.
var tableLineMarkersAttribute = []byte("logseq-line-markers")

// quoteMarkersOfLines gets the number of `>` markers at the start of each
// line of a table in a quote. Returns nil for a table that is not in a quote.
func quoteMarkersOfLines(table ast.Node, lines []text.Segment, source []byte) []int {
	if !inQuote(table) {
		return nil
	}

	markers := make([]int, len(lines))
	for i, line := range lines {
		pos := bytes.LastIndexByte(source[:line.Start], '\n') + 1
		for pos < line.Start {
			switch source[pos] {
			case '>':
				markers[i]++
			case ' ', '\t':
			default:
				pos = line.Start
				continue
			}
			pos++
		}
	}

	return markers
}

// tableSourceAttribute is the name of the attribute that holds the text of a
// table, as it is in the file.
const tableSourceAttribute = "logseq-source"

// tableBetween finds the table that a paragraph was made into. The paragraph
// was between previous and next, and previous is nil if it was the first
// child of parent.
func tableBetween(parent ast.Node, previous ast.Node, next ast.Node) *east.Table {
	start := parent.FirstChild()
	if previous != nil {
		start = previous.NextSibling()
	}

	for c := start; c != nil && c != next; c = c.NextSibling() {
		if table, ok := c.(*east.Table); ok {
			return table
		}
	}

	return nil
}

// addExtraTableCells adds the cells of a row that come after the columns of
// the header. Logseq keeps them, but Goldmark drops them as GitHub Flavored
// Markdown does. The rows are the lines below the delimiter row.
func addExtraTableCells(table *east.Table, rows []text.Segment, source []byte) {
	i := 0
	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		row, ok := child.(*east.TableRow)
		if !ok {
			// The header is not a body row.
			continue
		}

		if i >= len(rows) {
			return
		}

		line := rows[i]
		i++

		cells := tableRowCells(line.Value(source))
		for c := row.ChildCount(); c < len(cells); c++ {
			cell := east.NewTableCell()
			cell.Lines().Append(text.NewSegment(line.Start+cells[c][0], line.Start+cells[c][1]))
			row.AppendChild(row, cell)
		}
	}
}

// tableSource joins the lines of a table, without their line breaks.
func tableSource(lines []text.Segment, source []byte) []byte {
	out := make([]byte, 0)
	for i, line := range lines {
		if i > 0 {
			out = append(out, '\n')
		}

		out = append(out, bytes.TrimRight(line.Value(source), "\r\n")...)
	}

	return out
}

// tableDelimiterColumns returns the number of columns of a delimiter row, in
// the same way as Goldmark reads it. It returns -1 if the line is not a
// delimiter row.
func tableDelimiterColumns(line []byte) int {
	if w, _ := util.IndentWidth(line, 0); w > 3 {
		return -1
	}

	for _, b := range line {
		if !(util.IsSpace(b) || b == '-' || b == '|' || b == ':') {
			return -1
		}
	}

	columns := bytes.Split(line, []byte{'|'})
	if util.IsBlank(columns[0]) {
		columns = columns[1:]
	}

	if len(columns) > 0 && util.IsBlank(columns[len(columns)-1]) {
		columns = columns[:len(columns)-1]
	}

	for _, column := range columns {
		if !tableDelimiterColumnRegexp.Match(column) {
			return -1
		}
	}

	return len(columns)
}

// tableDelimiterColumnRegexp matches one column of a delimiter row.
var tableDelimiterColumnRegexp = regexp.MustCompile(`^\s*:?-+:?\s*$`)

// tableRowCells splits a row of a table into its cells, in the same way as
// Goldmark. It returns the start and end of the text of each cell in the
// line, without the spaces around it. A pipe after a backslash does not end a
// cell.
func tableRowCells(line []byte) [][2]int {
	pos := util.TrimLeftSpaceLength(line)
	limit := len(line) - util.TrimRightSpaceLength(line)
	if pos < limit && line[pos] == '|' {
		pos++
	}

	if pos < limit && line[limit-1] == '|' {
		limit--
	}

	cells := make([][2]int, 0)
	for pos < limit {
		end := pos
		for ; end < limit; end++ {
			if line[end] == '|' && (end == 0 || line[end-1] != '\\') {
				break
			}
		}

		start, stop := pos, end
		for start < stop && util.IsSpace(line[start]) {
			start++
		}

		for stop > start && util.IsSpace(line[stop-1]) {
			stop--
		}

		cells = append(cells, [2]int{start, stop})
		pos = end + 1
	}

	return cells
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
func addPropertyLinesAfterTable(parent ast.Node, previous ast.Node, next ast.Node, node *ast.Paragraph, properties []text.Segment, source []byte) {
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

	// The paragraph parser takes the spaces off the start of the lines of a
	// paragraph, and off the end of its last line, so the same is done
	// here. Otherwise the lines are not read as properties.
	paragraph := ast.NewParagraph()
	for i, line := range properties {
		line = line.TrimLeftSpace(source)
		if i == len(properties)-1 {
			line = line.TrimRightSpace(source)
		}
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
