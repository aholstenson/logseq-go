package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// fencedCodeBlockParser is the fenced code block parser of Goldmark, changed
// so that a fence can be indented four columns or more further than the text
// of its block, such as the fence in "- a\n      ```js\n      x()\n      ```".
//
// In CommonMark such a line is an indented code block, or more text of the
// paragraph before it. Logseq reads a fence at any indentation as the start of
// a code block, and ends the code block at the next fence line, at any
// indentation.
//
// Fences that are indented less than four columns are read as in CommonMark.
type fencedCodeBlockParser struct {
	parser.BlockParser
}

func newFencedCodeBlockParser() *fencedCodeBlockParser {
	return &fencedCodeBlockParser{
		BlockParser: parser.NewFencedCodeBlockParser(),
	}
}

// deepFence is a code block whose opening fence is indented four columns or
// more.
type deepFence struct {
	node   ast.Node
	char   byte
	length int
}

var deepFenceKey = parser.NewContextKey()

func (p *fencedCodeBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos, indent := pc.BlockOffset(), pc.BlockIndent()

	node, state := p.BlockParser.Open(parent, reader, pc)
	if node == nil || indent < 4 {
		return node, state
	}

	length := 0
	for i := pos; i < len(line) && line[i] == line[pos]; i++ {
		length++
	}

	pc.Set(deepFenceKey, &deepFence{
		node:   node,
		char:   line[pos],
		length: length,
	})
	return node, state
}

func (p *fencedCodeBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	if fence, ok := pc.Get(deepFenceKey).(*deepFence); ok && fence.node == node {
		line, segment := reader.PeekLine()
		if closesDeepFence(line, fence) {
			newline := 0
			if len(line) > 0 && line[len(line)-1] == '\n' {
				newline = 1
			}
			reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
			return parser.Close
		}
	}

	return p.BlockParser.Continue(node, reader, pc)
}

func (p *fencedCodeBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	if fence, ok := pc.Get(deepFenceKey).(*deepFence); ok && fence.node == node {
		pc.Set(deepFenceKey, nil)
	}

	p.BlockParser.Close(node, reader, pc)
}

func (p *fencedCodeBlockParser) CanAcceptIndentedLine() bool {
	return true
}

// closesDeepFence checks if a line is a fence that closes the code block, at
// any indentation.
func closesDeepFence(line []byte, fence *deepFence) bool {
	pos := util.FirstNonSpacePosition(line)
	if pos < 0 {
		return false
	}

	i := pos
	for ; i < len(line) && line[i] == fence.char; i++ {
	}

	return i-pos >= fence.length && util.IsBlank(line[i:])
}

var _ parser.BlockParser = (*fencedCodeBlockParser)(nil)
