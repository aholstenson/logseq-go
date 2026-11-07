package markdown

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// lineIndentationKey is where the parser keeps the lineIndentation of the
// paragraphs that it closes.
var lineIndentationKey = parser.NewContextKey()

// lineIndentationAttribute is the name of the document attribute that holds
// the lineIndentation of a parsed document.
var lineIndentationAttribute = []byte("logseqLineIndentation")

// lineIndentation holds the spaces and tabs at the start of the continuation
// lines of paragraphs, such as the two spaces before `b` in "a\n  b". Only
// the indentation after the indentation of the block or quote that holds the
// paragraph is kept. The key is the offset in the source of the start of the
// line.
//
// Goldmark removes these spaces from the lines of a paragraph. Logseq keeps
// them as text, so they are written back to not change the line on save.
type lineIndentation map[int]string

// paragraphParser is the paragraph parser of Goldmark, changed to record the
// indentation of continuation lines, see lineIndentation.
type paragraphParser struct {
	parser.BlockParser
}

func newParagraphParser() *paragraphParser {
	return &paragraphParser{
		BlockParser: parser.NewParagraphParser(),
	}
}

func (p *paragraphParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	recordLineIndentation(node, reader, pc, "")
	p.BlockParser.Close(node, reader, pc)
}

// recordLineIndentation records the indentation of the continuation lines of
// a paragraph before the paragraph parser removes it. The base indentation is
// written for each line of the paragraph, so it is not recorded. A line with
// less indentation than the base is recorded as having no indentation.
func recordLineIndentation(node ast.Node, reader text.Reader, pc parser.Context, base string) {
	src := reader.Source()
	lines := node.Lines()

	// The first line has no indentation, as the paragraph starts after it.
	for i := 1; i < lines.Len(); i++ {
		line := lines.At(i)
		trimmed := line.TrimLeftSpace(src)
		if line.Padding == 0 && trimmed.Start == line.Start {
			continue
		}

		value, ok := strings.CutPrefix(strings.Repeat(" ", line.Padding)+string(src[line.Start:trimmed.Start]), base)
		if !ok || value == "" {
			continue
		}

		indentation, _ := pc.Get(lineIndentationKey).(lineIndentation)
		if indentation == nil {
			indentation = lineIndentation{}
			pc.Set(lineIndentationKey, indentation)
		}

		lineStart := bytes.LastIndexByte(src[:line.Start], '\n') + 1
		indentation[lineStart] = value
	}
}

// nextLineIndentationOf returns the indentation of the line after the line
// break of a text, see lineIndentation.
func nextLineIndentationOf(src []byte, node *ast.Text) string {
	if !node.SoftLineBreak() && !node.HardLineBreak() {
		return ""
	}

	var doc ast.Node = node
	for doc.Parent() != nil {
		doc = doc.Parent()
	}

	value, ok := doc.Attribute(lineIndentationAttribute)
	if !ok {
		return ""
	}

	indentation, _ := value.(lineIndentation)
	i := bytes.IndexByte(src[node.Segment.Stop:], '\n')
	if i < 0 {
		return ""
	}

	return indentation[node.Segment.Stop+i+1]
}

var _ parser.BlockParser = (*paragraphParser)(nil)
