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

		It("does not read :current-block", func() {
			_, err := query.ParseAdvanced(`{:query [:find ?b :in $ ?c :where [?b :block/parent ?c]] :inputs [:current-block]}`,
				query.WithCurrentPage("Project"))
			Expect(errors.Is(err, query.ErrUnsupported)).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("the input :current-block is not supported")))
		})
	})
})
