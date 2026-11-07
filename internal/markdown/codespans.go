package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// codeSpanSourceAttribute is the name of the attribute that holds the text of
// a code span as it is in the file, with its backticks and spaces.
const codeSpanSourceAttribute = "logseq-code-span"

// codeSpanSourceParser is the code span parser of Goldmark, changed so that
// it keeps the text of each code span that is on one line. The code span is
// then written back with the same backticks and spaces, see
// content.CodeSpan.CodeSpanSource. Goldmark reads code spans with the rules
// of CommonMark, which remove a space at each end of the code and let a
// longer run of backticks hold a shorter one.
type codeSpanSourceParser struct {
	parser.InlineParser
}

func newCodeSpanSourceParser() *codeSpanSourceParser {
	return &codeSpanSourceParser{
		InlineParser: parser.NewCodeSpanParser(),
	}
}

func (p *codeSpanSourceParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	startLine, start := block.Position()
	result := p.InlineParser.Parse(parent, block, pc)
	if code, ok := result.(*ast.CodeSpan); ok {
		stopLine, stop := block.Position()
		source := block.Source()
		if startLine == stopLine {
			code.SetAttributeString(codeSpanSourceAttribute, string(source[start.Start:stop.Start]))
		} else if item := codeSpanBlockItem(parent); item != nil {
			if value, ok := blockRelativeCodeSpanSource(source, item, start.Start, stop.Start); ok {
				code.SetAttributeString(codeSpanSourceAttribute, value)
			}
		} else if lines := blockLinesOf(parent); lines != nil && stopLine < lines.Len() {
			// A code span over more than one line keeps its line breaks.
			// The lines are taken without the indentation and the markers
			// of the containers, which the writer writes again.
			var buf []byte
			for i := startLine; i <= stopLine; i++ {
				line := lines.At(i)
				from, to := line.Start, line.Stop
				if i == startLine {
					from = start.Start
				}
				if i == stopLine {
					to = stop.Start
				}
				if from > to {
					return result
				}

				buf = append(buf, source[from:to]...)
			}

			code.SetAttributeString(codeSpanSourceAttribute, string(buf))
		}
	}

	return result
}

// codeSpanBlockItem gets the list item of the block that holds an inline
// node, if only paragraphs and list items are between them. The lines of a
// code span over more than one line in such a block are kept with their
// indentation from the text of the block, see blockRelativeCodeSpanSource.
// Returns nil if the node is not in a block, or is in a quote or a footnote.
func codeSpanBlockItem(node ast.Node) ast.Node {
	for ; node != nil; node = node.Parent() {
		switch n := node.(type) {
		case *ast.Blockquote, *footnoteDefinition:
			return nil
		case *ast.ListItem:
			if item, ok := blockItemOf(n); ok {
				return item
			}
		}
	}

	return nil
}

// blockRelativeCodeSpanSource gets the text of a code span over more than one
// line from start to stop, where each line after the first has the
// indentation that it has after the column of the `-` of the block. A lazy
// line of a list item or of the block keeps its column in this way. Returns
// false if a line is indented less than the `-` of the block.
func blockRelativeCodeSpanSource(src []byte, item ast.Node, start int, stop int) (string, bool) {
	var buf []byte
	pos := start
	for {
		end := bytes.IndexByte(src[pos:stop], '\n')
		if end < 0 {
			buf = append(buf, src[pos:stop]...)
			return string(buf), true
		}

		buf = append(buf, src[pos:pos+end+1]...)
		lineStart := pos + end + 1
		value, _ := item.Attribute(bulletLineStartAttribute)
		bulletStart, ok := value.(int)
		dash := 0
		if ok {
			dash = indentationWidth(src, bulletStart)
		}
		if !ok || indentationWidth(src, lineStart) < dash {
			return "", false
		}

		indentation := indentationAfterColumn(src, lineStart, dash)

		pos = lineStart
		for pos < stop && (src[pos] == ' ' || src[pos] == '\t') {
			pos++
		}

		buf = append(buf, indentation...)
	}
}

// blockLinesOf gets the lines of the block that holds the inline node, such
// as the paragraph that a code span is in.
func blockLinesOf(node ast.Node) *text.Segments {
	for ; node != nil; node = node.Parent() {
		if node.Type() == ast.TypeBlock {
			return node.Lines()
		}
	}

	return nil
}

// codeSpanParser reads only paragraphs and code spans. It is used to check
// what the text of a code span is read as.
var codeSpanParser = parser.NewParser(
	parser.WithBlockParsers(util.Prioritized(parser.NewParagraphParser(), 1000)),
	parser.WithInlineParsers(util.Prioritized(parser.NewCodeSpanParser(), 100)),
)

// codeSpanReadsBack returns true if the text is read as one code span with
// the value.
func codeSpanReadsBack(source string, value string) bool {
	src := []byte(source)
	doc := codeSpanParser.Parse(text.NewReader(src))
	paragraph := doc.FirstChild()
	if paragraph == nil || paragraph.FirstChild() == nil || paragraph.FirstChild() != paragraph.LastChild() {
		return false
	}

	code, ok := paragraph.FirstChild().(*ast.CodeSpan)
	return ok && string(code.Text(src)) == value
}
