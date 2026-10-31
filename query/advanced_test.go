package query_test

import (
	"errors"
	"strings"

	"github.com/aholstenson/logseq-go/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const overdueQuery = `#+BEGIN_QUERY
{:title [:b "🔥 Overdue"]
 :query [:find (pull ?b [*])
   :in $ ?day
   :where
     [?b :block/marker ?m]
     [(contains? #{"LATER" "TODO"} ?m)]
     (or-join [?b ?d]
            [?b :block/scheduled ?d]
            [?b :block/deadline ?d])
     [(< ?d ?day)]
 ]
 :inputs [:today]
 :result-transform (fn [result] (sort-by (fn [b] (get b :block/priority "Z")) result))
 :breadcrumb-show? false
 :table-view? false
 :collapsed? false
}
#+END_QUERY`

const todayQuery = `#+BEGIN_QUERY
{:title [:b "📅 Today"]
 :query [:find (pull ?b [*])
        :in $ ?today
        :where
        [?b :block/marker ?m]
        [(contains? #{"LATER" "TODO"} ?m)]
        (or-join [?b ?d]
            (and
                [?b :block/page ?p]
                [?p :block/journal-day ?d]
                (not [?b :block/scheduled _])
            )
            [?b :block/scheduled ?d]
            [?b :block/deadline ?d])
        (not [?b :block/path-refs [:block/name "shopping"]])
        (not [?b :block/path-refs [:block/name "reading list"]])
        [(= ?d ?today)]
 ]
 :inputs [:today]
 :result-transform (fn [result] (sort-by (fn [b] (get b :block/priority "Z")) result))
 :breadcrumb-show? false
 :table-view? false
 :collapsed? false
}
#+END_QUERY`

const nextDaysQuery = `#+BEGIN_QUERY
{:title [:b "🛫 Next 3 days"]
 :query [:find (pull ?b [*])
        :in $ ?next ?today
        :where
        [?b :block/marker ?m]
        [(contains? #{"LATER" "TODO"} ?m)]
        (or-join [?b ?d]
            (and
                [?b :block/page ?p]
                [?p :block/journal-day ?d]
                (not [?b :block/scheduled _])
            )
            [?b :block/scheduled ?d]
            [?b :block/deadline ?d])
        [(<= ?d ?next)]
        [(> ?d ?today)]
 ]
 :inputs [:3d-after :today]
 :result-transform (fn [result] (sort-by (fn [b] (get b :block/priority "Z")) result))
 :breadcrumb-show? false
 :table-view? false
 :collapsed? false
}
#+END_QUERY`

const recipesQuery = `#+BEGIN_QUERY
{
:title [:b "All recipes"]
:query [
    :find (pull ?p [*])
    :where
        [?b :block/page ?p]
        [?p :page/properties ?props]
        [(get ?props :type) ?t]
        [(get ?t "Cognitive Recipe")]
]
}
#+END_QUERY`

var _ = Describe("Advanced queries", func() {
	Translates := func(name string, text string, expected string) {
		It(name, func() {
			advanced, err := query.ParseAdvanced(text)
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal(expected))
		})
	}

	// where translates a query with the given :where clauses that finds
	// blocks.
	where := func(clauses string) string {
		return "{:query [:find (pull ?b [*]) :where " + clauses + "]}"
	}

	Unsupported := func(name string, text string, message string) {
		It(name, func() {
			_, err := query.ParseAdvanced(text)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, query.ErrUnsupported)).To(BeTrue(), err.Error())

			var unsupported *query.UnsupportedError
			Expect(errors.As(err, &unsupported)).To(BeTrue())
			Expect(unsupported.Message).To(ContainSubstring(message))
			Expect(text[unsupported.Offset:]).To(HavePrefix(strings.Fields(unsupported.Form)[0]))
		})
	}

	Describe("Example queries", func() {
		Translates("overdue", overdueQuery,
			"(and (task LATER TODO) (or (scheduled < today) (deadline < today)) (sort-by priority asc))")
		Translates("today", todayQuery,
			"(and (task LATER TODO) (or (and (journal today) (not (scheduled))) (scheduled today) (deadline today)) (not [[shopping]]) (not [[reading list]]) (sort-by priority asc))")
		Translates("next 3 days", nextDaysQuery,
			"(and (task LATER TODO) (or (and (between +1d +3d) (not (scheduled))) (scheduled +1d +3d) (deadline +1d +3d)) (sort-by priority asc))")
		Translates("all recipes", recipesQuery,
			`(and (page-property type "Cognitive Recipe") (has-block))`)

		It("finds pages for the recipes and blocks for the others", func() {
			for _, text := range []string{overdueQuery, todayQuery, nextDaysQuery} {
				advanced, err := query.ParseAdvanced(text)
				Expect(err).ToNot(HaveOccurred())
				Expect(advanced.Query.FindsBlocks()).To(BeTrue())
			}

			advanced, err := query.ParseAdvanced(recipesQuery)
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.FindsBlocks()).To(BeFalse())
		})

		It("reads the title and collapsed", func() {
			advanced, err := query.ParseAdvanced(overdueQuery)
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Title).To(Equal("🔥 Overdue"))
			Expect(advanced.Collapsed).To(BeFalse())

			advanced, err = query.ParseAdvanced(`{:title "Plain" :collapsed? true :query (task TODO)}`)
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Title).To(Equal("Plain"))
			Expect(advanced.Collapsed).To(BeTrue())
		})
	})

	Describe("Display options", func() {
		groupByPage := func(text string) bool {
			advanced, err := query.ParseAdvanced(text)
			Expect(err).ToNot(HaveOccurred())
			return advanced.GroupByPage
		}

		breadcrumbShow := func(text string) bool {
			advanced, err := query.ParseAdvanced(text)
			Expect(err).ToNot(HaveOccurred())
			return advanced.BreadcrumbShow
		}

		It("groups by page by default", func() {
			Expect(groupByPage(`{:query (task TODO)}`)).To(BeTrue())
		})

		It("does not group by page with a result transform", func() {
			Expect(groupByPage(overdueQuery)).To(BeFalse())
		})

		It("does not group by page with (by-page false)", func() {
			Expect(groupByPage(`{:query "(and (task TODO) (by-page false))"}`)).To(BeFalse())
			Expect(groupByPage(`{:query (and (task TODO) (by-page false))}`)).To(BeFalse())
		})

		It("uses :group-by-page? before the rest of the query", func() {
			Expect(groupByPage(`{:query (and (task TODO) (by-page false)) :group-by-page? true}`)).To(BeTrue())
			Expect(groupByPage(`{:query (task TODO) :group-by-page? false}`)).To(BeFalse())
			Expect(groupByPage(`{:query (task TODO) :group-by-page? nil}`)).To(BeFalse())
			Expect(groupByPage(`{:query [:find (pull ?b [*]) :where [?b :block/marker "TODO"]]
				:result-transform (fn [r] (sort-by (fn [b] (get b :block/priority "Z")) r))
				:group-by-page? true}`)).To(BeTrue())
		})

		It("shows breadcrumbs unless :breadcrumb-show? is false", func() {
			Expect(breadcrumbShow(`{:query (task TODO)}`)).To(BeTrue())
			Expect(breadcrumbShow(`{:query (task TODO) :breadcrumb-show? true}`)).To(BeTrue())
			Expect(breadcrumbShow(overdueQuery)).To(BeFalse())
		})
	})

	Describe("Block attributes", func() {
		Translates("marker value", where(`[?b :block/marker "TODO"]`), "(task TODO)")
		Translates("any marker", where(`[?b :block/marker _]`),
			"(task TODO DOING DONE LATER NOW CANCELLED CANCELED IN-PROGRESS WAIT WAITING)")
		Translates("marker not equal", where(`[?b :block/marker ?m] [(not= ?m "DONE")]`),
			"(and (task TODO DOING DONE LATER NOW CANCELLED CANCELED IN-PROGRESS WAIT WAITING) (not (task DONE)))")
		Translates("marker in a not", where(`[?b :block/marker ?m] (not [(contains? #{"DONE" "CANCELED"} ?m)])`),
			"(and (task TODO DOING DONE LATER NOW CANCELLED CANCELED IN-PROGRESS WAIT WAITING) (not (task DONE CANCELED)))")
		Translates("priority", where(`[?b :block/priority ?p] [(contains? #{"A" "B"} ?p)]`), "(priority A B)")
		Translates("priority equal", where(`[?b :block/priority ?p] [(= "A" ?p)]`), "(priority A)")
		Translates("content", where(`[?b :block/content ?c] [(clojure.string/includes? ?c "bank")]`), `(content includes "bank")`)
		Translates("content start", where(`[?b :block/content ?c] [(clojure.string/starts-with? ?c "TODO")]`), `(content starts-with "TODO")`)
		Translates("content end", where(`[?b :block/content ?c] [(clojure.string/ends-with? ?c "!")]`), `(content ends-with "!")`)
		Translates("content equal", where(`[?b :block/content ?c] [(= "Call the bank" ?c)]`), `(content = "Call the bank")`)
		Translates("content as a value", where(`[?b :block/content "Call the bank"]`), `(content = "Call the bank")`)
		Translates("content with a reference", where(`[?b :block/content ?c] [(clojure.string/includes? ?c "[[Money]]")]`),
			`(content includes "[[Money]]")`)
		Translates("content with a regex", where(`[?b :block/content ?c] [(re-find #"bank\d" ?c)]`), `(content matches "bank\\d")`)
		Translates("content with re-pattern", where(`[(re-pattern "bank\\d") ?re] [?b :block/content ?c] [(re-find ?re ?c)]`),
			`(content matches "bank\\d")`)
		Translates("content with re-matches", where(`[?b :block/content ?c] [(re-matches #"TODO .*" ?c)]`), `(content matches "^(?:TODO .*)$")`)
		Translates("content with a regex in a not",
			where(`[?b :block/marker "TODO"] [?b :block/content ?c] (not [(re-find #"(?i)bank" ?c)])`),
			`(and (task TODO) (not (content matches "(?i)bank")))`)
		Translates("path refs with a name", where(`[?b :block/path-refs ?p] [?p :block/name "project"]`), "[[project]]")
		Translates("path refs with a lookup", where(`[?b :block/path-refs [:block/original-name "Project"]]`), "[[Project]]")
		Translates("parent with a lookup", where(`[?b :block/parent [:block/uuid #uuid "abc-123"]]`), "(parent ((abc-123)))")
		Translates("refs with a name", where(`[?b :block/refs ?p] [?p :block/name "project"]`), "(refs [[project]])")
		Translates("refs with a lookup", where(`[?b :block/refs [:block/name "project"]]`), "(refs [[project]])")
		Translates("page by name", where(`[?b :block/page ?p] [?p :block/name "project"] [?b :block/marker "TODO"]`),
			"(and (page [[project]]) (task TODO))")
		Translates("page by lookup", where(`[?b :block/page [:block/name "project"]]`), "(page [[project]])")
		Translates("on a journal", where(`[?b :block/page ?p] [?p :block/journal? true]`), "(journal)")
		Translates("scheduled on a journal day", where(`[?b :block/scheduled 20240115]`), "(scheduled 2024-01-15)")
		Translates("scheduled compared from the other side", where(`[?b :block/scheduled ?d] [(>= 20240101 ?d)]`), "(scheduled <= 2024-01-01)")
		Translates("deadline in a range", where(`[?b :block/deadline ?d] [(>= ?d 20240101)] [(< ?d 20240201)]`), "(deadline 2024-01-01 2024-01-31)")
		Translates("block property", where(`[?b :block/properties ?props] [(get ?props :status) ?s] [(= ?s "draft")]`), "(property status draft)")
		Translates("block property with get-in", where(`[(get-in ?b [:block/properties :status]) ?s] [(= ?s "draft")]`), "(property status draft)")
		Translates("has block property", where(`[?b :block/properties ?props] [(get ?props :status) ?s]`), "(property status)")
		Translates("property with a default that is compared",
			where(`[?b :block/properties ?props] [(get ?props :type "none") ?t] [(= ?t "none")]`),
			"(or (property type none) (not (property type)))")
		Translates("property with a default that is not compared",
			where(`[?b :block/properties ?props] [(get ?props :type "none") ?t] [(= ?t "book")]`),
			"(property type book)")
		Translates("property with a default without conditions",
			where(`[?b :block/marker "TODO"] [?b :block/properties ?props] [(get ?props :type "none") ?t]`),
			"(task TODO)")
		Translates("property with a default in a not",
			where(`[?b :block/marker "TODO"] [?b :block/properties ?props] [(get ?props :type "none") ?t] (not [(= ?t "none")])`),
			"(and (task TODO) (not (or (property type none) (not (property type)))))")
		Translates("property with nil as the default",
			where(`[?b :block/properties ?props] [(get ?props :type nil) ?t]`),
			"(property type)")
		Translates("missing", where(`[?b :block/marker "TODO"] [(missing? $ ?b :block/scheduled)]`), "(and (task TODO) (not (scheduled)))")
		Translates("missing in a not", where(`[?b :block/marker "TODO"] (not [(missing? $ ?b :block/deadline)])`),
			"(and (task TODO) (not (not (deadline))))")
		Translates("missing journal day", where(`[?b :block/marker "TODO"] [?b :block/page ?p] [(missing? $ ?p :block/journal-day)]`),
			"(and (task TODO) (not (journal)))")
		Translates("property as a condition", where(`[?b :block/properties ?props] [(get ?props :type)]`), "(property type)")
		Translates("property with a default as a condition",
			where(`[?b :block/marker "TODO"] [?b :block/properties ?props] [(get ?props :type "x")]`), "(task TODO)")
		Translates("property in a set", where(`[?b :block/properties ?props] [(get ?props :type) ?t] [(contains? #{"book" "film"} ?t)]`),
			"(or (property type book) (property type film))")
		Translates("property in a set with a default",
			where(`[?b :block/properties ?props] [(get ?props :type "none") ?t] [(contains? #{"none" "book"} ?t)]`),
			"(or (or (property type none) (not (property type))) (property type book))")
		Translates("property with get-in and a default",
			where(`[(get-in ?b [:block/properties :type] "none") ?t] [(= ?t "none")]`),
			"(or (property type none) (not (property type)))")
	})

	Describe("Pages of blocks", func() {
		Translates("a second variable for the page in an or",
			where(`[?b :block/page ?p] [?p :block/journal? true] (or [?b :block/marker "TODO"] [?b :block/page ?q])`),
			"(journal)")
		Translates("a second variable for the page in a not",
			where(`[?b :block/marker "TODO"] [?b :block/page ?p] (not [?b :block/page ?q] [?q :block/journal? true])`),
			"(and (task TODO) (not (journal)))")
		Translates("a page with a title in a not keeps its variable",
			where(`[?b :block/marker "TODO"] [?b :block/page ?p] [?p :block/journal? true] (not [?b :block/page ?q] [?q :block/name "x"])`),
			"(and (task TODO) (journal) (not (page [[x]])))")
		Translates("page property of the page of a block",
			where(`[?b :block/marker "TODO"] [?b :block/page ?p] [?p :block/properties ?props] [(get ?props :type) ?t] [(= ?t "book")]`),
			"(and (task TODO) (page-property type book))")
		Translates("page tags", "{:query [:find (pull ?p [*]) :where [?p :block/tags ?t] [?t :block/name \"book\"]]}", "(page-tags [[book]])")
		Translates("title only given a variable",
			"{:query [:find (pull ?p [*]) :where [?p :block/name ?name] [?p :block/properties ?props] [(get ?props :type) ?t] [(= ?t \"book\")]]}",
			"(page-property type book)")
		Translates("namespace", "{:query [:find ?p :where [?p :block/namespace [:block/name \"projects\"]]]}", "(namespace [[projects]])")
		Translates("a second variable for the page of the blocks",
			`{:query [:find (pull ?q [*]) :where [?b :block/page ?p] [?p :block/journal? true] [?b :block/page ?q] [?b :block/marker "TODO"]]}`,
			"(and (journal-page) (has-block (task TODO)))")
		Translates("page property as a condition in a not",
			`{:query [:find (pull ?p [*]) :where [?p :block/name] [?p :block/properties ?pp] (not [(get ?pp :type)])]}`,
			"(not (page-property type))")
		Translates("page property with contains?",
			`{:query [:find (pull ?p [*]) :where [?p :block/name] [?p :block/properties ?pp] [(contains? ?pp :type)]]}`,
			"(page-property type)")
		Translates("tags with titles in a set",
			`{:query [:find (pull ?p [*]) :where [?p :block/tags ?t] [?t :block/name ?n] [(contains? #{"book" "film"} ?n)]]}`,
			"(page-tags [[book]] [[film]])")
		Translates("page of a block with titles in a set",
			where(`[?b :block/marker "TODO"] [?b :block/page ?p] [?p :block/name ?n] [(contains? #{"a" "b"} ?n)]`),
			"(and (task TODO) (or (page [[a]]) (page [[b]])))")
		Translates("path refs with titles in a set", where(`[?b :block/path-refs ?p] [?p :block/name ?n] [(contains? #{"a" "b"} ?n)]`),
			"(or [[a]] [[b]])")
		Translates("refs with a title compared", where(`[?b :block/refs ?p] [?p :block/original-name ?n] [(= ?n "Project")]`),
			"(refs [[Project]])")
		Translates("titles in a set for each branch of an or",
			where(`[?p :block/name ?n] [(contains? #{"a" "b"} ?n)] (or [?b :block/refs ?p] [?b :block/page ?p])`),
			"(or (or (refs [[a]]) (refs [[b]])) (or (page [[a]]) (page [[b]])))")
		Translates("page property with a default",
			`{:query [:find (pull ?p [*]) :where [?p :block/name] [?p :block/properties ?pp] [(get ?pp :status "open") ?s] [(= ?s "open")]]}`,
			"(or (page-property status open) (not (page-property status)))")
	})

	Describe("Page queries", func() {
		pagesWhere := func(clauses string) string {
			return "{:query [:find (pull ?p [*]) :where " + clauses + "]}"
		}

		Translates("pages by title", pagesWhere(`[?p :block/name "x"]`), `(title = "x")`)
		Translates("pages by the start of their title",
			pagesWhere(`[?p :block/name ?n] [(clojure.string/starts-with? ?n "projects/")]`), `(title starts-with "projects/")`)
		Translates("title with a regex", pagesWhere(`[?p :block/name ?n] [(re-find #"^proj" ?n)]`), `(title matches "^proj")`)
		Translates("titles in an or",
			pagesWhere(`[?p :block/name ?n] (or [(clojure.string/starts-with? ?n "a")] [(clojure.string/ends-with? ?n "b")])`),
			`(or (title starts-with "a") (title ends-with "b"))`)
		Translates("title of the page of a block",
			where(`[?b :block/marker "TODO"] [?b :block/page ?p] [?p :block/original-name ?n] [(clojure.string/includes? ?n "Meeting")]`),
			`(and (task TODO) (title includes "Meeting"))`)
		Translates("journal pages", pagesWhere(`[?p :block/journal? true]`), "(journal-page)")
		Translates("pages that are not journals", pagesWhere(`[?p :block/name] (not [?p :block/journal? true])`), "(not (journal-page))")
		Translates("pages with journal? false", pagesWhere(`[?p :block/journal? false]`), "(not (journal-page))")
		Translates("pages without a journal day", pagesWhere(`[?p :block/name] [(missing? $ ?p :block/journal-day)]`), "(not (journal-page))")
		Translates("journal pages by date",
			`{:query [:find (pull ?p [*]) :in $ ?start ?end :where [?p :block/journal-day ?d] [(>= ?d ?start)] [(<= ?d ?end)]] :inputs [:-7d :today]}`,
			"(journal-page -7d today)")
		Translates("pages by alias", pagesWhere(`[?p :block/alias ?a] [?a :block/name "acme corp"]`), "(alias [[acme corp]])")
		Translates("pages by alias with a lookup", pagesWhere(`[?p :block/alias [:block/name "acme corp"]]`), "(alias [[acme corp]])")
		Translates("pages by their blocks", pagesWhere(`[?b :block/page ?p] [?b :block/marker "TODO"]`), "(has-block (task TODO))")
		Translates("pages with a block", pagesWhere(`[?b :block/page ?p]`), "(has-block)")
		Translates("pages with a block and a condition", pagesWhere(`[?b :block/page ?p] [?p :block/journal? true]`), "(and (journal-page) (has-block))")
		Translates("pages without a block", pagesWhere(`[?p :block/name] (not [?b :block/page ?p])`), "(not (has-block))")
		Translates("pages without a condition on blocks", pagesWhere(`[?p :block/name]`), "")
		Translates("pages by their blocks in an or",
			pagesWhere(`[?b :block/page ?p] (or [?b :block/marker "TODO"] [?b :block/marker "DOING"])`),
			"(has-block (or (task TODO) (task DOING)))")
		Translates("pages by their blocks and their own properties",
			pagesWhere(`[?b :block/page ?p] [?b :block/marker "TODO"] [?p :block/properties ?pp] [(get ?pp :type) ?t] [(= ?t "book")]`),
			"(and (page-property type book) (has-block (task TODO)))")
		Translates("pages by their blocks and their title",
			pagesWhere(`[?b :block/page ?p] [?p :block/name "x"] [?b :block/marker "TODO"]`),
			`(and (title = "x") (has-block (task TODO)))`)
		Translates("pages without blocks that match",
			pagesWhere(`[?p :block/name] (not-join [?p] [?b :block/page ?p] [?b :block/marker "TODO"])`),
			"(not (has-block (task TODO)))")
		Translates("pages without blocks that match in a not", pagesWhere(`[?p :block/name] (not [?b :block/page ?p] [?b :block/marker "TODO"])`),
			"(not (has-block (task TODO)))")
		Translates("pages with blocks that match and without others",
			pagesWhere(`[?b :block/page ?p] [?b :block/marker "TODO"] (not-join [?p] [?c :block/page ?p] [?c :block/marker "DONE"])`),
			"(and (not (has-block (task DONE))) (has-block (task TODO)))")
		Translates("pages with blocks that match in a branch of an or",
			pagesWhere(`[?p :block/name] (or (and [?b :block/page ?p] [?b :block/marker "TODO"]) [?p :block/journal? true])`),
			"(or (has-block (task TODO)) (journal-page))")
		Translates("the same block in a not", pagesWhere(`[?b :block/page ?p] [?b :block/marker "TODO"] (not [?b :block/priority "A"])`),
			"(has-block (and (task TODO) (not (priority A))))")
		Translates("journal pages by date and their blocks",
			`{:query [:find (pull ?p [*]) :in $ ?s :where [?p :block/journal-day ?d] [(>= ?d ?s)] [?b :block/page ?p] [?b :block/marker "TODO"]] :inputs [:-7d]}`,
			"(and (journal-page >= -7d) (has-block (task TODO)))")
	})

	Describe("References to pages that match", func() {
		Translates("refs to tagged pages",
			where(`[?b :block/marker "TODO"] [?b :block/refs ?p] [?p :block/tags ?t] [?t :block/name "project"]`),
			"(and (refs (page-tags [[project]])) (task TODO))")
		Translates("refs to pages with a property",
			where(`[?b :block/refs ?p] [?p :block/properties ?pp] [(get ?pp :status) ?s] [(= ?s "active")]`),
			"(refs (page-property status active))")
		Translates("refs to pages by the start of their title",
			where(`[?b :block/refs ?p] [?p :block/name ?n] [(clojure.string/starts-with? ?n "x")]`),
			`(refs (title starts-with "x"))`)
		Translates("refs to pages that include a text",
			where(`[?b :block/refs ?p] [?p :block/name ?n] [(clojure.string/includes? ?n "x")]`),
			`(refs (title includes "x"))`)
		Translates("references with a regex", where(`[?b :block/refs ?p] [?p :block/name ?n] [(re-matches #"proj.*" ?n)]`),
			`(refs (title matches "^(?:proj.*)$"))`)
		Translates("path refs to pages with a rule",
			where(`[?b :block/marker "TODO"] [?b :block/path-refs ?p] (page-tags ?p #{"project"})`),
			"(and (page-ref (page-tags [[project]])) (task TODO))")
		Translates("refs to journals",
			`{:query [:find (pull ?b [*]) :in $ ?s :where [?b :block/refs ?p] [?p :block/journal-day ?d] [(>= ?d ?s)]] :inputs [:-7d]}`,
			"(refs (journal-page >= -7d))")
		Translates("refs to pages with blocks that match",
			where(`[?b :block/refs ?p] [?c :block/page ?p] [?c :block/marker "TODO"]`),
			"(refs (has-block (task TODO)))")
		Translates("refs in a not",
			where(`[?b :block/marker "TODO"] (not [?b :block/refs ?p] [?p :block/tags ?t] [?t :block/name "someday"])`),
			"(and (task TODO) (not (refs (page-tags [[someday]]))))")
		Translates("pages with blocks that refer to pages that match",
			"{:query [:find (pull ?p [*]) :where [?b :block/page ?p] [?b :block/refs ?r] [?r :block/tags ?t] [?t :block/name \"project\"]]}",
			"(has-block (refs (page-tags [[project]])))")
	})

	Describe("Comparisons of property values", func() {
		withRating := func(clauses string) string {
			return where(`[?b :block/properties ?p] [(get ?p :rating) ?r] ` + clauses)
		}

		Translates("a number", withRating(`[(> ?r 7)]`), "(property rating > 7)")
		Translates("a number on the other side", withRating(`[(< 7 ?r)]`), "(property rating > 7)")
		Translates("a range of numbers", withRating(`[(>= ?r 6)] [(< ?r 10)]`), "(and (property rating >= 6) (property rating < 10))")
		Translates("a number in a not", withRating(`(not [(> ?r 7)])`), "(and (property rating) (not (property rating > 7)))")
		Translates("a number as an input",
			`{:query [:find (pull ?b [*]) :in $ ?min :where [?b :block/properties ?p] [(get ?p :rating) ?r] [(>= ?r ?min)]] :inputs [8]}`,
			"(property rating >= 8)")
		Translates("= with a number compares the text", withRating(`[(= ?r 8)]`), "(property rating 8)")
		Translates("a date input",
			`{:query [:find (pull ?b [*]) :in $ ?today :where [?b :block/properties ?p] [(get ?p :due) ?d] [(< ?d ?today)]] :inputs [:today]}`,
			"(property due < today)")
		Translates("= with a date input",
			`{:query [:find (pull ?b [*]) :in $ ?today :where [?b :block/properties ?p] [(get ?p :due) ?d] [(= ?d ?today)]] :inputs [:today]}`,
			"(property due = today)")
		Translates("a date as text", where(`[?b :block/properties ?p] [(get ?p :due) ?d] [(>= ?d "2024-01-12")]`), "(property due >= 2024-01-12)")
		Translates("a page property",
			`{:query [:find (pull ?p [*]) :where [?p :block/name] [?p :block/properties ?pp] [(get ?pp :pages) ?n] [(> ?n 300)]]}`,
			"(page-property pages > 300)")

		Translates("a default that passes", where(`[?b :block/properties ?p] [(get ?p :rating 0) ?r] [(< ?r 7)]`),
			"(or (property rating < 7) (not (property rating)))")
		Translates("a default that does not pass", where(`[?b :block/properties ?p] [(get ?p :rating 0) ?r] [(> ?r 7)]`),
			"(property rating > 7)")
		Translates("a default in a range", where(`[?b :block/properties ?p] [(get ?p :rating 5) ?r] [(>= ?r 5)] [(< ?r 7)]`),
			"(and (or (property rating >= 5) (not (property rating))) (or (property rating < 7) (not (property rating))))")
		Translates("a default in a not", where(`[?b :block/properties ?p] [(get ?p :rating 0) ?r] (not [(< ?r 7)])`),
			"(not (or (property rating < 7) (not (property rating))))")
		Translates("a default of a page property",
			`{:query [:find (pull ?p [*]) :where [?p :block/name] [?p :block/properties ?pp] [(get ?pp :pages 0) ?n] [(< ?n 300)]]}`,
			"(or (page-property pages < 300) (not (page-property pages)))")

		Unsupported("a date with a default",
			`{:query [:find (pull ?b [*]) :in $ ?today :where [?b :block/properties ?p] [(get ?p :due 0) ?d] [(< ?d ?today)]] :inputs [:today]}`,
			"comparing a date property that has a default value")
		Unsupported("a default that is not a number", where(`[?b :block/properties ?p] [(get ?p :rating "none") ?r] [(< ?r 7)]`),
			"expected a number as the default value")
		Unsupported("text", withRating(`[(< ?r "m")]`), "expected a number or a date")
	})

	Describe("Blocks related to other blocks", func() {
		below := `:rules [[(below ?b ?a) [?b :block/parent ?a]] [(below ?b ?a) [?b :block/parent ?c] (below ?c ?a)]]`

		Translates("blocks with only conditions on their page",
			where(`[?b :block/page ?p] [?p :block/original-name ?n] [(clojure.string/includes? ?n "Meeting")]`),
			`(page (title includes "Meeting"))`)
		Translates("children of tasks", where(`[?b :block/parent ?p] [?p :block/marker "TODO"]`), "(parent (task TODO))")
		Translates("children of blocks that refer to a page",
			where(`[?b :block/parent ?p] [?p :block/refs ?t] [?t :block/name "meeting"]`), "(parent (refs [[meeting]]))")
		Translates("grandchildren", where(`[?b :block/parent ?p] [?p :block/parent ?g] [?g :block/marker "TODO"]`),
			"(parent (parent (task TODO)))")
		Translates("children of blocks on a page with a title",
			where(`[?b :block/parent ?p] [?p :block/page ?pg] [?pg :block/name ?n] [(clojure.string/starts-with? ?n "meet")]`),
			`(parent (page (title starts-with "meet")))`)
		Translates("blocks with children that match", where(`[?c :block/parent ?b] [?c :block/marker "TODO"]`), "(has-child (task TODO))")
		Translates("blocks that are not children of blocks that match",
			where(`[?b :block/marker "TODO"] (not [?b :block/parent ?p] [?p :block/marker "DONE"])`),
			"(and (task TODO) (not (parent (task DONE))))")
		Translates("blocks below blocks that match with a rule",
			`{:query [:find (pull ?b [*]) :in $ % :where [?b :block/marker "TODO"] (below ?b ?a) [?a :block/refs ?t] [?t :block/name "meeting"]] `+below+`}`,
			"(and (ancestor (refs [[meeting]])) (task TODO))")
		Translates("blocks with blocks below them that match with a rule",
			`{:query [:find (pull ?b [*]) :in $ % :where (below ?c ?b) [?c :block/marker "TODO"]] `+below+`}`,
			"(has-descendant (task TODO))")
		Translates("pages with blocks below blocks that match",
			"{:query [:find (pull ?p [*]) :where [?b :block/page ?p] [?b :block/marker \"TODO\"] [?b :block/parent ?q] [?q :block/refs ?t] [?t :block/name \"meeting\"]]}",
			"(has-block (and (parent (refs [[meeting]])) (task TODO)))")

		Translates("a page as the parent", where(`[?b :block/parent ?p] [?p :block/name "x"]`), "(parent [[x]])")
		Translates("a page as the parent with a lookup", where(`[?b :block/parent [:block/name "x"]]`), "(parent [[x]])")
		Translates("the page of the block as the parent",
			`{:query [:find (pull ?b [*]) :in $ ?cp :where [?b :block/page ?p] [?b :block/parent ?p] [?p :block/name ?cp]] :inputs ["Notes"]}`,
			"(and (parent [[Notes]]) (page [[Notes]]))")

		Translates("the page of the block without a title as the parent", where(`[?b :block/page ?p] [?b :block/parent ?p]`), "(top-level)")
		Translates("the page of the block with conditions as the parent",
			where(`[?b :block/page ?p] [?p :block/journal? true] [?b :block/parent ?p]`), "(and (journal) (top-level))")
		Translates("the page of the blocks on the pages a query finds as the parent",
			`{:query [:find (pull ?p [*]) :where [?b :block/page ?p] [?b :block/parent ?p] [?b :block/marker "TODO"]]}`,
			"(has-block (and (top-level) (task TODO)))")

		Unsupported("a page as the parent that is not the page of the block",
			where(`[?b :block/parent ?p] [?p :block/journal? true]`), "or a page with a title")
	})

	Describe("Timestamps", func() {
		Translates("pages changed recently",
			`{:query [:find (pull ?p [*]) :in $ ?since :where [?p :block/name] [?p :block/updated-at ?u] [(> ?u ?since)]] :inputs [:-7d-ms]}`,
			"(page-updated-at > -7d-ms)")
		Translates("blocks created before now",
			`{:query [:find (pull ?b [*]) :in $ ?now :where [?b :block/created-at ?c] [(< ?c ?now)]] :inputs [:right-now-ms]}`,
			"(created-at < right-now-ms)")
		Translates("blocks with a time", where(`[?b :block/updated-at]`), "(updated-at)")
		Translates("blocks on pages changed recently",
			`{:query [:find (pull ?b [*]) :in $ ?since :where [?b :block/marker "TODO"] [?b :block/page ?p] [?p :block/updated-at ?u] [(>= ?u ?since)]] :inputs [:today-start]}`,
			"(and (task TODO) (page-updated-at >= today-start))")
		Translates("a range of times",
			`{:query [:find (pull ?b [*]) :in $ ?a ?z :where [?b :block/updated-at ?u] [(>= ?u ?a)] [(< ?u ?z)]] :inputs [:-7d-ms :right-now-ms]}`,
			"(and (updated-at >= -7d-ms) (updated-at < right-now-ms))")
		Translates("sorted by the time of the last change",
			`{:query (page-updated-at >= -7d-ms) :result-transform (fn [r] (reverse (sort-by :block/updated-at r)))}`,
			"(and (page-updated-at >= -7d-ms) (sort-by updated-at desc))")
	})

	Describe("Inputs", func() {
		Translates("older date inputs",
			`{:query [:find (pull ?b [*]) :in $ ?start ?end :where [?b :block/scheduled ?d] [(>= ?d ?start)] [(<= ?d ?end)]] :inputs [:14d :7d-after]}`,
			"(scheduled -14d +7d)")
		Translates("relative date inputs",
			`{:query [:find (pull ?b [*]) :in $ ?start :where [?b :block/deadline ?d] [(> ?d ?start)]] :inputs [:-2w]}`,
			"(deadline > -2w)")
		Translates("a page name input",
			`{:query [:find (pull ?b [*]) :in $ ?name :where [?b :block/path-refs ?p] [?p :block/name ?name]] :inputs ["[[Project]]"]}`,
			"[[Project]]")
		Translates("an input as the value of a clause",
			`{:query [:find (pull ?b [*]) :in $ ?day :where [?b :block/scheduled ?day]] :inputs [:tomorrow]}`,
			"(scheduled tomorrow)")
	})

	Describe("Rules", func() {
		Translates("task", where(`(task ?b #{"TODO" "DOING"})`), "(task TODO DOING)")
		Translates("priority", where(`(priority ?b #{"A"})`), "(priority A)")
		Translates("page-ref", where(`(page-ref ?b "project")`), "[[project]]")
		Translates("page", where(`(page ?b "project")`), "(page [[project]])")
		Translates("property", where(`(property ?b :type "book")`), "(property type book)")
		Translates("between", where(`(between ?b 20240101 20240131)`), "(between 2024-01-01 2024-01-31)")
		Translates("page-property", "{:query [:find (pull ?p [*]) :where (page-property ?p :type \"book\")]}", "(page-property type book)")
		Translates("page-tags", "{:query [:find (pull ?p [*]) :where (page-tags ?p #{\"book\" \"film\"})]}", "(page-tags [[book]] [[film]])")
	})

	Describe("Custom rules", func() {
		below := `[[(below ?b ?a) [?b :block/parent ?a]]
			[(below ?b ?a) [?b :block/parent ?c] (below ?c ?a)]]`
		withRules := func(rules string, clauses string) string {
			return "{:query [:find (pull ?b [*]) :in $ % :where " + clauses + "] :rules " + rules + "}"
		}

		Translates("blocks below a block", withRules(below, `[?b :block/marker "TODO"] (below ?b [:block/uuid #uuid "abc"])`),
			"(and (task TODO) (ancestor ((abc))))")

		inNamespace := `[[(in-ns ?p ?ns) [?p :block/namespace ?ns]]
			[(in-ns ?p ?ns) [?p :block/namespace ?mid] (in-ns ?mid ?ns)]]`
		withPageRules := func(rules string, clauses string) string {
			return "{:query [:find (pull ?p [*]) :in $ % :where " + clauses + "] :rules " + rules + "}"
		}

		Translates("pages in a namespace at any depth", withPageRules(inNamespace, `(in-ns ?p ?ns) [?ns :block/name "projects"]`),
			"(under-namespace [[projects]])")
		Translates("pages in a namespace with a lookup", withPageRules(inNamespace, `(in-ns ?p [:block/name "projects"])`),
			"(under-namespace [[projects]])")
		Translates("pages in a namespace that steps down",
			withPageRules(`[[(in-ns ?p ?ns) [?p :block/namespace ?ns]] [(in-ns ?p ?ns) [?mid :block/namespace ?ns] (in-ns ?p ?mid)]]`,
				`(in-ns ?p ?ns) [?ns :block/name "projects"]`),
			"(under-namespace [[projects]])")
		Translates("blocks on pages in a namespace",
			withRules(inNamespace, `[?b :block/marker "TODO"] [?b :block/page ?p] (in-ns ?p ?ns) [?ns :block/name "projects"]`),
			"(and (task TODO) (under-namespace [[projects]]))")
		Translates("the namespaces of a page",
			"{:query [:find (pull ?ns [*]) :in $ % :where (in-ns ?p ?ns) [?p :block/name \"projects/a/b\"]] :rules "+inNamespace+"}",
			`(or (title = "projects/a") (title = "projects"))`)
		Unsupported("a namespace without a title", withPageRules(inNamespace, `(in-ns ?p ?ns)`), "expected a namespace with a title")
		Translates("the recursive definition first",
			withRules(`[[(below ?b ?a) [?b :block/parent ?c] (below ?c ?a)] [(below ?b ?a) [?b :block/parent ?a]]]`,
				`(below ?b [:block/uuid "abc"])`),
			"(ancestor ((abc)))")
		Translates("the block above first, stepping down",
			withRules(`[[(above ?a ?b) [?b :block/parent ?a]] [(above ?a ?b) [?c :block/parent ?a] (above ?c ?b)]]`,
				`(above [:block/uuid "abc"] ?b)`),
			"(ancestor ((abc)))")
		Translates("a rule with the name of a built-in rule",
			withRules(`[[(task ?b ?a) [?b :block/parent ?a]] [(task ?b ?a) [?b :block/parent ?c] (task ?c ?a)]]`,
				`(task ?b [:block/uuid "abc"])`),
			"(ancestor ((abc)))")

		Translates("a rule in place of its use", withRules(`[[(todo ?b) [?b :block/marker "TODO"]]]`, `(todo ?b)`), "(task TODO)")
		Translates("only the direct step",
			withRules(`[[(below ?b ?a) [?b :block/parent ?a]]]`, `(below ?b [:block/uuid "abc"])`), "(parent ((abc)))")
		Translates("a rule with a value as the argument",
			withRules(`[[(tagged ?b ?tag) [?b :block/refs ?p] [?p :block/name ?tag]]]`, `[?b :block/marker "TODO"] (tagged ?b "project")`),
			"(and (task TODO) (refs [[project]]))")
		Translates("each use with variables of its own",
			withRules(`[[(tagged ?b ?tag) [?b :block/refs ?p] [?p :block/name ?tag]]]`, `(tagged ?b "a") (tagged ?b "b")`),
			"(and (refs [[a]]) (refs [[b]]))")
		Translates("a rule with more than one definition",
			withRules(`[[(active ?b) [?b :block/marker "TODO"]] [(active ?b) [?b :block/marker "DOING"]]]`, `(active ?b)`),
			"(or (task TODO) (task DOING))")
		Translates("a definition with more than one clause",
			withRules(`[[(urgent ?b) [?b :block/marker "TODO"] [?b :block/priority "A"]] [(urgent ?b) [?b :block/marker "NOW"]]]`, `(urgent ?b)`),
			"(or (and (task TODO) (priority A)) (task NOW))")
		Translates("a rule that uses another rule",
			withRules(`[[(todo ?b) [?b :block/marker "TODO"]] [(open ?b) (todo ?b) (not [?b :block/priority "C"])]]`, `(open ?b)`),
			"(and (task TODO) (not (priority C)))")
		Translates("a rule in a not",
			withRules(`[[(todo ?b) [?b :block/marker "TODO"]]]`, `[?b :block/priority "A"] (not (todo ?b))`),
			"(and (priority A) (not (task TODO)))")
		Translates("a rule with the variables it requires in a vector",
			withRules(`[[(tagged [?b] ?tag) [?b :block/refs ?p] [?p :block/name ?tag]]]`, `(tagged ?b "x")`),
			"(refs [[x]])")
		Translates("a rule about a page that a block refers to",
			withRules(`[[(project? ?p) [?p :block/tags ?t] [?t :block/name "project"]]]`, `[?b :block/refs ?p] (project? ?p)`),
			"(refs (page-tags [[project]]))")
		Translates("a rule that uses the rule for the blocks below a block",
			withRules(below[:len(below)-1]+` [(open-below ?b ?a) [?b :block/marker "TODO"] (below ?b ?a)]]`, `(open-below ?b [:block/uuid "abc"])`),
			"(and (task TODO) (ancestor ((abc))))")

		Unsupported("a step that does not go up",
			withRules(`[[(below ?b ?a) [?b :block/parent ?a]] [(below ?b ?a) [?c :block/parent ?b] (below ?c ?a)]]`,
				`(below ?b [:block/uuid "abc"])`), "rules that use themselves")
		Unsupported("rules that use each other",
			withRules(`[[(a ?b) (b ?b)] [(b ?b) (a ?b)]]`, `(a ?b)`), "rules that use themselves")
		Unsupported("a use with the wrong number of arguments",
			withRules(`[[(todo ?b) [?b :block/marker "TODO"]]]`, `(todo ?b "x")`), "takes 1 arguments")
		Unsupported("definitions with different numbers of arguments",
			withRules(`[[(todo ?b) [?b :block/marker "TODO"]] [(todo ?b ?x) [?b :block/marker "DOING"]]]`, `(todo ?b)`), "different numbers of arguments")
		Unsupported("a block above that is not known", withRules(below, `[?b :block/marker "TODO"] (below ?b ?a)`), "expected a block above")
	})

	Describe("Combinations", func() {
		Translates("or without join", where(`(or [?b :block/marker "TODO"] [?b :block/priority "A"])`), "(or (task TODO) (priority A))")
		Translates("not-join", where(`[?b :block/marker "TODO"] (not-join [?b] [?b :block/path-refs [:block/name "later"]])`),
			"(and (task TODO) (not [[later]]))")
		Translates("page with a title in a branch of an or",
			where(`[?p :block/name "project"] (or [?b :block/refs ?p] [?b :block/page ?p]) [?b :block/marker "TODO"]`),
			"(and (task TODO) (or (refs [[project]]) (page [[project]])))")
		Translates("title of the page in a branch of an or",
			where(`[?b :block/marker "TODO"] [?b :block/page ?p] (or [?p :block/name "x"] [?p :block/journal? true])`),
			"(and (task TODO) (or (page [[x]]) (journal)))")
		Translates("page in a not", where(`[?b :block/marker "TODO"] (not [?b :block/page ?p] [?p :block/journal? true])`),
			"(and (task TODO) (not (journal)))")
		Translates("pages with titles in each branch",
			where(`(or (and [?b :block/page ?p] [?p :block/name "a"]) (and [?b :block/page ?q] [?q :block/name "b"]))`),
			"(or (page [[a]]) (page [[b]]))")
	})

	Describe("Simple queries in :query", func() {
		Translates("as a list", `{:query (and (task TODO) [[project]])}`, "(and (task TODO) [[project]])")
		Translates("as a string", `{:query "(task NOW)"}`, "(task NOW)")
		Translates("with a sort", `{:query (task TODO) :result-transform (fn [r] (sort-by :block/scheduled r))}`,
			"(and (task TODO) (sort-by scheduled asc))")

		It("reports errors at their place in the advanced query", func() {
			text := `{:query (and (nope))}`
			_, err := query.ParseAdvanced(text)

			var parseErr *query.ParseError
			Expect(errors.As(err, &parseErr)).To(BeTrue())
			Expect(text[parseErr.Offset:]).To(HavePrefix("nope"))
		})
	})

	Describe("Result transforms", func() {
		withTransform := func(transform string) string {
			return `{:query (task TODO) :result-transform ` + transform + `}`
		}

		Translates("reverse", withTransform(`(fn [r] (reverse (sort-by :block/priority r)))`), "(and (task TODO) (sort-by priority desc))")
		Translates("keyword function", withTransform(`(fn [r] (sort-by (fn [b] (:block/deadline b)) r))`), "(and (task TODO) (sort-by deadline asc))")
		Translates("anonymous function", withTransform(`(fn [r] (sort-by #(get % :block/priority) r))`), "(and (task TODO) (sort-by priority asc))")
		Translates("journal day", withTransform(`(fn [r] (sort-by (fn [b] (get-in b [:block/page :block/journal-day])) r))`), "(and (task TODO) (sort-by journal-day asc))")
		Translates("page title", withTransform(`(fn [r] (sort-by (fn [b] (get-in b [:block/page :block/original-name])) r))`), "(and (task TODO) (sort-by title asc))")
		Translates("property", withTransform(`(fn [r] (sort-by (fn [b] (get-in b [:block/properties :rating])) r))`), "(and (task TODO) (sort-by rating asc))")
		Translates("sorts in turn", withTransform(`(fn [r] (sort-by :block/priority (sort-by :block/deadline r)))`),
			"(and (task TODO) (sort-by priority asc) (sort-by deadline asc))")
		Translates("filter", withTransform(`(fn [r] (filter (fn [b] (= "A" (:block/priority b))) r))`), "(and (task TODO) (priority A))")
		Translates("filter with a keyword", withTransform(`(fn [r] (filter :block/scheduled r))`), "(and (task TODO) (scheduled))")
		Translates("remove", withTransform(`(fn [r] (remove #(contains? #{"A" "B"} (:block/priority %)) r))`),
			"(and (task TODO) (not (priority A B)))")
		Translates("not= is true without the value", withTransform(`(fn [r] (filter (fn [b] (not= "A" (:block/priority b))) r))`),
			"(and (task TODO) (not (priority A)))")
		Translates("and, nil? and some?",
			withTransform(`(fn [r] (filter (fn [b] (and (some? (:block/scheduled b)) (nil? (:block/deadline b)))) r))`),
			"(and (task TODO) (scheduled) (not (deadline)))")
		Translates("or", withTransform(`(fn [r] (filter (fn [b] (or (:block/scheduled b) (:block/deadline b))) r))`),
			"(and (task TODO) (or (scheduled) (deadline)))")
		Translates("a property", withTransform(`(fn [r] (filter (fn [b] (= "book" (:type (:block/properties b)))) r))`),
			"(and (task TODO) (property type book))")
		Translates("a property as a number", withTransform(`(fn [r] (filter (fn [b] (> (get-in b [:block/properties :rating]) 7)) r))`),
			"(and (task TODO) (property rating > 7))")
		Translates("the title of the page",
			withTransform(`(fn [r] (filter (fn [b] (clojure.string/includes? (get-in b [:block/page :block/original-name]) "Meet")) r))`),
			`(and (task TODO) (page (title includes "Meet")))`)
		Translates("true or false", withTransform(`(fn [r] (filter (fn [b] (:block/journal? (:block/page b))) r))`), "(and (task TODO) (journal))")
		Translates("->> with filter and sort-by",
			withTransform(`(fn [r] (->> r (filter (fn [b] (contains? #{"A" "B"} (:block/priority b)))) (sort-by :block/priority) reverse))`),
			"(and (task TODO) (priority A B) (sort-by priority desc))")
		Translates("filter on pages", `{:query (page-property type) :result-transform (fn [r] (remove (fn [p] (:block/journal? p)) r))}`,
			"(and (page-property type) (not (journal-page)))")
		Translates("filter on the pages of a Datalog query",
			`{:query [:find (pull ?p [*]) :where [?p :block/name]] :result-transform (fn [r] (filter (fn [p] (= "book" (get-in p [:block/properties :type]))) r))}`,
			"(page-property type book)")

		Translates("take", withTransform(`(fn [r] (take 5 r))`), "(and (task TODO) (limit 5))")
		Translates("take after sort-by and reverse", withTransform(`(fn [r] (take 3 (reverse (sort-by :block/priority r))))`),
			"(and (task TODO) (sort-by priority desc) (limit 3))")
		Translates("take in ->>", withTransform(`(fn [r] (->> r (filter :block/scheduled) (sort-by :block/scheduled) (take 2)))`),
			"(and (task TODO) (scheduled) (sort-by scheduled asc) (limit 2))")
		Translates("take of take", withTransform(`(fn [r] (take 5 (take 2 r)))`), "(and (task TODO) (limit 2))")
		Translates("take on a query with a sample", `{:query (and (task TODO) (sample 10)) :result-transform (fn [r] (take 2 r))}`,
			"(and (task TODO) (sample 10) (limit 2))")

		Unsupported("other transforms", withTransform(`(fn [r] (map :block/content r))`), "only a :result-transform that sorts")
		Unsupported("take before sort-by", withTransform(`(fn [r] (sort-by :block/priority (take 5 r)))`), "take is only supported as the last step")
		Unsupported("take before filter", withTransform(`(fn [r] (filter :block/scheduled (take 5 r)))`), "take is only supported as the last step")
		Unsupported("take before reverse", withTransform(`(fn [r] (reverse (take 5 (sort-by :block/priority r))))`),
			"take is only supported as the last step")
		Unsupported("take with a count that is not a number", withTransform(`(fn [r] (take 0 r))`), "expected a number above zero")
		Unsupported("filter on a query with a sample", `{:query (and (task TODO) (sample 10)) :result-transform (fn [r] (filter :block/scheduled r))}`,
			"not supported for a query with sample or limit")
		Unsupported("sort-by on a query with a limit", `{:query (and (task TODO) (limit 10)) :result-transform (fn [r] (sort-by :block/priority r))}`,
			"sort-by is not supported for a query with limit")
		Unsupported("other sort keys", withTransform(`(fn [r] (sort-by :block/content r))`), "only sorting by")
		Unsupported("reverse without a sort", withTransform(`(fn [r] (reverse (filter :block/scheduled r)))`), "reverse is only supported")
		Unsupported("two values of the result",
			withTransform(`(fn [r] (filter (fn [b] (= (:block/scheduled b) (:block/deadline b))) r))`), "two values of the result")
		Unsupported("a default in filter", withTransform(`(fn [r] (filter (fn [b] (= "A" (get b :block/priority "A"))) r))`),
			"default values are not supported")
		Unsupported("a condition on something else", withTransform(`(fn [r] (filter (fn [b] (= 1 1)) r))`), "expected a condition on a value")
	})

	Describe("EDN", func() {
		Translates("comments, commas and discarded values",
			"{:query [:find (pull ?b [*]) ; blocks\n :where [?b :block/marker \"TODO\"], #_[?b :block/priority \"A\"]]}",
			"(task TODO)")

		It("reports EDN errors with their place", func() {
			_, err := query.ParseAdvanced(`{:query [:find ?b :where [?b :block/marker "TODO"]}`)

			var parseErr *query.ParseError
			Expect(errors.As(err, &parseErr)).To(BeTrue())
			Expect(parseErr.Message).To(ContainSubstring("unexpected `}`"))
		})

		It("requires a map with a query", func() {
			_, err := query.ParseAdvanced(`[:find ?b]`)
			Expect(err).To(MatchError(ContainSubstring("expected a map")))

			_, err = query.ParseAdvanced(`{:title "x"}`)
			Expect(err).To(MatchError(ContainSubstring("has no :query")))
		})
	})

	Describe("Unsupported", func() {
		Unsupported("current page", `{:query [:find (pull ?b [*]) :in $ ?p :where [?b :block/page ?p]] :inputs [:current-page]}`, "the input :current-page")
		Unsupported("time inputs", `{:query [:find (pull ?b [*]) :in $ ?t :where [?b :block/scheduled ?d] [(< ?d ?t)]] :inputs [:right-now-ms]}`, "time of day")
		Unsupported("dates for times", `{:query [:find (pull ?b [*]) :in $ ?t :where [?b :block/updated-at ?u] [(> ?u ?t)]] :inputs [:today]}`,
			"only compared with inputs that have a time of day")
		Translates("times of referenced pages",
			`{:query [:find (pull ?b [*]) :in $ ?since :where [?b :block/refs ?p] [?p :block/updated-at ?u] [(> ?u ?since)]] :inputs [:-7d-ms]}`,
			"(refs (page-updated-at > -7d-ms))")
		Unsupported("two variables in find", `{:query [:find ?b ?p :where [?b :block/page ?p]]}`, "one variable or pull")
		Unsupported("comparing variables", where(`[?b :block/scheduled ?s] [?b :block/deadline ?d] [(< ?s ?d)]`), "two variables")
		Unsupported("parents that are not inputs", where(`[?b :block/parent ?parent]`), "expected a block")
		Unsupported("other attributes", where(`[?b :block/left ?left]`), "attribute :block/left")
		Unsupported("other blocks", where(`[?b :block/page ?p] [?c :block/page ?p] [?c :block/marker "TODO"]`), "other blocks")
		Unsupported("unknown rules", where(`(my-rule ?b)`), "rule my-rule")
		Unsupported("unbound variables", where(`[?b :block/marker "TODO"] [(= ?x "y")]`), "?x does not get a value")
		Unsupported("missing inputs", `{:query [:find ?b :in $ ?day :where [?b :block/scheduled ?day]]}`, "1 variables in :in but 0 :inputs")
		Unsupported("empty lists", `{:query [:find ?b :where ()]}`, "expected a clause")
		Unsupported("empty sets", where(`[?b :block/marker ?m] [(contains? #{} ?m)]`), "expected a set of values")
		Unsupported("other conditions on titles", "{:query [:find ?p :where [?p :block/name ?n] [(clojure.string/blank? ?n)]]}", "for the title of a page")
		Unsupported("a referenced page that is also about the block",
			where(`[?b :block/page ?q] [?b :block/refs ?p] [?p :block/namespace ?q]`), "expected a page")
		Unsupported("defaults that are not values", where(`[?b :block/properties ?p] [(get ?p :type [1]) ?t]`), "expected a value as the default")
		Unsupported("missing for other attributes", where(`[?b :block/marker "TODO"] [(missing? $ ?b :block/properties)]`), "missing? is only supported")
		Unsupported("regex that Go can not read", where(`[?b :block/content ?c] [(re-find #"(?<=a)b" ?c)]`), "the regular expression is not supported")
		Unsupported("regex from a value that is not a text", where(`[(re-pattern 1) ?re] [?b :block/content ?c] [(re-find ?re ?c)]`),
			"expected the text of a regular expression")
		Unsupported("regex that is not a regex", where(`[?b :block/content ?c] [(re-find "bank" ?c)]`), "expected a regular expression")
	})
})
