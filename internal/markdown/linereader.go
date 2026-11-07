package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// lineCachingReader is a text.Reader that keeps facts about the line that it
// is on, so that the block parsers do not read the start of the line again
// for each block that holds the line.
//
// Each block that holds a line, such as a list and a list item for each level
// of an outline, reads the line in turn. The reader of Goldmark finds the
// column of its position from the start of the line each time, and the line
// is checked for text from the position each time. In an outline with one
// bullet for each level, the time to parse then grows with the cube of the
// depth. This reader finds the column from where it found it last, and finds
// the end of the text of a line only once. The block list parsers use it to
// skip the full checks of Goldmark for the usual lines, see
// continuesUnderLastItem and continuesItem.
//
// Goldmark itself still checks the line for text once for each block that
// holds it, so the time still grows with the cube of the depth, but much
// more slowly.
type lineCachingReader struct {
	text.Reader
	source []byte

	// lineStop is the end of the line that the fields below are for, or -1.
	lineStop int

	// lineHead is the offset where the line starts. The position of the
	// reader can be before it, see position.
	lineHead int

	// textEnd is the offset after the last character of the line that is not
	// a space, or lineHead if the line has only spaces.
	textEnd int

	// offsetWidth is the width in columns of the line from lineHead to
	// offsetAt.
	offsetAt    int
	offsetWidth int

	// lineOffset is the last value of LineOffset, if lineOffsetValid is set
	// and the reader has not moved since, see LineOffset.
	lineOffset      int
	lineOffsetValid bool
	lineOffsetStart int
	lineOffsetStop  int
}

func newLineCachingReader(source []byte) *lineCachingReader {
	return &lineCachingReader{
		Reader:   text.NewReader(source),
		source:   source,
		lineStop: -1,
	}
}

// position gets the position of the reader, and reads the line if the reader
// is on a new line.
func (r *lineCachingReader) position() text.Segment {
	_, pos := r.Reader.Position()
	if pos.Stop == r.lineStop {
		return pos
	}

	// The line is found from its end, as the reader of Goldmark does. The
	// position can be before the start of the line, as a parser can move it
	// back onto the line break of the line before.
	end := pos.Stop
	if end > len(r.source) {
		end = len(r.source)
	}

	head := end
	if head > 0 && r.source[head-1] == '\n' {
		head--
	}
	for head > 0 && r.source[head-1] != '\n' {
		head--
	}

	for end > head && util.IsSpace(r.source[end-1]) {
		end--
	}

	r.lineStop = pos.Stop
	r.lineHead = head
	r.textEnd = end
	r.offsetAt = head
	r.offsetWidth = 0
	return pos
}

// LineOffset gets the column of the position of the reader, in the same way
// as the reader of Goldmark. As in Goldmark, the value is kept until the
// reader moves, also when SetPadding changes the padding.
func (r *lineCachingReader) LineOffset() int {
	_, current := r.Reader.Position()
	if r.lineOffsetValid && current.Start == r.lineOffsetStart && current.Stop == r.lineOffsetStop {
		return r.lineOffset
	}

	pos := r.position()
	if pos.Start < r.offsetAt {
		r.offsetAt = r.lineHead
		r.offsetWidth = 0
	}

	for ; r.offsetAt < pos.Start; r.offsetAt++ {
		if r.source[r.offsetAt] == '\t' {
			r.offsetWidth += util.TabWidth(r.offsetWidth)
		} else {
			r.offsetWidth++
		}
	}

	r.lineOffset = r.offsetWidth - pos.Padding
	r.lineOffsetValid = true
	r.lineOffsetStart = pos.Start
	r.lineOffsetStop = pos.Stop
	return r.lineOffset
}

// The methods below move the reader, so that LineOffset finds the column
// again, as in Goldmark.

func (r *lineCachingReader) ResetPosition() {
	r.lineOffsetValid = false
	r.Reader.ResetPosition()
}

func (r *lineCachingReader) Advance(n int) {
	r.lineOffsetValid = false
	r.Reader.Advance(n)
}

func (r *lineCachingReader) AdvanceAndSetPadding(n, padding int) {
	r.lineOffsetValid = false
	r.Reader.AdvanceAndSetPadding(n, padding)
}

func (r *lineCachingReader) AdvanceLine() {
	r.lineOffsetValid = false
	r.Reader.AdvanceLine()
}

func (r *lineCachingReader) SetPosition(line int, pos text.Segment) {
	r.lineOffsetValid = false
	r.Reader.SetPosition(line, pos)
}

// restIsBlank returns true if the line has only spaces from the position of
// the reader to its end.
func (r *lineCachingReader) restIsBlank() bool {
	pos := r.position()
	return pos.Start >= r.textEnd
}

// restOfLineIsBlank returns true if the line that the reader is on has only
// spaces from the position of the reader. It is the same as util.IsBlank for
// the line from PeekLine, but does not read the line again for a
// lineCachingReader.
func restOfLineIsBlank(reader text.Reader) bool {
	if r, ok := reader.(*lineCachingReader); ok {
		return r.restIsBlank()
	}

	line, _ := reader.PeekLine()
	return util.IsBlank(line)
}

// indentWidthUpTo is util.IndentWidth that stops when the width is limit or
// more. A width less than limit is the full width of the indentation.
func indentWidthUpTo(line []byte, currentPos int, limit int) (int, int) {
	width := 0
	pos := 0
	for ; pos < len(line) && width < limit; pos++ {
		switch line[pos] {
		case ' ':
			width++
		case '\t':
			width += util.TabWidth(currentPos + width)
		default:
			return width, pos
		}
	}

	return width, pos
}

var _ text.Reader = (*lineCachingReader)(nil)

// appendContentLine adds the line that the reader is on to the lines of a
// block that keeps its lines as they are, such as a math block or a drawer,
// and moves the reader to the end of the line. The indentation of the first
// line of the block, indent columns, is taken from the start of the line. A
// line with less indentation keeps the text from its first character that is
// not a space.
func appendContentLine(node ast.Node, reader text.Reader, indent int) {
	line, segment := reader.PeekLine()
	pos, padding := util.IndentPositionPadding(line, reader.LineOffset(), segment.Padding, indent)
	if pos < 0 {
		// The line from PeekLine starts with the padding of the reader as
		// spaces, which are not in the source.
		pos = util.FirstNonSpacePosition(line) - segment.Padding
		if pos < 0 {
			pos = 0
		}
		padding = 0
	}

	length := segment.Stop - segment.Start
	if pos >= length {
		// Nothing of the line is left, and Goldmark cannot read an empty
		// line of a block.
		return
	}

	node.Lines().Append(text.NewSegmentPadding(segment.Start+pos, segment.Stop, padding))
	reader.AdvanceAndSetPadding(length-pos-1, padding)
}
