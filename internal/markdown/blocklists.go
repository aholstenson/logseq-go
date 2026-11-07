package markdown

import (
	"bytes"

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
	expandTabsBeforeBlockMarker(reader)

	if _, ok := indentedUnderHeading(parent, reader, pc); ok {
		// The bullets are children of the heading, see indentedBlocksParser.
		return nil, parser.NoChildren
	}

	if interruptsWithEmptyBlock(parent, reader, pc) {
		// The list item parser of Goldmark adds the empty item to the list.
		return ast.NewList('-'), parser.HasChildren
	}

	if _, inList := pc.LastOpenedBlock().Node.(*ast.List); !inList && holdsBlocks(parent) && !continuesQuoteParagraph(parent, pc) {
		if _, _, ok := headingBulletPosition(reader); ok {
			// blockListItemParser adds the item to the list.
			pc.Set(emptyItemWithBlankLinesKey, nil)
			return ast.NewList('-'), parser.HasChildren
		}
	}

	node, state := p.BlockParser.Open(parent, reader, pc)
	if node != nil {
		// Goldmark clears its flag for an empty item with blank lines.
		pc.Set(emptyItemWithBlankLinesKey, nil)
	}

	return node, state
}

// emptyItemWithBlankLinesKey is set when a list of Goldmark gets a blank line
// after an empty item, in the same way as Goldmark sets a flag of its own. The
// flag of Goldmark cannot be read, and while it is set, a list ends at the next
// line with text.
var emptyItemWithBlankLinesKey = parser.NewContextKey()

func (p *blockListParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	if !isBlockList(node) && inBlockContent(node) && startsWithBlockMarker(reader) {
		// Logseq reads each line that starts with "- " as a block, at any
		// indentation. Thus a list with "*", "+" or numbers ends before it,
		// also when CommonMark would nest the bullet in the list.
		return parser.Close
	}

	if isBlockList(node) {
		expandTabsBeforeBlockMarker(reader)
	}

	if continuesUnderLastItem(node, reader, pc) {
		return parser.Continue | parser.HasChildren
	}

	if _, _, ok := headingBulletPosition(reader); ok && isBlockList(node) {
		// The line is the next block of the list, or a child of its last
		// block, see blockListItemParser.
		return parser.Continue | parser.HasChildren
	}

	if restOfLineIsBlank(reader) && node.LastChild() != nil && node.LastChild().ChildCount() == 0 {
		pc.Set(emptyItemWithBlankLinesKey, true)
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

// continuesUnderLastItem returns true if the list continues at a line with
// text that is indented under the text of its last item, which is not empty.
// The list parser of Goldmark gives the same result, but reads all of the
// indentation to find it, which is slow for deeply nested outlines.
func continuesUnderLastItem(list ast.Node, reader text.Reader, pc parser.Context) bool {
	item, ok := list.LastChild().(*ast.ListItem)
	if !ok || item.ChildCount() == 0 || pc.Get(emptyItemWithBlankLinesKey) != nil || restOfLineIsBlank(reader) {
		return false
	}

	line, _ := reader.PeekLine()
	indent, _ := indentWidthUpTo(line, reader.LineOffset(), item.Offset)
	return indent >= item.Offset
}

// expandTabsBeforeBlockMarker moves the reader past the spaces and tabs
// before a bullet "- ", and keeps their width as padding. The list parsers
// of Goldmark do not read a bullet after a tab, so "\t- b" in the text of a
// block, such as the second line of "- a\n  \t- b", is read as text without
// this. Logseq reads such a line as a block. The width in columns is kept, so
// Goldmark nests the bullet in the same way as with spaces.
func expandTabsBeforeBlockMarker(reader text.Reader) {
	line, _ := reader.PeekLine()
	indent, pos := indentWidthUpTo(line, reader.LineOffset(), 4)
	if indent >= 4 || pos >= len(line) || !isBlockMarker(line[pos:]) {
		return
	}

	if bytes.IndexByte(line[:pos], '\t') < 0 {
		return
	}

	reader.Advance(pos)
	reader.SetPadding(indent)
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
// parser of Goldmark. It keeps the indentation of each bullet, see
// bulletIndent, and it has one difference that Logseq has: a block without
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

func (p *blockListItemParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	indent := lineIndentCharacters(reader)
	start := readerLineStart(reader)
	expandTabsBeforeBlockMarker(reader)
	var node ast.Node
	var state parser.State
	if !continuesQuoteParagraph(parent, pc) {
		node, state = openHeadingBulletItem(parent, reader)
	}
	if node == nil {
		node, state = p.BlockParser.Open(parent, reader, pc)
	}
	if node != nil {
		node.SetAttribute(bulletIndentAttribute, indent)
		node.SetAttribute(bulletLineStartAttribute, start)

		// Goldmark clears its flag for an empty item with blank lines.
		pc.Set(emptyItemWithBlankLinesKey, nil)
	}

	return node, state
}

// bulletLineStartAttribute is the attribute of a list item that holds the
// offset into the source where the line of its bullet starts.
var bulletLineStartAttribute = []byte("logseq-bullet-line-start")

// hasBlankLineBeforeBullet checks if the line before the bullet of a list
// item is blank, so that the blank line can be kept when the block is
// written. Goldmark does not tell this for each item, as it marks the items
// of a list with blank lines between them in another way.
func hasBlankLineBeforeBullet(src []byte, item ast.Node) bool {
	value, ok := item.Attribute(bulletLineStartAttribute)
	if !ok {
		return false
	}

	start, ok := value.(int)
	if !ok || start == 0 {
		return false
	}

	return isBlankLine(src[lineStart(src, start-1):start-1], inQuote(item))
}

// isBlankLine checks if a line has only spaces and tabs. In a quote, a line
// with only `>` markers and spaces is blank too, as in the second line of
// "> a\n>\n> b".
func isBlankLine(line []byte, quoted bool) bool {
	cutset := " \t\r"
	if quoted {
		cutset += ">"
	}

	return len(bytes.Trim(line, cutset)) == 0
}

// inQuote checks if the node is in a quote.
func inQuote(node ast.Node) bool {
	for p := node.Parent(); p != nil; p = p.Parent() {
		if _, ok := p.(*ast.Blockquote); ok {
			return true
		}
	}

	return false
}

// hasBlankQuoteLineBefore checks if the line before the first line of a node
// in a quote is blank, such as the line before `b` in "> a\n>\n> b". Goldmark
// does not mark such a line as a blank line before the node.
func hasBlankQuoteLineBefore(src []byte, node ast.Node) bool {
	if node.Lines().Len() == 0 || !inQuote(node) {
		return false
	}

	start := lineStart(src, node.Lines().At(0).Start)
	if start == 0 {
		return false
	}

	return isBlankLine(src[lineStart(src, start-1):start-1], true)
}

// bulletIndentAttribute is the attribute of a list item that holds the
// number of spaces and tabs before its bullet marker, from the start of the
// line. See lineIndentCharacters.
var bulletIndentAttribute = []byte("logseq-bullet-indent")

// bulletIndent gets the number of spaces and tabs before the bullet marker of
// a list item, as kept by blockListItemParser.
func bulletIndent(item ast.Node) (int, bool) {
	value, ok := item.Attribute(bulletIndentAttribute)
	if !ok {
		return 0, false
	}

	indent, ok := value.(int)
	return indent, ok
}

// lineIndentCharacters counts the spaces and tabs at the start of the line
// that the reader is on, from the start of the line and not from the position
// of the reader. Logseq gives each bullet a level from this count, where a tab
// and a space each count as one character.
func lineIndentCharacters(reader text.Reader) int {
	source := reader.Source()
	start := readerLineStart(reader)

	count := 0
	for i := start; i < len(source) && (source[i] == ' ' || source[i] == '\t'); i++ {
		count++
	}

	return count
}

// emptyBulletLine checks if the line of the bullet of a list item has no
// text after the `-`, and gets the spaces and tabs that follow the `-`.
func emptyBulletLine(src []byte, item ast.Node) (string, bool) {
	value, ok := item.Attribute(bulletLineStartAttribute)
	if !ok {
		return "", false
	}

	start, ok := value.(int)
	if !ok {
		return "", false
	}

	line := src[start:]
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = bytes.TrimSuffix(bytes.TrimLeft(line, " \t"), []byte("\r"))

	if len(line) == 0 || line[0] != '-' || len(bytes.Trim(line[1:], " \t")) > 0 {
		return "", false
	}

	return string(line[1:]), true
}

// bulletSpace gets the spaces and tabs between the `-` of the bullet of a list
// item and the text on the line of the bullet, see listMarkerSpace.
func bulletSpace(src []byte, item ast.Node) string {
	marker, space := listMarkerSpace(src, item)
	if marker != "-" {
		return ""
	}

	return space
}

// listMarkerSpace gets the marker of a list item, such as "-", "*" or "1.",
// and the spaces and tabs between it and the text on the line of the marker.
// Only spaces and tabs that are at most four columns wide are found, as more
// of them start the text with indentation of its own. The space is "" when
// there is one space, no text, or no space, as in "-## Title".
func listMarkerSpace(src []byte, item ast.Node) (string, string) {
	value, ok := item.Attribute(bulletLineStartAttribute)
	if !ok {
		return "", ""
	}

	start, ok := value.(int)
	if !ok {
		return "", ""
	}

	line := src[start:]
	if i := bytes.IndexAny(line, "\r\n"); i >= 0 {
		line = line[:i]
	}

	// The column of the marker, where a tab moves to the next multiple of
	// four.
	column := 0
	i := 0
	for ; i < len(line) && (line[i] == ' ' || line[i] == '\t'); i++ {
		column = nextColumn(column, line[i])
	}

	markerEnd := i
	for markerEnd < len(line) && line[markerEnd] >= '0' && line[markerEnd] <= '9' {
		markerEnd++
	}

	switch {
	case markerEnd > i && markerEnd < len(line) && (line[markerEnd] == '.' || line[markerEnd] == ')'):
		markerEnd++
	case markerEnd == i && i < len(line) && (line[i] == '-' || line[i] == '*' || line[i] == '+'):
		markerEnd++
	default:
		return "", ""
	}

	marker := string(line[i:markerEnd])
	textColumn := column + markerEnd - i
	column = textColumn
	end := markerEnd
	for ; end < len(line) && (line[end] == ' ' || line[end] == '\t'); end++ {
		column = nextColumn(column, line[end])
	}

	space := string(line[markerEnd:end])
	if space == " " || end >= len(line) || column-textColumn > 4 {
		return marker, ""
	}

	return marker, space
}

// nextColumn gets the column after a space or a tab at the column. A tab
// moves to the next multiple of four.
func nextColumn(column int, c byte) int {
	if c == '\t' {
		return column + 4 - column%4
	}

	return column + 1
}

// readerLineStart gets the offset into the source where the line that the
// reader is on starts.
func readerLineStart(reader text.Reader) int {
	source := reader.Source()
	_, segment := reader.Position()

	start := segment.Start
	for start > 0 && source[start-1] != '\n' && source[start-1] != '\r' {
		start--
	}

	return start
}

func (p *blockListItemParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	item, ok := node.(*ast.ListItem)
	if !ok {
		return p.BlockParser.Continue(node, reader, pc)
	}

	if keepsLine(item, reader) {
		line, _ := reader.PeekLine()
		pos, padding := util.IndentPosition(line, reader.LineOffset(), item.Offset)
		reader.AdvanceAndSetPadding(pos, padding)
		return parser.Continue | parser.HasChildren
	}

	if offset, ok := continuesItem(item, reader); ok {
		// The same result as the list item parser of Goldmark, which reads
		// all of the indentation to find it. That is slow for deeply nested
		// outlines.
		line, _ := reader.PeekLine()
		pos, padding := util.IndentPosition(line, reader.LineOffset(), offset)
		reader.AdvanceAndSetPadding(pos, padding)
		return parser.Continue | parser.HasChildren
	}

	if indent, _, ok := headingBulletPosition(reader); ok {
		// The same rule as the list item parser of Goldmark has for a line
		// with a bullet that it reads: the line starts the next item.
		if item.ChildCount() == 0 || indent < lastItemOffset(item.Parent()) {
			return parser.Close
		}
	}

	return p.BlockParser.Continue(node, reader, pc)
}

// headingBulletPosition checks if the line is a bullet that is directly
// followed by the `#` marks of a heading, see isHeadingBullet, and that has
// less than four columns of indentation. It returns the width of the
// indentation in columns, and the position of the dash in the line.
func headingBulletPosition(reader text.Reader) (int, int, bool) {
	line, _ := reader.PeekLine()
	indent, pos := util.IndentWidth(line, reader.LineOffset())
	if indent >= 4 || pos >= len(line) || !isHeadingBullet(line[pos:]) {
		return 0, 0, false
	}

	return indent, pos, true
}

// openHeadingBulletItem opens a list item for a bullet that is directly
// followed by the `#` marks of a heading, such as "-## Title", in a list of
// blocks. The list item parser of Goldmark needs a space after the dash. The
// item gets the same offset as the item of "- ## Title", and its content
// starts at the `#` marks. Returns nil if the line is not such a bullet.
func openHeadingBulletItem(parent ast.Node, reader text.Reader) (ast.Node, parser.State) {
	if !isBlockList(parent) {
		return nil, parser.NoChildren
	}

	indent, pos, ok := headingBulletPosition(reader)
	if !ok || indent-lastItemOffset(parent) > 3 {
		return nil, parser.NoChildren
	}

	reader.Advance(pos + 1)
	return ast.NewListItem(indent + 2), parser.HasChildren
}

// continuesQuoteParagraph checks if the line is a lazy continuation line of a
// paragraph in a quote: the paragraph is open, and it is not a direct child of
// the parent that a new block would get. Logseq reads a line such as "-## b"
// right after "> q" as more text of the quote, while a bullet "- b" ends the
// quote.
func continuesQuoteParagraph(parent ast.Node, pc parser.Context) bool {
	last := pc.LastOpenedBlock().Node
	if !ast.IsParagraph(last) || last.Parent() == parent {
		return false
	}

	for p := last.Parent(); p != nil && p != parent; p = p.Parent() {
		if _, ok := p.(*ast.Blockquote); ok {
			return true
		}
	}

	return false
}

// lastItemOffset gets the offset of the text of the last item of a list, or 0
// if the list has no items.
func lastItemOffset(list ast.Node) int {
	if item, ok := list.LastChild().(*ast.ListItem); ok {
		return item.Offset
	}

	return 0
}

// continuesItem returns true if the line has text and is indented under the
// text of the last item of the list of the item, and the item is not empty.
// The item holds such a line. Also returns the offset of the text of that
// last item.
func continuesItem(item *ast.ListItem, reader text.Reader) (int, bool) {
	last, ok := item.Parent().LastChild().(*ast.ListItem)
	if !ok || item.ChildCount() == 0 || restOfLineIsBlank(reader) {
		return 0, false
	}

	line, _ := reader.PeekLine()
	indent, _ := indentWidthUpTo(line, reader.LineOffset(), last.Offset)
	return last.Offset, indent >= last.Offset
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

	line, _ := reader.PeekLine()
	marker := string(bytes.TrimRight(bytes.TrimLeft(line, " \t"), "\r\n"))
	node, state := p.BlockParser.Open(parent, reader, pc)
	if node != nil {
		node.SetAttributeString(thematicBreakMarkerAttribute, marker)
	}

	return node, state
}

// thematicBreakMarkerAttribute is the name of the attribute that holds the
// text of a thematic break, such as `***`, as it is in the file.
const thematicBreakMarkerAttribute = "logseq-marker"

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
