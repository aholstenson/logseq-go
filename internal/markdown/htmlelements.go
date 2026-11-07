package markdown

import (
	"bytes"
	"sort"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// knownHTMLTags are the names of the HTML tags that Logseq reads as the start
// of an HTML element in text. The names are lower case.
var knownHTMLTags = map[string]struct{}{}

func init() {
	for _, tag := range strings.Fields(`a abbr address area article aside audio b
		base bdi bdo blockquote body br button canvas caption cite code col
		colgroup data datalist dd del dfn div dl dt em embed fieldset figcaption
		figure footer form h1 h2 h3 h4 h5 h6 head header hr html i iframe img
		input ins kbd keygen label legend li link main map mark meta meter nav
		noscript object ol optgroup option output p param pre progress q rb rp
		rt rtc ruby s samp script section select small source span strong style
		sub sup table tbody td template textarea tfoot th thead time title tr
		track u ul var video details summary wbr`) {
		knownHTMLTags[tag] = struct{}{}
	}
}

// htmlElementParser reads a whole HTML element in text, such as
// `<span class="x">[[a]]</span>`, as one piece of raw HTML. Logseq shows such
// an element as HTML, and reads no references, tags or other Markdown in it.
// Goldmark reads only the tags as raw HTML, and the text between them as
// Markdown.
//
// The rules are the ones of Logseq:
//
//   - The tag name must be a known HTML tag, in any case. `<foo>x</foo>` is
//     text.
//   - The element ends at the closing tag, in any case, that closes the
//     opening tag. An opening tag of the same name in the element, written as
//     `<span>` or `<span ...`, must be closed first.
//   - An element without a closing tag ends at the first `/>` after the tag
//     name, such as in `<img src="a.png"/>`.
//   - The element can continue on the lines after it. On the first line of a
//     block, it must end on that line, except when it is at the start of the
//     line.
//
// When the element does not end, the parsers after this one read the text.
type htmlElementParser struct {
}

func (p *htmlElementParser) Trigger() []byte {
	return []byte{'<'}
}

func (p *htmlElementParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	tag := htmlElementTag(line)
	if tag == "" {
		return nil
	}

	paragraph := htmlParagraphOf(parent, block.Source(), pc)
	lineIndex, segment := block.Position()
	if paragraph == nil || lineIndex < 0 || lineIndex >= len(paragraph.lineStarts) {
		return nil
	}

	start := paragraph.lineStarts[lineIndex] + segment.Start - parent.Lines().At(lineIndex).Start
	if start < 0 || start >= len(paragraph.text) || paragraph.text[start] != '<' {
		return nil
	}

	// On the first line of a block, the element must end on that line,
	// unless it starts the line.
	limit := len(paragraph.text)
	if lineIndex == 0 && isTitleParagraph(parent, block.Source(), pc) && !atParagraphStart(parent, block) {
		limit = paragraph.lineEnd(0)
	}

	end := paragraph.elementEnd(start, tag, limit)
	if end < 0 {
		return nil
	}

	node := ast.NewRawHTML()
	remaining := end - start
	for remaining > 0 {
		line, segment := block.PeekLine()
		if line == nil {
			break
		}

		if segment.Len() >= remaining {
			node.Segments.Append(segment.WithStop(segment.Start + remaining))
			block.Advance(remaining)
			break
		}

		node.Segments.Append(segment)
		remaining -= segment.Len()
		block.AdvanceLine()
	}

	return node
}

// htmlElementTag gets the tag name at the start of the text, such as `span`
// in `<span class="x">`, if it is a known HTML tag. It is empty otherwise.
func htmlElementTag(line []byte) string {
	if len(line) < 2 || line[0] != '<' {
		return ""
	}

	end := 1
	for end < len(line) && line[end] != '>' && !isHTMLTagSpace(line[end]) {
		end++
	}

	tag := string(line[1:end])
	if _, ok := knownHTMLTags[strings.ToLower(tag)]; !ok {
		return ""
	}

	return tag
}

// isHTMLTagSpace checks if a character ends the name of a tag.
func isHTMLTagSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

var htmlParagraphsKey = parser.NewContextKey()

// htmlParagraph is the text of a paragraph, with what is needed to find where
// its HTML elements end. Each paragraph is read once, so that the time to
// find the ends grows with the length of the paragraph and not with its
// square.
type htmlParagraph struct {
	// text is the lines of the paragraph, one after the other.
	text []byte

	// lineStarts is where each line starts in text.
	lineStarts []int

	// selfClosings is where each `/>` starts in text, in order.
	selfClosings []int

	// tags are the opening and closing tags of each tag name, as written.
	tags map[string]*htmlTagEvents
}

// htmlTagEvents are the opening tags (`<span>` and `<span `) and the closing
// tags (`</span>`, in any case) of one tag name in a paragraph, in order.
type htmlTagEvents struct {
	// starts is where each tag starts, and closing whether it is a closing
	// tag.
	starts  []int
	closing []bool

	// nextLower is, for each number of tags j from the start of the
	// paragraph, the smallest number of tags after j at which there are more
	// closing tags after the first j tags than opening tags. It is -1 when
	// there is none.
	nextLower []int
}

// htmlParagraphOf gets the paragraph that the node holds the lines of, and
// reads it the first time.
func htmlParagraphOf(node ast.Node, source []byte, pc parser.Context) *htmlParagraph {
	paragraphs, _ := pc.Get(htmlParagraphsKey).(map[ast.Node]*htmlParagraph)
	if paragraphs == nil {
		paragraphs = map[ast.Node]*htmlParagraph{}
		pc.Set(htmlParagraphsKey, paragraphs)
	}

	if paragraph, ok := paragraphs[node]; ok {
		return paragraph
	}

	lines := node.Lines()
	if lines == nil || lines.Len() == 0 {
		return nil
	}

	paragraph := &htmlParagraph{tags: map[string]*htmlTagEvents{}}
	for i := 0; i < lines.Len(); i++ {
		paragraph.lineStarts = append(paragraph.lineStarts, len(paragraph.text))
		segment := lines.At(i)
		paragraph.text = append(paragraph.text, segment.Value(source)...)
	}

	for offset := 0; ; {
		index := bytes.Index(paragraph.text[offset:], []byte("/>"))
		if index < 0 {
			break
		}

		paragraph.selfClosings = append(paragraph.selfClosings, offset+index)
		offset += index + 2
	}

	paragraphs[node] = paragraph
	return paragraph
}

// lineEnd gets where the line ends in text, after its line break.
func (p *htmlParagraph) lineEnd(line int) int {
	if line+1 < len(p.lineStarts) {
		return p.lineStarts[line+1]
	}

	return len(p.text)
}

// elementEnd finds where the HTML element that starts at the offset ends,
// see htmlElementParser. The element must end at or before limit. It is -1
// when the element does not end.
func (p *htmlParagraph) elementEnd(start int, tag string, limit int) int {
	events := p.eventsOf(tag)

	// The tags after the name of the opening tag. The element ends at the
	// first closing tag after which there are more closing tags than opening
	// tags.
	first := sort.SearchInts(events.starts, start+1)
	if next := events.nextLower[first]; next > 0 {
		end := events.starts[next-1] + len("</"+tag+">")
		if end <= limit {
			return end
		}
	}

	// Without a closing tag, the element ends at the first `/>`.
	after := start + 1 + len(tag)
	i := sort.SearchInts(p.selfClosings, after)
	if i < len(p.selfClosings) && p.selfClosings[i]+2 <= limit {
		return p.selfClosings[i] + 2
	}

	return -1
}

// eventsOf gets the opening and closing tags of the tag name, and finds them
// the first time.
func (p *htmlParagraph) eventsOf(tag string) *htmlTagEvents {
	if events, ok := p.tags[tag]; ok {
		return events
	}

	type event struct {
		start   int
		closing bool
	}

	var all []event
	find := func(text []byte, pattern string, closing bool) {
		for offset := 0; ; {
			index := bytes.Index(text[offset:], []byte(pattern))
			if index < 0 {
				return
			}

			all = append(all, event{start: offset + index, closing: closing})
			offset += index + len(pattern)
		}
	}

	find(p.text, "<"+tag+">", false)
	find(p.text, "<"+tag+" ", false)
	find(bytes.ToLower(p.text), "</"+strings.ToLower(tag)+">", true)
	sort.Slice(all, func(i, j int) bool { return all[i].start < all[j].start })

	events := &htmlTagEvents{}
	for _, e := range all {
		events.starts = append(events.starts, e.start)
		events.closing = append(events.closing, e.closing)
	}

	// sums[j] is the number of opening tags minus the number of closing tags
	// in the first j tags. nextLower[j] is the first j' > j with a smaller
	// sum, found with a stack from the end.
	sums := make([]int, len(all)+1)
	for j, e := range all {
		if e.closing {
			sums[j+1] = sums[j] - 1
		} else {
			sums[j+1] = sums[j] + 1
		}
	}

	events.nextLower = make([]int, len(sums))
	var stack []int
	for j := len(sums) - 1; j >= 0; j-- {
		for len(stack) > 0 && sums[stack[len(stack)-1]] >= sums[j] {
			stack = stack[:len(stack)-1]
		}

		events.nextLower[j] = -1
		if len(stack) > 0 {
			events.nextLower[j] = stack[len(stack)-1]
		}
		stack = append(stack, j)
	}

	p.tags[tag] = events
	return events
}

// isTitleParagraph checks if the node is the first line of a block and the
// lines after it, which is the paragraph that starts on the line of the
// bullet. The first line of a page that is read as a page is not the title
// of a block.
func isTitleParagraph(node ast.Node, source []byte, pc parser.Context) bool {
	container := node.Parent()
	if container == nil || container.FirstChild() != node || !holdsBlocks(container) || node.Lines().Len() == 0 {
		return false
	}

	if _, ok := container.(*ast.Document); ok {
		options, _ := pc.Get(parseOptionsKey).(*parseOptions)
		return options == nil || !options.page
	}

	first := node.Lines().At(0)
	start := lineStart(source, first.Start)
	return len(bytes.Trim(source[start:first.Start], " \t")) > 0
}

// atParagraphStart checks if the reader is at the start of the first line of
// the paragraph.
func atParagraphStart(node ast.Node, block text.Reader) bool {
	line, segment := block.Position()
	return line == 0 && node.Lines().Len() > 0 && segment.Start == node.Lines().At(0).Start
}

var _ parser.InlineParser = (*htmlElementParser)(nil)
