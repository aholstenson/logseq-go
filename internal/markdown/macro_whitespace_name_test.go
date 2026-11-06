package markdown_test

import (
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Text that the reader accepts must also be accepted by the writer, or no
// page that holds the text can be saved.
var _ = Describe("Macros with a tab in the name", func() {
	savesUnchanged := func(name string, input string) {
		It(name, func() {
			// Given a block with a tab after `{{`
			block, err := markdown.ParseString(input)
			Expect(err).ToNot(HaveOccurred())

			// When it is written
			written, err := markdown.AsString(block)

			// Then there is no error and the file text is unchanged
			Expect(err).ToNot(HaveOccurred())
			Expect(written).To(Equal(input))
		})
	}

	savesUnchanged("saves a macro whose name and argument are split by a tab", "- {{foo\ta}}")
	savesUnchanged("saves a tab and a macro after an opening brace pair", "- {{\t{{m a, b}}x")
})
