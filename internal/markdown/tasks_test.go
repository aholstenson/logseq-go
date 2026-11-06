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

	Describe("Clocks", func() {
		start := time.Date(2024, time.January, 10, 11, 0, 0, 0, time.Local)
		stop := start.Add(90 * time.Minute)

		It("starts and stops a clock", func() {
			out := editTask("- TODO Task\n  SCHEDULED: <2024-01-10 Wed>", func(block *content.Block) {
				block.SetTaskStatusWithClock(content.TaskStatusDoing, start)
			})
			Expect(out).To(Equal("- DOING Task\n  SCHEDULED: <2024-01-10 Wed>\n  :LOGBOOK:\n  CLOCK: [2024-01-10 Wed 11:00:00]\n  :END:"))

			out = editTask(out, func(block *content.Block) {
				block.SetTaskStatusWithClock(content.TaskStatusTodo, stop)
			})
			Expect(out).To(Equal("- TODO Task\n  SCHEDULED: <2024-01-10 Wed>\n  :LOGBOOK:\n  CLOCK: [2024-01-10 Wed 11:00:00]--[2024-01-10 Wed 12:30:00] =>  01:30:00\n  :END:"))
		})
	})
})

var _ = Describe("A task marker alone on its line", func() {
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

	It("does not join the marker with the next line on save", func() {
		// Given a block with TODO alone on the first line
		input := "- TODO\n  x"
		root := parse(input)

		// Then the text does not change on save
		Expect(write(root)).To(Equal(input))

		// And Logseq reads no task, since no space follows the marker
		Expect(root.Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusNone))
	})

	It("does not join a marker and a link on the next line on save", func() {
		input := "- DONE\n  [[x]]"
		Expect(write(parse(input))).To(Equal(input))
	})

	It("keeps the space after a marker that is followed by a logbook", func() {
		// Given a DOING task with no title and a logbook
		input := "- DOING \n  :LOGBOOK:\n  CLOCK: [2024-01-01 Mon 10:00:00]\n  :END:"
		root := parse(input)

		// Then it is a DOING task
		Expect(root.Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusDoing))

		// And the space after the marker is kept on save
		Expect(write(root)).To(Equal(input))
	})

	It("keeps the line break after a marker and a space", func() {
		input := "- TODO \n  x"
		Expect(write(parse(input))).To(Equal(input))
	})

	It("keeps the space after a marker before the next block", func() {
		input := "- TODO \n- b"
		Expect(write(parse(input))).To(Equal(input))
	})

	It("does not read a marker that is followed by the next block", func() {
		// Given TODO alone on the line, then another block
		// Then Logseq reads no task
		Expect(parse("- TODO\n- b").Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusNone))
	})

	It("reads a marker at the end of the file as a task", func() {
		Expect(parse("- TODO").Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("writes a task with no title so that it is read back as a task", func() {
		// Given a task with no title made with the API
		root := content.NewBlock(content.NewBlock(content.NewParagraph(content.NewTaskMarker(content.TaskStatusTodo))))

		// When it is written, and a line break follows it in the file
		written := write(root) + "\n"

		// Then it is read back as a task
		Expect(parse(written).Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusTodo), "written text: %q", written)
	})

	It("does not read a marker that is followed by a link without a space", func() {
		Expect(parse("- TODO[[x]]").Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusNone))
	})

	It("does not read a marker that is followed by a property line", func() {
		root := parse("- TODO\n  id:: 65e5a8b0-0000-4000-8000-000000000000")
		Expect(root.Blocks()[0].TaskStatus()).To(Equal(content.TaskStatusNone))
	})
})

var _ = Describe("Task markers after a heading", func() {
	firstBlock := func(input string) *content.Block {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Blocks()).ToNot(BeEmpty())
		return root.Blocks()[0]
	}

	It("reads the marker of a heading in a block, the same as Logseq", func() {
		// Given a block whose first line is a heading with a marker
		// Then the block is a TODO task
		Expect(firstBlock("- ## TODO x").TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("reads the marker and priority of a heading in a block", func() {
		// Given a heading with a marker and a priority
		block := firstBlock("- # DONE [#A] x")

		// Then the block is a DONE task with priority A
		Expect(block.TaskStatus()).To(Equal(content.TaskStatusDone))
		Expect(block.Priority()).To(Equal(content.PriorityA))
	})

	It("reads the marker of a heading that is not in a bullet", func() {
		// Given page content that is a heading with a marker
		root, err := markdown.ParseString("## TODO x")
		Expect(err).ToNot(HaveOccurred())

		// Then that content is a TODO task
		Expect(root.TaskStatus()).To(Equal(content.TaskStatusTodo))
	})

	It("does not read a marker that is followed by a line break, the same as Logseq", func() {
		// Given a heading that is only a marker, and a next block
		// Then the heading is text and not a task
		Expect(firstBlock("- ## TODO\n- y").TaskStatus()).To(Equal(content.TaskStatusNone))
	})

	DescribeTable("writes a heading task with no change",
		func(input string) {
			root, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			v, err := markdown.AsString(root)
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(input))
		},
		Entry("marker", "- ## TODO x"),
		Entry("marker and priority", "- # DONE [#A] x"),
		Entry("priority only", "- ## [#B] x"),
	)

	It("keeps the heading when the status changes", func() {
		// Given a heading task
		// When the status is set to DONE
		v := editTask("- ## TODO x", func(block *content.Block) {
			block.SetTaskStatus(content.TaskStatusDone)
		})

		// Then the marker changes and the heading stays
		Expect(v).To(Equal("- ## DONE x"))
	})

	It("puts a new marker after the heading marks", func() {
		// Given a heading block that is not a task
		// When it is made a task
		v := editTask("- ## x", func(block *content.Block) {
			block.SetTaskStatus(content.TaskStatusTodo)
		})

		// Then the marker goes after the `#` marks, where Logseq reads it
		Expect(v).To(Equal("- ## TODO x"))
	})
})
