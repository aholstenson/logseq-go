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

// relaxedTaskDateRegexp matches the task date lines that only
// WithRelaxedTaskDates reads, in addition to the lines that Logseq reads (see
// parseLogseqTaskDate). The weekday can be missing or in any language, and
// the parts can have more than one space between them.
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

	// Segment is the source of the date, from `SCHEDULED` or `DEADLINE` to
	// the end of the timestamp. When only white space follows the timestamp,
	// the segment goes to the end of the line, without the line break. The
	// other content after the timestamp is in the lines of the node, and is
	// parsed as its inline children.
	Segment text.Segment

	// length is the length of the date in the line it was parsed from, see
	// Segment.
	length int
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
	return []byte{'S', 'D', 's', 'd'}
}

func (b *taskDateParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	options, _ := pc.Get(parseOptionsKey).(*parseOptions)
	relaxed := options != nil && options.relaxedTaskDates

	node := parseTaskDate(string(line[pos:]), relaxed)
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
	if dateStop := start + node.length; dateStop < stop {
		node.Segment = text.NewSegment(start, dateStop)
		node.Lines().Append(text.NewSegment(dateStop, stop))
	}

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

// taskDateReadsBack returns true if Logseq reads the text of a task date line
// as the same date: the same type, day, time of day and repeater. The text is
// read in the form that Logseq reads, see parseLogseqTaskDate. A line that only
// WithRelaxedTaskDates reads is not kept, so that it is written in a form
// that Logseq reads.
func taskDateReadsBack(source string, date *content.TaskDate) bool {
	if strings.ContainsAny(source, "\r\n") {
		return false
	}

	parsed := parseTaskDate(source, false)
	if parsed == nil || parsed.length != len(source) || parsed.DateType != date.Type || parsed.HasTime != date.HasTime || !parsed.Date.Equal(date.Date) {
		return false
	}

	if (parsed.Repeater == nil) != (date.Repeater == nil) {
		return false
	}

	return parsed.Repeater == nil || *parsed.Repeater == *date.Repeater
}

// parseTaskDate parses a single line as a task date. The line is read in the
// way Logseq reads it, see parseLogseqTaskDate, and with relaxed also in the
// forms of relaxedTaskDateRegexp. It returns nil if the line is not a task
// date.
func parseTaskDate(line string, relaxed bool) *taskDate {
	line = strings.TrimRight(line, "\r\n")
	if node := parseLogseqTaskDate(line); node != nil {
		if strings.TrimRight(line[node.length:], " \t") == "" {
			// Only white space follows, which stays with the date.
			node.length = len(line)
		}

		return node
	}

	if !relaxed {
		return nil
	}

	return parseRelaxedTaskDate(line)
}

// parseLogseqTaskDate reads a `SCHEDULED:` or `DEADLINE:` line with the
// timestamp rules of mldoc, the parser of Logseq:
//
//   - The keyword can be in any case, and one or more spaces or tabs come
//     before the timestamp.
//   - The timestamp is in `<` and `>`, or in `[` and `]`.
//   - The date is the first part. Logseq reads the leading numbers of
//     `year-month-day`, so `2024-1-1` is a date and text after the day is
//     ignored. Exactly one space comes after the date.
//   - A weekday of ASCII letters follows. It is required, but its value is not
//     used.
//   - Up to two more parts follow, each after exactly one space. With two
//     parts, the first is the time of day and the second the repeater. With
//     one part, it is a repeater when it starts with `+` or `.`, and the time
//     otherwise.
//   - Logseq reads the leading `hour:minute` numbers of the time, so
//     `10:00-11:00`, `10:00:30` and `10:00am` are 10:00. A time that cannot be
//     read is ignored, and the timestamp is still a date.
//   - Logseq reads the leading number and unit of a repeater, so `+1d/3d` is
//     `+1d`. A repeater that cannot be read is ignored.
//
// Other content can follow the timestamp on the line. The length of the node
// is where the timestamp ends.
//
// The model of a task date cannot hold all that Logseq reads. A day that does
// not exist, such as 2024-02-30, makes the line text. A time that is not a
// time of day, such as 24:00, is ignored. A repeater of one character makes
// mldoc fail, so the line is text. In all these cases the line is written back
// as it is, unless the date is changed.
func parseLogseqTaskDate(line string) *taskDate {
	node := &taskDate{}

	rest, ok := cutPrefixFold(line, "SCHEDULED:")
	if ok {
		node.DateType = content.TaskDateTypeScheduled
	} else if rest, ok = cutPrefixFold(line, "DEADLINE:"); ok {
		node.DateType = content.TaskDateTypeDeadline
	} else {
		return nil
	}

	afterKeyword := strings.TrimLeftFunc(rest, isTimestampSpace)
	if len(afterKeyword) == len(rest) || afterKeyword == "" {
		return nil
	}

	var closer rune
	switch afterKeyword[0] {
	case '<':
		closer = '>'
	case '[':
		closer = ']'
	default:
		return nil
	}

	r := &timestampReader{value: afterKeyword[1:], closer: closer}

	// The date and exactly one space.
	dateText := r.part()
	if dateText == "" || !r.space() {
		return nil
	}

	numbers, ok := leadingNumbers(dateText, '-', 3)
	if !ok {
		return nil
	}

	// The weekday.
	if r.letters() == "" {
		return nil
	}

	// The time and the repeater.
	// A space must be followed by a part, as in `<2024-01-01 Mon >` the space
	// is where the closing character must be.
	var first, second string
	if r.space() {
		if first = r.part(); first == "" {
			return nil
		}

		if r.space() {
			if second = r.part(); second == "" {
				return nil
			}
		}
	}

	if !r.close() {
		return nil
	}

	node.length = len(line) - len(r.value)

	timeText, repeaterText := "", ""
	switch {
	case second != "":
		timeText, repeaterText = first, second
	case first != "" && (first[0] == '+' || first[0] == '.'):
		repeaterText = first
	default:
		timeText = first
	}

	hour, minute := 0, 0
	if timeText != "" {
		if parts, ok := leadingNumbers(timeText, ':', 2); ok && parts[0] <= 23 && parts[1] <= 59 {
			node.HasTime = true
			hour, minute = parts[0], parts[1]
		}
	}

	year, month, day := numbers[0], numbers[1], numbers[2]
	wall := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
	if wall.Year() != year || int(wall.Month()) != month || wall.Day() != day {
		// The date does not exist, such as 2024-02-31, so this is not a date
		// we can represent.
		return nil
	}

	node.Date = inLocalZone(wall)

	if repeaterText != "" {
		if len(repeaterText) < 2 {
			return nil
		}

		node.Repeater = parseRepeater(repeaterText)
	}

	return node
}

// parseRepeater reads a repeater in the way mldoc reads it. When the second
// character is not `+`, the repeater is `+` and the value starts after the
// first character, so `.1d` is `+1d`. Otherwise the repeater is `++` when the
// first character is `+`, and `.+` for any other first character. The value
// is a number and a unit, with any text after the unit ignored. It returns nil
// if the value cannot be read.
func parseRepeater(value string) *content.Repeater {
	repeater := &content.Repeater{}
	if value[1] != '+' {
		repeater.Type = content.RepeaterTypeCumulate
		value = value[1:]
	} else {
		if value[0] == '+' {
			repeater.Type = content.RepeaterTypeCatchUp
		} else {
			repeater.Type = content.RepeaterTypeRestart
		}
		value = value[2:]
	}

	digits := 0
	for digits < len(value) && value[digits] >= '0' && value[digits] <= '9' {
		digits++
	}

	if digits == 0 || digits == len(value) {
		return nil
	}

	number, err := strconv.Atoi(value[:digits])
	if err != nil {
		return nil
	}

	repeater.Value = number
	switch value[digits] {
	case 'h':
		repeater.Unit = content.RepeaterUnitHour
	case 'd':
		repeater.Unit = content.RepeaterUnitDay
	case 'w':
		repeater.Unit = content.RepeaterUnitWeek
	case 'm':
		repeater.Unit = content.RepeaterUnitMonth
	case 'y':
		repeater.Unit = content.RepeaterUnitYear
	default:
		return nil
	}

	return repeater
}

// parseRelaxedTaskDate reads a line that matches relaxedTaskDateRegexp.
func parseRelaxedTaskDate(line string) *taskDate {
	matches := relaxedTaskDateRegexp.FindStringSubmatch(strings.TrimRight(line, " \t"))
	if matches == nil {
		return nil
	}

	node := &taskDate{length: len(line)}
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

	wall := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
	if wall.Year() != year || wall.Month() != time.Month(month) || wall.Day() != day {
		return nil
	}

	node.Date = inLocalZone(wall)

	if matches[7] != "" {
		node.Repeater = parseRepeater(matches[7] + matches[8] + matches[9])
	}

	return node
}

// isTimestampSpace checks if the rune is a space in a timestamp to mldoc: a
// space, a tab, a form feed or the SUB control character.
func isTimestampSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\x1a' || r == '\f'
}

// cutPrefixFold removes the prefix from the value, ignoring ASCII case.
func cutPrefixFold(value string, prefix string) (string, bool) {
	if len(value) < len(prefix) || !strings.EqualFold(value[:len(prefix)], prefix) {
		return value, false
	}

	return value[len(prefix):], true
}

// leadingNumbers reads count numbers of ASCII digits with the separator
// between them from the start of the value, and ignores the text after the
// last number. This is how mldoc reads dates and times.
func leadingNumbers(value string, separator byte, count int) ([]int, bool) {
	numbers := make([]int, 0, count)
	for i := 0; i < count; i++ {
		if i > 0 {
			if value == "" || value[0] != separator {
				return nil, false
			}
			value = value[1:]
		}

		digits := 0
		for digits < len(value) && value[digits] >= '0' && value[digits] <= '9' {
			digits++
		}

		if digits == 0 {
			return nil, false
		}

		number, err := strconv.Atoi(value[:digits])
		if err != nil {
			return nil, false
		}

		numbers = append(numbers, number)
		value = value[digits:]
	}

	return numbers, true
}

// timestampReader reads the parts of a timestamp after its opening `<` or
// `[`.
type timestampReader struct {
	value  string
	closer rune
}

// part reads the text up to the next space or the closing character.
func (r *timestampReader) part() string {
	end := strings.IndexFunc(r.value, func(c rune) bool {
		return isTimestampSpace(c) || c == r.closer
	})
	if end < 0 {
		end = len(r.value)
	}

	part := r.value[:end]
	r.value = r.value[end:]
	return part
}

// space reads one space, and returns false if there is none.
func (r *timestampReader) space() bool {
	if r.value == "" || !isTimestampSpace(rune(r.value[0])) {
		return false
	}

	r.value = r.value[1:]
	return true
}

// letters reads ASCII letters.
func (r *timestampReader) letters() string {
	end := 0
	for end < len(r.value) && (r.value[end] >= 'a' && r.value[end] <= 'z' || r.value[end] >= 'A' && r.value[end] <= 'Z') {
		end++
	}

	letters := r.value[:end]
	r.value = r.value[end:]
	return letters
}

// close reads the closing character, and returns false if it is not next.
func (r *timestampReader) close() bool {
	if r.value == "" || rune(r.value[0]) != r.closer {
		return false
	}

	r.value = r.value[1:]
	return true
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
		paragraph.Lines().AppendAll(date.Lines().Sliced(0, date.Lines().Len()))
		paragraph.AppendChild(paragraph, ast.NewTextSegment(date.Segment))
		for child := date.FirstChild(); child != nil; {
			next := child.NextSibling()
			paragraph.AppendChild(paragraph, child)
			child = next
		}

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
