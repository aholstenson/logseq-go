package logseq

import (
	"time"

	"github.com/aholstenson/logseq-go/internal/utils"
)

// DateInput is the date or time that a query input such as `:today` or
// `:3d-after` stands for. See ParseDateInput.
type DateInput = utils.DateInput

// ParseDateInput reads one of the date inputs of a Logseq query, such as the
// `:today` in `:inputs [:today]`, relative to now. Pass time.Now() for now, so
// that the dates are in the local time zone, the same as the dates that
// queries such as ScheduledBetween and JournalBetween match on. The leading
// colon is optional. These inputs are read the same way as Logseq does:
//
//   - `today`, `yesterday` and `tomorrow`
//   - an offset from today in days, weeks, months or years, such as `+3d`,
//     `-1w`, `+2m` or `-1y`
//   - an offset with a time of day, such as `+3d-1230` for 12:30 three days
//     from today. The time is `start`, `end`, or two, four, six or nine digits
//     as hours, minutes, seconds and milliseconds. `ms` is the end of the day
//     for an offset into the future and the start of the day otherwise
//   - `today-` with a time of day, such as `today-end`
//   - `right-now-ms`, `start-of-today-ms` and `end-of-today-ms`
//   - the older forms `3d` and `3d-before` for three days ago, and
//     `3d-after` for three days from today, with an optional `-ms`
//
// Inputs that are a whole day give midnight of that day. Adding months or
// years to a day that the target month does not have gives the last day of
// that month, so one month after January 31 is the end of February.
//
// Returns false for inputs that are not dates, such as `:current-page`.
func ParseDateInput(input string, now time.Time) (DateInput, bool) {
	return utils.ParseQueryDateInput(input, now)
}
