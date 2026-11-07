package markdown

import (
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
