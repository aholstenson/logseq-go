package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
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

func isUUIDChar(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || c == '-'
}
