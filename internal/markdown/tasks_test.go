package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// editTask parses the input, changes its first block and returns the output.
func editTask(input string, edit func(block *content.Block)) string {
	root, err := markdown.ParseString(input)
	Expect(err).ToNot(HaveOccurred())

	blocks := root.Blocks()
	Expect(blocks).ToNot(BeEmpty())
	edit(blocks[0])

	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	return v
}

var _ = Describe("Editing tasks", func() {
	It("reads the status and priority of a parsed task", func() {
		root, err := markdown.ParseString("- DOING [#B] Task")
		Expect(err).ToNot(HaveOccurred())

		block := root.Blocks()[0]
		Expect(block.TaskStatus()).To(Equal(content.TaskStatusDoing))
		Expect(block.Priority()).To(Equal(content.PriorityB))
	})

	It("reads the status of a task with properties", func() {
		root, err := markdown.ParseString("- TODO Task\n  id:: 1")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("changes the status of a task", func() {
		out := editTask("- TODO [#A] Task\n  SCHEDULED: <2024-01-15 Mon>", func(block *content.Block) {
			block.SetTaskStatus(content.TaskStatusDone)
		})

		Expect(out).To(Equal("- DONE [#A] Task\n  SCHEDULED: <2024-01-15 Mon>"))
	})

	It("makes a block a task", func() {
		out := editTask("- Task", func(block *content.Block) {
			block.SetTaskStatus(content.TaskStatusLater)
			block.SetPriority(content.PriorityC)
		})

		Expect(out).To(Equal("- LATER [#C] Task"))
	})

	It("removes the status of a task", func() {
		out := editTask("- TODO Task", func(block *content.Block) {
			block.SetTaskStatus(content.TaskStatusNone)
		})

		Expect(out).To(Equal("- Task"))
	})

	It("removes the priority of a task", func() {
		out := editTask("- TODO [#A] Task", func(block *content.Block) {
			block.SetPriority(content.PriorityNone)
		})

		Expect(out).To(Equal("- TODO Task"))
	})
})
