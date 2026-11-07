package markdown

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// drawerPropertyRegexp matches a `:name: value` line in a property drawer. The
// value is optional.
var drawerPropertyRegexp = regexp.MustCompile(`^:([^\s:]+):(?:[ \t]+([^\r\n]*?))?[ \t]*\r?\n?$`)

// propertyDrawerParser parses a property drawer, which is the form of the
// properties of a block that Logseq wrote before it used `name:: value`
// lines, and that it still reads:
//
//	:PROPERTIES:
//	:id: 6500e8a3-0000-4000-8000-000000000000
//	:END:
//
// The markers are read in any case. A drawer must end with `:END:` and hold
// only `:name: value` lines, otherwise its lines are text.
type propertyDrawerParser struct {
}

func (p *propertyDrawerParser) Trigger() []byte {
	return []byte{':'}
}

func (p *propertyDrawerParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || !isDrawerMarker(line[pos:], ":PROPERTIES:") {
		return nil, parser.NoChildren
	}

	if !hasPropertyDrawerEnd(reader.Source()[segment.Stop:]) {
		return nil, parser.NoChildren
	}

	lowerCase := bytes.HasPrefix(line[pos:], []byte(":properties:"))
	return &properties{drawer: true, drawerLowerCase: lowerCase}, parser.NoChildren
}

func (p *propertyDrawerParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	_, pos := util.IndentWidth(line, reader.LineOffset())

	if isDrawerMarker(line[pos:], ":END:") {
		newline := 1
		if line[len(line)-1] != '\n' {
			newline = 0
		}

		reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
		return parser.Close
	}

	match := drawerPropertyRegexp.FindSubmatchIndex(line[pos:])
	if match == nil {
		// The drawer was checked to only hold property lines when it was
		// opened, so this does not happen.
		return parser.Close
	}

	property := &property{Name: string(line[pos+match[2] : pos+match[3]])}
	lineEnd := len(bytes.TrimRight(line, "\r\n"))
	if match[4] >= 0 && match[5] > match[4] {
		start := segment.Start + pos + match[4]
		property.Lines().Append(text.NewSegment(start, start+match[5]-match[4]))

		// The spaces around the value are kept, so that the line is
		// written back as it is. One space after the name is written
		// anyway.
		if space := line[pos+match[3]+1 : pos+match[4]]; len(space) > 1 && space[0] == ' ' {
			property.SpaceBeforeValue = string(space[1:])
		}
		if pos+match[5] < lineEnd {
			property.TrailingSpace = string(line[pos+match[5] : lineEnd])
		}
	} else if after := pos + match[3] + 1; after < lineEnd {
		property.TrailingSpace = string(line[after:lineEnd])
	}

	node.AppendChild(node, property)

	reader.Advance(segment.Len() - 1)
	return parser.Continue | parser.NoChildren
}

func (p *propertyDrawerParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (p *propertyDrawerParser) CanInterruptParagraph() bool {
	return true
}

func (p *propertyDrawerParser) CanAcceptIndentedLine() bool {
	return false
}

// hasPropertyDrawerEnd checks if the lines after `:PROPERTIES:` are property
// lines followed by `:END:`.
func hasPropertyDrawerEnd(source []byte) bool {
	for len(source) > 0 {
		line := source
		if i := bytes.IndexByte(source, '\n'); i >= 0 {
			line = source[:i+1]
		}
		source = source[len(line):]

		trimmed := bytes.TrimLeft(line, " \t")
		if isDrawerMarker(trimmed, ":END:") {
			return true
		}

		if !drawerPropertyRegexp.Match(trimmed) {
			return false
		}
	}

	return false
}

// isDrawerMarker checks if the rest of a line is the given marker of a drawer,
// in any case, with nothing after it.
func isDrawerMarker(line []byte, marker string) bool {
	return len(line) >= len(marker) &&
		bytes.EqualFold(line[:len(marker)], []byte(marker)) &&
		util.IsBlank(line[len(marker):])
}

var _ parser.BlockParser = (*propertyDrawerParser)(nil)
