package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var drawerKind = ast.NewNodeKind("Drawer")

// drawer is a drawer that is not a logbook or properties, see drawerParser.
type drawer struct {
	ast.BaseBlock

	// Name is the name between the colons of the first line.
	Name string

	// EndLine is the last line, without the indentation of the first line
	// and without its line break.
	EndLine string
}

func (n *drawer) Kind() ast.NodeKind {
	return drawerKind
}

func (n *drawer) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type drawerBlockData struct {
	indent int
	node   ast.Node
}

var drawerInfoKey = parser.NewContextKey()

// drawerStartRegexp matches the first line of a drawer, which is a name
// between colons and nothing after it, not even a space. The name has no
// colon and no space.
var drawerStartRegexp = regexp.MustCompile(`^:([^: \r\n]+):\r?\n?$`)

// drawerParser parses a drawer that the logbook and property drawer parsers do
// not take, such as:
//
//	:notes:
//	Some text with [[a link]]
//	:end:
//
// Logseq keeps the content of such a drawer as text, so it has no references.
// The drawer ends at a line that is `:end:` in any case, with spaces around it
// or not. As for a logbook, the end must come before the block ends, see
// hasLogbookEnd. Otherwise the lines are text.
type drawerParser struct {
}

func (p *drawerParser) Trigger() []byte {
	return []byte{':'}
}

func (p *drawerParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	match := drawerStartRegexp.FindSubmatch(line[pos:])
	if match == nil {
		return nil, parser.NoChildren
	}

	if !hasDrawerEnd(reader.Source()[segment.Stop:], pos) {
		return nil, parser.NoChildren
	}

	node := &drawer{Name: string(match[1])}
	pc.Set(drawerInfoKey, &drawerBlockData{indent: pos, node: node})

	reader.Advance(segment.Len() - 1)
	return node, parser.NoChildren
}

// hasDrawerEnd checks if an end line comes after the first line of a drawer
// before the block that holds the drawer ends. The source starts at the line
// after the first line, and indent is the number of characters before the
// first line. The block ends as for a logbook, see hasLogbookEnd.
func hasDrawerEnd(source []byte, indent int) bool {
	for len(source) > 0 {
		line := source
		if i := bytes.IndexByte(source, '\n'); i >= 0 {
			line = source[:i+1]
		}
		source = source[len(line):]

		trimmed := bytes.TrimLeft(line, " \t")
		if util.IsBlank(trimmed) {
			continue
		}

		if len(line)-len(trimmed) < indent || isBulletLine(trimmed) {
			return false
		}

		if isDrawerEnd(trimmed) {
			return true
		}
	}

	return false
}

// isDrawerEnd checks if a line ends a drawer: `:end:` in any case, with only
// spaces around it.
func isDrawerEnd(line []byte) bool {
	return strings.EqualFold(strings.TrimSpace(string(line)), ":end:")
}

func (p *drawerParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	data := pc.Get(drawerInfoKey).(*drawerBlockData)

	_, pos := util.IndentWidth(line, reader.LineOffset())
	if isDrawerEnd(line[pos:]) {
		// The spaces before the end line that are more than the indentation
		// of the first line are kept.
		start := pos
		if relative, _ := util.IndentPositionPadding(line, reader.LineOffset(), segment.Padding, data.indent); relative >= 0 && relative < pos {
			start = relative
		}
		node.(*drawer).EndLine = strings.TrimRight(string(line[start:]), "\r\n")

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

	node.Lines().Append(text.NewSegmentPadding(segment.Start+pos, segment.Stop, padding))
	reader.AdvanceAndSetPadding(segment.Stop-segment.Start-pos-1, padding)
	return parser.Continue | parser.NoChildren
}

func (p *drawerParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	if data, ok := pc.Get(drawerInfoKey).(*drawerBlockData); ok && data.node == node {
		pc.Set(drawerInfoKey, nil)
	}
}

func (p *drawerParser) CanInterruptParagraph() bool {
	return true
}

func (p *drawerParser) CanAcceptIndentedLine() bool {
	return false
}

var _ parser.BlockParser = (*drawerParser)(nil)
