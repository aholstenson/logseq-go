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

	entry := parseLogbookClock(line)
	if entry == nil {
		entry = parseLogbookStateChange(line)
	}

	if entry != nil {
		if aware, ok := entry.(content.LogbookEntrySourceAware); ok {
			aware.SetEntrySource(strings.TrimSuffix(value, "\r"))
		}

		return entry
	}

	return content.NewLogbookEntryRaw(value)
}

// logbookEntryReadsBack returns true if the text of a logbook entry is read
// as the same entry: a clock entry with the same start and end, or a state
// change with the same statuses and time.
func logbookEntryReadsBack(source string, entry content.LogbookEntry) bool {
	if strings.ContainsAny(source, "\r\n") {
		return false
	}

	switch e := entry.(type) {
	case *content.LogbookEntryClock:
		other, ok := parseLogbookEntry(source).(*content.LogbookEntryClock)
		return ok && other.Start.Equal(e.Start) && other.End.Equal(e.End)
	case *content.LogbookEntryStateChange:
		other, ok := parseLogbookEntry(source).(*content.LogbookEntryStateChange)
		return ok && other.From == e.From && other.To == e.To && other.Time.Equal(e.Time)
	}

	return false
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

	// StartLineText is the text after `:LOGBOOK:` on its line, without the
	// line break.
	StartLineText string

	// EndLine is the `:END:` line as it is in the file, without the line
	// break.
	EndLine string

	// indent is the position of `:LOGBOOK:` in the line that holds it, which
	// is the indentation that is taken from the lines inside.
	indent int
}

func (n *logbook) Kind() ast.NodeKind {
	return logbookKind
}

func (n *logbook) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

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

	source := reader.Source()
	lineStart := bytes.LastIndexByte(source[:segment.Start], '\n') + 1
	if markerAt := segment.Start + pos - segment.Padding; markerAt >= lineStart && len(bytes.Trim(source[lineStart:markerAt], " \t")) == 0 {
		// The line has only spaces and tabs before `:LOGBOOK:`, so the
		// lines after it are compared in columns, as a tab can be split
		// by the column of the block, as in "- a\n\t:LOGBOOK:". See
		// hasDrawerEndAtColumn.
		if !hasDrawerEndAtColumn(source[segment.Stop:], indentationWidth(source, lineStart), isLogbookEnd) {
			return nil, parser.NoChildren
		}
	} else if !hasLogbookEnd(source[segment.Stop:], indent) {
		return nil, parser.NoChildren
	}

	// Create the block
	rest := line[pos+len(":LOGBOOK:"):]
	block := &logbook{
		StartLineText: strings.TrimRight(string(rest), "\r\n"),
		indent:        indent,
	}

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

// isLogbookEnd checks if a line, without its indentation, ends a logbook.
func isLogbookEnd(line []byte) bool {
	return bytes.HasPrefix(line, []byte(":END:")) && util.IsBlank(line[len(":END:"):])
}

func (b *logbookParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	block := node.(*logbook)

	_, pos := util.IndentWidth(line, reader.LineOffset())

	// Check if the line starts with :END:
	searchFor := []byte(":END:")
	if bytes.HasPrefix(line[pos:], searchFor) && util.IsBlank(line[pos+len(searchFor):]) {
		// The END line is kept from the column of `:LOGBOOK:`, which the
		// writer writes at the column of the block.
		block.EndLine = strings.TrimRight(string(line), "\r\n")
		if block.indent > 0 {
			block.EndLine = indentationAfterColumn([]byte(block.EndLine), 0, block.indent)
			block.EndLine += strings.TrimLeft(strings.TrimRight(string(line), "\r\n"), " \t")
		}
		newline := 1
		if line[len(line)-1] != '\n' {
			newline = 0
		}
		reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
		return parser.Close
	}

	appendContentLine(node, reader, block.indent)
	return parser.Continue | parser.NoChildren
}

func (b *logbookParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (b *logbookParser) CanInterruptParagraph() bool {
	return true
}

func (b *logbookParser) CanAcceptIndentedLine() bool {
	return false
}
