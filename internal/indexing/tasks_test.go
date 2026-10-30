package indexing_test

import (
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// taskBlock creates a block with the given status, priority and text. Pass
// TaskStatusNone or PriorityNone to leave them out.
func taskBlock(status content.TaskStatus, priority content.Priority, text string, dates ...*content.TaskDate) *content.Block {
	block := content.NewBlock(content.NewParagraph(content.NewText(text)))
	block.SetTaskStatus(status)
	block.SetPriority(priority)
	for _, date := range dates {
		if date.Type == content.TaskDateTypeScheduled {
			block.SetScheduled(date)
		} else {
			block.SetDeadline(date)
		}
	}
	return block
}

func previewsOf(blocks []*indexing.Block) []string {
	previews := make([]string, 0, len(blocks))
	for _, block := range blocks {
		previews = append(previews, block.Preview)
	}
	return previews
}

var _ = Describe("Task queries", func() {
	var idx *indexing.BlugeIndex

	jan15 := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)
	jan16 := jan15.AddDate(0, 0, 1)
	jan20 := time.Date(2024, time.January, 20, 0, 0, 0, 0, time.Local)

	BeforeEach(func() {
		idx = createIndex()

		indexPage(idx, "pages/a.md", "Page A",
			taskBlock(content.TaskStatusTodo, content.PriorityA, "todo",
				content.NewScheduled(jan15),
			),
			taskBlock(content.TaskStatusDoing, content.PriorityNone, "doing",
				content.NewDeadline(jan20),
			),
			taskBlock(content.TaskStatusDone, content.PriorityB, "done",
				content.NewScheduledWithTime(jan15.Add(9*time.Hour)),
				content.NewDeadline(jan15),
			),
			taskBlock(content.TaskStatusCanceled, content.PriorityNone, "canceled"),
			taskBlock(content.TaskStatusNone, content.PriorityC, "plain"),
		)
	})

	AfterEach(func() {
		Expect(idx.Close()).To(Succeed())
	})

	Describe("IsTask", func() {
		It("matches all blocks with a marker", func() {
			results := searchBlocks(idx, indexing.IsTask())
			Expect(previewsOf(results)).To(ConsistOf("todo", "doing", "done", "canceled"))
		})

		It("does not match pages", func() {
			Expect(searchPages(idx, indexing.IsTask())).To(BeEmpty())
		})
	})

	Describe("IsOpenTask", func() {
		It("matches tasks that are not finished", func() {
			results := searchBlocks(idx, indexing.IsOpenTask())
			Expect(previewsOf(results)).To(ConsistOf("todo", "doing"))
		})
	})

	Describe("IsClosedTask", func() {
		It("matches done and cancelled tasks", func() {
			results := searchBlocks(idx, indexing.IsClosedTask())
			Expect(previewsOf(results)).To(ConsistOf("done", "canceled"))
		})
	})

	Describe("TaskStatusIs", func() {
		It("matches a single status", func() {
			results := searchBlocks(idx, indexing.TaskStatusIs(content.TaskStatusDoing))
			Expect(previewsOf(results)).To(ConsistOf("doing"))
		})

		It("matches any of several statuses", func() {
			results := searchBlocks(idx, indexing.TaskStatusIs(content.TaskStatusTodo, content.TaskStatusDone))
			Expect(previewsOf(results)).To(ConsistOf("todo", "done"))
		})

		It("keeps the two spellings of cancelled apart", func() {
			Expect(searchBlocks(idx, indexing.TaskStatusIs(content.TaskStatusCancelled))).To(BeEmpty())
		})

		It("matches nothing without a status", func() {
			Expect(searchBlocks(idx, indexing.TaskStatusIs())).To(BeEmpty())
			Expect(searchBlocks(idx, indexing.TaskStatusIs(content.TaskStatusNone))).To(BeEmpty())
		})
	})

	Describe("PriorityIs", func() {
		It("matches blocks with the priority", func() {
			results := searchBlocks(idx, indexing.PriorityIs(content.PriorityA))
			Expect(previewsOf(results)).To(ConsistOf("todo"))
		})

		It("matches a priority on a block without a marker", func() {
			results := searchBlocks(idx, indexing.PriorityIs(content.PriorityC))
			Expect(previewsOf(results)).To(ConsistOf("plain"))
		})

		It("matches nothing for no priority", func() {
			Expect(searchBlocks(idx, indexing.PriorityIs(content.PriorityNone))).To(BeEmpty())
		})
	})

	Describe("ScheduledBetween", func() {
		It("matches dates and times within a day", func() {
			results := searchBlocks(idx, indexing.ScheduledBetween(jan15, jan16))
			Expect(previewsOf(results)).To(ConsistOf("todo", "done"))
		})

		It("excludes the end of the range", func() {
			results := searchBlocks(idx, indexing.ScheduledBetween(jan15.Add(time.Hour), jan15.Add(9*time.Hour)))
			Expect(results).To(BeEmpty())
		})

		It("includes the start of the range", func() {
			results := searchBlocks(idx, indexing.ScheduledBetween(jan15.Add(9*time.Hour), jan16))
			Expect(previewsOf(results)).To(ConsistOf("done"))
		})

		It("matches every scheduled block for an open range", func() {
			results := searchBlocks(idx, indexing.ScheduledBetween(time.Time{}, time.Time{}))
			Expect(previewsOf(results)).To(ConsistOf("todo", "done"))
		})
	})

	Describe("DeadlineBetween", func() {
		It("matches deadlines before a time", func() {
			results := searchBlocks(idx, indexing.DeadlineBetween(time.Time{}, jan16))
			Expect(previewsOf(results)).To(ConsistOf("done"))
		})

		It("matches deadlines after a time", func() {
			results := searchBlocks(idx, indexing.DeadlineBetween(jan16, time.Time{}))
			Expect(previewsOf(results)).To(ConsistOf("doing"))
		})
	})

	Describe("Combined with other queries", func() {
		It("finds open tasks with a deadline before a time", func() {
			results := searchBlocks(idx, indexing.And(
				indexing.IsOpenTask(),
				indexing.DeadlineBetween(time.Time{}, jan20.AddDate(0, 0, 1)),
			))
			Expect(previewsOf(results)).To(ConsistOf("doing"))
		})
	})

	Describe("Updates", func() {
		It("stops matching a task that is completed", func() {
			indexPage(idx, "pages/a.md", "Page A",
				taskBlock(content.TaskStatusDone, content.PriorityA, "todo"),
			)

			Expect(searchBlocks(idx, indexing.IsOpenTask())).To(BeEmpty())
			Expect(previewsOf(searchBlocks(idx, indexing.IsClosedTask()))).To(ConsistOf("todo"))
		})
	})
})
