package markdown

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var blockRefKind = ast.NewNodeKind("BlockRef")

type blockRef struct {
	ast.BaseInline
	ID string
}

func (*blockRef) Kind() ast.NodeKind {
	return blockRefKind
}

func (n *blockRef) Dump(src []byte, level int) {
}

var blockRefTextKind = ast.NewNodeKind("BlockRefText")

// blockRefText is text that Logseq reads as a block reference to an id that is
// not a uuid, such as `((see [[x]]))`. Logseq does not use such a reference,
// and reads nothing in it as markup, so it is kept as text exactly as it is
// written, backslashes included.
type blockRefText struct {
	ast.BaseInline
	Segment text.Segment
}

func (*blockRefText) Kind() ast.NodeKind {
	return blockRefTextKind
}

func (n *blockRefText) Dump(src []byte, level int) {
}

// blockRefParser parses Logseq block references links which are in the form
// of ((block-id)).
//
// Logseq (mldoc) reads `((` and the text up to the first `))` as one block
// reference, if the text is not empty and has no `)` in it. Logseq uses only
// a reference with a uuid as its id, see isBlockID. Such text with another id,
// such as `((abc))` or `((see [[x]]))`, is kept as one piece of text, so the
// page links and tags in it are not references, the same as in Logseq.
type blockRefParser struct {
}

func (t *blockRefParser) Trigger() []byte {
	return []byte{'('}
}

func (t *blockRefParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()

	if len(line) < 2 || line[0] != '(' || line[1] != '(' {
		return nil
	}

	if !canStartBlockRef(parent, block) {
		return nil
	}

	length := blockRefTextLength(line)
	if length == 0 {
		return nil
	}

	// The value of the block reference is the text between the (( and )).
	value := line[2 : length-2]

	// Advance the block reader to the end of the block reference.
	block.Advance(length)

	if !isBlockID(value) {
		// Logseq reads this as a reference to a block that does not exist,
		// so the text in it is not read for references.
		return &blockRefText{Segment: segment.WithStop(segment.Start + length)}
	}

	return &blockRef{
		ID: string(value),
	}
}

// blockRefPrefixes are the characters after which Logseq can start a block
// reference. They are the characters that end a run of plain text in the
// Markdown parser of Logseq (mldoc).
const blockRefPrefixes = " \t\n\r\f\\_^[*~`=$#"

// canStartBlockRef returns true if a block reference can start at the current
// position. Logseq reads a run of plain text as one piece, so `ref:((uuid))`
// and `a((uuid))` are text. A block reference can start at the start of a
// line, after one of blockRefPrefixes, or directly after a different inline
// element, such as `[[page]]((uuid))`. Logseq also reads an escaped character
// such as `\)` and a timestamp such as `<2024-01-01 Mon>` as their own
// elements, but they are text here.
func canStartBlockRef(parent ast.Node, block text.Reader) bool {
	return canStartAfterText(parent, block, blockRefPrefixes)
}

// blockRefCanFollow returns true if a block reference can start directly
// after the Markdown text.
func blockRefCanFollow(text []byte) bool {
	return canFollowText(text, blockRefPrefixes)
}

// canStartAfterText returns true if an element that Logseq does not read
// inside a run of plain text, such as a block reference or a macro, can start
// at the current position. It can start at the start of a line, after one of
// the prefixes, or directly after a different inline element.
func canStartAfterText(parent ast.Node, block text.Reader, prefixes string) bool {
	prev := block.PrecendingCharacter()
	if strings.ContainsRune(prefixes, prev) {
		return true
	}

	// The inline parser adds the text before the current position to the
	// parent before it calls the parser. If the last child is text, the
	// element continues a run of plain text.
	last, isText := parent.LastChild().(*ast.Text)
	if !isText {
		return true
	}

	return canFollowText(last.Segment.Value(block.Source()), prefixes)
}

// canFollowText returns true if an element that Logseq does not read inside
// a run of plain text can start directly after the Markdown text: after one
// of the prefixes, an escaped character or a timestamp.
func canFollowText(text []byte, prefixes string) bool {
	return len(text) > 0 && strings.IndexByte(prefixes, text[len(text)-1]) >= 0 ||
		endsWithEscape(text) || endsWithTimestamp(text)
}

// endsWithEscape returns true if the text ends with a backslash and an ASCII
// punctuation character, such as `\)`.
func endsWithEscape(text []byte) bool {
	if len(text) < 2 || !util.IsPunct(text[len(text)-1]) {
		return false
	}

	// An even number of backslashes are escaped backslashes.
	backslashes := 0
	for i := len(text) - 2; i >= 0 && text[i] == '\\'; i-- {
		backslashes++
	}

	return backslashes%2 == 1
}

// maxTimestampLength is the length of the longest timestamp that
// endsWithTimestamp finds. Real timestamps, such as
// `<2024-01-01 Mon 10:00 .+1w>`, are much shorter.
const maxTimestampLength = 128

// endsWithTimestamp returns true if the text ends with a timestamp, such as
// `<2024-01-01 Mon>`, that Logseq reads. A timestamp in `[` and `]` is not
// looked for, as `[2024-01-01 Mon]((id))` is a link.
func endsWithTimestamp(text []byte) bool {
	if !bytes.HasSuffix(text, []byte(">")) {
		return false
	}

	// Only the end of the text is searched, so that a long line with many
	// block references stays fast to read.
	if len(text) > maxTimestampLength {
		text = text[len(text)-maxTimestampLength:]
	}

	start := bytes.LastIndexByte(text, '<')
	if start < 0 {
		return false
	}

	// Logseq does not end a run of plain text at a `<`, so a `<` directly
	// after a word is text.
	if start > 0 && strings.IndexByte(blockRefPrefixes, text[start-1]) < 0 {
		return false
	}

	timestamp, ok := scanLogseqTimestamp(string(text[start:]))
	return ok && timestamp.rest == ""
}

// blockRefTextLength gets the length of the text at the start of the line
// that Logseq reads as a block reference: `((`, text that is not empty and has
// no `)` in it, and `))`. The id in it does not have to be a uuid. Returns 0
// if the line does not start with such text.
func blockRefTextLength(line []byte) int {
	if len(line) < 5 || line[0] != '(' || line[1] != '(' {
		return 0
	}

	for i := 2; i < len(line)-1; i++ {
		if line[i] == ')' {
			if line[i+1] != ')' || i == 2 {
				return 0
			}

			return i + 2
		}
	}

	return 0
}

// isBlockID checks if the text is a uuid that Logseq uses as the id of a
// block reference: 32 hexadecimal digits in groups of 8-4-4-4-12, without
// spaces around them. Logseq reads a uuid without regard to case, so `A` to
// `F` are allowed too.
func isBlockID(value []byte) bool {
	if len(value) != 36 {
		return false
	}

	for i, c := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !isHexDigit(c) {
			return false
		}
	}

	return true
}

// isHexDigit checks if the character is a hexadecimal digit, in upper or
// lower case.
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
