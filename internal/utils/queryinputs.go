package utils

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DateInput is the date or time that a query input such as `:today` or
// `:3d-after` stands for.
type DateInput struct {
	// Time is midnight of the date for inputs that are a whole day, and the
	// exact time for inputs that have a time of day.
	Time time.Time

	// HasTime is true for inputs that have a time of day, such as
	// `:right-now-ms` or `:+1d-1230`. Logseq compares these with timestamps,
	// and the other inputs with dates such as the journal day of a page.
	HasTime bool
}

var (
	relativeDateInput     = regexp.MustCompile(`^([+-])(\d+)([dwmy])$`)
	relativeDateTimeInput = regexp.MustCompile(`^([+-])(\d+)([dwmy])-(ms|start|end|\d{2}|\d{4}|\d{6}|\d{9})$`)
	todayTimeInput        = regexp.MustCompile(`^today-(start|end|\d{2}|\d{4}|\d{6}|\d{9})$`)
	legacyRelativeInput   = regexp.MustCompile(`^(\d+)d(-before|-after|-before-ms|-after-ms)?$`)
)

// ParseQueryDateInput reads one of the date inputs of a Logseq query, such as
// the `:today` in `:inputs [:today]`, relative to now. The leading colon is
// optional. These inputs are read the same way as Logseq does:
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
// Dates and times are in the location of now. Adding months or years to a day
// that the target month does not have gives the last day of that month, so
// one month after January 31 is the end of February.
//
// Returns false for inputs that are not dates, such as `:current-page`.
func ParseQueryDateInput(input string, now time.Time) (DateInput, bool) {
	input = strings.TrimPrefix(input, ":")
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	switch input {
	case "today":
		return DateInput{Time: today}, true
	case "yesterday":
		return DateInput{Time: today.AddDate(0, 0, -1)}, true
	case "tomorrow":
		return DateInput{Time: today.AddDate(0, 0, 1)}, true
	case "right-now-ms":
		return DateInput{Time: now, HasTime: true}, true
	case "start-of-today-ms":
		return DateInput{Time: atTimeOfDay(today, "start", ""), HasTime: true}, true
	case "end-of-today-ms":
		return DateInput{Time: atTimeOfDay(today, "end", ""), HasTime: true}, true
	}

	if m := legacyRelativeInput.FindStringSubmatch(input); m != nil {
		// The older forms are written in the newer form and read again, as
		// Logseq does.
		sign := "-"
		if strings.Contains(m[2], "after") {
			sign = "+"
		}

		rewritten := sign + m[1] + "d"
		if strings.HasSuffix(m[2], "-ms") {
			rewritten += "-ms"
		}

		return ParseQueryDateInput(rewritten, now)
	}

	if m := relativeDateInput.FindStringSubmatch(input); m != nil {
		date, ok := offsetDate(today, m[1], m[2], m[3])
		return DateInput{Time: date}, ok
	}

	if m := relativeDateTimeInput.FindStringSubmatch(input); m != nil {
		date, ok := offsetDate(today, m[1], m[2], m[3])
		if !ok {
			return DateInput{}, false
		}

		return DateInput{Time: atTimeOfDay(date, m[4], m[1]), HasTime: true}, true
	}

	if m := todayTimeInput.FindStringSubmatch(input); m != nil {
		return DateInput{Time: atTimeOfDay(today, m[1], ""), HasTime: true}, true
	}

	return DateInput{}, false
}

// offsetDate moves a date by a number of days, weeks, months or years in the
// direction of sign, which is `+` or `-`.
func offsetDate(date time.Time, sign string, count string, unit string) (time.Time, bool) {
	n, err := strconv.Atoi(count)
	if err != nil {
		return time.Time{}, false
	}

	if sign == "-" {
		n = -n
	}

	switch unit {
	case "d":
		return date.AddDate(0, 0, n), true
	case "w":
		return date.AddDate(0, 0, 7*n), true
	case "m":
		return addMonthsClamped(date, n), true
	case "y":
		return addMonthsClamped(date, 12*n), true
	}

	return time.Time{}, false
}

// addMonthsClamped adds months to a date and keeps it in the target month. A
// day that the target month does not have becomes the last day of that month,
// unlike time.AddDate which moves it into the month after.
func addMonthsClamped(date time.Time, months int) time.Time {
	first := time.Date(date.Year(), date.Month()+time.Month(months), 1, 0, 0, 0, 0, date.Location())
	lastDay := first.AddDate(0, 1, -1).Day()

	day := date.Day()
	if day > lastDay {
		day = lastDay
	}

	return time.Date(first.Year(), first.Month(), day, date.Hour(), date.Minute(), date.Second(), date.Nanosecond(), date.Location())
}

// atTimeOfDay returns the time of day on a date, where the time is `start`,
// `end`, `ms` or digits as hours, minutes, seconds and milliseconds. `ms` is
// the end of the day for the sign `+` and the start of the day otherwise. Each
// part of the digits is capped at its highest value, so `99` is 23:00.
func atTimeOfDay(date time.Time, value string, sign string) time.Time {
	var parts [4]int
	switch value {
	case "start":
	case "end":
		parts = [4]int{23, 59, 59, 999}
	case "ms":
		if sign == "+" {
			parts = [4]int{23, 59, 59, 999}
		}
	default:
		digits := value + strings.Repeat("0", 9-len(value))
		limits := [4]int{23, 59, 59, 999}
		bounds := [5]int{0, 2, 4, 6, 9}
		for i := range parts {
			n, _ := strconv.Atoi(digits[bounds[i]:bounds[i+1]])
			if n > limits[i] {
				n = limits[i]
			}
			parts[i] = n
		}
	}

	return time.Date(date.Year(), date.Month(), date.Day(), parts[0], parts[1], parts[2], parts[3]*int(time.Millisecond), date.Location())
}
