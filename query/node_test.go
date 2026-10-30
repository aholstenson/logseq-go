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
			Expect(err).To(MatchError(ContainSubstring("needs the UUID of the block, given with WithCurrentBlock")))

			_, err = query.ParseAdvanced(siblings)
			Expect(err).To(MatchError(ContainSubstring("given with WithParentBlock")))
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
