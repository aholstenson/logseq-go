package utils_test

import (
	"time"

	"github.com/aholstenson/logseq-go/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseRelativeDate", func() {
	// wednesday is in the middle of the week, so that weekdays before and
	// after it can be tested. The clock time is not part of the result.
	wednesday := time.Date(2026, 10, 28, 15, 30, 0, 0, time.UTC)

	day := func(month time.Month, d int) time.Time {
		return time.Date(2026, month, d, 0, 0, 0, 0, time.UTC)
	}

	Parses := func(text string, ref time.Time, expected time.Time) {
		It(text, func() {
			date, ok := utils.ParseRelativeDate(text, ref)
			Expect(ok).To(BeTrue())
			Expect(date).To(Equal(expected))
		})
	}

	DoesNotParse := func(text string) {
		It(text, func() {
			_, ok := utils.ParseRelativeDate(text, wednesday)
			Expect(ok).To(BeFalse())
		})
	}

	Describe("Days", func() {
		Parses("today", wednesday, day(10, 28))
		Parses("tomorrow", wednesday, day(10, 29))
		Parses("yesterday", wednesday, day(10, 27))
		Parses("Tomorrow", wednesday, day(10, 29))
		Parses("2026-11-05", wednesday, day(11, 5))
	})

	Describe("Weekdays", func() {
		Parses("friday", wednesday, day(10, 30))
		Parses("monday", wednesday, day(10, 26))
		Parses("wednesday", wednesday, day(10, 28))
		Parses("fri", wednesday, day(10, 30))
		Parses("this friday", wednesday, day(10, 30))
		Parses("this monday", wednesday, day(11, 2))
		Parses("next friday", wednesday, day(11, 6))
		Parses("next monday", wednesday, day(11, 2))
		Parses("next sunday", wednesday, day(11, 8))
		Parses("next wednesday", wednesday, day(11, 4))
		Parses("last friday", wednesday, day(10, 23))
		Parses("last wednesday", wednesday, day(10, 21))

		Describe("from a Sunday", func() {
			sunday := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
			Parses("next friday", sunday, day(11, 6))
			Parses("next sunday", sunday, day(11, 8))
		})

		Describe("from a Saturday", func() {
			saturday := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
			Parses("next friday", saturday, day(11, 6))
			Parses("next saturday", saturday, day(11, 7))
			Parses("next sunday", saturday, day(11, 8))
		})
	})

	Describe("Periods", func() {
		Parses("next week", wednesday, day(11, 4))
		Parses("last month", wednesday, day(9, 28))
		Parses("next year", wednesday, time.Date(2027, 10, 28, 0, 0, 0, 0, time.UTC))
		Parses("in 3 days", wednesday, day(10, 31))
		Parses("in one week", wednesday, day(11, 4))
		Parses("two weeks ago", wednesday, day(10, 14))
		Parses("1 day ago", wednesday, day(10, 27))
		Parses("a month from now", wednesday, day(11, 28))
	})

	Describe("Other text", func() {
		DoesNotParse("")
		DoesNotParse("current page")
		DoesNotParse("banana")
		DoesNotParse("in many days")
		DoesNotParse("next fortnight")
	})
})
