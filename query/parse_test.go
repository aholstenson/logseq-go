package query_test

import (
	"errors"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parsing queries", func() {
	Parses := func(text string, expected *query.Query) {
		It(text, func() {
			q, err := query.Parse(text)
			Expect(err).ToNot(HaveOccurred())
			Expect(q).To(Equal(expected))
		})
	}

	ParsesFilter := func(text string, expected query.Filter) {
		Parses(text, query.New(expected))
	}

	Fails := func(text string, offset int, message string) {
		It(text, func() {
			_, err := query.Parse(text)
			Expect(err).To(HaveOccurred())

			var parseErr *query.ParseError
			Expect(errors.As(err, &parseErr)).To(BeTrue())
			Expect(parseErr.Message).To(ContainSubstring(message))
			Expect(parseErr.Offset).To(Equal(offset))
		})
	}

	Describe("Logseq filters", func() {
		ParsesFilter("[[page]]", query.PageRef("page"))
		ParsesFilter("#tag", query.PageRef("tag"))
		ParsesFilter("#[[multi word]]", query.PageRef("multi word"))
		ParsesFilter("[[a [[b]]]]", query.PageRef("a [[b]]"))
		ParsesFilter(`(page-ref "a]")`, query.PageRef("a]"))
		ParsesFilter(`"some text"`, query.FullText("some text"))
		ParsesFilter(`"say \"hi\" \\ there"`, query.FullText(`say "hi" \ there`))
		ParsesFilter("(and [[a]] [[b]])", query.And(query.PageRef("a"), query.PageRef("b")))
		ParsesFilter("(AND [[a]] #b)", query.And(query.PageRef("a"), query.PageRef("b")))
		ParsesFilter("(or [[a]] [[b]])", query.Or(query.PageRef("a"), query.PageRef("b")))
		ParsesFilter("(not [[a]])", query.Not(query.PageRef("a")))
		ParsesFilter("(not [[a]] [[b]])", query.Not(query.And(query.PageRef("a"), query.PageRef("b"))))
		ParsesFilter("(task TODO LATER)", query.Task(content.TaskStatusTodo, content.TaskStatusLater))
		ParsesFilter("(task [NOW DOING])", query.Task(content.TaskStatusNow, content.TaskStatusDoing))
		ParsesFilter("(todo now later)", query.Task(content.TaskStatusNow, content.TaskStatusLater))
		ParsesFilter("(task in-progress waiting)", query.Task(content.TaskStatusInProgress, content.TaskStatusWaiting))
		ParsesFilter("(priority a b)", query.Priority(content.PriorityA, content.PriorityB))
		ParsesFilter("(priority [C])", query.Priority(content.PriorityC))
		ParsesFilter("(page [[My Page]])", query.Page("My Page"))
		ParsesFilter(`(page "My Page")`, query.Page("My Page"))
		ParsesFilter("(page mypage)", query.Page("mypage"))
		ParsesFilter("(property type book)", query.Property("type", "book"))
		ParsesFilter("(property type [[Cognitive Recipe]])", query.Property("type", "Cognitive Recipe"))
		ParsesFilter("(property type #book)", query.Property("type", "book"))
		ParsesFilter(`(property type "a b")`, query.Property("type", "a b"))
		ParsesFilter("(property :type book)", query.Property("type", "book"))
		ParsesFilter("(property page_type book)", query.Property("page-type", "book"))
		ParsesFilter("(property type [a b])", query.Property("type", "a"))
		ParsesFilter("(property type)", query.HasProperty("type"))
		ParsesFilter("(page-property type [[Cognitive Recipe]])", query.PageProperty("type", "Cognitive Recipe"))
		ParsesFilter("(page-property type)", query.HasPageProperty("type"))
		ParsesFilter("(page-tags book #reading [[to read]])", query.PageTags("book", "reading", "to read"))
		ParsesFilter("(page-tags [book reading])", query.PageTags("book", "reading"))
		ParsesFilter("(all-page-tags)", query.AllPageTags())
		ParsesFilter("(top-level)", query.TopLevel())
		ParsesFilter("(refs [[My Page]])", query.Refs("My Page"))
		ParsesFilter("(refs #tag)", query.Refs("tag"))
		ParsesFilter("(parent ((6523a0f8-1111-4c3b-9b1e-000000000001)))", query.Parent("6523a0f8-1111-4c3b-9b1e-000000000001"))
		ParsesFilter(`(parent "abc")`, query.Parent("abc"))
		ParsesFilter("(ancestor ((6523a0f8-1111-4c3b-9b1e-000000000001)))", query.Ancestor("6523a0f8-1111-4c3b-9b1e-000000000001"))
		ParsesFilter("(parent [[Outline]] 0 1)", query.ParentAt("Outline", 0, 1))
		ParsesFilter("(parent [[Outline]])", query.ParentAt("Outline"))
		ParsesFilter("(ancestor [[Outline]])", query.AncestorAt("Outline"))
		ParsesFilter("(journal-page)", query.IsJournalPage())
		ParsesFilter("(journal-page >= -7d)", query.JournalPage(query.GreaterOrEqual, query.Relative("-7d")))
		ParsesFilter(`(title starts-with "projects/")`, query.Title(query.TitleStartsWith, "projects/"))
		ParsesFilter(`(title = "Book")`, query.Title(query.TitleEquals, "Book"))
		ParsesFilter(`(content starts-with "TODO")`, query.Content(query.TitleStartsWith, "TODO"))
		ParsesFilter(`(content includes "the Bank")`, query.Content(query.TitleIncludes, "the Bank"))
		ParsesFilter(`(content matches "bank\\d")`, query.Content(query.TitleMatches, `bank\d`))
		ParsesFilter(`(title matches "^proj")`, query.Title(query.TitleMatches, "^proj"))
		ParsesFilter("(alias [[ACME Corp]])", query.Alias("ACME Corp"))
		ParsesFilter("(has-block (task TODO))", query.HasBlock(query.Task(content.TaskStatusTodo)))
		ParsesFilter("(refs (page-tags [[project]]))", query.RefsTo(query.PageTags("project")))
		ParsesFilter(`(page-ref (title starts-with "x"))`, query.PageRefTo(query.Title(query.TitleStartsWith, "x")))
		ParsesFilter("(refs [[project]])", query.Refs("project"))
		ParsesFilter(`(page (title = "x"))`, query.OnPage(query.Title(query.TitleEquals, "x")))
		ParsesFilter("(property rating > 7)", query.PropertyNumber("rating", query.Greater, 7))
		ParsesFilter("(property rating <= -1.5)", query.PropertyNumber("rating", query.LessOrEqual, -1.5))
		ParsesFilter("(property due < today)", query.PropertyDate("due", query.Less, query.Relative("today")))
		ParsesFilter("(page-property pages >= 300)", query.PagePropertyNumber("pages", query.GreaterOrEqual, 300))
		ParsesFilter("(page-property due = 2024-01-15)", query.PagePropertyDate("due", query.Equal, query.On(time.Date(2024, time.January, 15, 0, 0, 0, 0, time.Local))))
		ParsesFilter("(parent (task TODO))", query.ChildOf(query.Task(content.TaskStatusTodo)))
		ParsesFilter("(ancestor (task TODO))", query.DescendantOf(query.Task(content.TaskStatusTodo)))
		ParsesFilter("(has-child (task TODO))", query.HasChild(query.Task(content.TaskStatusTodo)))
		ParsesFilter("(has-descendant (task TODO))", query.HasDescendant(query.Task(content.TaskStatusTodo)))
		ParsesFilter("(ancestor [[Outline]] 2)", query.AncestorAt("Outline", 2))
		ParsesFilter("(and (task TODO) (parent ((abc))))", query.And(query.Task(content.TaskStatusTodo), query.Parent("abc")))
		ParsesFilter("(namespace [[Projects]])", query.Namespace("Projects"))
		ParsesFilter("(between -7d today)", query.Between(query.Relative("-7d"), query.Today()))
		ParsesFilter("(between -7 +7)", query.Between(query.Relative("-7d"), query.Relative("+7d")))
		ParsesFilter("(between :-2w :+1m)", query.Between(query.Relative("-2w"), query.Relative("+1m")))
		ParsesFilter("(between [[Jan 1st, 2024]] yesterday)", query.Between(query.JournalDate("Jan 1st, 2024"), query.Relative("yesterday")))
	})

	Describe("Text that is not a query", func() {
		ParsesFilter("hello world", query.FullText("hello world"))
		Parses("", &query.Query{})
		Parses("   ", &query.Query{})
	})

	Describe("Date filters", func() {
		ParsesFilter("(scheduled)", query.HasScheduled())
		ParsesFilter("(scheduled today)", query.Scheduled(query.Equal, query.Today()))
		ParsesFilter("(scheduled = today)", query.Scheduled(query.Equal, query.Today()))
		ParsesFilter("(scheduled < today)", query.Scheduled(query.Less, query.Today()))
		ParsesFilter("(scheduled <= +3d)", query.Scheduled(query.LessOrEqual, query.Relative("+3d")))
		ParsesFilter("(scheduled > today)", query.Scheduled(query.Greater, query.Today()))
		ParsesFilter("(scheduled >= Today)", query.Scheduled(query.GreaterOrEqual, query.Today()))
		ParsesFilter("(scheduled +1d +3d)", query.ScheduledBetween(query.Relative("+1d"), query.Relative("+3d")))
		ParsesFilter("(deadline)", query.HasDeadline())
		ParsesFilter("(deadline < 3d-after)", query.Deadline(query.Less, query.Relative("3d-after")))
		ParsesFilter("(journal)", query.OnJournal())
		ParsesFilter("(journal today)", query.Journal(query.Equal, query.Today()))
		ParsesFilter("(journal -7d today)", query.Between(query.Relative("-7d"), query.Today()))
		ParsesFilter(`(scheduled < "Jan 15th, 2024")`, query.Scheduled(query.Less, query.JournalDate("Jan 15th, 2024")))
		ParsesFilter("(scheduled < 2024-01-15)", query.Scheduled(query.Less, query.On(time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local))))
	})

	Describe("Sorting and sampling", func() {
		Parses("(and (task TODO) (sort-by priority asc))",
			query.New(query.And(query.Task(content.TaskStatusTodo))).SortBy(query.SortPriority, query.Asc))
		Parses("(and (task TODO) (sort-by priority))",
			query.New(query.And(query.Task(content.TaskStatusTodo))).SortBy(query.SortPriority, query.Desc))
		Parses("(and (task TODO) (sort-by created-at desc) (sort-by title asc))",
			query.New(query.And(query.Task(content.TaskStatusTodo))).
				SortByProperty("created-at", query.Desc).
				SortBy(query.SortTitle, query.Asc))
		Parses("(and (task TODO) (sort-by :journal-day :asc))",
			query.New(query.And(query.Task(content.TaskStatusTodo))).SortBy(query.SortJournalDay, query.Asc))
		Parses("(and (sort-by scheduled asc) (sort-by deadline asc))",
			query.New(nil).SortBy(query.SortScheduled, query.Asc).SortBy(query.SortDeadline, query.Asc))
		Parses("(sort-by priority asc)", query.New(nil).SortBy(query.SortPriority, query.Asc))
		Parses("(and [[a]] (sample 5))", &query.Query{Filter: query.And(query.PageRef("a")), Sample: 5})
	})

	Describe("Whitespace and comments", func() {
		ParsesFilter("(and\n  [[a]], ; first page\n  [[b]])", query.And(query.PageRef("a"), query.PageRef("b")))
	})

	Describe("Errors", func() {
		Fails("(and [[a]]", 0, "missing `)`")
		Fails("(and [[a)", 5, "missing `]]`")
		Fails(`(and "text)`, 5, "missing `\"`")
		Fails("(and [[a]]))", 11, "unexpected text after the query")
		Fails("[[a]] [[b]]", 6, "unexpected text after the query")
		Fails("(foo [[a]])", 1, `unknown filter "foo"`)
		Fails("()", 0, "expected the name of a filter")
		Fails("(and foo)", 5, "expected a filter")
		Fails("(not)", 0, "not takes one or more filters")
		Fails("(task TODO NOPE)", 11, `unknown task status "NOPE"`)
		Fails("(task)", 0, "task takes one or more values")
		Fails("(priority D)", 10, `unknown priority "D"`)
		Fails("(page)", 0, "page takes one name")
		Fails("(property)", 0, "property takes a name")
		Fails(`(property "")`, 10, "expected a name")
		Fails("(property :)", 10, "expected a name")
		Fails("(all-page-tags x)", 0, "all-page-tags takes no arguments")
		Fails("(top-level x)", 0, "top-level takes no arguments")
		Fails("(parent)", 0, "parent takes one block")
		Fails("(parent ((a b)))", 8, "expected the UUID of a block")
		Fails("(parent ((abc)", 8, "missing `))`")
		Fails("(ancestor)", 0, "ancestor takes one block")
		Fails("(parent [[page]] 0 x)", 19, "expected the position of a block")
		Fails(`(title "x")`, 0, "title takes a comparison and a text")
		Fails(`(title has "x")`, 7, "expected =, starts-with")
		Fails(`(content "x")`, 0, "content takes a comparison and a text")
		Fails(`(content has "x")`, 9, "expected =, starts-with")
		Fails(`(content matches "(")`, 17, "expected a regular expression")
		Fails("(has-block)", 0, "has-block takes one filter")
		Fails("(has-child)", 0, "has-child takes one filter")
		Fails("(property rating ~ 7)", 17, "expected =, <")
		Fails("(property rating > nope)", 19, "expected a number or a date")
		Fails("(parent [[page]] -1)", 17, "expected the position of a block")
		Fails("((abc))", 0, "expected a filter")
		Fails("(between today)", 0, "between takes two dates")
		Fails("(between created-at -7d today)", 0, "timestamp property is not supported")
		Fails("(scheduled < soon)", 13, `"soon" is not a date`)
		Fails("(scheduled < right-now-ms)", 13, "is not a date")
		Fails("(scheduled < today tomorrow)", 0, "scheduled takes")
		Fails("(scheduled 2024-13-45)", 11, "is not a valid date")
		Fails("(or [[a]] (sort-by priority))", 10, "sort-by can only be used at the top")
		Fails("(and (sort-by priority up))", 23, "expected `asc` or `desc`")
		Fails("(sample 0)", 8, "above zero")
		Fails("(and #)", 5, "expected a tag")
		Fails("(and ])", 5, "unexpected `]`")
		Fails("(and [[ ]])", 5, "expected a page title")
		Fails("(between [[]] today)", 9, "expected a page title")
		Fails("#[[]]", 1, "expected a page title")
	})

	Describe("Round trips", func() {
		RoundTrips := func(name string, text string) {
			It(name, func() {
				q, err := query.Parse(text)
				Expect(err).ToNot(HaveOccurred())
				Expect(q.String()).To(Equal(text))

				again, err := query.Parse(q.String())
				Expect(err).ToNot(HaveOccurred())
				Expect(again).To(Equal(q))
			})
		}

		RoundTrips("overdue", "(and (task TODO LATER) (or (scheduled < today) (deadline < today)) (sort-by priority asc))")
		RoundTrips("today", "(and (task TODO LATER) (or (and (journal today) (not (scheduled))) (scheduled today) (deadline today)) (not [[shopping]]) (not [[reading list]]) (sort-by priority asc))")
		RoundTrips("next 3 days", "(and (task TODO LATER) (or (and (between +1d +3d) (not (scheduled))) (scheduled +1d +3d) (deadline +1d +3d)) (sort-by priority asc))")
		RoundTrips("all recipes", `(page-property type "Cognitive Recipe")`)
		RoundTrips("sample", "(and (task NOW) (sample 3))")
		RoundTrips("text", `"needle"`)
	})
})
