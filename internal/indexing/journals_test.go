package indexing_test

import (
	"context"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func indexJournal(idx *indexing.BlugeIndex, subPath string, date time.Time, blocks ...*content.Block) {
	ctx := context.Background()
	page := &indexing.Page{
		SubPath:      subPath,
		Type:         indexing.PageTypeJournal,
		LastModified: time.Now(),
		Title:        date.Format("Jan 2, 2006"),
		Date:         date,
		Blocks:       blocks,
	}
	Expect(idx.IndexPage(ctx, page)).To(Succeed())
	Expect(idx.Sync()).To(Succeed())
}

// journalDatesOf returns the dates of the pages as `2006-01-02` in the local
// time zone, as the index returns dates in UTC.
func journalDatesOf(pages []*indexing.Page) []string {
	dates := make([]string, 0, len(pages))
	for _, page := range pages {
		dates = append(dates, page.Date.Local().Format("2006-01-02"))
	}
	return dates
}

var _ = Describe("JournalBetween", func() {
	var idx *indexing.BlugeIndex

	jan15 := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)
	jan16 := jan15.AddDate(0, 0, 1)
	jan17 := jan15.AddDate(0, 0, 2)

	BeforeEach(func() {
		idx = createIndex()

		indexJournal(idx, "journals/2024_01_15.md", jan15,
			textBlock(text("monday"),
				textBlock(text("monday child")),
			),
		)
		indexJournal(idx, "journals/2024_01_16.md", jan16,
			textBlock(text("tuesday")),
		)
		indexPage(idx, "pages/a.md", "Page A",
			textBlock(text("not a journal")),
		)
	})

	AfterEach(func() {
		Expect(idx.Close()).To(Succeed())
	})

	It("matches journal pages within the range", func() {
		results := searchPages(idx, indexing.JournalBetween(jan15, jan16))
		Expect(journalDatesOf(results)).To(ConsistOf("2024-01-15"))
	})

	It("matches the blocks on journals within the range", func() {
		results := searchBlocks(idx, indexing.JournalBetween(jan15, jan16))
		Expect(previewsOf(results)).To(ConsistOf("monday", "monday child"))
	})

	It("includes the start and excludes the end of the range", func() {
		results := searchBlocks(idx, indexing.JournalBetween(jan16, jan17))
		Expect(previewsOf(results)).To(ConsistOf("tuesday"))
	})

	It("matches all journals for an open range", func() {
		Expect(searchPages(idx, indexing.JournalBetween(time.Time{}, time.Time{}))).To(HaveLen(2))

		results := searchBlocks(idx, indexing.JournalBetween(time.Time{}, time.Time{}))
		Expect(previewsOf(results)).To(ConsistOf("monday", "monday child", "tuesday"))
	})

	It("finds tasks for a day by journal, scheduled date or deadline", func() {
		indexJournal(idx, "journals/2024_01_15.md", jan15,
			taskBlock(content.TaskStatusTodo, content.PriorityNone, "on journal"),
			taskBlock(content.TaskStatusTodo, content.PriorityNone, "moved to later",
				content.NewScheduled(jan17),
			),
		)
		indexPage(idx, "pages/a.md", "Page A",
			taskBlock(content.TaskStatusTodo, content.PriorityNone, "scheduled",
				content.NewScheduled(jan15),
			),
			taskBlock(content.TaskStatusTodo, content.PriorityNone, "deadline",
				content.NewDeadline(jan15),
			),
			taskBlock(content.TaskStatusTodo, content.PriorityNone, "other day",
				content.NewScheduled(jan16),
			),
		)

		results := searchBlocks(idx, indexing.And(
			indexing.IsOpenTask(),
			indexing.Or(
				indexing.And(
					indexing.JournalBetween(jan15, jan16),
					indexing.Not(indexing.ScheduledBetween(time.Time{}, time.Time{})),
				),
				indexing.ScheduledBetween(jan15, jan16),
				indexing.DeadlineBetween(jan15, jan16),
			),
		))
		Expect(previewsOf(results)).To(ConsistOf("on journal", "scheduled", "deadline"))
	})
})
