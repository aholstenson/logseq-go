package markdown

import (
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var tagKind = ast.NewNodeKind("Hashtag")

type tag struct {
	ast.BaseInline
	Page string
}

func (*tag) Kind() ast.NodeKind {
	return tagKind
}

func (n *tag) Dump(src []byte, level int) {
}

// tagDelimiters are the characters that Logseq removes from the end of a tag
// name, so "#tag," has the tag "tag" followed by the text ",".
const tagDelimiters = ",;.!?'\":#"

// tagPrefixes are the characters that can come directly before the # of a
// tag, in addition to Unicode spaces. Logseq accepts the tag delimiters, so
// "a,#tag" has the tag "tag". Logseq also makes a tag after "[", so
// "[#[[a tag]]]" has the tag "a tag".
const tagPrefixes = tagDelimiters + "["

// tagParser parses Logseq style tags in Goldmark.
//
// A # starts a tag only at the start of the text, after a Unicode space or
// after one of tagPrefixes. Thus "C#Sharp" and "https://a.com/#top" do not
// contain tags.
type tagParser struct {
}

func (t *tagParser) Trigger() []byte {
	return []byte{'#'}
}

func (t *tagParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, seg := block.PeekLine()

	if len(line) == 0 || line[0] != '#' {
		return nil
	}

	prev := block.PrecendingCharacter()
	if !unicode.IsSpace(prev) && !strings.ContainsRune(tagPrefixes, prev) {
		return nil
	}

	line = line[1:]

	// Logseq does not make tags in the text of a link, so
	// "[Teams:#](https://a.com/)" is a link and not a tag.
	if insideLinkLabel(parent) && closesAsInlineLink(line) {
		return nil
	}

	// Two # in sequence is not a tag, but the second # can start a tag.
	if len(line) > 0 && line[0] == '#' {
		return nil
	}

	end := 0
	var value []byte

	// Check for extended tag syntax which wraps tags in [[...]], this is how
	// Logseq supports tags with spaces.
	if len(line) > 1 && line[0] == '[' && line[1] == '[' {
		// This is an extended tag so scan until the closing ]].
		for i := 1; i < len(line)-1; i++ {
			if line[i] == ']' && line[i+1] == ']' {
				end = i
				break
			}
		}

		// Didn't find the closing ]], so this isn't a tag.
		if end == 0 {
			return nil
		}

		// The value of the tag is the text between the [[ and ]].
		value = line[2:end]
		end += 2
	} else {
		// Scan until a Unicode space character is found, or use the rest of the
		// line if there is none.
		end = len(line)
		for i, r := range string(line) {
			if unicode.IsSpace(r) {
				end = i
				break
			}
		}

		// Delimiters at the end are not part of the tag name, they stay as
		// text after the tag.
		end = len(strings.TrimRight(string(line[:end]), tagDelimiters))
		value = line[:end]
	}

	if len(value) == 0 {
		// A tag needs a name, so a # on its own is just text.
		return nil
	}

	seg = seg.WithStop(seg.Start + end + 1)

	n := tag{
		Page: string(value),
	}
	block.Advance(seg.Len())
	return &n
}

// insideLinkLabel returns true if the inline content of parent has a "[" or
// "![" that Goldmark has not closed yet. Goldmark keeps each open bracket as
// a node of the kind "LinkLabelState" until it finds the closing "]".
func insideLinkLabel(parent ast.Node) bool {
	for c := parent.LastChild(); c != nil; c = c.PreviousSibling() {
		if c.Kind().String() == "LinkLabelState" {
			return true
		}
	}
	return false
}

// closesAsInlineLink returns true if rest has the "]" that closes the open
// link label, and a "(" comes directly after it. Brackets in rest must be in
// balanced pairs, and a backslash escapes the character after it.
func closesAsInlineLink(rest []byte) bool {
	depth := 0
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '\\':
			i++
		case '[':
			depth++
		case ']':
			if depth == 0 {
				return i+1 < len(rest) && rest[i+1] == '('
			}
			depth--
		}
	}
	return false
}

var _ parser.InlineParser = (*tagParser)(nil)
