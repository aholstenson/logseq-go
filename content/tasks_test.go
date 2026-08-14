package content_test

import (
	"time"

	"github.com/aholstenson/logseq-go/content"
	. "github.com/aholstenson/logseq-go/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tasks", func() {
	date := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)
	otherDate := time.Date(2024, time.January, 20, 0, 0, 0, 0, time.Local)

	Describe("Dates", func() {
		It("drops the time of day of a date without a time", func() {
			scheduled := content.NewScheduled(time.Date(2024, time.January, 15, 9, 30, 0, 0, time.Local))

			Expect(scheduled.Date).To(Equal(date))
			Expect(scheduled.HasTime).To(BeFalse())
		})

		It("keeps the time of day of a date with a time", func() {
			scheduled := content.NewScheduledWithTime(time.Date(2024, time.January, 15, 9, 30, 12, 500, time.Local))

			Expect(scheduled.Date).To(Equal(time.Date(2024, time.January, 15, 9, 30, 0, 0, time.Local)))
			Expect(scheduled.HasTime).To(BeTrue())
		})
	})

	Describe("Dates on blocks", func() {
		It("has no dates on an empty block", func() {
			block := content.NewBlock()

			Expect(block.Scheduled()).To(BeNil())
			Expect(block.Deadline()).To(BeNil())
		})

		It("can get the dates of a block", func() {
			scheduled := content.NewScheduled(date)
			deadline := content.NewDeadline(otherDate)
			block := content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
				scheduled,
				deadline,
			)

			Expect(block.Scheduled()).To(BeIdenticalTo(scheduled))
			Expect(block.Deadline()).To(BeIdenticalTo(deadline))
		})

		It("adds a date after the content of the block", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
			)
			block.SetScheduled(content.NewScheduled(date))

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
				content.NewScheduled(date),
			)))
		})

		It("adds a date after the properties and content of the block", func() {
			block := content.NewBlock(
				content.NewProperties(content.NewProperty("id", content.NewText("1"))),
				content.NewParagraph(content.NewText("Task")),
			)
			block.SetDeadline(content.NewDeadline(date))

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewProperties(content.NewProperty("id", content.NewText("1"))),
				content.NewParagraph(content.NewText("Task")),
				content.NewDeadline(date),
			)))
		})

		It("adds a date before the sub blocks", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
				content.NewBlock(content.NewParagraph(content.NewText("Child"))),
			)
			block.SetScheduled(content.NewScheduled(date))

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
				content.NewScheduled(date),
				content.NewBlock(content.NewParagraph(content.NewText("Child"))),
			)))
		})

		It("adds a deadline after a scheduled date", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
				content.NewScheduled(date),
			)
			block.SetDeadline(content.NewDeadline(otherDate))

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
				content.NewScheduled(date),
				content.NewDeadline(otherDate),
			)))
		})

		It("replaces a date that is already there", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
				content.NewScheduled(date),
			)
			block.SetScheduled(content.NewScheduled(otherDate))

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
				content.NewScheduled(otherDate),
			)))
		})

		It("removes a date when set to nil", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
				content.NewScheduled(date),
			)
			block.SetScheduled(nil)

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(content.NewText("Task")),
			)))
		})
	})
})
