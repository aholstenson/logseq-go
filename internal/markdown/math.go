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
		if !closesInParagraph(parent, block, pc) {
			return nil
		}

		return parseBackslashMath(block)
	}

	value, length, displayed, ok := m.formulaAt(block, pc)
	if !ok {
		return nil
	}

	block.Advance(length)
	return &math{
		Value:     string(value),
		Displayed: displayed,
	}
}

// mathScannerKey is where the mathScanner of the current parse is kept.
var mathScannerKey = parser.NewContextKey()

// formulaAt reads the formula that starts at the position of the reader, see
// mathScanner.at. All formulas of a parse use the same mathScanner, which is
// kept in the parse context.
func (m *mathParser) formulaAt(block text.Reader, pc parser.Context) ([]byte, int, bool, bool) {
	line, segment := block.PeekLine()
	end := len(line)
	for end > 0 && (line[end-1] == '\n' || line[end-1] == '\r') {
		end--
	}

	if segment.Padding != 0 {
		// The line does not match the source, so it is read on its own.
		return newMathScanner(line).at(0, end)
	}

	source := block.Source()
	scanner, _ := pc.Get(mathScannerKey).(*mathScanner)
	if scanner == nil || len(scanner.src) != len(source) || (len(source) > 0 && &scanner.src[0] != &source[0]) {
		scanner = newMathScanner(source)
		pc.Set(mathScannerKey, scanner)
	}

	return scanner.at(segment.Start, segment.Start+end)
}

var _ parser.InlineParser = (*mathParser)(nil)

// closesInParagraph checks if the `\)` or `\]` that closes a formula that
// starts with `\(` or `\[` at the position of the reader comes later in the
// paragraph. It returns true if this cannot be known, and the formula is then
// read to find out. See nextInSource for why this is checked first.
func closesInParagraph(parent ast.Node, block text.Reader, pc parser.Context) bool {
	line, segment := block.PeekLine()
	if len(line) < 2 || (line[1] != '(' && line[1] != '[') || segment.Padding != 0 {
		return true
	}

	lines := parent.Lines()
	if lines == nil || lines.Len() == 0 {
		return true
	}

	closing := `\)`
	if line[1] == '[' {
		closing = `\]`
	}

	end := lines.At(lines.Len() - 1).Stop
	at := nextInSource(block.Source(), pc, closing, segment.Start+2)
	return at >= 0 && at+len(closing) <= end
}

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

// mathScanner finds the formulas in a text. It remembers the last match of
// each search, so that looking for the end of a formula at each dollar sign
// of a long line reads the line about once instead of once for each dollar
// sign.
type mathScanner struct {
	src []byte

	lineEnds nextMatch
	doubles  nextMatch
	dollars  nextMatch
}

func newMathScanner(src []byte) *mathScanner {
	return &mathScanner{src: src}
}

// lineEnd finds where the line that holds the offset ends, as a formula does
// not carry on past the line it starts on.
func (s *mathScanner) lineEnd(offset int) int {
	end := s.lineEnds.find(offset, func(from int) int {
		if i := bytes.IndexByte(s.src[from:], '\n'); i >= 0 {
			return from + i
		}

		return -1
	})

	if end < 0 {
		return len(s.src)
	}

	return end
}

// at reads the formula that starts at the offset and ends before end,
// returning its LaTeX, how much of the text it takes up and whether it was
// written with two dollar signs.
//
// A formula between single dollar signs ends at the next dollar sign, the same
// as in Logseq, so `$5 [[Bob]] $x$` has the formula `x` and a page link. The
// rules keep the amounts in a sentence such as `$5 and $10` out of it: the
// opening dollar sign is followed by something other than a space, and the
// closing one comes after something other than a space. A digit never follows
// the closing one, so `US$5 and US$6` has no formula either. Logseq does not
// have this last rule.
func (s *mathScanner) at(offset int, end int) (value []byte, length int, displayed bool, ok bool) {
	line := s.src[offset:end]
	if bytes.HasPrefix(line, []byte("$$")) {
		closing := s.doubles.find(offset+2, func(from int) int {
			if i := bytes.Index(s.src[from:], []byte("$$")); i >= 0 {
				return from + i
			}

			return -1
		})

		if closing <= offset+2 || closing+2 > end {
			return nil, 0, false, false
		}

		return s.src[offset+2 : closing], closing + 2 - offset, true, true
	}

	if len(line) < 3 || line[0] != '$' || isMathSpace(line[1]) {
		return nil, 0, false, false
	}

	closing := s.dollars.find(offset+1, func(from int) int {
		if i := bytes.IndexByte(s.src[from:], '$'); i >= 0 {
			return from + i
		}

		return -1
	})

	if closing < offset+2 || closing >= end || isMathSpace(s.src[closing-1]) {
		return nil, 0, false, false
	}

	// A dollar sign at the end of the line closes the formula also when a
	// digit comes after the end, such as in the next cell of a table.
	if closing != end-1 && closing+1 < len(s.src) && util.IsNumeric(s.src[closing+1]) {
		return nil, 0, false, false
	}

	return s.src[offset+1 : closing], closing + 1 - offset, false, true
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

	// Logseq reads a `$$` line that is not closed as text, see
	// hasClosingLine.
	if !hasClosingLine(reader, pc, "math", isMathBlockMarker) {
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

	appendContentLine(node, reader, node.(*mathBlock).indent)
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

	appendContentLine(node, reader, node.(*mathBlock).indent)
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
	return bytes.Equal(line, []byte("-")) || bytes.HasPrefix(line, []byte("- ")) || bytes.HasPrefix(line, []byte("-\t")) || isHeadingBullet(line)
}

var _ parser.BlockParser = (*latexEnvironmentParser)(nil)
