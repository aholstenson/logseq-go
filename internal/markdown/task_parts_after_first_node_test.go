package markdown_test

import (
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq reads the marker and the priority of a task only on the line of the
// bullet, and its dates and logbook only on the lines directly after it. A
// change through the API must put them there, also when the block starts
// with properties, a heading or a table.
var _ = Describe("Task parts added to a block that starts with another node", func() {
	at := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	type change struct {
		apply func(*content.Block)
		reads func(*content.Block) bool
	}

	changes := map[string]change{
		"priority": {
			func(b *content.Block) { b.SetPriority(content.PriorityA) },
			func(b *content.Block) bool { return b.Priority() == content.PriorityA },
		},
		"task status": {
			func(b *content.Block) { b.SetTaskStatus(content.TaskStatusTodo) },
			func(b *content.Block) bool { return b.TaskStatus() == content.TaskStatusTodo },
		},
		"scheduled date": {
			func(b *content.Block) { b.SetScheduled(content.NewScheduled(at)) },
			func(b *content.Block) bool { return b.Scheduled() != nil },
		},
		"deadline": {
			func(b *content.Block) { b.SetDeadline(content.NewDeadline(at)) },
			func(b *content.Block) bool { return b.Deadline() != nil },
		},
		"clock": {
			func(b *content.Block) { b.SetTaskStatusWithClock(content.TaskStatusDoing, at) },
			func(b *content.Block) bool { return b.RunningClock() != nil && b.TaskStatus() == content.TaskStatusDoing },
		},
		"property": {
			func(b *content.Block) { b.Properties().Set("k", content.NewText("v")) },
			func(b *content.Block) bool {
				properties := b.FindProperties()
				return properties != nil && len(properties.Get("k")) > 0
			},
		},
	}

	inputs := []string{
		"- a:: b",
		"- a:: b\n  text",
		"- a:: b\n  c:: d",
		"- a:: b\n  # h",
		"- # h",
		"- # h\n  text",
		"- # h\n  a:: b",
		"- | a | b |\n  | --- | --- |",
		"- | a | b |\n  | --- | --- |\n  a:: b",
		"- TODO x\n  SCHEDULED: <2024-01-01 Mon>\n  a:: b",
		"-\n  a:: b",
	}

	for _, input := range inputs {
		for name, c := range changes {
			It("reads back the "+name+" of "+input, func() {
				// Given the block
				root, err := markdown.ParseString(input)
				Expect(err).ToNot(HaveOccurred())

				// When the change is made and the block is written and read again
				c.apply(root.Blocks()[0])
				output, err := markdown.AsString(root)
				Expect(err).ToNot(HaveOccurred())
				again, err := markdown.ParseString(output)
				Expect(err).ToNot(HaveOccurred())

				// Then it is one block that has the change, and it is written the same again
				Expect(again.Blocks()).To(HaveLen(1), "output: %q", output)
				Expect(c.reads(again.Blocks()[0])).To(BeTrue(), "output: %q", output)
				Expect(markdown.AsString(again)).To(Equal(output))
			})
		}
	}

	It("puts the marker on the line of the bullet before properties", func() {
		// Given a block that starts with properties
		root, err := markdown.ParseString("- a:: b\n  text")
		Expect(err).ToNot(HaveOccurred())

		// When it gets a task status and a priority
		root.Blocks()[0].SetTaskStatus(content.TaskStatusTodo)
		root.Blocks()[0].SetPriority(content.PriorityA)

		// Then they are on the line of the bullet
		Expect(markdown.AsString(root)).To(Equal("- TODO [#A]\n  a:: b\n  text"))
	})

	It("puts a date after the properties that start the block", func() {
		// Given a block that starts with properties
		root, err := markdown.ParseString("- a:: b\n  text")
		Expect(err).ToNot(HaveOccurred())

		// When it gets a scheduled date
		root.Blocks()[0].SetScheduled(content.NewScheduled(at))

		// Then the date is directly after the properties
		Expect(markdown.AsString(root)).To(Equal("- a:: b\n  SCHEDULED: <2024-01-01 Mon>\n  text"))
	})
})
