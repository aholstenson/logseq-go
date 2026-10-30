package utils

import (
	"strconv"
	"strings"
	"time"
)

var weekdayNames = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"sun":       time.Sunday,
	"monday":    time.Monday,
	"mon":       time.Monday,
	"tuesday":   time.Tuesday,
	"tue":       time.Tuesday,
	"tues":      time.Tuesday,
	"wednesday": time.Wednesday,
	"wed":       time.Wednesday,
	"thursday":  time.Thursday,
	"thu":       time.Thursday,
	"thur":      time.Thursday,
	"thurs":     time.Thursday,
	"friday":    time.Friday,
	"fri":       time.Friday,
	"saturday":  time.Saturday,
	"sat":       time.Saturday,
}

var numberWords = map[string]int{
	"a":      1,
	"an":     1,
	"one":    1,
	"two":    2,
	"three":  3,
	"four":   4,
	"five":   5,
	"six":    6,
	"seven":  7,
	"eight":  8,
	"nine":   9,
	"ten":    10,
	"eleven": 11,
	"twelve": 12,
}

// ParseRelativeDate reads a date written in words, such as `next friday` or
// `in 3 days`, relative to the day of ref. Logseq reads these dates with the
// chrono library, and this reads the most common forms the same way:
//
//   - `today`, `tomorrow` and `yesterday`
//   - a weekday, such as `friday`, which is the closest one, before or after
//   - `this friday`, `next friday` and `last friday`
//   - `next week`, `last month` and the same with `year`
//   - `in 3 days`, `2 weeks ago` and `a month from now`
//   - a date such as `2026-11-05`
//
// Adding months or years to a day that the target month does not have gives
// the last day of that month, so `next month` on January 31 is the end of
// February. Chrono does the same.
//
// The returned time is midnight of the date, in the location of ref. If the
// text is not a date this returns false.
func ParseRelativeDate(text string, ref time.Time) (time.Time, bool) {
	ref = time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location())

	text = strings.ToLower(strings.TrimSpace(text))
	words := strings.Fields(text)

	switch len(words) {
	case 1:
		switch words[0] {
		case "today":
			return ref, true
		case "tomorrow":
			return ref.AddDate(0, 0, 1), true
		case "yesterday":
			return ref.AddDate(0, 0, -1), true
		}

		if weekday, ok := weekdayNames[words[0]]; ok {
			return ref.AddDate(0, 0, daysToClosestWeekday(ref.Weekday(), weekday)), true
		}

		if date, err := time.ParseInLocation("2006-01-02", words[0], ref.Location()); err == nil {
			return date, true
		}
	case 2:
		if weekday, ok := weekdayNames[words[1]]; ok {
			switch words[0] {
			case "this":
				return ref.AddDate(0, 0, daysForwardToWeekday(ref.Weekday(), weekday)), true
			case "next":
				return ref.AddDate(0, 0, daysToNextWeekday(ref.Weekday(), weekday)), true
			case "last":
				return ref.AddDate(0, 0, daysBackToWeekday(ref.Weekday(), weekday)), true
			}
		}

		var sign int
		switch words[0] {
		case "next":
			sign = 1
		case "last":
			sign = -1
		default:
			return time.Time{}, false
		}

		return addUnits(ref, sign, words[1])
	case 3:
		if words[0] == "in" {
			// in 3 days
			n, ok := parseCount(words[1])
			if !ok {
				return time.Time{}, false
			}

			return addUnits(ref, n, words[2])
		}

		if words[2] == "ago" {
			// 3 days ago
			n, ok := parseCount(words[0])
			if !ok {
				return time.Time{}, false
			}

			return addUnits(ref, -n, words[1])
		}
	case 4:
		if words[2] == "from" && words[3] == "now" {
			// 3 days from now
			n, ok := parseCount(words[0])
			if !ok {
				return time.Time{}, false
			}

			return addUnits(ref, n, words[1])
		}
	}

	return time.Time{}, false
}

// parseCount reads a count written as digits or as a word, such as `3` or
// `three`.
func parseCount(word string) (int, bool) {
	if n, ok := numberWords[word]; ok {
		return n, true
	}

	n, err := strconv.Atoi(word)
	if err != nil || n < 0 {
		return 0, false
	}

	return n, true
}

// addUnits moves date by n of the unit, such as `days` or `week`.
func addUnits(date time.Time, n int, unit string) (time.Time, bool) {
	switch strings.TrimSuffix(unit, "s") {
	case "day":
		return date.AddDate(0, 0, n), true
	case "week":
		return date.AddDate(0, 0, 7*n), true
	case "month":
		return addMonthsClamped(date, n), true
	case "year":
		return addMonthsClamped(date, 12*n), true
	}

	return time.Time{}, false
}

// daysForwardToWeekday counts the days from one weekday forward to the next
// time it is the other weekday, which is zero for the same weekday.
func daysForwardToWeekday(from time.Weekday, to time.Weekday) int {
	days := int(to) - int(from)
	if days < 0 {
		days += 7
	}

	return days
}

// daysBackToWeekday counts the days back from one weekday to the last time it
// was the other weekday, which is a week back for the same weekday.
func daysBackToWeekday(from time.Weekday, to time.Weekday) int {
	days := int(to) - int(from)
	if days >= 0 {
		days -= 7
	}

	return days
}

// daysToClosestWeekday counts the days to the closest time it is the weekday,
// which can be before or after. This is what a weekday on its own means.
func daysToClosestWeekday(from time.Weekday, to time.Weekday) int {
	forward := daysForwardToWeekday(from, to)
	backward := daysBackToWeekday(from, to)
	if forward < -backward {
		return forward
	}

	return backward
}

// daysToNextWeekday counts the days to the weekday in `next friday`, which is
// the weekday of the next week. Weeks start on Sunday, as they do in chrono.
func daysToNextWeekday(from time.Weekday, to time.Weekday) int {
	switch from {
	case time.Sunday:
		if to == time.Sunday {
			return 7
		}

		return int(to)
	case time.Saturday:
		if to == time.Saturday {
			return 7
		}

		if to == time.Sunday {
			return 8
		}

		return 1 + int(to)
	}

	if to < from && to != time.Sunday {
		return daysForwardToWeekday(from, to)
	}

	return daysForwardToWeekday(from, to) + 7
}
