package markdown

import (
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

// tagParser parses Logseq style tags in Goldmark.
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
	line = line[1:]

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
		// TODO: Does Logseq support Unicode tags?
		// Scan until a Unicode space character is found.
		for i, r := range line {
			if unicode.IsSpace(rune(r)) {
				end = i
				break
			}
		}

		if end == 0 {
			// No space found, assume the tag is until end of line.
			end = len(line)
		}

		value = line[:end]
	}

	seg = seg.WithStop(seg.Start + end + 1)

	n := tag{
		Page: string(value),
	}
	block.Advance(seg.Len())
	return &n
}

var _ parser.InlineParser = (*tagParser)(nil)
