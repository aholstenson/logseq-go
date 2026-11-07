package markdown

import (
	"bytes"
	"strings"

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

	// beginLine and endLine are the BEGIN and END lines as they are in the
	// file, see content.BeginEndSourceAware.
	beginLine string
	endLine   string

	// indent is the position of the BEGIN line in the line that holds it,
	// which is the indentation that is taken from the lines inside.
	indent int
}

func (n *beginEnd) Kind() ast.NodeKind {
	return beginEndKind
}

func (n *beginEnd) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

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
// parameters of the block. The line is only read as the start of the block if
// an END line of the same type closes it.
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

	// Logseq reads a BEGIN line that no END line in the same block closes
	// as text, see hasClosingLine.
	searchFor := []byte(string(endPrefix) + blockType)
	kind := "end:" + strings.ToLower(blockType)
	closed := hasClosingLine(reader, pc, kind, func(line []byte) bool {
		return hasPrefixFold(line, searchFor) && util.IsBlank(line[len(searchFor):])
	})
	if !closed {
		return nil, parser.NoChildren
	}

	options, _ := pc.Get(parseOptionsKey).(*parseOptions)
	node := &beginEnd{
		Variant:    blockType,
		Parameters: string(util.TrimRightSpace(util.TrimLeftSpace(line[i:]))),
		LowerCase:  bytes.Equal(line[pos:pos+len(beginPrefix)], beginPrefix),
		options:    options,
		indent:     indent,
		beginLine:  strings.TrimRight(string(line[pos:]), "\r\n"),
	}
	return node, parser.NoChildren
}

func (b *beginEndParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	block := node.(*beginEnd)

	_, pos := util.IndentWidth(line, reader.LineOffset())

	// Check if the line starts with #+END_ and the type, without regard for
	// case
	searchFor := []byte(string(endPrefix) + block.Variant)
	if hasPrefixFold(line[pos:], searchFor) && util.IsBlank(line[pos+len(searchFor):]) {
		block.endLine = strings.TrimRight(string(line), "\r\n")
		newline := 1
		if line[len(line)-1] != '\n' {
			newline = 0
		}
		reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
		return parser.Close
	}

	appendContentLine(node, reader, block.indent)
	return parser.Continue | parser.NoChildren
}

func (b *beginEndParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (b *beginEndParser) CanInterruptParagraph() bool {
	return true
}

func (b *beginEndParser) CanAcceptIndentedLine() bool {
	return false
}
