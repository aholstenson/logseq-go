package markdown

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var mathKind = ast.NewNodeKind("Math")

type math struct {
	ast.BaseInline

	// Value is the LaTeX of the formula.
	Value string

	// Displayed is whether the formula was written with two dollar signs, or
	// between `\[` and `\]`.
	Displayed bool

	// Backslashes is whether the formula was written between `\(` and `\)`,
	// or between `\[` and `\]`.
	Backslashes bool
}

func (*math) Kind() ast.NodeKind {
	return mathKind
}

func (n *math) Dump(src []byte, level int) {
}

// mathParser parses the LaTeX that Logseq renders with KaTeX, which is a
// formula between dollar signs. Two dollar signs set the formula apart from
// the text around it, one keeps it within the line. A formula can also be
// written between `\(` and `\)`, which keeps it within the line, or between
// `\[` and `\]`, which sets it apart.
type mathParser struct {
}

func (m *mathParser) Trigger() []byte {
	return []byte{'$', '\\'}
}

func (m *mathParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) > 0 && line[0] == '\\' {
		return parseBackslashMath(block)
	}

	value, length, displayed, ok := mathAt(line)
	if !ok {
		return nil
	}

	block.Advance(length)
	return &math{
		Value:     string(value),
		Displayed: displayed,
	}
}

var _ parser.InlineParser = (*mathParser)(nil)

// parseBackslashMath reads a formula between `\(` and `\)`, or between `\[`
// and `\]`. The formula ends at the first closing delimiter, and it can go on
// over more than one line of a paragraph, in the same way as in Logseq.
func parseBackslashMath(block text.Reader) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 2 || (line[1] != '(' && line[1] != '[') {
		return nil
	}

	displayed := line[1] == '['
	closing := []byte(`\)`)
	if displayed {
		closing = []byte(`\]`)
	}

	savedLine, savedPosition := block.Position()
	value := make([]byte, 0)
	offset := 2
	for {
		if i := bytes.Index(line[offset:], closing); i >= 0 {
			value = append(value, line[offset:offset+i]...)
			block.Advance(offset + i + len(closing))
			return &math{
				Value:       string(value),
				Displayed:   displayed,
				Backslashes: true,
			}
		}

		value = append(value, line[offset:]...)
		block.AdvanceLine()
		line, _ = block.PeekLine()
		if line == nil {
			// The formula is not closed, so the delimiter is text.
			block.SetPosition(savedLine, savedPosition)
			return nil
		}

		offset = 0
	}
}

// mathAt reads the formula that starts at the beginning of the given text,
// returning its LaTeX, how much of the text it takes up and whether it was
// written with two dollar signs.
//
// A formula ends with the line it starts on. The rules for a formula between
// single dollar signs keep the amounts in a sentence such as `$5 and $10` out
// of it: the opening dollar sign is followed by something other than a space,
// the closing one comes after something other than a space, and a digit never
// follows the closing one.
func mathAt(line []byte) (value []byte, length int, displayed bool, ok bool) {
	if end := lineEnd(line); end < len(line) {
		line = line[:end]
	}

	if bytes.HasPrefix(line, []byte("$$")) {
		closing := bytes.Index(line[2:], []byte("$$"))
		if closing <= 0 {
			return nil, 0, false, false
		}

		return line[2 : 2+closing], closing + 4, true, true
	}

	if len(line) < 3 || line[0] != '$' || isMathSpace(line[1]) {
		return nil, 0, false, false
	}

	for i := 2; i < len(line); i++ {
		if line[i] != '$' || isMathSpace(line[i-1]) {
			continue
		}

		if i+1 < len(line) && util.IsNumeric(line[i+1]) {
			continue
		}

		return line[1:i], i + 1, false, true
	}

	return nil, 0, false, false
}

// lineEnd finds where the first line of the given text ends, as a formula does
// not carry on past the line it starts on.
func lineEnd(line []byte) int {
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		return i
	}

	return len(line)
}

func isMathSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

var mathBlockKind = ast.NewNodeKind("MathBlock")

type mathBlock struct {
	ast.BaseBlock

	// indent is how far the line that opened the block was indented, which the
	// lines of the formula have taken off them.
	indent int

	// environment is the name of the LaTeX environment of the block, or empty
	// for a block between two lines of `$$`.
	environment string
}

func (*mathBlock) Kind() ast.NodeKind {
	return mathBlockKind
}

func (n *mathBlock) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// mathBlockParser parses the math block of Logseq, which is a formula written
// between two lines that hold nothing but two dollar signs:
//
//	$$
//	E = mc^2
//	$$
//
// A formula written entirely on one line is left to the inline parser, so that
// it is written back out the way it was found.
type mathBlockParser struct {
}

func (m *mathBlockParser) Trigger() []byte {
	return []byte{'$'}
}

func (m *mathBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || !isMathBlockMarker(line[pos:]) {
		return nil, parser.NoChildren
	}

	return &mathBlock{indent: pos}, parser.NoChildren
}

func (m *mathBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()

	_, pos := util.IndentWidth(line, reader.LineOffset())
	if isMathBlockMarker(line[pos:]) {
		newline := 1
		if line[len(line)-1] != '\n' {
			newline = 0
		}

		reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
		return parser.Close
	}

	pos, padding := util.IndentPositionPadding(line, reader.LineOffset(), segment.Padding, node.(*mathBlock).indent)
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

func (m *mathBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (m *mathBlockParser) CanInterruptParagraph() bool {
	return true
}

func (m *mathBlockParser) CanAcceptIndentedLine() bool {
	return false
}

// isMathBlockMarker checks if the rest of a line marks the start or the end of
// a math block, which is two dollar signs on their own.
func isMathBlockMarker(line []byte) bool {
	return bytes.HasPrefix(line, []byte("$$")) && util.IsBlank(line[2:])
}

var _ parser.BlockParser = (*mathBlockParser)(nil)

// latexEnvironmentBeginRegexp matches the line that starts a LaTeX
// environment, and finds the name of the environment.
var latexEnvironmentBeginRegexp = regexp.MustCompile(`^\\begin\{([^{}\s]+)\}[ \t]*\r?\n?$`)

// latexEnvironmentParser parses a LaTeX environment, which Logseq renders in
// the same way as a math block:
//
//	\begin{equation}
//	E = mc^2
//	\end{equation}
//
// The lines that start and end the environment hold nothing else. An
// environment that does not end before the next bullet is text, so that it
// does not take in the blocks after it.
type latexEnvironmentParser struct {
}

func (m *latexEnvironmentParser) Trigger() []byte {
	return []byte{'\\'}
}

func (m *latexEnvironmentParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	match := latexEnvironmentBeginRegexp.FindSubmatch(line[pos:])
	if match == nil {
		return nil, parser.NoChildren
	}

	environment := string(match[1])
	if !hasLatexEnvironmentEnd(reader.Source()[segment.Stop:], environment) {
		return nil, parser.NoChildren
	}

	return &mathBlock{indent: pos, environment: environment}, parser.NoChildren
}

func (m *latexEnvironmentParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	environment := node.(*mathBlock).environment

	_, pos := util.IndentWidth(line, reader.LineOffset())
	if isLatexEnvironmentEnd(line[pos:], environment) {
		newline := 1
		if line[len(line)-1] != '\n' {
			newline = 0
		}

		reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
		return parser.Close
	}

	pos, padding := util.IndentPositionPadding(line, reader.LineOffset(), segment.Padding, node.(*mathBlock).indent)
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

func (m *latexEnvironmentParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (m *latexEnvironmentParser) CanInterruptParagraph() bool {
	return true
}

func (m *latexEnvironmentParser) CanAcceptIndentedLine() bool {
	return false
}

// hasLatexEnvironmentEnd checks if the source after the line that starts an
// environment has the line that ends it, before the next bullet.
func hasLatexEnvironmentEnd(source []byte, environment string) bool {
	for len(source) > 0 {
		line := source
		if i := bytes.IndexByte(source, '\n'); i >= 0 {
			line = source[:i+1]
		}
		source = source[len(line):]

		trimmed := bytes.TrimLeft(line, " \t")
		if isLatexEnvironmentEnd(trimmed, environment) {
			return true
		}

		if isBulletLine(trimmed) {
			return false
		}
	}

	return false
}

// isLatexEnvironmentEnd checks if the rest of a line ends the environment with
// the given name.
func isLatexEnvironmentEnd(line []byte, environment string) bool {
	end := []byte(`\end{` + environment + `}`)
	return bytes.HasPrefix(line, end) && util.IsBlank(line[len(end):])
}

// isBulletLine checks if a line, without its indentation, starts a block.
func isBulletLine(line []byte) bool {
	line = bytes.TrimRight(line, "\r\n")
	return bytes.Equal(line, []byte("-")) || bytes.HasPrefix(line, []byte("- ")) || bytes.HasPrefix(line, []byte("-\t"))
}

var _ parser.BlockParser = (*latexEnvironmentParser)(nil)
