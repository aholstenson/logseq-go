package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The END line of a logbook can be indented more than its other lines, and
// a line of a property drawer can have extra spaces. Saving the block keeps
// these lines as they are in the file.
var _ = Describe("Parsing then outputting the whitespace of logbooks and property drawers", func() {
	// Given a logbook whose END line is indented more, When written, Then
	// the indentation stays.
	FullyEqual("Logbook END line with more indentation", "- DONE x\n  :LOGBOOK:\n  CLOCK: [2024-01-01 Mon 10:00:00]--[2024-01-01 Mon 11:00:00] =>  01:00:00\n    :END:")

	// Given a property drawer line with extra spaces, When written, Then
	// the spaces stay and the value is `bar`.
	It("keeps the spaces of a property drawer line", func() {
		input := "- x\n  :PROPERTIES:\n  :foo:   bar  \n  :END:"
		Expect(parseAndOutput(input)).To(Equal(input))

		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		properties := root.Children().FindDeep(content.IsOfType[*content.Properties]())
		Expect(properties).ToNot(BeNil())
		value := ""
		for _, node := range properties.(*content.Properties).Get("foo").FilterDeep(content.IsOfType[*content.Text]()) {
			value += node.(*content.Text).Value
		}
		Expect(value).To(Equal("bar"))
	})
})
