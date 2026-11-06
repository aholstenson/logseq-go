package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// labelledRefLinkParser is the link parser of Goldmark, changed so that it
// also reads `[label]([[Page Name]])` when the name of the page has spaces in
// it. Logseq reads this as a link to the page with a label. Goldmark stops the
// destination of a link at the first space, so without this change the label
// stays as text next to a page link without a label.
//
// When the closing `]` of a label is directly followed by `([[...]])`, the
// reader that Goldmark gets shows the spaces in the page name as another
// character. The destination of the link that Goldmark returns is then set
// back to the text of the source.
type labelledRefLinkParser struct {
	parser.InlineParser
}

func newLabelledRefLinkParser() *labelledRefLinkParser {
	return &labelledRefLinkParser{
		InlineParser: parser.NewLinkParser(),
	}
}

func (p *labelledRefLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	startLine, start := block.Position()
	result := p.parse(parent, block, pc)
	switch result.(type) {
	case *ast.Link, *ast.Image:
		stopLine, stop := block.Position()
		if startLine == stopLine {
			setDestinationSource(result, block.Source()[start.Start:stop.Start])
		}
	}

	return result
}

// destinationSourceAttribute is the name of the attribute that holds the
// text between the parentheses of a link or an image, as it is in the file.
const destinationSourceAttribute = "logseq-destination"

// setDestinationSource keeps the text between the parentheses of a link or
// an image. The text is what the link parser read, from the `]` of the label
// to the `)` at the end. A reference link, such as `[a][b]`, has no such
// text.
func setDestinationSource(node ast.Node, consumed []byte) {
	if len(consumed) < 3 || consumed[0] != ']' || consumed[1] != '(' || consumed[len(consumed)-1] != ')' {
		return
	}

	node.SetAttributeString(destinationSourceAttribute, string(consumed[2:len(consumed)-1]))
}

func (p *labelledRefLinkParser) parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()
	if len(line) < 2 || line[0] != ']' || line[1] != '(' {
		return p.InlineParser.Parse(parent, block, pc)
	}

	// The page link starts after the `](` and the link ends directly after it.
	length := pageRefLength(line[2:])
	if length == 0 || len(line) < 2+length+1 || line[2+length] != ')' {
		return p.InlineParser.Parse(parent, block, pc)
	}

	start := segment.Start + 2
	stop := start + length
	result := p.InlineParser.Parse(parent, &spacelessReader{
		Reader: block,
		start:  start,
		stop:   stop,
	}, pc)

	destination := block.Value(text.NewSegment(start, stop))
	switch node := result.(type) {
	case *ast.Link:
		node.Destination = destination
	case *ast.Image:
		node.Destination = destination
	}

	return result
}

func (p *labelledRefLinkParser) CloseBlock(parent ast.Node, block text.Reader, pc parser.Context) {
	if closer, ok := p.InlineParser.(parser.CloseBlocker); ok {
		closer.CloseBlock(parent, block, pc)
	}
}

// pageRefLength gets the length of the `[[...]]` at the start of the value,
// including the brackets. The page reference ends in the same way as a page
// link, see pageLinkClose. Returns 0 if the value does not start with a page
// reference.
func pageRefLength(value []byte) int {
	if len(value) < 4 || value[0] != '[' || value[1] != '[' {
		return 0
	}

	end := pageLinkClose(value)
	if end == 0 {
		return 0
	}

	return end + 2
}

// blockRefLength gets the length of the `((uuid))` at the start of the value,
// including the parentheses. Returns 0 if the value does not start with a
// block reference.
func blockRefLength(value []byte) int {
	if len(value) < 4 || value[0] != '(' || value[1] != '(' {
		return 0
	}

	for i := 2; i < len(value)-1; i++ {
		if value[i] == ')' && value[i+1] == ')' {
			return i + 2
		} else if !isUUIDChar(value[i]) {
			return 0
		}
	}

	return 0
}

// spacelessReader is a reader that shows the spaces between start and stop in
// the source as `_`, so that Goldmark reads that part as one destination of a
// link. Only the lines returned by PeekLine are changed.
type spacelessReader struct {
	text.Reader

	start int
	stop  int
}

func (r *spacelessReader) PeekLine() ([]byte, text.Segment) {
	line, segment := r.Reader.PeekLine()

	var changed []byte
	for i, c := range line {
		position := segment.Start + i
		if position < r.start || position >= r.stop || !util.IsSpace(c) {
			continue
		}

		if changed == nil {
			changed = make([]byte, len(line))
			copy(changed, line)
		}

		changed[i] = '_'
	}

	if changed == nil {
		return line, segment
	}

	return changed, segment
}

// destinationParser reads only paragraphs and links. It is used to check what
// the text between the parentheses of a link is read as.
var destinationParser = parser.NewParser(
	parser.WithBlockParsers(util.Prioritized(parser.NewParagraphParser(), 1000)),
	parser.WithInlineParsers(util.Prioritized(parser.NewLinkParser(), 200)),
)

// destinationReadsBack returns true if the text between the parentheses of a
// link is read as the URL and the title.
func destinationReadsBack(source string, url string, title string) bool {
	if strings.ContainsAny(source, "\r\n") {
		return false
	}

	doc := destinationParser.Parse(text.NewReader([]byte("[x](" + source + ")")))
	paragraph := doc.FirstChild()
	if paragraph == nil || paragraph.FirstChild() == nil || paragraph.FirstChild() != paragraph.LastChild() {
		return false
	}

	link, ok := paragraph.FirstChild().(*ast.Link)
	return ok && unescapePunctuation(link.Destination) == url && unescapePunctuation(link.Title) == title
}
