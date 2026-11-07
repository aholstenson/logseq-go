package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A `#+BEGIN_` line in a quote without its `#+END_` line is text for
// Logseq. Reading keeps it text, and saving does not add an `#+END_` line.
var _ = Describe("Parsing then outputting unclosed BEGIN lines in quotes", func() {
	// Given an unclosed NOTE in a quote, When written, Then the text is the
	// same and no command is read.
	It("keeps an unclosed NOTE in a quote as text", func() {
		input := "- > #+BEGIN_NOTE\n  > x\n- b"
		Expect(parseAndOutput(input)).To(Equal(input))

		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Children().FilterDeep(content.IsOfType[*content.AdvancedCommand]())).To(BeEmpty())
	})

	// Given a page with an unclosed SRC in a quote with more quote lines,
	// When written, Then no END line is added.
	It("does not add an END line to an unclosed SRC in a quote", func() {
		input := "- t\n  > #+BEGIN_SRC js\n  > x\n  >\n  > y\n- b\n"
		root, err := markdown.ParseString(input, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		if !strings.HasSuffix(v, "\n") {
			v += "\n"
		}
		Expect(v).To(Equal(input))
	})

	// Given a closed NOTE in a quote, When written, Then the text is the
	// same and one command is read. This works now and must keep working.
	It("keeps a closed NOTE in a quote", func() {
		input := "- > #+BEGIN_NOTE\n  > x\n  > #+END_NOTE\n- b"
		Expect(parseAndOutput(input)).To(Equal(input))

		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(root.Children().FilterDeep(content.IsOfType[*content.AdvancedCommand]())).To(HaveLen(1))
	})
})
