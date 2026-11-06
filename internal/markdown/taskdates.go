package markdown

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// taskDateRegexp matches a full `SCHEDULED:` or `DEADLINE:` line in the form
// that Logseq reads: one space between the parts, and a weekday of ASCII
// letters after the date. The weekday is only there for readability and is
// regenerated on output, so it is matched but not captured. Lines that do not
// match in full are left alone and parsed as regular text, so nothing is lost
// for syntax this library does not model.
var taskDateRegexp = regexp.MustCompile(
	`^(SCHEDULED|DEADLINE): +<(\d{4})-(\d{2})-(\d{2}) [A-Za-z]+(?: (\d{1,2}):(\d{2}))?(?: (\+\+|\.\+|\+)(\d+)([hdwmy]))?>$`,
)

// relaxedTaskDateRegexp is taskDateRegexp for WithRelaxedTaskDates. The
// weekday can be missing or in any language, and the parts can have more
// than one space between them.
var relaxedTaskDateRegexp = regexp.MustCompile(
	`^(SCHEDULED|DEADLINE): +<(\d{4})-(\d{2})-(\d{2})(?: +\p{L}+)?(?: +(\d{1,2}):(\d{2}))?(?: +(\+\+|\.\+|\+)(\d+)([hdwmy]))?>$`,
)

var taskDateKind = ast.NewNodeKind("TaskDate")

type taskDate struct {
	ast.BaseBlock

	DateType content.TaskDateType
	Date     time.Time
	HasTime  bool
	Repeater *content.Repeater

	// Segment is the source of the line, from `SCHEDULED` or `DEADLINE` to
	// the end of the line, without the line break.
	Segment text.Segment
}

func (n *taskDate) Kind() ast.NodeKind {
	return taskDateKind
}

func (n *taskDate) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type taskDateParser struct {
}

func (b *taskDateParser) Trigger() []byte {
	return []byte{'S', 'D'}
}

func (b *taskDateParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	pattern := taskDateRegexp
	if options, ok := pc.Get(parseOptionsKey).(*parseOptions); ok && options.relaxedTaskDates {
		pattern = relaxedTaskDateRegexp
	}

	node := parseTaskDate(string(line[pos:]), pattern)
	if node == nil {
		return nil, parser.NoChildren
	}

	start := segment.Start + pos - segment.Padding
	if start < segment.Start {
		start = segment.Start
	}
	stop := segment.Stop
	for stop > start && (reader.Source()[stop-1] == '\n' || reader.Source()[stop-1] == '\r') {
		stop--
	}
	node.Segment = text.NewSegment(start, stop)

	reader.Advance(segment.Stop - segment.Start - 1 - segment.Padding)
	return node, parser.NoChildren
}

func (b *taskDateParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Close
}

func (b *taskDateParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (b *taskDateParser) CanInterruptParagraph() bool {
	return true
}

func (b *taskDateParser) CanAcceptIndentedLine() bool {
	return false
}

// parseTaskDate parses a single line with the given pattern, see
// taskDateRegexp. It returns nil if the line is not a task date.
func parseTaskDate(line string, pattern *regexp.Regexp) *taskDate {
	matches := pattern.FindStringSubmatch(strings.TrimRight(line, " \t\r\n"))
	if matches == nil {
		return nil
	}

	node := &taskDate{}
	if matches[1] == "DEADLINE" {
		node.DateType = content.TaskDateTypeDeadline
	} else {
		node.DateType = content.TaskDateTypeScheduled
	}

	year, _ := strconv.Atoi(matches[2])
	month, _ := strconv.Atoi(matches[3])
	day, _ := strconv.Atoi(matches[4])

	hour := 0
	minute := 0
	if matches[5] != "" {
		node.HasTime = true
		hour, _ = strconv.Atoi(matches[5])
		minute, _ = strconv.Atoi(matches[6])

		if hour > 23 || minute > 59 {
			return nil
		}
	}

	node.Date = time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local)
	if node.Date.Year() != year || node.Date.Month() != time.Month(month) || node.Date.Day() != day {
		// The date does not exist, such as 2024-02-31, so this is not a date
		// we can represent.
		return nil
	}

	if matches[7] != "" {
		repeater := &content.Repeater{}
		switch matches[7] {
		case "+":
			repeater.Type = content.RepeaterTypeCumulate
		case "++":
			repeater.Type = content.RepeaterTypeCatchUp
		case ".+":
			repeater.Type = content.RepeaterTypeRestart
		}

		repeater.Value, _ = strconv.Atoi(matches[8])

		switch matches[9] {
		case "h":
			repeater.Unit = content.RepeaterUnitHour
		case "d":
			repeater.Unit = content.RepeaterUnitDay
		case "w":
			repeater.Unit = content.RepeaterUnitWeek
		case "m":
			repeater.Unit = content.RepeaterUnitMonth
		case "y":
			repeater.Unit = content.RepeaterUnitYear
		}

		node.Repeater = repeater
	}

	return node
}

// taskDatePositions turns the task dates that Logseq ignores because of where
// they are into text. It does nothing for WithRelaxedTaskDates.
//
// Logseq reads the content of a block after its first line as paragraphs.
// Blank lines and text lines are in the same paragraph, and properties, code,
// a logbook and other content that is not text end it. Logseq uses the dates
// of the first paragraph whose first part is a date, and ignores all other
// dates. Thus the date is used in "- TODO a\n  SCHEDULED: <2024-01-01 Mon>".
// It is text in "- TODO a\n  note\n  SCHEDULED: <2024-01-01 Mon>", as "note"
// comes first, and in "- TODO a\n\n  SCHEDULED: <2024-01-01 Mon>", as the
// blank line comes first.
//
// A date on the first line of a block, as in `- SCHEDULED: <...>`, is the
// title of the block and is text too.
//
// Only the content of a block before its child blocks is checked. In a page,
// the dates after the first bullet or heading that are not in a bullet are
// kept as they are.
type taskDatePositions struct {
}

func (t *taskDatePositions) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	options, _ := pc.Get(parseOptionsKey).(*parseOptions)
	if options != nil && options.relaxedTaskDates {
		return
	}

	page := options != nil && options.page
	var ignored []*taskDate
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if n.Kind() == ast.KindDocument || (n.Kind() == ast.KindListItem && holdsBlocks(n)) {
			ignored = append(ignored, ignoredTaskDates(n, reader.Source(), page)...)
		}

		return ast.WalkContinue, nil
	})

	for _, date := range ignored {
		paragraph := ast.NewParagraph()
		paragraph.Lines().Append(date.Segment)
		paragraph.AppendChild(paragraph, ast.NewTextSegment(date.Segment))
		paragraph.SetBlankPreviousLines(date.HasBlankPreviousLines())
		date.Parent().ReplaceChild(date.Parent(), date, paragraph)
	}
}

// ignoredTaskDates finds the task dates in the content of a block, or of a
// page, that Logseq ignores. See taskDatePositions.
func ignoredTaskDates(container ast.Node, source []byte, page bool) []*taskDate {
	_, isDocument := container.(*ast.Document)

	// The first line of a block is its title. The content of a page has no
	// title.
	hasTitle := !isDocument || !page

	var ignored []*taskDate

	// inParagraph is whether a paragraph as Logseq reads it is open, and
	// startsWithDate whether its first part is a date. used is whether a
	// paragraph that starts with a date has ended, so that its dates are
	// the ones of the block.
	inParagraph := false
	startsWithDate := false
	used := false
	endParagraph := func() {
		if inParagraph && startsWithDate {
			used = true
		}

		inParagraph = false
		startsWithDate = false
	}

	for child := container.FirstChild(); child != nil; child = child.NextSibling() {
		first := child == container.FirstChild()
		switch n := child.(type) {
		case *ast.Paragraph, *ast.TextBlock:
			if first && hasTitle {
				// The lines after the title start a paragraph of text.
				if hasLineBreak(n) {
					inParagraph = true
				}

				continue
			}

			if !inParagraph {
				inParagraph = true
				startsWithDate = false
			}
		case *taskDate:
			if first && hasTitle && (isDocument || onBulletLine(n, source)) {
				// The date is the title of the block.
				ignored = append(ignored, n)
				continue
			}

			if !inParagraph {
				inParagraph = true
				startsWithDate = !n.HasBlankPreviousLines() &&
					(!first || isDocument || afterBareBullet(n, source))
			}

			if !startsWithDate || used {
				ignored = append(ignored, n)
			}
		case *ast.List, *indentedBlocks, *ast.Heading:
			if _, ok := n.(*ast.Heading); ok && !(isDocument && page) {
				endParagraph()
				continue
			}

			// Content after this belongs to other blocks.
			return ignored
		default:
			endParagraph()
		}
	}

	return ignored
}

// hasLineBreak checks if the text of a paragraph has more than one line.
func hasLineBreak(paragraph ast.Node) bool {
	found := false
	_ = ast.Walk(paragraph, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := n.(*ast.Text); ok && entering && (t.SoftLineBreak() || t.HardLineBreak()) {
			found = true
			return ast.WalkStop, nil
		}

		return ast.WalkContinue, nil
	})

	return found
}

// onBulletLine checks if the date is on the same line as the bullet of its
// block.
func onBulletLine(date *taskDate, source []byte) bool {
	start := lineStart(source, date.Segment.Start)
	return len(bytes.Trim(source[start:date.Segment.Start], " \t")) > 0
}

// afterBareBullet checks if the line before the date is a bullet with
// nothing after it, not even a space. Logseq reads a space after the bullet
// as text before the date.
func afterBareBullet(date *taskDate, source []byte) bool {
	start := lineStart(source, date.Segment.Start)
	if start == 0 {
		return false
	}

	previous := source[lineStart(source, start-1):start]
	previous = bytes.TrimRight(previous, "\r\n")
	return string(bytes.TrimLeft(previous, " \t")) == "-"
}

// lineStart finds where the line that holds the offset starts.
func lineStart(source []byte, offset int) int {
	for offset > 0 && source[offset-1] != '\n' {
		offset--
	}

	return offset
}
