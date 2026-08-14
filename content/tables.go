package content

import (
	"strings"
)

// TableAlignment is how the text of a column in a table lines up, as set by
// the row of dashes below the header.
type TableAlignment int

const (
	// TableAlignmentNone leaves the alignment of a column to whoever renders
	// the table, and is written as `---`.
	TableAlignmentNone TableAlignment = iota
	// TableAlignmentLeft lines the text of a column up to the left, and is
	// written as `:--`.
	TableAlignmentLeft
	// TableAlignmentRight lines the text of a column up to the right, and is
	// written as `--:`.
	TableAlignmentRight
	// TableAlignmentCenter centers the text of a column, and is written as
	// `:-:`.
	TableAlignmentCenter
)

func (a TableAlignment) String() string {
	switch a {
	case TableAlignmentNone:
		return "none"
	case TableAlignmentLeft:
		return "left"
	case TableAlignmentRight:
		return "right"
	case TableAlignmentCenter:
		return "center"
	}

	return ""
}

// Table is a table written in the GitHub Flavored Markdown syntax:
//
//	| Name | Value |
//	| ---- | ----- |
//	| a    | 1     |
//
// The first row of a table names its columns, the rows below it hold the data.
type Table struct {
	baseNodeWithChildren
	previousLineAwareImpl

	// Alignments is how the text of each column lines up, in the order the
	// columns appear. A column without an entry is not aligned.
	Alignments []TableAlignment
}

// NewTable creates a table out of its rows, the first of which names the
// columns of the table.
func NewTable(rows ...*TableRow) *Table {
	t := &Table{}
	t.self = t
	t.childValidator = allowOnlyTableRows
	for _, row := range rows {
		t.AddChild(row)
	}
	return t
}

// WithAlignments sets how the text of each column lines up, in the order the
// columns appear.
func (t *Table) WithAlignments(alignments ...TableAlignment) *Table {
	t.Alignments = alignments
	return t
}

// Header gets the row that names the columns of the table, which is nil if the
// table has no rows at all.
func (t *Table) Header() *TableRow {
	row, _ := t.FirstChild().(*TableRow)
	return row
}

// BodyRows gets the rows that hold the data of the table, which is every row
// below the header.
func (t *Table) BodyRows() []*TableRow {
	rows := make([]*TableRow, 0)

	header := t.Header()
	if header == nil {
		return rows
	}

	for child := header.NextSibling(); child != nil; child = child.NextSibling() {
		if row, ok := child.(*TableRow); ok {
			rows = append(rows, row)
		}
	}

	return rows
}

// AlignmentOf gets how the text of the column with the given index lines up.
// Columns that the table does not have an alignment for are not aligned.
func (t *Table) AlignmentOf(column int) TableAlignment {
	if column < 0 || column >= len(t.Alignments) {
		return TableAlignmentNone
	}

	return t.Alignments[column]
}

func (t *Table) debug(p *debugPrinter) {
	p.StartType("Table")
	if len(t.Alignments) > 0 {
		names := make([]string, len(t.Alignments))
		for i, alignment := range t.Alignments {
			names[i] = alignment.String()
		}
		p.Field("alignments", strings.Join(names, ", "))
	}
	debugPreviousLineAware(p, t)
	p.Children(t)
	p.EndType()
}

func (t *Table) WithPreviousLineType(lineType PreviousLineType) *Table {
	t.previousLineType = lineType
	return t
}

func (t *Table) isBlock() {}

var _ BlockNode = (*Table)(nil)

// TableRow is a single row of a table, holding one cell per column.
type TableRow struct {
	baseNodeWithChildren
}

func NewTableRow(cells ...*TableCell) *TableRow {
	r := &TableRow{}
	r.self = r
	r.childValidator = allowOnlyTableCells
	for _, cell := range cells {
		r.AddChild(cell)
	}
	return r
}

func (r *TableRow) debug(p *debugPrinter) {
	p.StartType("TableRow")
	p.Children(r)
	p.EndType()
}

var _ HasChildren = (*TableRow)(nil)

// TableCell is the content of one column of a row.
type TableCell struct {
	baseNodeWithChildren
}

func NewTableCell(children ...Node) *TableCell {
	c := &TableCell{}
	c.self = c
	c.childValidator = allowOnlyInlineNodes
	c.AddChildren(children...)
	return c
}

func (c *TableCell) debug(p *debugPrinter) {
	p.StartType("TableCell")
	p.Children(c)
	p.EndType()
}

var _ HasChildren = (*TableCell)(nil)

func allowOnlyTableRows(node Node) bool {
	_, ok := node.(*TableRow)
	return ok
}

func allowOnlyTableCells(node Node) bool {
	_, ok := node.(*TableCell)
	return ok
}
