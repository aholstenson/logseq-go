package query

import (
	"strconv"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/internal/utils"
)

// Time is a time in a query, for a TimestampFilter. It is a Logseq input with
// a time of day, such as `right-now-ms`, `-7d-ms` or `today-start`, or a fixed
// time. Only one of the fields is set. The zero Time is not a valid time.
type Time struct {
	// Input is an input with a time of day, relative to now, in the forms
	// that Logseq reads, without the leading colon.
	Input string

	// Fixed is a fixed time.
	Fixed time.Time
}

// Now is the time when the query runs.
func Now() Time {
	return Time{Input: "right-now-ms"}
}

// TimeInput is a time relative to now, such as `-7d-ms` or `today-start`.
func TimeInput(input string) Time {
	return Time{Input: strings.TrimPrefix(input, ":")}
}

// At is a fixed time.
func At(t time.Time) Time {
	return Time{Fixed: t}
}

// IsZero returns true if no field of the time is set.
func (t Time) IsZero() bool {
	return t.Input == "" && t.Fixed.IsZero()
}

// String returns the time in the text form that Parse reads.
func (t Time) String() string {
	switch {
	case t.Input != "":
		return t.Input
	case !t.Fixed.IsZero():
		local := t.Fixed.Local()
		if local.Nanosecond() != 0 {
			return local.Format(timeLayouts[0])
		}
		return local.Format(timeLayouts[1])
	}

	return ""
}

// timeLayouts are the forms of a fixed time in the text form, in the local
// time zone.
var timeLayouts = []string{
	"2006-01-02T15:04:05.000",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
}

// parseTime reads a time: an input with a time of day, such as `-7d-ms`, a
// fixed time such as `2024-01-15T10:30`, or milliseconds since 1970, which is
// how Logseq stores timestamps. ok is false for other text.
func parseTime(text string) (Time, bool) {
	text = strings.TrimPrefix(text, ":")
	if input, ok := utils.ParseQueryDateInput(strings.ToLower(text), time.Now()); ok && input.HasTime {
		return TimeInput(strings.ToLower(text)), true
	}

	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return At(t), true
		}
	}

	if ms, err := strconv.ParseInt(text, 10, 64); err == nil {
		return At(time.UnixMilli(ms)), true
	}

	return Time{}, false
}

// TimestampField is the time that a TimestampFilter matches on.
type TimestampField int

const (
	// CreatedAt is the time when a block or page was created.
	CreatedAt TimestampField = iota
	// UpdatedAt is the time when a block or page was last changed.
	UpdatedAt
)

var timestampFieldNames = map[TimestampField]string{
	CreatedAt: "created-at",
	UpdatedAt: "updated-at",
}

// TimestampFilter matches the blocks or pages with a time of creation or of
// the last change that compares with a time. It is written as
// `(updated-at > -7d-ms)`, and for pages as `(page-updated-at > -7d-ms)`.
// Without a comparison, as `(updated-at)`, it matches when there is a time.
//
// The times of a block are its `created-at::` and `updated-at::` properties,
// in milliseconds since 1970. Logseq writes these when the setting
// `:feature/enable-block-timestamps?` is on. The time of the last change of a
// page is the time its file was last changed, the same as Logseq uses. The
// time a page was created is not known, so a filter for it returns an error
// when the query runs.
type TimestampFilter struct {
	// Field is the time to compare.
	Field TimestampField

	// Op is how to compare the time: Exists, Equal, Less, LessOrEqual,
	// Greater or GreaterOrEqual.
	Op Op

	// Time is the time to compare with, when Op is not Exists.
	Time Time

	// Page is true to match pages instead of blocks.
	Page bool
}

// Timestamp creates a filter that matches the blocks with a time that
// compares with the time.
func Timestamp(field TimestampField, op Op, t Time) *TimestampFilter {
	return &TimestampFilter{Field: field, Op: op, Time: t}
}

// PageTimestamp creates a filter that matches the pages with a time that
// compares with the time.
func PageTimestamp(field TimestampField, op Op, t Time) *TimestampFilter {
	return &TimestampFilter{Field: field, Op: op, Time: t, Page: true}
}

func (*TimestampFilter) isFilter() {}

func (f *TimestampFilter) String() string {
	name := timestampFieldNames[f.Field]
	if f.Page {
		name = "page-" + name
	}

	if f.Op == Exists {
		return list(name)
	}
	return list(name, opNames[f.Op], f.Time.String())
}

// parseTimestampBetween reads `(between created-at -7d today)`, which matches
// the blocks with a time from the start of the first day to the end of the
// last day, the same as in Logseq.
func parseTimestampBetween(e *expr, args []*expr) (Filter, error) {
	field := CreatedAt
	switch strings.ToLower(args[0].text) {
	case "created-at":
	case "updated-at":
		field = UpdatedAt
	default:
		return nil, errorAt(args[0].pos, "expected created-at or updated-at")
	}

	var times [2]Time
	for i, arg := range args[1:] {
		date, err := parseDate(arg)
		if err != nil {
			return nil, err
		}

		t, ok := dayTime(date, i == 1)
		if !ok {
			return nil, errorAt(arg.pos, "expected a date such as `-7d` or `2024-01-15`")
		}
		times[i] = t
	}

	return And(Timestamp(field, GreaterOrEqual, times[0]), Timestamp(field, LessOrEqual, times[1])), nil
}

// dayTime returns the start of a date as a time, or the end of it if end is
// true. A journal date can not be written as a time, so ok is false for it.
func dayTime(date Date, end bool) (Time, bool) {
	switch {
	case date.Relative != "":
		input := date.Relative
		switch input {
		case "yesterday":
			input = "-1d"
		case "tomorrow":
			input = "+1d"
		}

		if end {
			return TimeInput(input + "-end"), true
		}
		return TimeInput(input + "-start"), true
	case !date.Fixed.IsZero():
		if end {
			return At(date.Fixed.AddDate(0, 0, 1).Add(-time.Millisecond)), true
		}
		return At(date.Fixed), true
	}

	return Time{}, false
}

// parseTimestampFilter reads a filter such as `(updated-at > -7d-ms)`.
func parseTimestampFilter(e *expr, name string, args []*expr) (Filter, error) {
	filter := &TimestampFilter{Field: UpdatedAt, Page: strings.HasPrefix(name, "page-")}
	if strings.HasSuffix(name, "created-at") {
		filter.Field = CreatedAt
	}

	switch len(args) {
	case 0:
		filter.Op = Exists
		return filter, nil
	case 2:
		op, ok := ops[args[0].text]
		if args[0].kind != exprSymbol || !ok {
			return nil, errorAt(args[0].pos, "expected =, <, <=, > or >=")
		}

		t, ok := parseTime(args[1].text)
		if args[1].kind != exprSymbol || !ok {
			return nil, errorAt(args[1].pos, "expected a time, such as `-7d-ms`, `right-now-ms` or `2024-01-15T10:30`")
		}

		filter.Op = op
		filter.Time = t
		return filter, nil
	}

	return nil, errorAt(e.pos, "%s takes a comparison and a time, such as (%s > -7d-ms)", name, name)
}
