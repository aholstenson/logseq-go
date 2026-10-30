package indexing_test

import (
	"context"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func sortedBlocks(idx *indexing.BlugeIndex, query indexing.Query, opts indexing.SearchOptions) []string {
	results, err := idx.SearchBlocks(context.Background(), query, opts)
	Expect(err).ToNot(HaveOccurred())
	return previewsOf(results.Results())
}

func sortedPages(idx *indexing.BlugeIndex, query indexing.Query, sortBy ...indexing.SortField) []string {
	results, err := idx.SearchPages(context.Background(), query, indexing.SearchOptions{SortBy: sortBy})
	Expect(err).ToNot(HaveOccurred())

	titles := make([]string, 0, results.Size())
	for _, page := range results.Results() {
		titles = append(titles, page.Title)
	}
	return titles
}

func sortBy(field string, asc bool) indexing.SortField {
	return indexing.SortField{Field: field, Asc: asc}
}

var _ = Describe("Sorting", func() {
	var idx *indexing.BlugeIndex

	jan15 := time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local)
	jan16 := jan15.AddDate(0, 0, 1)
	jan17 := jan15.AddDate(0, 0, 2)

	BeforeEach(func() {
		idx = createIndex()
	})

	AfterEach(func() {
		Expect(idx.Close()).To(Succeed())
	})

	Describe("by priority", func() {
		BeforeEach(func() {
			indexPage(idx, "pages/a.md", "Page A",
				taskBlock(content.TaskStatusTodo, content.PriorityNone, "none"),
				taskBlock(content.TaskStatusTodo, content.PriorityC, "c"),
				taskBlock(content.TaskStatusTodo, content.PriorityA, "a"),
				taskBlock(content.TaskStatusTodo, content.PriorityB, "b"),
			)
		})

		It("sorts A first in ascending order, with no priority last", func() {
			results := sortedBlocks(idx, indexing.IsTask(), indexing.SearchOptions{
				SortBy: []indexing.SortField{sortBy(indexing.SortFieldPriority, true)},
			})
			Expect(results).To(Equal([]string{"a", "b", "c", "none"}))
		})

		It("sorts C first in descending order, with no priority last", func() {
			results := sortedBlocks(idx, indexing.IsTask(), indexing.SearchOptions{
				SortBy: []indexing.SortField{sortBy(indexing.SortFieldPriority, false)},
			})
			Expect(results).To(Equal([]string{"c", "b", "a", "none"}))
		})

		It("pages through the results in sorted order", func() {
			results := sortedBlocks(idx, indexing.IsTask(), indexing.SearchOptions{
				Size:   2,
				From:   1,
				SortBy: []indexing.SortField{sortBy(indexing.SortFieldPriority, true)},
			})
			Expect(results).To(Equal([]string{"b", "c"}))
		})
	})

	Describe("by task dates", func() {
		BeforeEach(func() {
			indexPage(idx, "pages/a.md", "Page A",
				taskBlock(content.TaskStatusTodo, content.PriorityNone, "no dates"),
				taskBlock(content.TaskStatusTodo, content.PriorityNone, "late",
					content.NewScheduled(jan17),
					content.NewDeadline(jan15),
				),
				taskBlock(content.TaskStatusTodo, content.PriorityNone, "early",
					content.NewScheduled(jan15),
					content.NewDeadline(jan17),
				),
				taskBlock(content.TaskStatusTodo, content.PriorityNone, "middle",
					content.NewScheduledWithTime(jan16.Add(9*time.Hour)),
				),
			)
		})

		It("sorts by scheduled date, with no date last", func() {
			results := sortedBlocks(idx, indexing.IsTask(), indexing.SearchOptions{
				SortBy: []indexing.SortField{sortBy(indexing.SortFieldScheduled, true)},
			})
			Expect(results).To(Equal([]string{"early", "middle", "late", "no dates"}))
		})

		It("sorts by deadline in descending order, with no date last", func() {
			results := sortedBlocks(idx, indexing.IsTask(), indexing.SearchOptions{
				SortBy: []indexing.SortField{sortBy(indexing.SortFieldDeadline, false)},
			})
			Expect(results[:2]).To(Equal([]string{"early", "late"}))
			Expect(results[2:]).To(ConsistOf("no dates", "middle"))
		})
	})

	It("sorts by a second field when the first is the same", func() {
		indexPage(idx, "pages/a.md", "Page A",
			taskBlock(content.TaskStatusTodo, content.PriorityB, "b later", content.NewScheduled(jan17)),
			taskBlock(content.TaskStatusTodo, content.PriorityA, "a", content.NewScheduled(jan17)),
			taskBlock(content.TaskStatusTodo, content.PriorityB, "b sooner", content.NewScheduled(jan15)),
		)

		results := sortedBlocks(idx, indexing.IsTask(), indexing.SearchOptions{
			SortBy: []indexing.SortField{
				sortBy(indexing.SortFieldPriority, true),
				sortBy(indexing.SortFieldScheduled, true),
			},
		})
		Expect(results).To(Equal([]string{"a", "b sooner", "b later"}))
	})

	Describe("by journal date", func() {
		BeforeEach(func() {
			indexJournal(idx, "journals/2024_01_16.md", jan16, textBlock(text("tuesday")))
			indexJournal(idx, "journals/2024_01_15.md", jan15, textBlock(text("monday")))
			indexJournal(idx, "journals/2024_01_17.md", jan17, textBlock(text("wednesday")))
			indexPage(idx, "pages/a.md", "Page A", textBlock(text("not a journal")))
		})

		It("sorts journals by date", func() {
			results, err := idx.SearchPages(context.Background(), indexing.All(), indexing.SearchOptions{
				SortBy: []indexing.SortField{sortBy(indexing.SortFieldJournalDate, false)},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(journalDatesOf(results.Results()[:3])).To(Equal([]string{"2024-01-17", "2024-01-16", "2024-01-15"}))
		})

		It("sorts blocks by the date of their journal, with other pages last", func() {
			results := sortedBlocks(idx, indexing.All(), indexing.SearchOptions{
				SortBy: []indexing.SortField{sortBy(indexing.SortFieldJournalDate, true)},
			})
			Expect(results).To(Equal([]string{"monday", "tuesday", "wednesday", "not a journal"}))
		})
	})

	Describe("by title", func() {
		BeforeEach(func() {
			indexPage(idx, "pages/b.md", "beta", textBlock(text("on beta")))
			indexPage(idx, "pages/c.md", "Gamma", textBlock(text("on gamma")))
			indexPage(idx, "pages/a.md", "Alpha", textBlock(text("on alpha")))
		})

		It("sorts pages by title without regard for case", func() {
			Expect(sortedPages(idx, indexing.All(), sortBy(indexing.SortFieldTitle, true))).
				To(Equal([]string{"Alpha", "beta", "Gamma"}))
			Expect(sortedPages(idx, indexing.All(), sortBy(indexing.SortFieldTitle, false))).
				To(Equal([]string{"Gamma", "beta", "Alpha"}))
		})

		It("sorts blocks by the title of their page", func() {
			results := sortedBlocks(idx, indexing.All(), indexing.SearchOptions{
				SortBy: []indexing.SortField{sortBy(indexing.SortFieldTitle, true)},
			})
			Expect(results).To(Equal([]string{"on alpha", "on beta", "on gamma"}))
		})
	})
})
