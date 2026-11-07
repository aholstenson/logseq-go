package markdown

import (
	"bytes"

	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// hasClosingLine checks if a line after the current line of the reader, in
// the same block, closes a construct that the current line opens, such as a
// code fence or a `#+BEGIN_` line.
//
// Logseq reads an opening line that is not closed as text, so the lines after
// it are read as text and child blocks as usual. CommonMark instead closes the
// construct at the end of the block, so the parsers check for the closing
// line before they open the construct.
//
// The block ends at the first line that is not blank and is indented less
// than the text of the current line. closes gets each line in the block with
// its indentation taken off.
//
// For a line in a quote the block is the lines after it that have the `>`
// markers of the quote, see hasClosingLineInQuote.
//
// kind names the closing lines that closes looks for, such as "fence:`:3".
// When no closing line is found, the lines that were checked are remembered
// for the kind, so that more opening lines of the same kind in the same block
// are not checked again. Without this, a page with many opening lines that
// are not closed takes quadratic time to parse.
func hasClosingLine(reader text.Reader, pc parser.Context, kind string, closes func(line []byte) bool) bool {
	source := reader.Source()
	_, segment := reader.PeekLine()

	lineStart := bytes.LastIndexByte(source[:segment.Start], '\n') + 1
	prefix := source[lineStart:segment.Start]
	if depth := bytes.Count(prefix, []byte{'>'}); depth > 0 {
		return hasClosingLineInQuote(source[segment.Stop:], depth, closes)
	}

	// The prefix holds the indentation and the markers of the blocks and
	// lists that start on this line. The lines after it are indented to
	// the text of the block, which starts where the prefix ends.
	column := columnsOf(prefix) - segment.Padding

	scans, _ := pc.Get(closingLineScansKey).(map[closingLineScan]checkedLines)
	if scans == nil {
		scans = map[closingLineScan]checkedLines{}
		pc.Set(closingLineScansKey, scans)
	}

	scan := closingLineScan{kind: kind, column: column}
	if checked, ok := scans[scan]; ok && checked.start <= segment.Stop && segment.Stop <= checked.end {
		return false
	}

	pos := segment.Stop
	for pos < len(source) {
		start := pos
		end := bytes.IndexByte(source[pos:], '\n')
		if end < 0 {
			end = len(source)
		} else {
			end += pos + 1
		}

		line := source[pos:end]
		pos = end

		if util.IsBlank(line) {
			continue
		}

		width, textStart := util.IndentWidth(line, 0)
		if width < column {
			pos = start
			break
		}

		if closes(line[textStart:]) {
			return true
		}
	}

	scans[scan] = checkedLines{start: segment.Stop, end: pos}
	return false
}

// hasClosingLineInQuote checks if a line in the text closes a construct that
// opens in a quote, where depth is the number of `>` markers before the
// construct. The quote ends at the first line with fewer markers, as the lines
// of a code block or of a `#+BEGIN_` block cannot be lazy lines of the quote.
// closes gets each line with its markers and indentation taken off.
func hasClosingLineInQuote(rest []byte, depth int, closes func(line []byte) bool) bool {
	for len(rest) > 0 {
		end := bytes.IndexByte(rest, '\n')
		if end < 0 {
			end = len(rest)
		} else {
			end++
		}

		line := rest[:end]
		rest = rest[end:]

		i := 0
		for markers := 0; markers < depth; markers++ {
			for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
				i++
			}
			if i >= len(line) || line[i] != '>' {
				return false
			}
			i++
		}

		line = line[i:]
		if util.IsBlank(line) {
			continue
		}

		_, textStart := util.IndentWidth(line, 0)
		if closes(line[textStart:]) {
			return true
		}
	}

	return false
}

var closingLineScansKey = parser.NewContextKey()

// closingLineScan is a kind of closing line in the block with the text at the
// given column.
type closingLineScan struct {
	kind   string
	column int
}

// checkedLines are the lines from start up to end that have no closing line
// of a kind. end is the start of the line that ends the block, or the end of
// the source.
type checkedLines struct {
	start int
	end   int
}

// columnsOf gets the width of the text in columns. A tab moves to the next
// multiple of four columns, and every other byte is one column.
func columnsOf(text []byte) int {
	columns := 0
	for _, c := range text {
		if c == '\t' {
			columns += 4 - columns%4
		} else {
			columns++
		}
	}

	return columns
}
