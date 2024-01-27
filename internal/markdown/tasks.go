package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var logbookKind = ast.NewNodeKind("Logbook")

type logbook struct {
	ast.BaseBlock
}

func (n *logbook) Kind() ast.NodeKind {
	return logbookKind
}

func (n *logbook) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type logbookBlockData struct {
	indent int
	node   ast.Node
}

var logbookInfoKey = parser.NewContextKey()

type logbookParser struct {
}

func (b *logbookParser) Trigger() []byte {
	return []byte{':'}
}

// Open looks for :LOGBOOK: at the beginning of a line
func (b *logbookParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	indent := pos

	// Check if the line starts with :LOGBOOK:
	if !bytes.HasPrefix(line[pos:], []byte(":LOGBOOK:")) {
		return nil, parser.NoChildren
	}

	// Create the block
	block := &logbook{}

	// Store the block in the context
	pc.Set(logbookInfoKey, &logbookBlockData{
		indent: indent,
		node:   block,
	})

	return block, parser.NoChildren
}

func (b *logbookParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	data := pc.Get(logbookInfoKey).(*logbookBlockData)

	_, pos := util.IndentWidth(line, reader.LineOffset())

	// Check if the line starts with :END:
	searchFor := []byte(":END:")
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

func (b *logbookParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	data := pc.Get(logbookInfoKey).(*logbookBlockData)
	if data.node == node {
		pc.Set(beginEndInfoKey, nil)
	}
}

func (b *logbookParser) CanInterruptParagraph() bool {
	return true
}

func (b *logbookParser) CanAcceptIndentedLine() bool {
	return false
}
