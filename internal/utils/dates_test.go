package utils_test

import (
	"time"

	"github.com/aholstenson/logseq-go/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dates", func() {
	// reference is a Wednesday, so the weekday is not the same as any of the
	// other parts of the date.
	reference := time.Date(2024, 3, 5, 13, 4, 9, 0, time.UTC)

	// Formats checks that a date format converts into a layout that formats the
	// reference date as expected.
	Formats := func(dateFormat string, expected string) {
		It(dateFormat, func() {
			layout := utils.ConvertDateFormat(dateFormat)
			Expect(reference.Format(layout)).To(Equal(expected))
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
		Formats("do", "5")
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
		Formats("EEE do, MMM yyyy", "Tue 5, Mar 2024")
		Formats("EEEE, MMMM do, yyyy", "Tuesday, March 5, 2024")
		Formats("MMM do, yyyy", "Mar 5, 2024")
	})

	It("keeps text that is not a token", func() {
		layout := utils.ConvertDateFormat("[yyyy]")
		Expect(reference.Format(layout)).To(Equal("[2024]"))
	})
})
