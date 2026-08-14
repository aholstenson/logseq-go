package markdown

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// taskDateRegexp matches a full `SCHEDULED:` or `DEADLINE:` line. The day name
// after the date is only there for readability and is regenerated on output,
// so it is matched but not captured. Lines that do not match in full are left
// alone and parsed as regular text, so nothing is lost for syntax this library
// does not model.
var taskDateRegexp = regexp.MustCompile(
	`^(SCHEDULED|DEADLINE): +<(\d{4})-(\d{2})-(\d{2})(?: +\p{L}+)?(?: +(\d{1,2}):(\d{2}))?(?: +(\+\+|\.\+|\+)(\d+)([hdwmy]))?>$`,
)

var taskDateKind = ast.NewNodeKind("TaskDate")

type taskDate struct {
	ast.BaseBlock

	DateType content.TaskDateType
	Date     time.Time
	HasTime  bool
	Repeater *content.Repeater
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

	node := parseTaskDate(string(line[pos:]))
	if node == nil {
		return nil, parser.NoChildren
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

// parseTaskDate parses a single line, returning nil if it is not a task date.
func parseTaskDate(line string) *taskDate {
	matches := taskDateRegexp.FindStringSubmatch(strings.TrimRight(line, " \t\r\n"))
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
