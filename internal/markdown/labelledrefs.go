package markdown

import (
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
// including the brackets. A `]]` with a `\` before it does not close the page
// reference. Returns 0 if the value does not start with a page reference.
func pageRefLength(value []byte) int {
	if len(value) < 4 || value[0] != '[' || value[1] != '[' {
		return 0
	}

	for i := 2; i < len(value)-1; i++ {
		if value[i] == ']' && value[i+1] == ']' && value[i-1] != '\\' {
			return i + 2
		}
	}

	return 0
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
