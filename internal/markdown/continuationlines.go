package markdown

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// lineIndentationKey is where the parser keeps the lineIndentation of the
// paragraphs that it closes.
var lineIndentationKey = parser.NewContextKey()

// lineIndentationAttribute is the name of the document attribute that holds
// the lineIndentation of a parsed document.
var lineIndentationAttribute = []byte("logseqLineIndentation")

// lineIndentation holds the spaces and tabs at the start of the lines of
// paragraphs, such as the two spaces before `b` in "a\n  b", or before `a` in
// "# H\n  a". Only the indentation after the indentation of the block or quote
// that holds the paragraph is kept. The key is the offset in the source of the
// start of the line.
//
// Goldmark removes these spaces from the lines of a paragraph. Logseq keeps
// them as text, so they are written back to not change the line on save.
type lineIndentation map[int]string

// paragraphParser is the paragraph parser of Goldmark, changed to record the
// indentation of continuation lines, see lineIndentation.
type paragraphParser struct {
	parser.BlockParser
}

func newParagraphParser() *paragraphParser {
	return &paragraphParser{
		BlockParser: parser.NewParagraphParser(),
	}
}

func (p *paragraphParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	// The paragraph parser of Goldmark removes the indentation of the first
	// line, so it is recorded before.
	line, segment := reader.PeekLine()
	// The line holds the padding of the reader as spaces.
	indentation := string(spacesAndTabs(line))

	// The first paragraph of a footnote definition starts on the line of
	// its label, and the space after the label belongs to the label.
	_, inFootnote := parent.(*footnoteDefinition)
	afterLabel := inFootnote && parent.FirstChild() == nil

	lineStart := bytes.LastIndexByte(reader.Source()[:segment.Start], '\n') + 1
	if item, ok := blockItemOf(parent); ok && !util.IsBlank(line) {
		// A paragraph that starts on a line after the bullet of its block
		// is measured from the column of the bullet.
		if value, _, ok := blockLineIndentation(reader.Source(), item, lineStart); ok {
			indentation = value
		}
	}

	node, state := p.BlockParser.Open(parent, reader, pc)
	if node != nil && indentation != "" && !afterLabel && !util.IsBlank(line) {
		addLineIndentation(pc, lineStart, indentation)
	}

	return node, state
}

// blockItemOf returns the node if it is the list item of a block.
func blockItemOf(node ast.Node) (ast.Node, bool) {
	if item, ok := node.(*ast.ListItem); ok && isBlockList(item.Parent()) {
		return item, true
	}

	return nil, false
}

// blockLineIndentation gets the indentation of a line in the text of a
// block, other than the line of its bullet. The text of a block is written
// at the column of the text on the line of its bullet, which is two columns
// after its `-`, or further for a bullet with more spaces as in "-   a", see
// bulletSpace. The indentation is measured from there.
//
// A line indented less than that is lazy: it is a part of the block, but is
// written without the indentation of the text of the block, and its
// indentation is measured from the column of the `-`. Returns false for the
// line of the bullet, or if the item has no bullet line.
func blockLineIndentation(src []byte, item ast.Node, lineStart int) (string, bool, bool) {
	value, ok := item.Attribute(bulletLineStartAttribute)
	bulletStart, isInt := value.(int)
	if !ok || !isInt || bulletStart == lineStart {
		return "", false, false
	}

	dash := 0
	i := bulletStart
	for ; i < len(src) && (src[i] == ' ' || src[i] == '\t'); i++ {
		dash = nextColumn(dash, src[i])
	}

	textColumn := dash + 2
	if space := bulletSpace(src, item); space != "" {
		textColumn = dash + 1
		for j := 0; j < len(space); j++ {
			textColumn = nextColumn(textColumn, space[j])
		}
	}

	if indentationWidth(src, lineStart) >= textColumn {
		return indentationAfterColumn(src, lineStart, textColumn), false, true
	}

	return indentationAfterColumn(src, lineStart, dash), true, true
}

// wideBulletLineIndentation gets the indentation of a line in the text of a
// block whose bullet has more than one space, as in "-   a", measured from two
// columns after the `-` of the block. Property lines and the dates of tasks
// are written from that column, as Logseq writes them. Returns false if the
// block has no such bullet, or if the line is indented less than that column.
func wideBulletLineIndentation(src []byte, item ast.Node, lineStart int) (string, bool) {
	value, ok := item.Attribute(bulletLineStartAttribute)
	bulletStart, isInt := value.(int)
	if !ok || !isInt || bulletStart == lineStart || bulletSpace(src, item) == "" {
		return "", false
	}

	dash := indentationWidth(src, bulletStart)
	if indentationWidth(src, lineStart) < dash+2 {
		return "", false
	}

	return indentationAfterColumn(src, lineStart, dash+2), true
}

// indentationWidth gets the width in columns of the spaces and tabs at the
// start of the line, where a tab moves to the next multiple of four.
func indentationWidth(src []byte, lineStart int) int {
	column := 0
	for i := lineStart; i < len(src) && (src[i] == ' ' || src[i] == '\t'); i++ {
		column = nextColumn(column, src[i])
	}

	return column
}

// indentationAfterColumn gets the spaces and tabs at the start of the line
// after the given column. The part of a tab after the column is written as
// spaces.
func indentationAfterColumn(src []byte, lineStart int, target int) string {
	column := 0
	i := lineStart
	for ; i < len(src) && (src[i] == ' ' || src[i] == '\t') && column < target; i++ {
		column = nextColumn(column, src[i])
	}

	end := i
	for end < len(src) && (src[end] == ' ' || src[end] == '\t') {
		end++
	}

	if column < target {
		return ""
	}

	return strings.Repeat(" ", column-target) + string(src[i:end])
}

// lazyBlockLinesKey is where the parser keeps the start offsets of the lines
// of blocks that are indented less than the text of the block, see
// content.Text.LazyBlockLine.
var lazyBlockLinesKey = parser.NewContextKey()

// lazyBlockLinesAttribute is the name of the document attribute that holds
// the lazy block lines of a parsed document.
var lazyBlockLinesAttribute = []byte("logseqLazyBlockLines")

// addLazyBlockLine records the line that starts at the offset as a lazy line
// of a block.
func addLazyBlockLine(pc parser.Context, lineStart int) {
	lines, _ := pc.Get(lazyBlockLinesKey).(map[int]bool)
	if lines == nil {
		lines = map[int]bool{}
		pc.Set(lazyBlockLinesKey, lines)
	}

	lines[lineStart] = true
}

// isLazyBlockLineAfter checks if the line after the line break of a text is
// a lazy line of a block, see content.Text.LazyBlockLine.
func isLazyBlockLineAfter(src []byte, node *ast.Text) bool {
	if !node.SoftLineBreak() && !node.HardLineBreak() {
		return false
	}

	doc := ast.Node(node)
	for doc.Parent() != nil {
		doc = doc.Parent()
	}

	value, _ := doc.Attribute(lazyBlockLinesAttribute)
	lines, _ := value.(map[int]bool)
	i := bytes.IndexByte(src[node.Segment.Stop:], '\n')
	return i >= 0 && lines[node.Segment.Stop+i+1]
}

// addLineIndentation records the indentation of the line that starts at the
// offset, see lineIndentation.
func addLineIndentation(pc parser.Context, lineStart int, value string) {
	indentation, _ := pc.Get(lineIndentationKey).(lineIndentation)
	if indentation == nil {
		indentation = lineIndentation{}
		pc.Set(lineIndentationKey, indentation)
	}

	indentation[lineStart] = value
}

// firstLineIndentationOf returns the indentation of the first line of a
// paragraph, see lineIndentation. The line of the first text of the paragraph
// is used, as a paragraph that is split from another paragraph, such as the
// text after properties, keeps the lines of that paragraph. Only text has
// line breaks, so the first text is on the first line.
func firstLineIndentationOf(src []byte, node ast.Node) string {
	start := -1
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if text, ok := n.(*ast.Text); ok && entering {
			start = text.Segment.Start
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})

	if lines := node.Lines(); start < 0 && lines.Len() > 0 {
		start = lines.At(0).Start
	}

	if start < 0 {
		return ""
	}

	lineStart := bytes.LastIndexByte(src[:start], '\n') + 1
	return documentLineIndentation(node)[lineStart]
}

// documentLineIndentation gets the lineIndentation of the document that holds
// the node. It is nil if no line has indentation.
func documentLineIndentation(node ast.Node) lineIndentation {
	doc := node
	for doc.Parent() != nil {
		doc = doc.Parent()
	}

	value, _ := doc.Attribute(lineIndentationAttribute)
	indentation, _ := value.(lineIndentation)
	return indentation
}

func (p *paragraphParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	recordLineIndentation(node, reader, pc, "")
	p.BlockParser.Close(node, reader, pc)
}

// recordLineIndentation records the indentation of the continuation lines of
// a paragraph before the paragraph parser removes it. The base indentation is
// written for each line of the paragraph, so it is not recorded. A line with
// less indentation than the base is recorded as having no indentation.
//
// The lines of a paragraph in a block are measured from the column of the
// bullet of the block instead, see blockLineIndentation.
func recordLineIndentation(node ast.Node, reader text.Reader, pc parser.Context, base string) {
	src := reader.Source()
	lines := node.Lines()

	item, inBlock := blockItemOf(node.Parent())

	// The first line has no indentation, as the paragraph starts after it.
	for i := 1; i < lines.Len(); i++ {
		line := lines.At(i)
		if inBlock && base == "" {
			lineStart := bytes.LastIndexByte(src[:line.Start], '\n') + 1
			if value, lazy, ok := blockLineIndentation(src, item, lineStart); ok {
				if value != "" {
					addLineIndentation(pc, lineStart, value)
				}
				if lazy {
					addLazyBlockLine(pc, lineStart)
				}
				continue
			}
		}

		trimmed := line.TrimLeftSpace(src)
		if line.Padding == 0 && trimmed.Start == line.Start {
			continue
		}

		value, ok := strings.CutPrefix(strings.Repeat(" ", line.Padding)+string(src[line.Start:trimmed.Start]), base)
		if !ok || value == "" {
			continue
		}

		lineStart := bytes.LastIndexByte(src[:line.Start], '\n') + 1
		addLineIndentation(pc, lineStart, value)
	}
}

// nextLineIndentationOf returns the indentation of the line after the line
// break of a text, see lineIndentation.
func nextLineIndentationOf(src []byte, node *ast.Text) string {
	if !node.SoftLineBreak() && !node.HardLineBreak() {
		return ""
	}

	indentation := documentLineIndentation(node)
	i := bytes.IndexByte(src[node.Segment.Stop:], '\n')
	if i < 0 {
		return ""
	}

	return indentation[node.Segment.Stop+i+1]
}

var _ parser.BlockParser = (*paragraphParser)(nil)
