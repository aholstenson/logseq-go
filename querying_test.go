package logseq_test

import (
	"context"
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
			"reading.md":          "- About reading\n",
			"acme.md":             "alias:: ACME Corp\n\n- A company\n",
			"ada.md":              "organization:: [[Acme]], [[Initech]]\n\n- Engineer\n",
			"bob.md":              "organization:: [[Initech]]\n\n- Manager\n",
			"films.md": "- Good film\n  rating:: 8\n" +
				"- Great film\n  rating:: 10\n" +
				"- Fine film\n  rating:: 6\n",
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

		It("finds only the blocks that reference a page themselves with refs", func() {
			Expect(blocks("(refs shopping)")).To(ConsistOf("#shopping"))
			Expect(blocks("[[shopping]]")).To(ConsistOf("#shopping", "Buy milk"))
		})

		It("finds the pages that other pages use as tags", func() {
			Expect(pages("(all-page-tags)")).To(ConsistOf("reading"))
		})

		It("finds the blocks on the pages that other pages use as tags", func() {
			Expect(blocks(`(and "about" (all-page-tags))`)).To(ConsistOf("About reading"))
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

	Describe("Page queries", func() {
		It("finds journal pages by their date", func() {
			Expect(pages("(journal-page)")).To(HaveLen(2))
			Expect(pages("(journal-page today)")).To(HaveLen(1))
			Expect(pages("(not (journal-page))")).To(ContainElements("tasks", "book"))
			Expect(blocks("(and (task TODO) (journal-page today))")).To(ConsistOf("Call the bank", "Moved task", "Buy milk"))
		})

		It("finds pages by their title", func() {
			Expect(pages(`(title starts-with "projects/")`)).To(ConsistOf("projects/alpha"))
			Expect(pages(`(title includes "recipe")`)).To(ConsistOf("recipe one", "recipe two"))
			Expect(pages(`(title ends-with "TWO")`)).To(ConsistOf("recipe two"))
			Expect(pages(`(title = "Book")`)).To(ConsistOf("book"))
			Expect(pages(`(title starts-with ".")`)).To(BeEmpty())
			Expect(blocks(`(and (task TODO) (title starts-with "boo"))`)).To(ConsistOf("Finish chapter"))
		})

		It("finds pages by their alias", func() {
			Expect(pages("(alias [[acme corp]])")).To(ConsistOf("acme"))
		})

		It("finds pages by the blocks on them", func() {
			Expect(pages("(has-block (priority A))")).To(ConsistOf("tasks"))
			Expect(pages("(not (has-block (task TODO)))")).To(And(ContainElement("films"), Not(ContainElement("book"))))
			Expect(blocks(`(and (task TODO) (has-block "chapter"))`)).To(ConsistOf("Finish chapter"))
		})
	})

	Describe("Sorting", func() {
		It("sorts blocks by a property with numbers in the order of their value", func() {
			Expect(blocks("(and (property rating) (sort-by rating asc))")).
				To(Equal([]string{"Fine film", "Good film", "Great film"}))
		})

		It("sorts by a property in descending order if no order is given", func() {
			Expect(blocks("(and (property rating) (sort-by rating))")).
				To(Equal([]string{"Great film", "Good film", "Fine film"}))
		})

		It("sorts pages by a page property", func() {
			Expect(pages("(and (page-property type) (sort-by type asc))")[0]).To(Equal("book"))
			Expect(pages("(and (page-property type) (sort-by type desc))")[2]).To(Equal("book"))
		})

		It("sorts by a property in an advanced query", func() {
			advanced, err := query.ParseAdvanced(`{:query (property rating)
				:result-transform (fn [r] (reverse (sort-by (fn [b] (get-in b [:block/properties :rating])) r)))}`)
			Expect(err).ToNot(HaveOccurred())

			results, err := graph.Query(ctx, advanced.Query, logseq.QueryAt(now))
			Expect(err).ToNot(HaveOccurred())

			previews := make([]string, 0, len(results.Blocks))
			for _, block := range results.Blocks {
				previews = append(previews, block.Preview())
			}
			Expect(previews).To(Equal([]string{"Great film", "Good film", "Fine film"}))
		})
	})

	Describe("Advanced queries", func() {
		runAdvanced := func(text string, opts ...query.ParseOption) *logseq.QueryResults {
			advanced, err := query.ParseAdvanced(text, opts...)
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

		It("finds the blocks with a property that is not the default", func() {
			results := runAdvanced(`{:query [:find (pull ?b [*])
				:where [?b :block/page ?pg] [?pg :block/name "tasks"] [?b :block/properties ?p]
				[(get ?p :status "none") ?s] (not [(= ?s "none")])]}`)
			Expect(blockPreviews(results)).To(ConsistOf("A note"))
		})

		It("finds the blocks on or about the current page", func() {
			// The page has no file, so only the branch for the refs matches.
			results := runAdvanced(`{:query [:find (pull ?b [*]) :in $ ?name
				:where [?p :block/name ?name] (or [?b :block/refs ?p] [?b :block/page ?p])]
				:inputs ["reading list"]}`)
			Expect(blockPreviews(results)).To(ConsistOf("Reading List"))
		})

		It("finds the tasks without a date", func() {
			results := runAdvanced(`{:query [:find (pull ?b [*])
				:where [?b :block/page ?pg] [?pg :block/name "tasks"] [?b :block/marker "TODO"]
				[(missing? $ ?b :block/scheduled)] [(missing? $ ?b :block/deadline)]]}`)
			Expect(blockPreviews(results)).To(ConsistOf("No dates"))
		})

		It("finds the pages with a property in a set of values", func() {
			results := runAdvanced(`{:query [:find (pull ?p [*])
				:where [?p :block/name] [?p :block/properties ?pp] [(get ?pp :type) ?t]
				[(contains? #{"book" "Cognitive Recipe"} ?t)]]}`)
			Expect(results.Type).To(Equal(logseq.QueryResultPages))

			titles := make([]string, 0, len(results.Pages))
			for _, page := range results.Pages {
				titles = append(titles, page.Title())
			}
			Expect(titles).To(ConsistOf("book", "recipe one", "recipe two"))
		})

		It("finds the blocks that reference a page in a set", func() {
			results := runAdvanced(`{:query [:find (pull ?b [*])
				:where [?b :block/refs ?p] [?p :block/name ?n] [(contains? #{"reading list" "shopping"} ?n)]]}`)
			Expect(blockPreviews(results)).To(ConsistOf("Reading List", "#shopping"))
		})

		It("finds the pages with open tasks", func() {
			results := runAdvanced(`{:query [:find (pull ?p [*])
				:where [?b :block/page ?p] [?b :block/marker "TODO"] [?b :block/priority "A"]]}`)
			Expect(results.Type).To(Equal(logseq.QueryResultPages))
			Expect(results.Pages).To(HaveLen(1))
			Expect(results.Pages[0].Title()).To(Equal("tasks"))
		})

		It("finds the pages by the start of their title", func() {
			results := runAdvanced(`{:query [:find (pull ?p [*])
				:where [?p :block/name ?n] [(clojure.string/starts-with? ?n "recipe")]]}`)
			Expect(results.Type).To(Equal(logseq.QueryResultPages))

			titles := make([]string, 0, len(results.Pages))
			for _, page := range results.Pages {
				titles = append(titles, page.Title())
			}
			Expect(titles).To(ConsistOf("recipe one", "recipe two"))
		})

		It("finds the people of the current page", func() {
			results := runAdvanced(`{:query [:find (pull ?p [*]) :in $ ?target
				:where [?b :block/page ?p] [?p :page/properties ?props]
				[(get ?props :organization) ?t]
				[?me-low :block/name ?target] [?me-low :block/original-name ?me]
				[(get ?t ?me)]]
				:inputs [:current-page]}`, query.WithCurrentPage("Acme"))
			Expect(results.Type).To(Equal(logseq.QueryResultPages))

			titles := make([]string, 0, len(results.Pages))
			for _, page := range results.Pages {
				titles = append(titles, page.Title())
			}
			Expect(titles).To(ConsistOf("ada"))
		})
	})

	Describe("Queries on pages", func() {
		It("finds the queries on a page and runs them", func() {
			Expect(os.WriteFile(filepath.Join(graph.Directory(), "pages", "project.md"), []byte(
				"- Open tasks {{query (and (task TODO) [[project]])}}\n"+
					"- #+BEGIN_QUERY\n"+
					"  {:title [:b \"Linked tasks\"]\n"+
					"   :query [:find (pull ?b [*]) :in $ ?page\n"+
					"           :where [?b :block/marker \"TODO\"] [?b :block/path-refs ?p] [?p :block/name ?page]]\n"+
					"   :inputs [:current-page]}\n"+
					"  #+END_QUERY\n",
			), 0o644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(graph.Directory(), "pages", "work.md"), []byte(
				"- TODO Write the docs for [[project]]\n- TODO Something else\n",
			), 0o644)).To(Succeed())
			graph.Close()

			var err error
			graph, err = logseq.Open(ctx, graph.Directory(), logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())

			page, err := graph.OpenPage("project")
			Expect(err).ToNot(HaveOccurred())

			var nodes content.NodeList
			for _, block := range page.Blocks() {
				nodes = append(nodes, block.Children().FilterDeep(content.IsQuery())...)
			}
			Expect(nodes).To(HaveLen(2))

			var titles, previews []string
			for _, node := range nodes {
				Expect(content.BlockOf(node)).ToNot(BeNil())

				advanced, err := query.ParseNode(node, query.WithCurrentPage(page.Title()))
				Expect(err).ToNot(HaveOccurred())
				titles = append(titles, advanced.Title)

				results, err := graph.Query(ctx, advanced.Query, logseq.QueryAt(now))
				Expect(err).ToNot(HaveOccurred())
				for _, block := range results.Blocks {
					previews = append(previews, block.Preview())
				}
			}

			Expect(titles).To(Equal([]string{"", "Linked tasks"}))
			Expect(previews).To(Equal([]string{"Write the docs for project", "Write the docs for project"}))
		})
	})

	Describe("Queries for the blocks around the query", func() {
		BeforeEach(func() {
			Expect(os.WriteFile(filepath.Join(graph.Directory(), "pages", "outline.md"), []byte(
				"- Section\n"+
					"  id:: 6523a0f8-0000-4c3b-9b1e-000000000001\n"+
					"  - Children {{query (task TODO)}}\n"+
					"    id:: 6523a0f8-0000-4c3b-9b1e-000000000002\n"+
					"    - TODO First child\n"+
					"      - TODO Grandchild\n"+
					"    - TODO Second child\n"+
					"  - TODO Sibling\n"+
					"- TODO Not below\n",
			), 0o644)).To(Succeed())
			graph.Close()

			var err error
			graph, err = logseq.Open(ctx, graph.Directory(), logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())
		})

		run := func(text string) []string {
			page, err := graph.OpenPage("outline")
			Expect(err).ToNot(HaveOccurred())

			nodes := page.Blocks()[0].Children().FilterDeep(content.IsQuery())
			Expect(nodes).To(HaveLen(1))
			block := content.BlockOf(nodes[0])
			parent := block.Parent().(*content.Block)

			advanced, err := query.ParseAdvanced(text,
				query.WithCurrentBlock(block.ID()),
				query.WithParentBlock(parent.ID()))
			Expect(err).ToNot(HaveOccurred())

			results, err := graph.Query(ctx, advanced.Query, logseq.QueryAt(now))
			Expect(err).ToNot(HaveOccurred())

			previews := make([]string, 0, len(results.Blocks))
			for _, result := range results.Blocks {
				previews = append(previews, result.Preview())
			}
			return previews
		}

		It("finds the blocks below the block with the query", func() {
			Expect(run(`{:query [:find (pull ?b [*]) :in $ ?c :where [?b :block/parent ?c]] :inputs [:current-block]}`)).
				To(ConsistOf("First child", "Second child"))
		})

		It("finds the blocks next to the block with the query", func() {
			Expect(run(`{:query [:find (pull ?b [*]) :in $ ?p :where [?b :block/marker "TODO"] [?b :block/parent ?p]] :inputs [:parent-block]}`)).
				To(ConsistOf("Sibling"))
		})

		It("finds the blocks below a block at any depth with a rule", func() {
			rules := `:rules [[(below ?b ?a) [?b :block/parent ?a]]
				[(below ?b ?a) [?b :block/parent ?c] (below ?c ?a)]]`

			Expect(run(`{:query [:find (pull ?b [*]) :in $ ?c % :where (below ?b ?c)] :inputs [:current-block] ` + rules + `}`)).
				To(ConsistOf("First child", "Grandchild", "Second child"))
			Expect(run(`{:query [:find (pull ?b [*]) :in $ ?p % :where [?b :block/marker "TODO"] (below ?b ?p)] :inputs [:parent-block] ` + rules + `}`)).
				To(ConsistOf("First child", "Grandchild", "Second child", "Sibling"))
		})
	})

	Describe("Queries for the blocks around the query without ids", func() {
		It("finds the action items below the parent of the query", func() {
			Expect(os.WriteFile(filepath.Join(graph.Directory(), "pages", "plain.md"), []byte(
				"- Intro\n"+
					"- Section\n"+
					"  - TODO Write [[action]]\n"+
					"    - TODO Nested [[action]]\n"+
					"  - #+BEGIN_QUERY\n"+
					"    {:title [:b \"Action items\"]\n"+
					"     :query [:find (pull ?b [*]) :in $ ?parent %\n"+
					"       :where [?p :block/name \"action\"] [?b :block/refs ?p] (check-parent ?b ?parent)]\n"+
					"     :rules [[(check-parent ?b ?parent) [?b :block/parent ?parent]]\n"+
					"             [(check-parent ?b ?parent) [?b :block/parent ?c] (check-parent ?c ?parent)]]\n"+
					"     :inputs [:parent-block]}\n"+
					"    #+END_QUERY\n"+
					"- TODO Outside [[action]]\n",
			), 0o644)).To(Succeed())
			graph.Close()

			var err error
			graph, err = logseq.Open(ctx, graph.Directory(), logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())

			page, err := graph.OpenPage("plain")
			Expect(err).ToNot(HaveOccurred())

			var nodes content.NodeList
			for _, block := range page.Blocks() {
				nodes = append(nodes, block.Children().FilterDeep(content.IsQuery())...)
			}
			Expect(nodes).To(HaveLen(1))

			advanced, err := query.ParseNode(nodes[0], query.WithQueryPage(page.Title()))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(and (refs [[action]]) (ancestor [[plain]] 1))"))

			results, err := graph.Query(ctx, advanced.Query, logseq.QueryAt(now))
			Expect(err).ToNot(HaveOccurred())

			previews := make([]string, 0, len(results.Blocks))
			for _, block := range results.Blocks {
				previews = append(previews, block.Preview())
			}
			Expect(previews).To(ConsistOf("Write action", "Nested action"))
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
		It("reports a sort by a property without a name in a query built in code", func() {
			_, err := graph.Query(ctx, query.New(nil).SortByProperty("", query.Asc))
			Expect(err).To(MatchError(ContainSubstring("has no name")))
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
