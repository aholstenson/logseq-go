package utils_test

import (
	"time"

	"github.com/aholstenson/logseq-go/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dates", func() {
	// reference is a Tuesday, so the weekday is not the same as any of the
	// other parts of the date.
	reference := time.Date(2024, 3, 5, 13, 4, 9, 0, time.UTC)

	// Formats checks that a date format writes the reference date as expected.
	Formats := func(dateFormat string, expected string) {
		It(dateFormat, func() {
			Expect(utils.NewDateFormat(dateFormat).Format(reference)).To(Equal(expected))
		})
	}

	// FormatsDay checks that a date format writes the given day of March 2024
	// as expected, for the formats where the day is not written as a plain
	// number.
	FormatsDay := func(dateFormat string, day int, expected string) {
		It(expected, func() {
			date := time.Date(2024, 3, day, 0, 0, 0, 0, time.UTC)
			Expect(utils.NewDateFormat(dateFormat).Format(date)).To(Equal(expected))
		})
	}

	Describe("Years", func() {
		Formats("yyyy", "2024")
		Formats("yy", "24")
	})

	Describe("Months", func() {
		Formats("MMMM", "March")
		Formats("MMM", "Mar")
		Formats("MM", "03")
		Formats("M", "3")
	})

	Describe("Days", func() {
		Formats("dd", "05")
		Formats("d", "5")
		Formats("do", "5th")
	})

	Describe("Ordinal days", func() {
		FormatsDay("do", 1, "1st")
		FormatsDay("do", 2, "2nd")
		FormatsDay("do", 3, "3rd")
		FormatsDay("do", 4, "4th")
		FormatsDay("do", 11, "11th")
		FormatsDay("do", 12, "12th")
		FormatsDay("do", 13, "13th")
		FormatsDay("do", 21, "21st")
		FormatsDay("do", 22, "22nd")
		FormatsDay("do", 23, "23rd")
		FormatsDay("do", 31, "31st")
	})

	Describe("Weekdays", func() {
		Formats("EEEE", "Tuesday")
		Formats("EEE", "Tue")
	})

	Describe("Times", func() {
		Formats("HH:mm", "13:04")
		Formats("hh:mm", "01:04")
		Formats("h:mm:ss", "1:04:09")
	})

	Describe("Full formats", func() {
		Formats("yyyy_MM_dd", "2024_03_05")
		Formats("yyyy-MM-dd", "2024-03-05")
		Formats("EEE do, MMM yyyy", "Tue 5th, Mar 2024")
		Formats("EEEE, MMMM do, yyyy", "Tuesday, March 5th, 2024")
		Formats("MMM do, yyyy", "Mar 5th, 2024")
	})

	It("keeps text that is not a token", func() {
		Expect(utils.NewDateFormat("[yyyy]").Format(reference)).To(Equal("[2024]"))
	})

	Describe("Parse", func() {
		// Parses checks that a date written in a format is read back as the
		// given date in March 2024.
		Parses := func(dateFormat string, value string, day int) {
			It(value, func() {
				date, err := utils.NewDateFormat(dateFormat).Parse(value)
				Expect(err).ToNot(HaveOccurred())
				Expect(date).To(Equal(time.Date(2024, 3, day, 0, 0, 0, 0, time.Local)))
			})
		}

		Parses("yyyy_MM_dd", "2024_03_05", 5)
		Parses("yyyy-MM-dd", "2024-03-05", 5)
		Parses("MMM do, yyyy", "Mar 1st, 2024", 1)
		Parses("MMM do, yyyy", "Mar 2nd, 2024", 2)
		Parses("MMM do, yyyy", "Mar 3rd, 2024", 3)
		Parses("MMM do, yyyy", "Mar 5th, 2024", 5)
		Parses("MMM do, yyyy", "Mar 22nd, 2024", 22)
		Parses("EEEE, MMMM do, yyyy", "Tuesday, March 5th, 2024", 5)

		It("reads back what it formats", func() {
			format := utils.NewDateFormat("MMM do, yyyy")
			for day := 1; day <= 31; day++ {
				expected := time.Date(2024, 3, day, 0, 0, 0, 0, time.Local)

				date, err := format.Parse(format.Format(expected))
				Expect(err).ToNot(HaveOccurred())
				Expect(date).To(Equal(expected))
			}
		})

		It("fails on a date that does not match the format", func() {
			_, err := utils.NewDateFormat("yyyy-MM-dd").Parse("not a date")
			Expect(err).To(HaveOccurred())
		})
	})
})
