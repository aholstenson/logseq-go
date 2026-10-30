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

	Describe("Next occurrence", func() {
		// The task is completed on 10 January at noon.
		at := time.Date(2024, time.January, 10, 12, 0, 0, 0, time.Local)
		jan1 := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.Local)

		repeating := func(date time.Time, repeaterType content.RepeaterType, value int, unit content.RepeaterUnit) *content.TaskDate {
			return content.NewScheduled(date).WithRepeater(content.NewRepeater(repeaterType, value, unit))
		}

		It("keeps a date that does not repeat", func() {
			Expect(content.NewScheduled(jan1).Next(at)).To(Equal(jan1))
		})

		It("moves + one interval, even into the past", func() {
			date := repeating(jan1, content.RepeaterTypeCumulate, 1, content.RepeaterUnitDay)
			Expect(date.Next(at)).To(Equal(jan1.AddDate(0, 0, 1)))
		})

		It("moves ++ one interval when it is in the past", func() {
			date := repeating(jan1, content.RepeaterTypeCatchUp, 1, content.RepeaterUnitWeek)
			Expect(date.Next(at)).To(Equal(jan1.AddDate(0, 0, 7)))
		})

		It("keeps ++ when it is in the future", func() {
			future := time.Date(2024, time.January, 12, 0, 0, 0, 0, time.Local)
			date := repeating(future, content.RepeaterTypeCatchUp, 1, content.RepeaterUnitDay)
			Expect(date.Next(at)).To(Equal(future))
		})

		It("moves .+ until it is in the future", func() {
			date := repeating(jan1, content.RepeaterTypeRestart, 1, content.RepeaterUnitDay)
			Expect(date.Next(at)).To(Equal(time.Date(2024, time.January, 11, 0, 0, 0, 0, time.Local)))
		})

		It("moves .+ one interval when it is already in the future", func() {
			future := time.Date(2024, time.January, 12, 0, 0, 0, 0, time.Local)
			date := repeating(future, content.RepeaterTypeRestart, 3, content.RepeaterUnitDay)
			Expect(date.Next(at)).To(Equal(time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)))
		})

		It("moves .+ past the time of day the task was completed", func() {
			date := content.NewScheduledWithTime(time.Date(2024, time.January, 9, 12, 0, 0, 0, time.Local)).
				WithRepeater(content.NewRepeater(content.RepeaterTypeRestart, 1, content.RepeaterUnitDay))
			Expect(date.Next(at)).To(Equal(time.Date(2024, time.January, 11, 12, 0, 0, 0, time.Local)))
		})

		It("adds hours to the time of day", func() {
			date := content.NewScheduledWithTime(time.Date(2024, time.January, 10, 9, 0, 0, 0, time.Local)).
				WithRepeater(content.NewRepeater(content.RepeaterTypeCumulate, 2, content.RepeaterUnitHour))
			Expect(date.Next(at)).To(Equal(time.Date(2024, time.January, 10, 11, 0, 0, 0, time.Local)))
		})

		It("keeps a date without a time of day on a day for hours", func() {
			date := repeating(jan1, content.RepeaterTypeCumulate, 25, content.RepeaterUnitHour)
			Expect(date.Next(at)).To(Equal(jan1.AddDate(0, 0, 1)))
		})

		It("uses the last day of a shorter month", func() {
			jan31 := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.Local)
			date := repeating(jan31, content.RepeaterTypeCumulate, 1, content.RepeaterUnitMonth)
			Expect(date.Next(at)).To(Equal(time.Date(2024, time.February, 29, 0, 0, 0, 0, time.Local)))
		})

		It("uses the last day of February for a leap day", func() {
			leapDay := time.Date(2024, time.February, 29, 0, 0, 0, 0, time.Local)
			date := repeating(leapDay, content.RepeaterTypeCumulate, 1, content.RepeaterUnitYear)
			Expect(date.Next(at)).To(Equal(time.Date(2025, time.February, 28, 0, 0, 0, 0, time.Local)))
		})

		It("keeps the day of the month when it can", func() {
			date := repeating(jan1, content.RepeaterTypeCumulate, 13, content.RepeaterUnitMonth)
			Expect(date.Next(at)).To(Equal(time.Date(2025, time.February, 1, 0, 0, 0, 0, time.Local)))
		})
	})

	Describe("Completing tasks", func() {
		at := time.Date(2024, time.January, 10, 12, 0, 0, 0, time.Local)
		jan1 := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.Local)
		daily := func() *content.Repeater {
			return content.NewRepeater(content.RepeaterTypeCumulate, 1, content.RepeaterUnitDay)
		}

		task := func(status content.TaskStatus, rest ...content.Node) *content.Block {
			children := append([]content.Node{content.NewParagraph(
				content.NewTaskMarker(status),
				content.NewText("Task"),
			)}, rest...)
			return content.NewBlock(children...)
		}

		It("marks a task that does not repeat as done", func() {
			block := task(content.TaskStatusTodo, content.NewScheduled(jan1))
			block.CompleteTask(at)

			Expect(block).To(EqualNode(task(content.TaskStatusDone, content.NewScheduled(jan1))))
		})

		It("stops a running clock", func() {
			start := at.Add(-time.Hour)
			block := task(content.TaskStatusDoing, content.NewLogbook(content.NewLogbookEntryClock(start, time.Time{})))
			block.CompleteTask(at)

			Expect(block).To(EqualNode(task(content.TaskStatusDone,
				content.NewLogbook(content.NewLogbookEntryClock(start, at)),
			)))
		})

		It("keeps a clock that is stopped", func() {
			start := at.Add(-2 * time.Hour)
			end := at.Add(-time.Hour)
			block := task(content.TaskStatusTodo, content.NewLogbook(content.NewLogbookEntryClock(start, end)))
			block.CompleteTask(at)

			Expect(block.Logbook().LastChild()).To(EqualNode(content.NewLogbookEntryClock(start, end)))
		})

		It("moves a repeating date and keeps the task open", func() {
			block := task(content.TaskStatusTodo, content.NewScheduled(jan1).WithRepeater(daily()))
			block.CompleteTask(at)

			Expect(block).To(EqualNode(task(content.TaskStatusTodo,
				content.NewScheduled(jan1.AddDate(0, 0, 1)).WithRepeater(daily()),
				content.NewLogbook(content.NewLogbookEntryStateChange(content.TaskStatusTodo, content.TaskStatusDone, at)).
					WithPreviousLineType(content.PreviousLineTypeNonBlank),
			)))
		})

		It("moves both repeating dates", func() {
			block := task(content.TaskStatusLater,
				content.NewScheduled(jan1).WithRepeater(daily()),
				content.NewDeadline(jan1.AddDate(0, 0, 2)).WithRepeater(daily()),
			)
			block.CompleteTask(at)

			Expect(block.Scheduled().Date).To(Equal(jan1.AddDate(0, 0, 1)))
			Expect(block.Deadline().Date).To(Equal(jan1.AddDate(0, 0, 3)))
		})

		It("only moves the date that repeats", func() {
			block := task(content.TaskStatusTodo,
				content.NewScheduled(jan1).WithRepeater(daily()),
				content.NewDeadline(jan1),
			)
			block.CompleteTask(at)

			Expect(block.Scheduled().Date).To(Equal(jan1.AddDate(0, 0, 1)))
			Expect(block.Deadline().Date).To(Equal(jan1))
		})

		It("changes DOING to TODO and NOW to LATER for a repeating task", func() {
			doing := task(content.TaskStatusDoing, content.NewScheduled(jan1).WithRepeater(daily()))
			doing.CompleteTask(at)
			Expect(doing.TaskStatus()).To(Equal(content.TaskStatusTodo))

			now := task(content.TaskStatusNow, content.NewScheduled(jan1).WithRepeater(daily()))
			now.CompleteTask(at)
			Expect(now.TaskStatus()).To(Equal(content.TaskStatusLater))
		})

		It("keeps other open statuses for a repeating task", func() {
			block := task(content.TaskStatusWaiting, content.NewScheduled(jan1).WithRepeater(daily()))
			block.CompleteTask(at)

			Expect(block.TaskStatus()).To(Equal(content.TaskStatusWaiting))
		})

		It("adds the state change after a stopped clock", func() {
			start := at.Add(-time.Hour)
			block := task(content.TaskStatusDoing,
				content.NewScheduled(jan1).WithRepeater(daily()),
				content.NewLogbook(content.NewLogbookEntryClock(start, time.Time{})),
			)
			block.CompleteTask(at)

			Expect(block.Logbook()).To(EqualNode(content.NewLogbook(
				content.NewLogbookEntryClock(start, at),
				content.NewLogbookEntryStateChange(content.TaskStatusDoing, content.TaskStatusDone, at),
			)))
		})

		It("adds the logbook after properties and before sub blocks", func() {
			block := task(content.TaskStatusTodo,
				content.NewScheduled(jan1).WithRepeater(daily()),
				content.NewProperties(content.NewProperty("id", content.NewText("1"))),
				content.NewBlock(content.NewParagraph(content.NewText("Child"))),
			)
			block.CompleteTask(at)

			Expect(block).To(EqualNode(task(content.TaskStatusTodo,
				content.NewScheduled(jan1.AddDate(0, 0, 1)).WithRepeater(daily()),
				content.NewProperties(content.NewProperty("id", content.NewText("1"))),
				content.NewLogbook(content.NewLogbookEntryStateChange(content.TaskStatusTodo, content.TaskStatusDone, at)).
					WithPreviousLineType(content.PreviousLineTypeNonBlank),
				content.NewBlock(content.NewParagraph(content.NewText("Child"))),
			)))
		})

		It("does nothing for a closed task", func() {
			block := task(content.TaskStatusCancelled, content.NewScheduled(jan1).WithRepeater(daily()))
			block.CompleteTask(at)

			Expect(block).To(EqualNode(task(content.TaskStatusCancelled, content.NewScheduled(jan1).WithRepeater(daily()))))
		})

		It("does nothing for a block that is not a task", func() {
			block := content.NewBlock(content.NewParagraph(content.NewText("Text")))
			block.CompleteTask(at)

			Expect(block).To(EqualNode(content.NewBlock(content.NewParagraph(content.NewText("Text")))))
		})
	})

	Describe("Clocks", func() {
		at := time.Date(2024, time.January, 10, 12, 0, 0, 0, time.Local)
		earlier := at.Add(-time.Hour)

		task := func(status content.TaskStatus, rest ...content.Node) *content.Block {
			children := append([]content.Node{content.NewParagraph(
				content.NewTaskMarker(status),
				content.NewText("Task"),
			)}, rest...)
			return content.NewBlock(children...)
		}

		newLogbook := func(entries ...content.LogbookEntry) *content.Logbook {
			return content.NewLogbook(entries...).WithPreviousLineType(content.PreviousLineTypeNonBlank)
		}

		It("has no running clock without a logbook", func() {
			Expect(task(content.TaskStatusDoing).RunningClock()).To(BeNil())
		})

		It("has no running clock when the last clock is stopped", func() {
			block := task(content.TaskStatusDoing, newLogbook(
				content.NewLogbookEntryClock(earlier.Add(-time.Hour), time.Time{}),
				content.NewLogbookEntryClock(earlier, at),
			))

			Expect(block.RunningClock()).To(BeNil())
		})

		It("starts a clock in a new logbook", func() {
			block := task(content.TaskStatusDoing)
			block.StartClock(at)

			Expect(block).To(EqualNode(task(content.TaskStatusDoing,
				newLogbook(content.NewLogbookEntryClock(at, time.Time{})),
			)))
			Expect(block.RunningClock()).ToNot(BeNil())
		})

		It("starts a clock after the entries that are there", func() {
			block := task(content.TaskStatusDoing, newLogbook(content.NewLogbookEntryClock(earlier.Add(-time.Hour), earlier)))
			block.StartClock(at)

			Expect(block.Logbook()).To(EqualNode(newLogbook(
				content.NewLogbookEntryClock(earlier.Add(-time.Hour), earlier),
				content.NewLogbookEntryClock(at, time.Time{}),
			)))
		})

		It("does not start a second clock", func() {
			block := task(content.TaskStatusDoing, newLogbook(content.NewLogbookEntryClock(earlier, time.Time{})))
			block.StartClock(at)

			Expect(block.Logbook()).To(EqualNode(newLogbook(content.NewLogbookEntryClock(earlier, time.Time{}))))
		})

		It("stops a running clock", func() {
			block := task(content.TaskStatusDoing, newLogbook(content.NewLogbookEntryClock(earlier, time.Time{})))
			block.StopClock(at)

			Expect(block.Logbook()).To(EqualNode(newLogbook(content.NewLogbookEntryClock(earlier, at))))
		})

		It("does nothing when stopping without a running clock", func() {
			block := task(content.TaskStatusTodo)
			block.StopClock(at)

			Expect(block).To(EqualNode(task(content.TaskStatusTodo)))
		})

		Describe("SetTaskStatusWithClock", func() {
			It("starts a clock for DOING and NOW", func() {
				for _, status := range []content.TaskStatus{content.TaskStatusDoing, content.TaskStatusNow} {
					block := task(content.TaskStatusTodo)
					block.SetTaskStatusWithClock(status, at)

					Expect(block).To(EqualNode(task(status, newLogbook(content.NewLogbookEntryClock(at, time.Time{})))))
				}
			})

			It("starts a clock for a block that was not a task", func() {
				block := content.NewBlock(content.NewParagraph(content.NewText("Task")))
				block.SetTaskStatusWithClock(content.TaskStatusDoing, at)

				Expect(block).To(EqualNode(task(content.TaskStatusDoing, newLogbook(content.NewLogbookEntryClock(at, time.Time{})))))
			})

			It("stops the clock for other statuses", func() {
				for _, status := range []content.TaskStatus{content.TaskStatusTodo, content.TaskStatusDone, content.TaskStatusCanceled, content.TaskStatusNone} {
					block := task(content.TaskStatusDoing, newLogbook(content.NewLogbookEntryClock(earlier, time.Time{})))
					block.SetTaskStatusWithClock(status, at)

					Expect(block.TaskStatus()).To(Equal(status))
					Expect(block.Logbook()).To(EqualNode(newLogbook(content.NewLogbookEntryClock(earlier, at))))
				}
			})

			It("keeps a running clock when changing between DOING and NOW", func() {
				block := task(content.TaskStatusDoing, newLogbook(content.NewLogbookEntryClock(earlier, time.Time{})))
				block.SetTaskStatusWithClock(content.TaskStatusNow, at)

				Expect(block.Logbook()).To(EqualNode(newLogbook(content.NewLogbookEntryClock(earlier, time.Time{}))))
			})

			It("does not add a logbook for a status without a clock", func() {
				block := task(content.TaskStatusTodo)
				block.SetTaskStatusWithClock(content.TaskStatusDone, at)

				Expect(block).To(EqualNode(task(content.TaskStatusDone)))
			})
		})
	})
})
