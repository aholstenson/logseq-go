package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// fencedCodeBlockParser is the fenced code block parser of Goldmark, changed
// to read fences as Logseq does:
//
//   - A fence can be indented four columns or more further than the text of
//     its block, such as the fence in "- a\n      ```js\n      x()\n      ```".
//     In CommonMark such a line is an indented code block, or more text of the
//     paragraph before it.
//   - The code block ends at the next fence line, at any indentation. In
//     CommonMark a closing fence must be indented less than four columns.
//   - A fence that no later fence line in the same block closes is text, see
//     hasClosingLine. In CommonMark the code block ends at the end of the
//     block.
type fencedCodeBlockParser struct {
	parser.BlockParser
}

func newFencedCodeBlockParser() *fencedCodeBlockParser {
	return &fencedCodeBlockParser{
		BlockParser: parser.NewFencedCodeBlockParser(),
	}
}

// openFence is the opening fence of the code block that is being parsed.
type openFence struct {
	node   ast.Node
	char   byte
	length int
}

var openFenceKey = parser.NewContextKey()

// codeFenceAttribute is the name of the attribute that holds the characters
// of the opening fence of a code block, when they are not "```".
const codeFenceAttribute = "logseq-code-fence"

// codeInfoSourceAttribute is the name of the attribute that holds the text
// after the opening fence of a code block, as it is in the file.
const codeInfoSourceAttribute = "logseq-code-info"

// codeIndentationAttribute is the name of the attribute that holds the spaces
// before the opening fence of a code block.
const codeIndentationAttribute = "logseq-code-indentation"

// codeSourceAttribute is the name of the attribute that holds the lines of the
// code of a code block as they are in the file, in a *bytes.Buffer.
var codeSourceAttribute = []byte("logseq-code-source")

// closingFenceAttribute is the name of the attribute that holds the line of
// the closing fence of a code block as it is in the file.
const closingFenceAttribute = "logseq-closing-fence"

func (p *fencedCodeBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || pos >= len(line) || (line[pos] != '`' && line[pos] != '~') {
		return nil, parser.NoChildren
	}

	length := 0
	for i := pos; i < len(line) && line[i] == line[pos]; i++ {
		length++
	}

	fence := &openFence{
		char:   line[pos],
		length: length,
	}
	kind := fmt.Sprintf("fence:%c:%d", fence.char, fence.length)
	closed := hasClosingLine(reader, pc, kind, func(line []byte) bool {
		return closesFence(line, fence)
	})
	if !closed {
		return nil, parser.NoChildren
	}

	node, state := p.BlockParser.Open(parent, reader, pc)
	if node == nil {
		return node, state
	}

	fence.node = node
	pc.Set(openFenceKey, fence)

	// The fence and the text after it are kept, so that the code block is
	// written back with them unchanged.
	if marker := string(line[pos : pos+length]); marker != "```" {
		node.SetAttributeString(codeFenceAttribute, marker)
	}
	info := strings.TrimRight(string(line[pos+length:]), "\r\n")
	node.SetAttributeString(codeInfoSourceAttribute, info)
	node.SetAttributeString(codeIndentationAttribute, string(line[:pos]))
	node.SetAttribute(codeSourceAttribute, &bytes.Buffer{})

	return node, state
}

func (p *fencedCodeBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	if fence, ok := pc.Get(openFenceKey).(*openFence); ok && fence.node == node {
		line, segment := reader.PeekLine()
		if closesFence(line, fence) {
			node.SetAttributeString(closingFenceAttribute, strings.TrimRight(string(line), "\r\n"))
			newline := 0
			if len(line) > 0 && line[len(line)-1] == '\n' {
				newline = 1
			}
			reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
			return parser.Close
		}

		// The line is kept as it is, before Goldmark removes the spaces of
		// the indentation of the fence from it.
		if value, ok := node.Attribute(codeSourceAttribute); ok {
			if buf, ok := value.(*bytes.Buffer); ok {
				buf.Write(line)
			}
		}
	}

	return p.BlockParser.Continue(node, reader, pc)
}

func (p *fencedCodeBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	if fence, ok := pc.Get(openFenceKey).(*openFence); ok && fence.node == node {
		pc.Set(openFenceKey, nil)
	}

	p.BlockParser.Close(node, reader, pc)
}

func (p *fencedCodeBlockParser) CanAcceptIndentedLine() bool {
	return true
}

// closesFence checks if a line is a fence that closes the code block, at any
// indentation.
func closesFence(line []byte, fence *openFence) bool {
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
