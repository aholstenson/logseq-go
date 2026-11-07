package markdown

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var priorityKind = ast.NewNodeKind("Priority")

type priority struct {
	ast.BaseInline
	Priority content.Priority

	// SpaceAfter is true when a space follows the closing `]`.
	SpaceAfter bool
}

func (*priority) Kind() ast.NodeKind {
	return priorityKind
}

func (n *priority) Dump(src []byte, level int) {
}

// priorityParser parses the `[#A]` priority Logseq puts at the start of a
// task. Logseq reads any one character between `[#` and `]` as the priority,
// such as `[#D]`, `[#a]` or `[#1]`, see content.Priority.
//
// Bracketed text that looks like a priority but is not one, either because of
// where it is or because of the letter used, is claimed as plain text. Leaving
// it to the other parsers would let the tag parser pick up the part after the
// `[` as a tag, which swallows the closing bracket.
type priorityParser struct {
}

func (t *priorityParser) Trigger() []byte {
	return []byte{'['}
}

func (t *priorityParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, seg := block.PeekLine()

	if len(line) < 4 || line[0] != '[' || line[1] != '#' {
		return nil
	}

	if line[3] == ']' && line[2] != '\n' && line[2] != '\r' && isPriorityPosition(parent, block.Source()) {
		return newPriority(block, line, content.PriorityOf(line[2]))
	}

	// Scan for the closing bracket. Anything with a space or a nested bracket
	// in it is left to the other parsers.
	end := 0
	for i := 2; i < len(line); i++ {
		if line[i] == ']' {
			end = i
			break
		}

		if line[i] == '[' || util.IsSpace(line[i]) {
			return nil
		}
	}

	if end == 0 {
		return nil
	}

	// The text is not a priority, as it has more than one character in the
	// brackets or is not where Logseq reads a priority.
	block.Advance(end + 1)
	return ast.NewTextSegment(seg.WithStop(seg.Start + end + 1))
}

// newPriority takes the priority at the start of the line, which is written
// as `[#A]`, from the reader.
func newPriority(block text.Reader, line []byte, value content.Priority) ast.Node {
	length := 4
	spaceAfter := length < len(line) && line[length] == ' '
	if spaceAfter {
		// The space after the priority separates it from the content and is
		// written back out with the priority.
		length++
	}

	block.Advance(length)
	return &priority{
		Priority:   value,
		SpaceAfter: spaceAfter,
	}
}

// isPriorityPosition checks if the parser is where Logseq looks for a
// priority, which is at the start of the content of a block, after the task
// marker if there is one. Logseq allows more than one space after the marker.
func isPriorityPosition(parent ast.Node, source []byte) bool {
	if parent.PreviousSibling() != nil {
		// Something else comes before this in the block.
		return false
	}

	switch parent.ChildCount() {
	case 0:
		return true
	case 1:
		// A task marker is still part of the text at this point, as it is
		// picked out after the whole block has been parsed.
		text, ok := parent.FirstChild().(*ast.Text)
		if !ok {
			return false
		}

		value := string(text.Segment.Value(source))
		if !strings.HasSuffix(value, " ") {
			return false
		}

		return taskStatusFor(strings.TrimRight(value, " ")) != content.TaskStatusNone
	}

	return false
}

var _ parser.InlineParser = (*priorityParser)(nil)
