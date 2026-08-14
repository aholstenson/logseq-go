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

// clockRegexp matches a clock entry. The duration at the end is not captured
// as it is derived from the two timestamps when the entry is written.
var clockRegexp = regexp.MustCompile(`^CLOCK: \[([^\]]+)\](?:--\[([^\]]+)\] +=> +\d+:\d{2}:\d{2})?$`)

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

func parseLogbookClock(line string) content.LogbookEntry {
	matches := clockRegexp.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	start, err := time.ParseInLocation(logbookTimeLayout, matches[1], time.Local)
	if err != nil {
		return nil
	}

	var end time.Time
	if matches[2] != "" {
		end, err = time.ParseInLocation(logbookTimeLayout, matches[2], time.Local)
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

	at, err := time.ParseInLocation(logbookTimeLayout, matches[3], time.Local)
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

// Open looks for :LOGBOOK: at the beginning of a line
func (b *logbookParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}

	indent := pos

	// Check if the line starts with :LOGBOOK:
	if !bytes.HasPrefix(line[pos:], []byte(":LOGBOOK:")) {
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
