package markdown

import (
	"bytes"
	"regexp"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// logbookTimeLayout is the timestamp format used inside logbook entries. The
// day name is only there for readability and is regenerated on output.
const logbookTimeLayout = "2006-01-02 Mon 15:04:05"

// logbookTimeLayoutWithoutSeconds is the timestamp format of graphs where
// `:logbook/settings :with-second-support?` is off.
const logbookTimeLayoutWithoutSeconds = "2006-01-02 Mon 15:04"

// clockRegexp matches a clock entry. The duration at the end is not captured
// as it is derived from the two timestamps when the entry is written.
var clockRegexp = regexp.MustCompile(`^CLOCK: \[([^\]]+)\](?:--\[([^\]]+)\] +=> +\d+:\d{2}(?::\d{2})?)?$`)

// stateChangeRegexp matches the entry Logseq writes when a repeating task
// changes status.
var stateChangeRegexp = regexp.MustCompile(`^\* State "([^"]+)"(?: +from "([^"]+)")? +\[([^\]]+)\]$`)

// parseLogbookEntry parses a single line of a logbook, falling back to a raw
// entry for anything this library does not model so that it survives being
// written back out.
func parseLogbookEntry(value string) content.LogbookEntry {
	line := strings.TrimRight(value, " \t\r")

	if entry := parseLogbookClock(line); entry != nil {
		return entry
	}

	if entry := parseLogbookStateChange(line); entry != nil {
		return entry
	}

	return content.NewLogbookEntryRaw(value)
}

// parseLogbookTime reads a timestamp of a logbook entry. Both precisions are
// accepted, as a graph without second support leaves the seconds out. The
// time is in the local time zone, see inLocalZone.
func parseLogbookTime(value string) (time.Time, error) {
	parsed, err := time.Parse(logbookTimeLayout, value)
	if err != nil {
		parsed, err = time.Parse(logbookTimeLayoutWithoutSeconds, value)
		if err != nil {
			return time.Time{}, err
		}
	}

	return inLocalZone(parsed), nil
}

// inLocalZone gets the time in the local time zone that has the same wall
// clock time (date, hour, minute and second) as the given time.
//
// When the clocks go forward, some wall clock times do not exist in the
// local zone, such as 02:30 on 2024-03-31 in Europe/Stockholm. time.Date
// moves such a time by the length of the gap, and the moved time would then
// be written to the page. Instead, such a time gets a fixed zone with the
// offset that gives the same instant as time.Date, so that the wall clock
// time that was written in the page stays the same.
func inLocalZone(wall time.Time) time.Time {
	local := time.Date(wall.Year(), wall.Month(), wall.Day(),
		wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), time.Local)
	if sameWallClock(local, wall) {
		return local
	}

	wallUTC := time.Date(wall.Year(), wall.Month(), wall.Day(),
		wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), time.UTC)
	offset := int(wallUTC.Sub(local) / time.Second)

	// Use the name of the local zone that has this offset, which is the zone
	// on one side of the gap.
	name := ""
	_, localOffset := local.Zone()
	gap := time.Duration(offset-localOffset) * time.Second
	for _, at := range []time.Time{local.Add(gap), local.Add(-gap)} {
		if zoneName, zoneOffset := at.Zone(); zoneOffset == offset {
			name = zoneName
			break
		}
	}

	return local.In(time.FixedZone(name, offset))
}

// sameWallClock checks if two times have the same date, hour, minute and
// second, in their own time zones.
func sameWallClock(a time.Time, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day() &&
		a.Hour() == b.Hour() && a.Minute() == b.Minute() && a.Second() == b.Second()
}

func parseLogbookClock(line string) content.LogbookEntry {
	matches := clockRegexp.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	start, err := parseLogbookTime(matches[1])
	if err != nil {
		return nil
	}

	var end time.Time
	if matches[2] != "" {
		end, err = parseLogbookTime(matches[2])
		if err != nil {
			return nil
		}

		if end.Before(start) {
			// The entry can not be written back out, so keep it as it is.
			return nil
		}
	}

	return content.NewLogbookEntryClock(start, end)
}

func parseLogbookStateChange(line string) content.LogbookEntry {
	matches := stateChangeRegexp.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	to := taskStatusFor(matches[1])
	if to == content.TaskStatusNone {
		return nil
	}

	var from content.TaskStatus
	if matches[2] != "" {
		from = taskStatusFor(matches[2])
		if from == content.TaskStatusNone {
			return nil
		}
	}

	at, err := parseLogbookTime(matches[3])
	if err != nil {
		return nil
	}

	return content.NewLogbookEntryStateChange(from, to, at)
}

var logbookKind = ast.NewNodeKind("Logbook")

type logbook struct {
	ast.BaseBlock
}

func (n *logbook) Kind() ast.NodeKind {
	return logbookKind
}

func (n *logbook) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type logbookBlockData struct {
	indent int
	node   ast.Node
}

var logbookInfoKey = parser.NewContextKey()

type logbookParser struct {
}

func (b *logbookParser) Trigger() []byte {
	return []byte{':'}
}

// Open looks for :LOGBOOK: at the beginning of a line. The line starts a
// logbook only if an :END: line follows in the same block, see
// hasLogbookEnd. Otherwise the lines are text.
func (b *logbookParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	indent := pos

	// Check if the line starts with :LOGBOOK:
	if !bytes.HasPrefix(line[pos:], []byte(":LOGBOOK:")) {
		return nil, parser.NoChildren
	}

	if !hasLogbookEnd(reader.Source()[segment.Stop:], indent) {
		return nil, parser.NoChildren
	}

	// Create the block
	block := &logbook{}

	// Store the block in the context
	pc.Set(logbookInfoKey, &logbookBlockData{
		indent: indent,
		node:   block,
	})

	return block, parser.NoChildren
}

// hasLogbookEnd checks if an :END: line comes after a :LOGBOOK: line before
// the block that holds the logbook ends. The source starts at the line after
// the :LOGBOOK: line, and indent is the number of characters before
// :LOGBOOK: on its line.
//
// The block ends at a bullet line, or at a line with fewer characters of
// indentation than the :LOGBOOK: line. Without an :END: line before that, the
// logbook would hide the text after it, and an :END: line would be added on
// save. Logseq reads such a logbook up to an :END: line in a later block,
// which hides those blocks too, so here the lines are read as text instead.
func hasLogbookEnd(source []byte, indent int) bool {
	for len(source) > 0 {
		line := source
		if i := bytes.IndexByte(source, '\n'); i >= 0 {
			line = source[:i+1]
		}
		source = source[len(line):]

		trimmed := bytes.TrimLeft(line, " \t")
		if util.IsBlank(trimmed) {
			continue
		}

		if len(line)-len(trimmed) < indent || isBulletLine(trimmed) {
			return false
		}

		if bytes.HasPrefix(trimmed, []byte(":END:")) && util.IsBlank(trimmed[len(":END:"):]) {
			return true
		}
	}

	return false
}

func (b *logbookParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	data := pc.Get(logbookInfoKey).(*logbookBlockData)

	_, pos := util.IndentWidth(line, reader.LineOffset())

	// Check if the line starts with :END:
	searchFor := []byte(":END:")
	if bytes.HasPrefix(line[pos:], searchFor) && util.IsBlank(line[pos+len(searchFor):]) {
		newline := 1
		if line[len(line)-1] != '\n' {
			newline = 0
		}
		reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
		return parser.Close
	}

	pos, padding := util.IndentPositionPadding(line, reader.LineOffset(), segment.Padding, data.indent)
	if pos < 0 {
		pos = util.FirstNonSpacePosition(line)
		if pos < 0 {
			pos = 0
		}
		padding = 0
	}

	seg := text.NewSegmentPadding(segment.Start+pos, segment.Stop, padding)
	node.Lines().Append(seg)
	reader.AdvanceAndSetPadding(segment.Stop-segment.Start-pos-1, padding)
	return parser.Continue | parser.NoChildren
}

func (b *logbookParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	data := pc.Get(logbookInfoKey).(*logbookBlockData)
	if data.node == node {
		pc.Set(beginEndInfoKey, nil)
	}
}

func (b *logbookParser) CanInterruptParagraph() bool {
	return true
}

func (b *logbookParser) CanAcceptIndentedLine() bool {
	return false
}
