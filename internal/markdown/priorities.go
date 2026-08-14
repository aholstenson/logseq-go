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
}

func (*priority) Kind() ast.NodeKind {
	return priorityKind
}

func (n *priority) Dump(src []byte, level int) {
}

// priorityParser parses the `[#A]` priority Logseq puts at the start of a
// task.
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

	value := priorityFor(string(line[2:end]))
	if value == content.PriorityNone || !isPriorityPosition(parent, block.Source()) {
		block.Advance(end + 1)
		return ast.NewTextSegment(seg.WithStop(seg.Start + end + 1))
	}

	length := end + 1
	if length < len(line) && line[length] == ' ' {
		// The space after the priority separates it from the content and is
		// written back out with the priority.
		length++
	}

	block.Advance(length)
	return &priority{
		Priority: value,
	}
}

func priorityFor(value string) content.Priority {
	switch value {
	case "A":
		return content.PriorityA
	case "B":
		return content.PriorityB
	case "C":
		return content.PriorityC
	}

	return content.PriorityNone
}

// isPriorityPosition checks if the parser is where Logseq looks for a
// priority, which is at the start of the content of a block, after the task
// marker if there is one.
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

		return taskStatusFor(strings.TrimSuffix(value, " ")) != content.TaskStatusNone
	}

	return false
}

var _ parser.InlineParser = (*priorityParser)(nil)
