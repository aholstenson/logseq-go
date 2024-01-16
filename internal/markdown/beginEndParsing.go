package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var beginEndKind = ast.NewNodeKind("BeginEnd")

type beginEnd struct {
	ast.BaseBlock

	Variant string
}

func (n *beginEnd) Kind() ast.NodeKind {
	return beginEndKind
}

func (n *beginEnd) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type beginEndBlockData struct {
	indent  int
	node    ast.Node
	keyword string
}

var beginEndInfoKey = parser.NewContextKey()

type beginEndParser struct {
}

func (b *beginEndParser) Trigger() []byte {
	return []byte{'#'}
}

// Open looks for the start of the org-mode block, which is a line starting with #+BEGIN_
func (b *beginEndParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	indent := pos

	// Check if the line starts with #+BEGIN_
	if !bytes.HasPrefix(line[pos:], []byte("#+BEGIN_")) {
		return nil, parser.NoChildren
	}

	// Fetch the type of the block
	blockType := ""
	i := pos + 8
	for ; i < len(line); i++ {
		if line[i] == ' ' || line[i] == '\t' || line[i] == '\n' {
			break
		}
		blockType += string(line[i])
	}

	// Make sure that the rest of the line is empty
	if !util.IsBlank(line[i:]) {
		return nil, parser.NoChildren
	}

	node := &beginEnd{
		Variant: blockType,
	}
	pc.Set(beginEndInfoKey, &beginEndBlockData{
		indent:  indent,
		node:    node,
		keyword: blockType,
	})
	return node, parser.NoChildren

}

func (b *beginEndParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	data := pc.Get(beginEndInfoKey).(*beginEndBlockData)

	_, pos := util.IndentWidth(line, reader.LineOffset())

	// Check if the line starts with #+END_ and the type
	searchFor := []byte("#+END_" + data.keyword)
	if bytes.HasPrefix(line[pos:], searchFor) && util.IsBlank(line[pos+len(searchFor):]) {
		newline := 1
		if line[len(line)-1] != '\n' {
			newline = 0
		}
		reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
		return parser.Close
	}

	pos, padding := util.IndentPositionPadding(line, reader.LineOffset(), segment.Padding, data.indent)
	if pos < 0 {
		pos = util.FirstNonSpacePosition(line)
		if pos < 0 {
			pos = 0
		}
		padding = 0
	}

	seg := text.NewSegmentPadding(segment.Start+pos, segment.Stop, padding)
	node.Lines().Append(seg)
	reader.AdvanceAndSetPadding(segment.Stop-segment.Start-pos-1, padding)
	return parser.Continue | parser.NoChildren
}

func (b *beginEndParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	data := pc.Get(beginEndInfoKey).(*beginEndBlockData)
	if data.node == node {
		pc.Set(beginEndInfoKey, nil)
	}
}

func (b *beginEndParser) CanInterruptParagraph() bool {
	return true
}

func (b *beginEndParser) CanAcceptIndentedLine() bool {
	return false
}
