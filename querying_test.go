package logseq_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Query", func() {
	var (
		graph *logseq.Graph
		ctx   context.Context
	)

	// now is a Monday afternoon, the day of the journal with today's tasks.
	now := time.Date(2024, time.January, 15, 14, 0, 0, 0, time.Local)

	BeforeEach(func() {
		dir := setupGraph()
		ctx = context.Background()

		journals := map[string]string{
			"2024_01_15.md": "- TODO [#B] Call the bank\n" +
				"- TODO Moved task\n  SCHEDULED: <2024-01-17 Wed>\n" +
				"- DONE Walk the dog\n" +
				"- #shopping\n  - TODO Buy milk\n",
			"2024_01_14.md": "- TODO Old journal task\n",
		}
		for name, text := range journals {
			Expect(os.WriteFile(filepath.Join(dir, "journals", name), []byte(text), 0o644)).To(Succeed())
		}

		graph = openGraphWithPages(dir, map[string]string{
			"tasks.md": "- TODO [#A] Overdue scheduled\n  SCHEDULED: <2024-01-10 Wed>\n" +
				"- LATER Overdue deadline\n  DEADLINE: <2024-01-12 Fri>\n" +
				"- TODO Due today\n  DEADLINE: <2024-01-15 Mon 09:00>\n" +
				"- TODO [#C] In two days\n  SCHEDULED: <2024-01-17 Wed>\n" +
				"- TODO In four days\n  SCHEDULED: <2024-01-19 Fri>\n" +
				"- TODO No dates\n" +
				"- [[Reading List]]\n  - TODO Read a book\n    SCHEDULED: <2024-01-15 Mon>\n" +
				"- A note\n  status:: draft\n",
			"recipe one.md":       "type:: [[Cognitive Recipe]]\n\n- step one\n",
			"recipe two.md":       "type:: Cognitive Recipe\n\n- step two\n",
			"book.md":             "type:: book\ntags:: reading\n\n- TODO Finish chapter\n",
			"projects___alpha.md": "- alpha\n",
		})
	})

	AfterEach(func() {
		graph.Close()
	})

	run := func(text string) *logseq.QueryResults {
		q, err := query.Parse(text)
		Expect(err).ToNot(HaveOccurred())

		results, err := graph.Query(ctx, q, logseq.QueryAt(now))
		Expect(err).ToNot(HaveOccurred())
		return results
	}

	blocks := func(text string) []string {
		results := run(text)
		Expect(results.Type).To(Equal(logseq.QueryResultBlocks))
		Expect(results.Pages).To(BeNil())

		previews := make([]string, 0, len(results.Blocks))
		for _, block := range results.Blocks {
			previews = append(previews, block.Preview())
		}
		return previews
	}

	pages := func(text string) []string {
		results := run(text)
		Expect(results.Type).To(Equal(logseq.QueryResultPages))
		Expect(results.Blocks).To(BeNil())

		titles := make([]string, 0, len(results.Pages))
		for _, page := range results.Pages {
			titles = append(titles, page.Title())
		}
		return titles
	}

	queryError := func(text string) error {
		q, err := query.Parse(text)
		Expect(err).ToNot(HaveOccurred())

		_, err = graph.Query(ctx, q, logseq.QueryAt(now))
		Expect(err).To(HaveOccurred())
		return err
	}

	Describe("Example queries", func() {
		It("finds overdue tasks", func() {
			Expect(blocks("(and (task TODO LATER) (or (scheduled < today) (deadline < today)) (sort-by priority asc))")).
				To(Equal([]string{"Overdue scheduled", "Overdue deadline"}))
		})

		It("finds the tasks for today", func() {
			Expect(blocks("(and (task TODO LATER)" +
				" (or (and (journal today) (not (scheduled))) (scheduled today) (deadline today))" +
				" (not [[shopping]]) (not [[reading list]])" +
				" (sort-by priority asc))")).
				To(Equal([]string{"Call the bank", "Due today"}))
		})

		It("finds the tasks for the next three days", func() {
			Expect(blocks("(and (task TODO LATER)" +
				" (or (and (between +1d +3d) (not (scheduled))) (scheduled +1d +3d) (deadline +1d +3d))" +
				" (sort-by priority asc))")).
				To(Equal([]string{"In two days", "Moved task"}))
		})

		It("finds pages by a property that references a page", func() {
			Expect(pages("(page-property type [[Cognitive Recipe]])")).
				To(ConsistOf("recipe one", "recipe two"))
		})
	})

	Describe("Filters", func() {
		It("finds blocks by text", func() {
			Expect(blocks(`"bank"`)).To(ConsistOf("Call the bank"))
		})

		It("finds blocks on a page", func() {
			Expect(blocks("(and (page [[Book]]) (task TODO))")).To(ConsistOf("Finish chapter"))
		})

		It("finds blocks by a property", func() {
			Expect(blocks("(property status draft)")).To(ConsistOf("A note"))
			Expect(blocks("(property status)")).To(ConsistOf("A note"))
		})

		It("finds blocks by priority", func() {
			Expect(blocks("(priority A C)")).To(ConsistOf("Overdue scheduled", "In two days"))
		})

		It("finds blocks on journals between journal titles", func() {
			Expect(blocks("(and (task TODO) (between [[Jan 14th, 2024]] [[Jan 15th, 2024]]))")).
				To(ConsistOf("Call the bank", "Moved task", "Buy milk", "Old journal task"))
		})

		It("accepts the dates of a range in either order", func() {
			Expect(blocks("(and (task TODO) (between today -1d))")).
				To(ConsistOf("Call the bank", "Moved task", "Buy milk", "Old journal task"))
		})

		It("finds blocks with a deadline or a scheduled date", func() {
			Expect(blocks("(and (task TODO LATER) (deadline))")).To(ConsistOf("Overdue deadline", "Due today"))
			Expect(blocks("(and (task TODO) (scheduled > +2d))")).To(ConsistOf("In four days"))
			Expect(blocks("(and (task TODO) (scheduled >= +2d))")).To(ConsistOf("In two days", "Moved task", "In four days"))
			Expect(blocks("(and (task TODO) (scheduled <= 2024-01-10))")).To(ConsistOf("Overdue scheduled"))
		})

		It("leaves out blocks with a filter on its own in not", func() {
			results := blocks("(not [[shopping]])")
			Expect(results).To(ContainElement("Call the bank"))
			Expect(results).ToNot(ContainElement("Buy milk"))
		})

		It("finds pages by tags", func() {
			Expect(pages("(page-tags reading)")).To(ConsistOf("book"))
		})

		It("finds pages in a namespace", func() {
			Expect(pages("(namespace [[projects]])")).To(ConsistOf("projects/alpha"))
		})

		It("finds all pages without a filter", func() {
			Expect(pages("")).To(ContainElements("tasks", "book", "Jan 15th, 2024"))
		})
	})

	Describe("Page filters in queries for blocks", func() {
		It("matches the page a block is on", func() {
			Expect(blocks("(and (task TODO) (page-property type book))")).To(ConsistOf("Finish chapter"))
		})

		It("leaves out the blocks on the pages with not", func() {
			results := blocks("(and (task TODO) (not (page-property type book)))")
			Expect(results).To(ContainElement("No dates"))
			Expect(results).ToNot(ContainElement("Finish chapter"))
		})

		It("matches nothing when no page matches", func() {
			Expect(blocks("(and (task TODO) (page-property type nothing))")).To(BeEmpty())
		})
	})

	Describe("Advanced queries", func() {
		runAdvanced := func(text string) *logseq.QueryResults {
			advanced, err := query.ParseAdvanced(text)
			Expect(err).ToNot(HaveOccurred())

			results, err := graph.Query(ctx, advanced.Query, logseq.QueryAt(now))
			Expect(err).ToNot(HaveOccurred())
			return results
		}

		blockPreviews := func(results *logseq.QueryResults) []string {
			Expect(results.Type).To(Equal(logseq.QueryResultBlocks))
			previews := make([]string, 0, len(results.Blocks))
			for _, block := range results.Blocks {
				previews = append(previews, block.Preview())
			}
			return previews
		}

		It("finds overdue tasks", func() {
			results := runAdvanced(`{:query [:find (pull ?b [*]) :in $ ?day
				:where [?b :block/marker ?m] [(contains? #{"LATER" "TODO"} ?m)]
				(or-join [?b ?d] [?b :block/scheduled ?d] [?b :block/deadline ?d])
				[(< ?d ?day)]]
				:inputs [:today]
				:result-transform (fn [result] (sort-by (fn [b] (get b :block/priority "Z")) result))}`)
			Expect(blockPreviews(results)).To(Equal([]string{"Overdue scheduled", "Overdue deadline"}))
		})

		It("finds the tasks for today", func() {
			results := runAdvanced(`{:query [:find (pull ?b [*]) :in $ ?today
				:where [?b :block/marker ?m] [(contains? #{"LATER" "TODO"} ?m)]
				(or-join [?b ?d]
					(and [?b :block/page ?p] [?p :block/journal-day ?d] (not [?b :block/scheduled _]))
					[?b :block/scheduled ?d]
					[?b :block/deadline ?d])
				(not [?b :block/path-refs [:block/name "shopping"]])
				(not [?b :block/path-refs [:block/name "reading list"]])
				[(= ?d ?today)]]
				:inputs [:today]
				:result-transform (fn [result] (sort-by (fn [b] (get b :block/priority "Z")) result))}`)
			Expect(blockPreviews(results)).To(Equal([]string{"Call the bank", "Due today"}))
		})

		It("finds the tasks for the next three days", func() {
			results := runAdvanced(`{:query [:find (pull ?b [*]) :in $ ?next ?today
				:where [?b :block/marker ?m] [(contains? #{"LATER" "TODO"} ?m)]
				(or-join [?b ?d]
					(and [?b :block/page ?p] [?p :block/journal-day ?d] (not [?b :block/scheduled _]))
					[?b :block/scheduled ?d]
					[?b :block/deadline ?d])
				[(<= ?d ?next)]
				[(> ?d ?today)]]
				:inputs [:3d-after :today]
				:result-transform (fn [result] (sort-by (fn [b] (get b :block/priority "Z")) result))}`)
			Expect(blockPreviews(results)).To(Equal([]string{"In two days", "Moved task"}))
		})

		It("finds the recipes", func() {
			results := runAdvanced(`{:query [:find (pull ?p [*])
				:where [?b :block/page ?p] [?p :page/properties ?props]
				[(get ?props :type) ?t] [(get ?t "Cognitive Recipe")]]}`)
			Expect(results.Type).To(Equal(logseq.QueryResultPages))

			titles := make([]string, 0, len(results.Pages))
			for _, page := range results.Pages {
				titles = append(titles, page.Title())
			}
			Expect(titles).To(ConsistOf("recipe one", "recipe two"))
		})
	})

	Describe("Results", func() {
		It("picks a sample of the results", func() {
			results := blocks("(and (task TODO) (sort-by priority asc) (sample 2))")
			Expect(results).To(HaveLen(2))
		})

		It("returns more results than one batch of the index", func() {
			var text string
			for i := 0; i < 600; i++ {
				text += "- TODO many\n"
			}
			Expect(os.WriteFile(filepath.Join(graph.Directory(), "pages", "many.md"), []byte(text), 0o644)).To(Succeed())
			graph.Close()

			var err error
			graph, err = logseq.Open(ctx, graph.Directory(), logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())

			Expect(blocks("(and (task TODO) (page [[many]]))")).To(HaveLen(600))
		})

		It("runs a query built in code", func() {
			q := query.New(query.And(
				query.Task(content.TaskStatusTodo, content.TaskStatusLater),
				query.Scheduled(query.Less, query.Today()),
			)).SortBy(query.SortPriority, query.Asc)

			results, err := graph.Query(ctx, q, logseq.QueryAt(now))
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Blocks).To(HaveLen(1))
			Expect(results.Blocks[0].Preview()).To(Equal("Overdue scheduled"))

			block, _, err := results.Blocks[0].Open()
			Expect(err).ToNot(HaveOccurred())
			Expect(block.TaskStatus()).To(Equal(content.TaskStatusTodo))
		})
	})

	Describe("Errors", func() {
		It("does not run all-page-tags yet", func() {
			Expect(errors.Is(queryError("(all-page-tags)"), logseq.ErrUnsupportedQuery)).To(BeTrue())
		})

		It("does not sort by properties yet", func() {
			err := queryError("(and (task TODO) (sort-by rating asc))")
			Expect(errors.Is(err, logseq.ErrUnsupportedQuery)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring(`"rating"`))
		})

		It("reports a journal title that is not a date", func() {
			Expect(queryError("(between [[Not a journal]] today)").Error()).To(ContainSubstring("not the title of a journal"))
		})

		It("reports a date that is missing in a query built in code", func() {
			_, err := graph.Query(ctx, query.New(query.Scheduled(query.Less, query.Date{})))
			Expect(err).To(MatchError(ContainSubstring("date is missing")))
		})
	})
})
