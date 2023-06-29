package markdown

import (
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

	end := 0
	var value []byte

	// Scan until the closing ]] while also dealing with escaped ] characters.
	for i := 2; i < len(line)-1; i++ {
		if line[i] == ']' && line[i+1] == ']' {
			// If the previous character was a \, then this is an escaped ] and
			// we should continue scanning.
			if line[i-1] == '\\' {
				continue
			}

			end = i
			break
		}
	}

	// Didn't find the closing ]], so this isn't a page link.
	if end == 0 {
		return nil
	}

	// The value of the page link is the text between the [[ and ]].
	value = line[2:end]

	// Advance the block reader to the end of the page link.
	block.Advance(end + 2)

	return &pageLink{
		Page: unescapeString(value),
	}
}
