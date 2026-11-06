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

	// Variant is the type of the command as written, such as `SRC` or
	// `src`. Logseq reads the type without regard for case.
	Variant string

	// Parameters is the text after the type on the BEGIN line, such as the
	// language of a source block.
	Parameters string

	// LowerCase is whether the BEGIN line is written in lower case, as in
	// `#+begin_src`.
	LowerCase bool

	// options are the parse options of the page, which are used to read the
	// text of a command that Logseq reads as Markdown.
	options *parseOptions
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

// beginPrefix and endPrefix start the lines around an org-mode block. Logseq
// reads them without regard for case.
var (
	beginPrefix = []byte("#+begin_")
	endPrefix   = []byte("#+end_")
)

// hasPrefixFold checks if the text starts with the prefix, without regard for
// case.
func hasPrefixFold(text []byte, prefix []byte) bool {
	return len(text) >= len(prefix) && bytes.EqualFold(text[:len(prefix)], prefix)
}

// Open looks for the start of the org-mode block, which is a line starting
// with `#+BEGIN_` and the type of the block, such as `#+BEGIN_SRC`. The text
// after the type, such as the language of a source block, is kept as the
// parameters of the block.
func (b *beginEndParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	indent := pos

	if !hasPrefixFold(line[pos:], beginPrefix) {
		return nil, parser.NoChildren
	}

	// Fetch the type of the block
	start := pos + len(beginPrefix)
	i := start
	for ; i < len(line); i++ {
		if util.IsSpace(line[i]) {
			break
		}
	}

	blockType := string(line[start:i])
	if blockType == "" {
		return nil, parser.NoChildren
	}

	options, _ := pc.Get(parseOptionsKey).(*parseOptions)
	node := &beginEnd{
		Variant:    blockType,
		Parameters: string(util.TrimRightSpace(util.TrimLeftSpace(line[i:]))),
		LowerCase:  bytes.Equal(line[pos:pos+len(beginPrefix)], beginPrefix),
		options:    options,
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

	// Check if the line starts with #+END_ and the type, without regard for
	// case
	searchFor := []byte(string(endPrefix) + data.keyword)
	if hasPrefixFold(line[pos:], searchFor) && util.IsBlank(line[pos+len(searchFor):]) {
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
