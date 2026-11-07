package markdown_test

import (
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A task can be a heading, such as `- ## TODO h`. When it is scheduled or
// clocked with the API, the date or the logbook must go after the heading
// line, as for a task in a paragraph, so that the block stays one task.
var _ = Describe("Dates and clocks of heading tasks", func() {
	parse := func(input string) *content.Block {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		return root
	}

	write := func(root *content.Block) string {
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		return v
	}

	scheduled := func() *content.TaskDate {
		return content.NewTaskDate(content.TaskDateTypeScheduled, time.Date(2024, 1, 2, 0, 0, 0, 0, time.Local))
	}

	It("puts the scheduled date after the heading line", func() {
		// Given a heading task
		root := parse("- ## TODO h")

		// When the task is scheduled and the block is written
		root.Blocks()[0].SetScheduled(scheduled())
		output := write(root)

		// Then the date is on the line after the heading
		Expect(output).To(Equal("- ## TODO h\n  SCHEDULED: <2024-01-02 Tue>"))

		// And the date is not on the bullet line
		Expect(output).ToNot(HavePrefix("- SCHEDULED"))
	})

	It("puts the logbook of a clocked heading task after the heading line", func() {
		// Given a heading task
		root := parse("- ## TODO h")

		// When the task is set to DOING with a clock and the block is written
		root.Blocks()[0].SetTaskStatusWithClock(content.TaskStatusDoing, time.Date(2024, 1, 2, 10, 0, 0, 0, time.Local))
		output := write(root)

		// Then the logbook is on the lines after the heading
		Expect(output).To(Equal("- ## DOING h\n  :LOGBOOK:\n  CLOCK: [2024-01-02 Tue 10:00:00]\n  :END:"))

		// And it reads back as one block with one heading that is a DOING task
		again := parse(output)
		Expect(again.Blocks()).To(HaveLen(1), "output: %q", output)
		block := again.Blocks()[0]
		Expect(block.Blocks()).To(BeEmpty(), "output: %q", output)
		Expect(block.Children().FilterDeep(content.IsOfType[*content.Heading]())).To(HaveLen(1), "output: %q", output)
		Expect(block.TaskStatus()).To(Equal(content.TaskStatusDoing), "output: %q", output)
	})

	It("puts the scheduled date of a paragraph task after the task line", func() {
		// Given a task in a paragraph
		root := parse("- TODO x")

		// When the task is scheduled and the block is written
		root.Blocks()[0].SetScheduled(scheduled())

		// Then the date is on the line after the task
		Expect(write(root)).To(Equal("- TODO x\n  SCHEDULED: <2024-01-02 Tue>"))
	})
})
