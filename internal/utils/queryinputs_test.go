package utils_test

import (
	"time"

	"github.com/aholstenson/logseq-go/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseQueryDateInput", func() {
	// now is in the afternoon at the end of a month, so that the time of day
	// is left out of dates and months of different lengths can be tested.
	now := time.Date(2026, 1, 31, 15, 30, 45, 0, time.UTC)

	day := func(year int, month time.Month, d int) time.Time {
		return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
	}

	at := func(month time.Month, d int, hour int, minute int, second int, ms int) time.Time {
		return time.Date(2026, month, d, hour, minute, second, ms*int(time.Millisecond), time.UTC)
	}

	ParsesDate := func(input string, expected time.Time) {
		It(input, func() {
			result, ok := utils.ParseQueryDateInput(input, now)
			Expect(ok).To(BeTrue())
			Expect(result.Time).To(Equal(expected))
			Expect(result.HasTime).To(BeFalse())
		})
	}

	ParsesTime := func(input string, expected time.Time) {
		It(input, func() {
			result, ok := utils.ParseQueryDateInput(input, now)
			Expect(ok).To(BeTrue())
			Expect(result.Time).To(Equal(expected))
			Expect(result.HasTime).To(BeTrue())
		})
	}

	DoesNotParse := func(input string) {
		It(input, func() {
			_, ok := utils.ParseQueryDateInput(input, now)
			Expect(ok).To(BeFalse())
		})
	}

	Describe("Days", func() {
		ParsesDate(":today", day(2026, 1, 31))
		ParsesDate("today", day(2026, 1, 31))
		ParsesDate(":yesterday", day(2026, 1, 30))
		ParsesDate(":tomorrow", day(2026, 2, 1))
	})

	Describe("Offsets", func() {
		ParsesDate(":+3d", day(2026, 2, 3))
		ParsesDate(":-3d", day(2026, 1, 28))
		ParsesDate(":+0d", day(2026, 1, 31))
		ParsesDate(":+1w", day(2026, 2, 7))
		ParsesDate(":-2w", day(2026, 1, 17))
		ParsesDate(":-1y", day(2025, 1, 31))
		ParsesDate(":+12m", day(2027, 1, 31))
	})

	Describe("Offsets in months and years", func() {
		ParsesDate(":+1m", day(2026, 2, 28))
		ParsesDate(":-2m", day(2025, 11, 30))
		ParsesDate(":+3m", day(2026, 4, 30))

		It("keeps the last day of February in leap years", func() {
			result, ok := utils.ParseQueryDateInput(":+1y", time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC))
			Expect(ok).To(BeTrue())
			Expect(result.Time).To(Equal(day(2025, 2, 28)))
		})
	})

	Describe("Older forms", func() {
		ParsesDate(":3d", day(2026, 1, 28))
		ParsesDate(":14d", day(2026, 1, 17))
		ParsesDate(":3d-before", day(2026, 1, 28))
		ParsesDate(":3d-after", day(2026, 2, 3))
		ParsesTime(":3d-before-ms", at(1, 28, 0, 0, 0, 0))
		ParsesTime(":3d-after-ms", at(2, 3, 23, 59, 59, 999))
	})

	Describe("Times", func() {
		ParsesTime(":right-now-ms", now)
		ParsesTime(":start-of-today-ms", at(1, 31, 0, 0, 0, 0))
		ParsesTime(":end-of-today-ms", at(1, 31, 23, 59, 59, 999))
		ParsesTime(":today-start", at(1, 31, 0, 0, 0, 0))
		ParsesTime(":today-end", at(1, 31, 23, 59, 59, 999))
		ParsesTime(":today-09", at(1, 31, 9, 0, 0, 0))
		ParsesTime(":today-1230", at(1, 31, 12, 30, 0, 0))
		ParsesTime(":today-123045", at(1, 31, 12, 30, 45, 0))
		ParsesTime(":today-123045250", at(1, 31, 12, 30, 45, 250))
		ParsesTime(":today-9999", at(1, 31, 23, 59, 0, 0))
	})

	Describe("Offsets with times", func() {
		ParsesTime(":+3d-1230", at(2, 3, 12, 30, 0, 0))
		ParsesTime(":-1d-start", at(1, 30, 0, 0, 0, 0))
		ParsesTime(":-1d-end", at(1, 30, 23, 59, 59, 999))
		ParsesTime(":+1d-ms", at(2, 1, 23, 59, 59, 999))
		ParsesTime(":-1d-ms", at(1, 30, 0, 0, 0, 0))
	})

	Describe("Not dates", func() {
		DoesNotParse(":current-page")
		DoesNotParse(":query-page")
		DoesNotParse(":current-block")
		DoesNotParse(":parent-block")
		DoesNotParse("")
		DoesNotParse(":+3")
		DoesNotParse(":3")
		DoesNotParse(":+3x")
		DoesNotParse(":3w-after")
		DoesNotParse(":today-ms")
		DoesNotParse(":today-1")
		DoesNotParse(":+3d-123")
		DoesNotParse(":+3d-")
		DoesNotParse(":Today")
		DoesNotParse(":+99999999999999999999d")
	})

	It("uses the location of now", func() {
		zone := time.FixedZone("UTC+9", 9*60*60)
		result, ok := utils.ParseQueryDateInput(":tomorrow", time.Date(2026, 1, 31, 23, 0, 0, 0, zone))
		Expect(ok).To(BeTrue())
		Expect(result.Time).To(Equal(time.Date(2026, 2, 1, 0, 0, 0, 0, zone)))
	})
})
