package markdown_test

import (
	"time"

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

	Describe("Completing", func() {
		at := time.Date(2024, time.January, 10, 12, 0, 0, 0, time.Local)
		complete := func(block *content.Block) {
			block.CompleteTask(at)
		}

		It("marks a task as done", func() {
			Expect(editTask("- TODO Task", complete)).To(Equal("- DONE Task"))
		})

		It("stops a running clock", func() {
			out := editTask("- DOING Task\n  :LOGBOOK:\n  CLOCK: [2024-01-10 Wed 11:00:00]\n  :END:", complete)

			Expect(out).To(Equal("- DONE Task\n  :LOGBOOK:\n  CLOCK: [2024-01-10 Wed 11:00:00]--[2024-01-10 Wed 12:00:00] =>  01:00:00\n  :END:"))
		})

		It("repeats a task with a new logbook", func() {
			out := editTask("- TODO Task\n  SCHEDULED: <2024-01-01 Mon .+1d>", complete)

			Expect(out).To(Equal("- TODO Task\n  SCHEDULED: <2024-01-11 Thu .+1d>\n  :LOGBOOK:\n  * State \"DONE\" from \"TODO\" [2024-01-10 Wed 12:00:00]\n  :END:"))
		})

		It("repeats a task with properties and sub blocks", func() {
			out := editTask("- NOW Task\n  DEADLINE: <2024-01-01 Mon 09:00 +1w>\n  id:: 1\n\t- Child", complete)

			Expect(out).To(Equal("- LATER Task\n  DEADLINE: <2024-01-08 Mon 09:00 +1w>\n  id:: 1\n  :LOGBOOK:\n  * State \"DONE\" from \"NOW\" [2024-01-10 Wed 12:00:00]\n  :END:\n\t- Child"))
		})

		It("adds to a logbook that is already there", func() {
			out := editTask("- TODO Task\n  SCHEDULED: <2024-01-09 Tue ++1d>\n  :LOGBOOK:\n  * State \"DONE\" from \"TODO\" [2024-01-08 Mon 12:00:00]\n  :END:", complete)

			Expect(out).To(Equal("- TODO Task\n  SCHEDULED: <2024-01-10 Wed ++1d>\n  :LOGBOOK:\n  * State \"DONE\" from \"TODO\" [2024-01-08 Mon 12:00:00]\n  * State \"DONE\" from \"TODO\" [2024-01-10 Wed 12:00:00]\n  :END:"))
		})
	})
})
