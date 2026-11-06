package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var pageLinkKind = ast.NewNodeKind("PageLink")

type pageLink struct {
	ast.BaseInline
	Page string
}

func (*pageLink) Kind() ast.NodeKind {
	return pageLinkKind
}

func (n *pageLink) Dump(src []byte, level int) {
}

// pageLinkParser parses Logseq page links which are in the form of [[Page Name]].
type pageLinkParser struct {
}

func (t *pageLinkParser) Trigger() []byte {
	return []byte{'['}
}

func (t *pageLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()

	if len(line) < 2 || line[0] != '[' || line[1] != '[' {
		return nil
	}

	end := pageLinkClose(line)

	// Didn't find the closing ]], so this isn't a page link.
	if end == 0 {
		return nil
	}

	// The value of the page link is the text between the [[ and ]].
	value := line[2:end]

	// Advance the block reader to the end of the page link.
	block.Advance(end + 2)

	return &pageLink{
		Page: unescapePunctuation(value),
	}
}

// pageLinkClose returns the position of the ]] that closes the page link at
// the start of line, or 0 if it has none. A ] after a backslash does not close
// the link.
//
// Page links can be nested, as in Logseq, so "[[a [[b]]]]" is a link to the
// page "a [[b]]". The [[ and ]] in the title are counted. If they do not
// balance, the first ]] closes the link, so "[[a [[b]]" is a link to "a [[b".
func pageLinkClose(line []byte) int {
	depth := 1
	first := 0
	for i := 2; i < len(line)-1; i++ {
		if line[i] == '[' && line[i+1] == '[' {
			depth++
			i++
		} else if line[i] == ']' && line[i+1] == ']' && line[i-1] != '\\' {
			if first == 0 {
				first = i
			}

			depth--
			if depth == 0 {
				return i
			}

			i++
		}
	}

	return first
}

// wholePageLink returns the title of the page if all of text is one page
// link, such as "[[a, b]]". Text such as "[[a]], [[b]]" is not one page link.
func wholePageLink(text string) (string, bool) {
	if !strings.HasPrefix(text, "[[") {
		return "", false
	}

	end := pageLinkClose([]byte(text))
	if end == 0 || end+2 != len(text) {
		return "", false
	}

	return unescapePunctuation([]byte(text[2:end])), true
}
