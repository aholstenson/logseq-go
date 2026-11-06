package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Logseq reads a line such as `#+title: My page` as a directive, which is
// the Org-mode form of a property. It does so in Markdown files too:
//
//   - A directive is never a tag, so `#+title: X` does not refer to the page
//     `+title`. See isDirective and atLineStart.
//   - A directive in the content before the first bullet of a page is a page
//     property. A directive line after a `name:: value` line is one more
//     property of the same group. See findDirectiveAt.
//
// Logseq also moves a directive that is in the text of a block to the
// properties of the page. Such a directive is kept as text of the block.

// isDirective checks if the line starts with a directive: `#+`, then a name
// that holds no colon, and then a colon. This is the rule of Logseq for the
// lines that are not tags. A directive with a name that is not a valid
// property name is kept as text.
func isDirective(line []byte) bool {
	if !bytes.HasPrefix(line, []byte("#+")) {
		return false
	}

	for i := 2; i < len(line); i++ {
		switch line[i] {
		case ':':
			return i > 2
		case '\n', '\r':
			return false
		}
	}

	return false
}

// findDirectiveAt finds a directive at the start of the line that starts at
// the given offset into the source, and returns its name and the offset
// where its value starts. The colon after the name must be followed by a
// space or by the end of the line, in the same way as `name::`, so that the
// directive is written back in the same form.
func findDirectiveAt(source []byte, start int) (name string, valueStart int, ok bool) {
	line := source[start:]
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = bytes.TrimSuffix(line, []byte("\r"))

	if !isDirective(line) {
		return "", 0, false
	}

	colon := bytes.IndexByte(line, ':')
	name = string(line[2:colon])
	if bytes.ContainsAny(line[2:colon], " \t") || !validPropertyName(name) {
		return "", 0, false
	}

	rest := line[colon+1:]
	switch {
	case len(rest) > 0 && rest[0] == ' ':
		return name, start + colon + 2, true
	case len(rest) == 0:
		return name, start + len(line), true
	default:
		return "", 0, false
	}
}

// atLineStart checks if the inline reader is at the start of a line of the
// paragraph that holds the parent, with only spaces and tabs before it on
// that line.
func atLineStart(parent ast.Node, reader text.Reader) bool {
	block := parent
	for block != nil && block.Type() != ast.TypeBlock {
		block = block.Parent()
	}

	if block == nil || (block.Kind() != ast.KindParagraph && block.Kind() != ast.KindTextBlock) {
		return false
	}

	_, position := reader.Position()
	source := reader.Source()
	lines := block.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		if position.Start < line.Start || position.Start >= line.Stop {
			continue
		}

		return len(bytes.Trim(source[line.Start:position.Start], " \t")) == 0
	}

	return false
}

// atPageStart checks if the paragraph is in the content before the first
// bullet of a page, where a directive is a page property.
func atPageStart(paragraph ast.Node) bool {
	if _, ok := paragraph.Parent().(*ast.Document); !ok {
		return false
	}

	for previous := paragraph.PreviousSibling(); previous != nil; previous = previous.PreviousSibling() {
		switch previous.(type) {
		case *ast.List, *indentedBlocks, *ast.Heading:
			return false
		}
	}

	return true
}
