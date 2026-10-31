package query_test

import (
	"errors"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/query"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Queries in nodes", func() {
	It("reads a simple query", func() {
		advanced, err := query.ParseNode(content.NewQuery("(and (task TODO) [[project]])"))
		Expect(err).ToNot(HaveOccurred())
		Expect(advanced.Title).To(BeEmpty())
		Expect(advanced.Query.String()).To(Equal("(and (task TODO) [[project]])"))
		Expect(advanced.GroupByPage).To(BeTrue())
		Expect(advanced.RemoveBlockChildren).To(BeTrue())
		Expect(advanced.BreadcrumbShow).To(BeTrue())
	})

	Describe("Table view", func() {
		// inBlock returns a simple query in a block with the value of the
		// property `query-table::`, or without the property if it is empty.
		inBlock := func(text string, queryTable string) *content.Query {
			q := content.NewQuery(text)
			block := content.NewBlock(content.NewParagraph(q))
			if queryTable != "" {
				block.Properties().Set("query-table", content.NewText(queryTable))
			}
			return q
		}

		It("reads `table` at the end of a simple query", func() {
			advanced, err := query.ParseNode(content.NewQuery("(task TODO) table"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(task TODO)"))
			Expect(advanced.TableView).To(BeTrue())
			Expect(advanced.GroupByPage).To(BeFalse())
		})

		It("reads the property query-table of the block", func() {
			advanced, err := query.ParseNode(inBlock("(task TODO)", "true"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.TableView).To(BeTrue())
			Expect(advanced.GroupByPage).To(BeFalse())

			advanced, err = query.ParseNode(inBlock("(task TODO)", "false"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.TableView).To(BeFalse())
			Expect(advanced.GroupByPage).To(BeTrue())

			advanced, err = query.ParseNode(inBlock("(task TODO)", ""))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.TableView).To(BeFalse())
		})

		It("reads the property for an advanced query", func() {
			command := content.NewQueryCommand(`{:query (task TODO) :group-by-page? true}`)
			block := content.NewBlock(command)
			block.Properties().Set("query-table", content.NewText("true"))

			advanced, err := query.ParseNode(command)
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.TableView).To(BeTrue())
			Expect(advanced.GroupByPage).To(BeFalse())
		})
	})

	It("reads (by-page false) in a simple query", func() {
		advanced, err := query.ParseNode(content.NewQuery("(and (task TODO) (by-page false))"))
		Expect(err).ToNot(HaveOccurred())
		Expect(advanced.GroupByPage).To(BeFalse())
	})

	It("reads an advanced query", func() {
		advanced, err := query.ParseNode(content.NewQueryCommand("{:title \"Tasks\"\n :query [:find (pull ?b [*]) :where [?b :block/marker \"TODO\"]]}\n"))
		Expect(err).ToNot(HaveOccurred())
		Expect(advanced.Title).To(Equal("Tasks"))
		Expect(advanced.Query.String()).To(Equal("(task TODO)"))
	})

	It("returns the errors of a simple query", func() {
		_, err := query.ParseNode(content.NewQuery("(nope)"))

		var parseErr *query.ParseError
		Expect(errors.As(err, &parseErr)).To(BeTrue())
	})

	It("returns an error for other nodes", func() {
		_, err := query.ParseNode(content.NewText("text"))
		Expect(err).To(MatchError(ContainSubstring("*content.Text is not a query")))
	})

	Describe("Inputs for pages", func() {
		currentPage := `{:query [:find (pull ?b [*]) :in $ ?page
			:where [?b :block/path-refs ?p] [?p :block/name ?page]]
			:inputs [:current-page]}`
		queryPage := `{:query [:find (pull ?b [*]) :in $ ?page
			:where [?b :block/page ?p] [?p :block/name ?page]]
			:inputs [:query-page]}`

		It("reads :current-page as the given page", func() {
			advanced, err := query.ParseAdvanced(currentPage, query.WithCurrentPage("Project"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("[[Project]]"))
		})

		It("reads :query-page as the given page", func() {
			advanced, err := query.ParseNode(content.NewQueryCommand(queryPage), query.WithQueryPage("Notes"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(page [[Notes]])"))
		})

		It("names the option when the page is not given", func() {
			_, err := query.ParseAdvanced(currentPage, query.WithQueryPage("Notes"))
			Expect(errors.Is(err, query.ErrUnsupported)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("needs the title of the page, given with WithCurrentPage")))

			_, err = query.ParseAdvanced(queryPage)
			Expect(err).To(MatchError(ContainSubstring("given with WithQueryPage")))
		})

		It("reads the title of the current page as a value", func() {
			advanced, err := query.ParseAdvanced(`{:query [:find (pull ?p [*]) :in $ ?target
				:where [?b :block/page ?p] [?p :page/properties ?props]
				[(get ?props :organization) ?t]
				[?me-low :block/name ?target] [?me-low :block/original-name ?me]
				[(get ?t ?me)]]
				:inputs [:current-page]}`, query.WithCurrentPage("Acme"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(and (page-property organization Acme) (has-block))"))
		})

		It("reads the title before the clause that names the page", func() {
			advanced, err := query.ParseAdvanced(`{:query [:find (pull ?b [*]) :in $ ?target
				:where [?p :block/original-name ?title] [?p :block/name ?target]
				[?b :block/properties ?props] [(get ?props :owner) ?o] [(contains? ?o ?title)]]
				:inputs [:current-page]}`, query.WithCurrentPage("Ada"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(property owner Ada)"))
		})

	})

	Describe("Inputs for blocks", func() {
		children := `{:query [:find (pull ?b [*]) :in $ ?c :where [?b :block/parent ?c]] :inputs [:current-block]}`
		siblings := `{:query [:find (pull ?b [*]) :in $ ?p :where [?b :block/marker "TODO"] [?b :block/parent ?p]] :inputs [:parent-block]}`

		It("reads :current-block as the parent of the blocks", func() {
			advanced, err := query.ParseAdvanced(children, query.WithCurrentBlock("6523a0f8-1111-4c3b-9b1e-000000000001"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(parent ((6523a0f8-1111-4c3b-9b1e-000000000001)))"))
		})

		It("reads :parent-block as the parent of the blocks", func() {
			advanced, err := query.ParseAdvanced(siblings, query.WithParentBlock("abc"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(and (task TODO) (parent ((abc))))"))
		})

		It("names the option when the block is not given", func() {
			_, err := query.ParseAdvanced(children, query.WithCurrentPage("Project"))
			Expect(errors.Is(err, query.ErrUnsupported)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("needs the block, given with WithCurrentBlock or WithCurrentBlockAt")))

			_, err = query.ParseAdvanced(siblings)
			Expect(err).To(MatchError(ContainSubstring("given with WithParentBlock or WithParentBlockAt")))
		})

		It("reads :parent-block as the block above the blocks in a rule", func() {
			advanced, err := query.ParseAdvanced(`{:query [:find (pull ?b [*]) :in $ ?parent %
				:where [?p :block/name "action"] [?b :block/refs ?p] (check-parent ?b ?parent)]
				:rules [[(check-parent ?b ?parent) [?b :block/parent ?parent]]
				        [(check-parent ?b ?parent) [?b :block/parent ?c] (check-parent ?c ?parent)]]
				:inputs [:parent-block]}`, query.WithParentBlock("abc"))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(and (refs [[action]]) (ancestor ((abc))))"))
		})

		It("reads blocks given by their location on a page", func() {
			advanced, err := query.ParseAdvanced(children, query.WithCurrentBlockAt("Outline", 1, 1))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(parent [[Outline]] 1 1)"))

			advanced, err = query.ParseAdvanced(`{:query [:find (pull ?b [*]) :in $ ?parent %
				:where [?p :block/name "action"] [?b :block/refs ?p] (check-parent ?b ?parent)]
				:rules [[(check-parent ?b ?parent) [?b :block/parent ?parent]]
				        [(check-parent ?b ?parent) [?b :block/parent ?c] (check-parent ?c ?parent)]]
				:inputs [:parent-block]}`, query.WithParentBlockAt("2023-08-21", 8))
			Expect(err).ToNot(HaveOccurred())
			Expect(advanced.Query.String()).To(Equal("(and (refs [[action]]) (ancestor [[2023-08-21]] 8))"))
		})

		Describe("found by ParseNode", func() {
			below := `{:query [:find (pull ?b [*]) :in $ ?p %
				:where (below ?b ?p)]
				:rules [[(below ?b ?a) [?b :block/parent ?a]]
				        [(below ?b ?a) [?b :block/parent ?c] (below ?c ?a)]]
				:inputs [:parent-block]}`

			// onPage puts the query in a block below the second block of a
			// page, and returns the query node.
			onPage := func(text string, parentProps ...string) content.Node {
				node := content.NewQueryCommand(text)
				parent := content.NewBlock(
					content.NewParagraph(content.NewText("section")),
					content.NewBlock(content.NewParagraph(content.NewText("first"))),
					content.NewBlock(node),
				)
				if len(parentProps) > 0 {
					parent.Properties().Set("id", content.NewText(parentProps[0]))
				}
				content.NewBlock(content.NewBlock(content.NewParagraph(content.NewText("intro"))), parent)
				return node
			}

			It("names blocks without a UUID by their location on the query page", func() {
				advanced, err := query.ParseNode(onPage(below), query.WithQueryPage("Outline"))
				Expect(err).ToNot(HaveOccurred())
				Expect(advanced.Query.String()).To(Equal("(ancestor [[Outline]] 1)"))

				advanced, err = query.ParseNode(onPage(children), query.WithQueryPage("Outline"))
				Expect(err).ToNot(HaveOccurred())
				Expect(advanced.Query.String()).To(Equal("(parent [[Outline]] 1 1)"))
			})

			It("names blocks with a UUID by it without the query page", func() {
				advanced, err := query.ParseNode(onPage(below, "abc"))
				Expect(err).ToNot(HaveOccurred())
				Expect(advanced.Query.String()).To(Equal("(ancestor ((abc)))"))
			})

			It("uses the blocks given as options first", func() {
				advanced, err := query.ParseNode(onPage(below), query.WithQueryPage("Outline"), query.WithParentBlock("given"))
				Expect(err).ToNot(HaveOccurred())
				Expect(advanced.Query.String()).To(Equal("(ancestor ((given)))"))

				advanced, err = query.ParseNode(onPage(below, "abc"), query.WithParentBlockAt("Other", 3))
				Expect(err).ToNot(HaveOccurred())
				Expect(advanced.Query.String()).To(Equal("(ancestor [[Other]] 3)"))
			})

			It("needs the query page for blocks without a UUID", func() {
				_, err := query.ParseNode(onPage(below), query.WithCurrentPage("Outline"))
				Expect(errors.Is(err, query.ErrUnsupported)).To(BeTrue())
				Expect(err).To(MatchError(ContainSubstring("given with WithQueryPage")))
			})

			It("has no parent block for a query at the top of a page", func() {
				node := content.NewQueryCommand(below)
				content.NewBlock(content.NewBlock(node))

				_, err := query.ParseNode(node, query.WithQueryPage("Outline"))
				Expect(err).To(MatchError(ContainSubstring("the input :parent-block needs the block")))
			})
		})

		It("does not read the block in other clauses", func() {
			_, err := query.ParseAdvanced(`{:query [:find (pull ?b [*]) :in $ ?c
				:where [?c :block/page ?p] [?b :block/page ?p]] :inputs [:current-block]}`,
				query.WithCurrentBlock("abc"))
			Expect(errors.Is(err, query.ErrUnsupported)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("only supported as the parent")))
		})
	})
})
