package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A block can have nothing on its bullet line, as in "-\n  a". Logseq reads
// a task marker or a priority only on the bullet line. When a task status
// or a priority is set with the API, it must be written there.
var _ = Describe("Task markers on a block with an empty bullet line", func() {
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

	It("writes a new task marker on the bullet line", func() {
		// Given a block with an empty bullet line and the text `a` on the next line
		root := parse("-\n  a")

		// When the task status is set to TODO and the block is written
		root.Blocks()[0].SetTaskStatus(content.TaskStatusTodo)
		output := write(root)

		// Then the marker and the text are on the bullet line
		Expect(output).To(Equal("- TODO a"))

		// And it reads back as a TODO task
		Expect(parse(output).Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("writes a new priority on the bullet line", func() {
		// Given a block with an empty bullet line and the text `a` on the next line
		root := parse("-\n  a")

		// When the priority is set to A and the block is written
		root.Blocks()[0].SetPriority(content.PriorityA)

		// Then the priority and the text are on the bullet line
		Expect(write(root)).To(Equal("- [#A] a"))
	})

	It("writes a new task marker before the text of a block with text on the bullet line", func() {
		// Given a block with the text `a` on the bullet line
		root := parse("- a")

		// When the task status is set to TODO and the block is written
		root.Blocks()[0].SetTaskStatus(content.TaskStatusTodo)

		// Then the marker is before the text
		Expect(write(root)).To(Equal("- TODO a"))
	})
})
