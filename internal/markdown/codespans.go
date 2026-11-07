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
		if startLine == stopLine {
			code.SetAttributeString(codeSpanSourceAttribute, string(block.Source()[start.Start:stop.Start]))
		}
	}

	return result
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
