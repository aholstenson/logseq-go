package utils

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// dateFormatToken maps a token of the date format Logseq uses in its config to
// the equivalent in a Go layout.
type dateFormatToken struct {
	// token is what the format uses.
	token string
	// layout is what Go uses for the same thing.
	layout string
	// ordinal marks the day written with an English ordinal suffix, such as
	// `15th`. A Go layout can not express it, so layout is only what the day
	// parses as.
	ordinal bool
}

// dateFormatTokens are the tokens that can appear in a date format, in the
// order they are matched. Tokens sharing a prefix are ordered longest first so
// the longest one wins, keeping `EEEE` from being read as `EEE` and a leftover
// `E`.
var dateFormatTokens = []dateFormatToken{
	{token: "yyyy", layout: "2006"},
	{token: "yy", layout: "06"},
	{token: "MMMM", layout: "January"},
	{token: "MMM", layout: "Jan"},
	{token: "MM", layout: "01"},
	{token: "M", layout: "1"},
	{token: "do", layout: "2", ordinal: true},
	{token: "dd", layout: "02"},
	{token: "d", layout: "2"},
	{token: "EEEE", layout: "Monday"},
	{token: "EEE", layout: "Mon"},
	{token: "HH", layout: "15"},
	{token: "H", layout: "15"},
	{token: "hh", layout: "03"},
	{token: "h", layout: "3"},
	{token: "mm", layout: "04"},
	{token: "m", layout: "4"},
	{token: "ss", layout: "05"},
	{token: "s", layout: "5"},
}

// ordinalSuffixes matches the suffix of an ordinal day, so it can be taken out
// of a date before Go parses it.
var ordinalSuffixes = regexp.MustCompile(`([0-9])(?:st|nd|rd|th)`)

// DateFormat is a date format as used by Logseq, prepared for formatting and
// parsing dates. Formats can ask for things a Go layout has no way to express,
// such as the ordinal day in the default title format `MMM do, yyyy`, so the
// format is kept as the segments it is made up of rather than as a single
// layout.
type DateFormat struct {
	// segments are the parts of the format in the order they are written,
	// which formatting fills in one at a time.
	segments []dateFormatSegment

	// layout is the whole format as a Go layout, with ordinal days as plain
	// days. Parsing uses it once the ordinal suffixes have been taken out of
	// the value.
	layout string

	// hasOrdinal is whether any segment writes an ordinal day.
	hasOrdinal bool
}

// dateFormatSegment is a part of a date format.
type dateFormatSegment struct {
	// layout is the Go layout of this segment, used unless it is an ordinal
	// day.
	layout string

	// ordinal is whether this segment is the day with its ordinal suffix.
	ordinal bool
}

// NewDateFormat prepares a date format as used by Logseq for formatting and
// parsing. Anything that is not a token is kept as it is.
func NewDateFormat(dateFormat string) *DateFormat {
	f := &DateFormat{}

	var layout strings.Builder
	// literal collects everything since the last ordinal day, so a segment
	// covers as much of the format as one Go layout can.
	var literal strings.Builder

	flush := func() {
		if literal.Len() == 0 {
			return
		}

		f.segments = append(f.segments, dateFormatSegment{layout: literal.String()})
		literal.Reset()
	}

	for i := 0; i < len(dateFormat); {
		// Tokens are replaced where they are found instead of one token at a
		// time over the whole format, so a replacement can not be picked up as
		// a token of its own.
		matched := false
		for _, t := range dateFormatTokens {
			if !strings.HasPrefix(dateFormat[i:], t.token) {
				continue
			}

			layout.WriteString(t.layout)
			if t.ordinal {
				flush()
				f.segments = append(f.segments, dateFormatSegment{ordinal: true})
				f.hasOrdinal = true
			} else {
				literal.WriteString(t.layout)
			}

			i += len(t.token)
			matched = true
			break
		}

		if !matched {
			layout.WriteByte(dateFormat[i])
			literal.WriteByte(dateFormat[i])
			i++
		}
	}

	flush()

	f.layout = layout.String()
	return f
}

// Format writes a date in this format.
func (f *DateFormat) Format(date time.Time) string {
	if !f.hasOrdinal {
		return date.Format(f.layout)
	}

	var out strings.Builder
	for _, segment := range f.segments {
		if segment.ordinal {
			out.WriteString(ordinalDay(date.Day()))
		} else {
			out.WriteString(date.Format(segment.layout))
		}
	}

	return out.String()
}

// Parse reads a date written in this format. Dates are interpreted in the local
// time zone, as the dates this format is used for are calendar days.
func (f *DateFormat) Parse(value string) (time.Time, error) {
	if f.hasOrdinal {
		// Go has no way to parse an ordinal suffix, so the day is left as the
		// plain number the layout expects.
		value = ordinalSuffixes.ReplaceAllString(value, "$1")
	}

	return time.ParseInLocation(f.layout, value, time.Local)
}

// ordinalDay writes a day of the month with the English ordinal suffix Logseq
// uses for `do`, such as `1st` or `15th`.
func ordinalDay(day int) string {
	suffix := "th"

	// The teens all end in `th`, everything else goes by the last digit.
	if day < 11 || day > 13 {
		switch day % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}

	return strconv.Itoa(day) + suffix
}
