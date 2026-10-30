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

	Describe("Logbook clocks", func() {
		start := time.Date(2023, time.June, 26, 17, 25, 56, 0, time.Local)

		It("is running when it has no end", func() {
			clock := content.NewLogbookEntryClock(start, time.Time{})

			Expect(clock.IsRunning()).To(BeTrue())
			Expect(clock.Duration()).To(BeZero())
		})

		It("has a duration when it has an end", func() {
			clock := content.NewLogbookEntryClock(start, start.Add(90*time.Second))

			Expect(clock.IsRunning()).To(BeFalse())
			Expect(clock.Duration()).To(Equal(90 * time.Second))
		})

		It("drops anything smaller than a second", func() {
			clock := content.NewLogbookEntryClock(start.Add(500*time.Millisecond), time.Time{})

			Expect(clock.Start).To(Equal(start))
		})

		It("becomes running again without an end", func() {
			clock := content.NewLogbookEntryClock(start, start.Add(time.Minute)).WithEnd(time.Time{})

			Expect(clock.IsRunning()).To(BeTrue())
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

	Describe("Statuses", func() {
		It("is open for a status that is not finished", func() {
			Expect(content.TaskStatusTodo.IsOpen()).To(BeTrue())
			Expect(content.TaskStatusTodo.IsClosed()).To(BeFalse())
			Expect(content.TaskStatusWaiting.IsOpen()).To(BeTrue())
		})

		It("is closed for a status that is finished", func() {
			for _, status := range []content.TaskStatus{
				content.TaskStatusDone,
				content.TaskStatusCancelled,
				content.TaskStatusCanceled,
			} {
				Expect(status.IsClosed()).To(BeTrue())
				Expect(status.IsOpen()).To(BeFalse())
			}
		})

		It("is neither open nor closed for no status", func() {
			Expect(content.TaskStatusNone.IsOpen()).To(BeFalse())
			Expect(content.TaskStatusNone.IsClosed()).To(BeFalse())
		})
	})

	Describe("Statuses on blocks", func() {
		It("has no status on a block without a marker", func() {
			block := content.NewBlock(content.NewParagraph(content.NewText("Task")))

			Expect(block.TaskStatus()).To(Equal(content.TaskStatusNone))
			Expect(block.IsTask()).To(BeFalse())
		})

		It("can get the status of a block", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusDoing),
				content.NewText("Task"),
			))

			Expect(block.TaskStatus()).To(Equal(content.TaskStatusDoing))
			Expect(block.IsTask()).To(BeTrue())
		})

		It("finds the marker after properties", func() {
			block := content.NewBlock(
				content.NewProperties(content.NewProperty("id", content.NewText("1"))),
				content.NewParagraph(
					content.NewTaskMarker(content.TaskStatusTodo),
					content.NewText("Task"),
				),
			)

			Expect(block.TaskStatus()).To(Equal(content.TaskStatusTodo))
		})

		It("ignores a marker that is not at the start of the paragraph", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewText("Task"),
				content.NewTaskMarker(content.TaskStatusTodo),
			))

			Expect(block.TaskStatus()).To(Equal(content.TaskStatusNone))
		})

		It("ignores a marker in a paragraph that is not the first", func() {
			block := content.NewBlock(
				content.NewCodeBlock("code"),
				content.NewParagraph(
					content.NewTaskMarker(content.TaskStatusTodo),
					content.NewText("Task"),
				),
			)

			Expect(block.TaskStatus()).To(Equal(content.TaskStatusNone))
		})

		It("adds a marker at the start of the paragraph", func() {
			block := content.NewBlock(content.NewParagraph(content.NewText("Task")))
			block.SetTaskStatus(content.TaskStatusTodo)

			Expect(block).To(EqualNode(content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewText("Task"),
			))))
		})

		It("adds a marker before the priority", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewTaskPriority(content.PriorityA),
				content.NewText("Task"),
			))
			block.SetTaskStatus(content.TaskStatusTodo)

			Expect(block).To(EqualNode(content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewTaskPriority(content.PriorityA),
				content.NewText("Task"),
			))))
		})

		It("adds a paragraph for the marker if the block has none", func() {
			child := content.NewBlock(content.NewParagraph(content.NewText("Child")))
			block := content.NewBlock(child)
			block.SetTaskStatus(content.TaskStatusTodo)

			Expect(block).To(EqualNode(content.NewBlock(
				content.NewParagraph(content.NewTaskMarker(content.TaskStatusTodo)),
				content.NewBlock(content.NewParagraph(content.NewText("Child"))),
			)))
		})

		It("replaces a marker that is already there", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewText("Task"),
			))
			block.SetTaskStatus(content.TaskStatusDone)

			Expect(block).To(EqualNode(content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusDone),
				content.NewText("Task"),
			))))
		})

		It("removes the marker but keeps the priority", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewTaskPriority(content.PriorityB),
				content.NewText("Task"),
			))
			block.SetTaskStatus(content.TaskStatusNone)

			Expect(block).To(EqualNode(content.NewBlock(content.NewParagraph(
				content.NewTaskPriority(content.PriorityB),
				content.NewText("Task"),
			))))
		})

		It("does nothing when removing a marker that is not there", func() {
			block := content.NewBlock(content.NewParagraph(content.NewText("Task")))
			block.SetTaskStatus(content.TaskStatusNone)

			Expect(block).To(EqualNode(content.NewBlock(content.NewParagraph(content.NewText("Task")))))
		})
	})

	Describe("Priorities on blocks", func() {
		It("has no priority on a block without one", func() {
			block := content.NewBlock(content.NewParagraph(content.NewText("Task")))

			Expect(block.Priority()).To(Equal(content.PriorityNone))
		})

		It("can get the priority after a marker", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewTaskPriority(content.PriorityA),
				content.NewText("Task"),
			))

			Expect(block.Priority()).To(Equal(content.PriorityA))
		})

		It("can get the priority without a marker", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewTaskPriority(content.PriorityC),
				content.NewText("Task"),
			))

			Expect(block.Priority()).To(Equal(content.PriorityC))
		})

		It("ignores a priority that is not at the start", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewText("Task"),
				content.NewTaskPriority(content.PriorityA),
			))

			Expect(block.Priority()).To(Equal(content.PriorityNone))
		})

		It("adds a priority after the marker", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewText("Task"),
			))
			block.SetPriority(content.PriorityA)

			Expect(block).To(EqualNode(content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewTaskPriority(content.PriorityA),
				content.NewText("Task"),
			))))
		})

		It("adds a priority at the start without a marker", func() {
			block := content.NewBlock(content.NewParagraph(content.NewText("Task")))
			block.SetPriority(content.PriorityB)

			Expect(block).To(EqualNode(content.NewBlock(content.NewParagraph(
				content.NewTaskPriority(content.PriorityB),
				content.NewText("Task"),
			))))
		})

		It("replaces a priority that is already there", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewTaskPriority(content.PriorityA),
				content.NewText("Task"),
			))
			block.SetPriority(content.PriorityC)

			Expect(block.Priority()).To(Equal(content.PriorityC))
			Expect(block.TaskStatus()).To(Equal(content.TaskStatusTodo))
		})

		It("removes the priority but keeps the marker", func() {
			block := content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewTaskPriority(content.PriorityA),
				content.NewText("Task"),
			))
			block.SetPriority(content.PriorityNone)

			Expect(block).To(EqualNode(content.NewBlock(content.NewParagraph(
				content.NewTaskMarker(content.TaskStatusTodo),
				content.NewText("Task"),
			))))
		})
	})
})
