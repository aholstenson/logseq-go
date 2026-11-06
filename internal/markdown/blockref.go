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

// blockRefParser parses Logseq block references links which are in the form
// of ((block-id)).
type blockRefParser struct {
}

func (t *blockRefParser) Trigger() []byte {
	return []byte{'('}
}

func (t *blockRefParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()

	if len(line) < 2 || line[0] != '(' || line[1] != '(' {
		return nil
	}

	if !canStartBlockRef(parent, block) {
		return nil
	}

	end := 0
	var value []byte

	// Scan until the closing )) only allowing valid UUID chars
	for i := 2; i < len(line)-1; i++ {
		if line[i] == ')' && line[i+1] == ')' {
			end = i
			break
		} else if !isUUIDChar(line[i]) {
			return nil
		}
	}

	// Didn't find the closing )), so this isn't a block reference.
	if end == 0 {
		return nil
	}

	// The value of the block reference is the text between the (( and )).
	value = line[2:end]

	// Advance the block reader to the end of the block reference.
	block.Advance(end + 2)

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
	prev := block.PrecendingCharacter()
	if strings.ContainsRune(blockRefPrefixes, prev) {
		return true
	}

	// The inline parser adds the text before the current position to the
	// parent before it calls this parser. If the last child is text, the
	// reference continues a run of plain text.
	last, isText := parent.LastChild().(*ast.Text)
	if !isText {
		return true
	}

	return blockRefCanFollow(last.Segment.Value(block.Source()))
}

// blockRefCanFollow returns true if a block reference can start directly
// after the Markdown text.
func blockRefCanFollow(text []byte) bool {
	return len(text) > 0 && strings.IndexByte(blockRefPrefixes, text[len(text)-1]) >= 0 ||
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

func isUUIDChar(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || c == '-'
}
