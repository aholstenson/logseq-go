package query_test

import (
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Writing queries", func() {
	Prints := func(name string, q *query.Query, expected string) {
		It(name, func() {
			Expect(q.String()).To(Equal(expected))
		})
	}

	Describe("Filters", func() {
		Prints("and", query.New(query.And(query.PageRef("a"), query.PageRef("b"))), "(and [[a]] [[b]])")
		Prints("or", query.New(query.Or(query.PageRef("a"), query.PageRef("b"))), "(or [[a]] [[b]])")
		Prints("not", query.New(query.Not(query.PageRef("a"))), "(not [[a]])")
		Prints("task", query.New(query.Task(content.TaskStatusTodo, content.TaskStatusInProgress)), "(task TODO IN-PROGRESS)")
		Prints("priority", query.New(query.Priority(content.PriorityA, content.PriorityC)), "(priority A C)")
		Prints("full text", query.New(query.FullText(`say "hi"`)), `"say \"hi\""`)
		Prints("page", query.New(query.Page("My Page")), "(page [[My Page]])")
		Prints("property", query.New(query.Property("type", "book")), "(property type book)")
		Prints("property with spaces", query.New(query.Property("type", "Cognitive Recipe")), `(property type "Cognitive Recipe")`)
		Prints("has property", query.New(query.HasProperty("type")), "(property type)")
		Prints("page property", query.New(query.PageProperty("type", "book")), "(page-property type book)")
		Prints("has page property", query.New(query.HasPageProperty("type")), "(page-property type)")
		Prints("page tags", query.New(query.PageTags("book", "to read")), "(page-tags [[book]] [[to read]])")
		Prints("all page tags", query.New(query.AllPageTags()), "(all-page-tags)")
		Prints("refs", query.New(query.Refs("My Page")), "(refs [[My Page]])")
		Prints("parent", query.New(query.Parent("abc-123")), "(parent ((abc-123)))")
		Prints("parent that is not a UUID", query.New(query.Parent("a b")), `(parent "a b")`)
		Prints("ancestor", query.New(query.Ancestor("abc-123")), "(ancestor ((abc-123)))")
		Prints("ancestor that is not a UUID", query.New(query.Ancestor("a b")), `(ancestor "a b")`)
		Prints("parent by location", query.New(query.ParentAt("Outline", 0, 1)), "(parent [[Outline]] 0 1)")
		Prints("page as the parent", query.New(query.ParentAt("Outline")), "(parent [[Outline]])")
		Prints("top-level", query.New(query.And(query.TopLevel(), query.PageTags("x"))), "(and (top-level) (page-tags [[x]]))")
		Prints("journal page", query.New(query.JournalPage(query.Less, query.Relative("today"))), "(journal-page < today)")
		Prints("title", query.New(query.Title(query.TitleIncludes, "a\"b")), `(title includes "a\"b")`)
		Prints("content", query.New(query.Content(query.TitleEndsWith, "a\nb")), `(content ends-with "a\nb")`)
		Prints("regex", query.New(query.Content(query.TitleMatches, `bank\d`)), `(content matches "bank\\d")`)
		Prints("alias", query.New(query.Alias("ACME Corp")), "(alias [[ACME Corp]])")
		Prints("has-block", query.New(query.HasBlock(query.Task(content.TaskStatusTodo))), "(has-block (task TODO))")
		Prints("refs to pages", query.New(query.RefsTo(query.PageTags("project"))), "(refs (page-tags [[project]]))")
		Prints("blocks on pages", query.New(query.OnPage(query.Alias("x"))), "(page (alias [[x]]))")
		Prints("property number", query.New(query.PropertyNumber("rating", query.Greater, 7.5)), "(property rating > 7.5)")
		Prints("page property date", query.New(query.PagePropertyDate("due", query.Less, query.Relative("today"))), "(page-property due < today)")
		Prints("children", query.New(query.ChildOf(query.Task(content.TaskStatusTodo))), "(parent (task TODO))")
		Prints("descendants", query.New(query.DescendantOf(query.Task(content.TaskStatusTodo))), "(ancestor (task TODO))")
		Prints("has-child", query.New(query.HasChild(query.Task(content.TaskStatusTodo))), "(has-child (task TODO))")
		Prints("has-descendant", query.New(query.HasDescendant(query.Task(content.TaskStatusTodo))), "(has-descendant (task TODO))")
		Prints("page refs to pages", query.New(query.PageRefTo(query.Alias("x"))), "(page-ref (alias [[x]]))")
		Prints("ancestor by location", query.New(query.AncestorAt("Outline", 2)), "(ancestor [[Outline]] 2)")
		Prints("refs with brackets", query.New(query.Refs("a ]] b")), `(refs "a ]] b")`)
		Prints("namespace", query.New(query.Namespace("Projects")), "(namespace [[Projects]])")
		Prints("page ref with brackets", query.New(query.PageRef("a ]] b")), `(page-ref "a ]] b")`)
		Prints("page with brackets", query.New(query.Page("a]")), `(page "a]")`)
	})

	Describe("Dates", func() {
		Prints("journal range as between", query.New(query.Between(query.DaysFromToday(-7), query.Today())), "(between -7d today)")
		Prints("journal on a day", query.New(query.Journal(query.Equal, query.Today())), "(journal today)")
		Prints("on a journal", query.New(query.OnJournal()), "(journal)")
		Prints("scheduled before", query.New(query.Scheduled(query.Less, query.Today())), "(scheduled < today)")
		Prints("scheduled on or before", query.New(query.Scheduled(query.LessOrEqual, query.DaysFromToday(3))), "(scheduled <= +3d)")
		Prints("scheduled after", query.New(query.Scheduled(query.Greater, query.Today())), "(scheduled > today)")
		Prints("scheduled on or after", query.New(query.Scheduled(query.GreaterOrEqual, query.Today())), "(scheduled >= today)")
		Prints("scheduled range", query.New(query.ScheduledBetween(query.Relative("+1d"), query.Relative("+3d"))), "(scheduled +1d +3d)")
		Prints("has scheduled", query.New(query.HasScheduled()), "(scheduled)")
		Prints("deadline", query.New(query.Deadline(query.Equal, query.Relative("tomorrow"))), "(deadline tomorrow)")
		Prints("has deadline", query.New(query.HasDeadline()), "(deadline)")
		Prints("deadline range", query.New(query.DeadlineBetween(query.Today(), query.Relative("+1w"))), "(deadline today +1w)")
		Prints("fixed date", query.New(query.Scheduled(query.Less, query.On(time.Date(2024, 1, 15, 13, 0, 0, 0, time.Local)))), "(scheduled < 2024-01-15)")
		Prints("journal date", query.New(query.Between(query.JournalDate("Jan 1st, 2024"), query.Today())), "(between [[Jan 1st, 2024]] today)")
	})

	Describe("Sorting and sampling", func() {
		Prints("sort added to a top-level and",
			query.New(query.And(query.Task(content.TaskStatusTodo), query.PageRef("a"))).SortBy(query.SortPriority, query.Asc),
			"(and (task TODO) [[a]] (sort-by priority asc))",
		)
		Prints("sort with another filter wraps it in and",
			query.New(query.Task(content.TaskStatusTodo)).SortBy(query.SortScheduled, query.Desc),
			"(and (task TODO) (sort-by scheduled desc))",
		)
		Prints("sort by property",
			query.New(query.Task(content.TaskStatusTodo)).SortByProperty("created-at", query.Desc),
			"(and (task TODO) (sort-by created-at desc))",
		)
		Prints("several sorts and sample",
			query.New(query.Task(content.TaskStatusTodo)).
				SortBy(query.SortPriority, query.Asc).
				SortBy(query.SortJournalDay, query.Desc).
				SortBy(query.SortTitle, query.Asc).
				SortBy(query.SortDeadline, query.Asc),
			"(and (task TODO) (sort-by priority asc) (sort-by journal-day desc) (sort-by title asc) (sort-by deadline asc))",
		)
		Prints("sort without a filter", query.New(nil).SortBy(query.SortPriority, query.Asc), "(sort-by priority asc)")
		Prints("sample", &query.Query{Filter: query.PageRef("a"), Sample: 5}, "(and [[a]] (sample 5))")
		Prints("nothing", query.New(nil), "")
	})
})
