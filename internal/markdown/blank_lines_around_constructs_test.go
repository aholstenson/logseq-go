package markdown_test

import (
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq shows a blank line inside a block as an empty line. Saving a page
// must not add blank lines around block constructs that the file did not
// have.
var _ = Describe("Parsing then outputting block constructs without blank lines around them", func() {
	// Given block constructs directly after or before text in a block, When
	// written, Then no blank line is added.
	FullyEqual("Quote block directly after text", "- quote\n  #+BEGIN_QUOTE\n  a\n  #+END_QUOTE")
	FullyEqual("Thematic break between two lines of text", "- above\n  ---\n  below")
	FullyEqual("HTML block directly after text", "- html block\n  <div>\n  x\n  </div>")
	FullyEqual("Text directly after a code block", "- a\n  ```js\n  x\n  ```\n  after")

	// Given text directly after a SCHEDULED line or a logbook, When
	// written, Then no blank line is added.
	FullyEqual("Text directly after a SCHEDULED line", "- TODO a\n  SCHEDULED: <2024-01-01 Mon>\n  note")
	FullyEqual("Text directly after an empty logbook", "- a\n  :LOGBOOK:\n  :END:\n  c")

	// Given a page link in text directly after a SCHEDULED line, When
	// parsed, Then the block has no page reference, as Logseq drops a
	// paragraph that starts with a timestamp from the block body.
	It("does not reference a page from text directly after a SCHEDULED line", func() {
		root, err := markdown.ParseString("- TODO a\n  SCHEDULED: <2024-01-01 Mon>\n  note [[x]]")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()).To(HaveLen(1))
		Expect(root.Blocks()[0].Content().PageReferences()).To(BeEmpty())
	})

	// Given blank lines around block constructs, When written, Then the
	// blank lines are kept.
	FullyEqual("Quote block after a blank line", "- a\n\n  #+BEGIN_QUOTE\n  a\n  #+END_QUOTE")
	FullyEqual("Thematic break with blank lines around it", "- above\n\n  ---\n\n  below")
	FullyEqual("HTML block after a blank line", "- html block\n\n  <div>\n  x\n  </div>")
	FullyEqual("Query directly after text", "- a\n  #+BEGIN_QUERY\n  {}\n  #+END_QUERY")

	// Given a page link in text after a blank line after a SCHEDULED line,
	// When parsed, Then the block has no page reference, as Logseq reads
	// the date and the text as one paragraph.
	It("does not reference a page from text after a blank line after a SCHEDULED line", func() {
		root, err := markdown.ParseString("- TODO a\n  SCHEDULED: <2024-01-01 Mon>\n\n  note [[x]]")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()[0].Content().PageReferences()).To(BeEmpty())
	})

	// Given a page link in text after a logbook after a SCHEDULED line,
	// When parsed, Then the block references the page, as the logbook ends
	// the paragraph of the date.
	It("references a page from text after a logbook after a SCHEDULED line", func() {
		root, err := markdown.ParseString("- TODO a\n  SCHEDULED: <2024-01-01 Mon>\n  :LOGBOOK:\n  :END:\n  note [[x]]")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()[0].Content().PageReferences()).To(HaveLen(1))
	})

	// Given a page link in the first line of a task with a SCHEDULED line,
	// When parsed, Then the block references the page.
	It("references a page from the first line of a task with a SCHEDULED line", func() {
		root, err := markdown.ParseString("- TODO a [[x]]\n  SCHEDULED: <2024-01-01 Mon>")
		Expect(err).ToNot(HaveOccurred())

		Expect(root.Blocks()[0].Content().PageReferences()).To(HaveLen(1))
	})
})
