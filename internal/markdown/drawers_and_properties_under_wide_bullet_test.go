package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// wideDrawerPageText reads a page and writes it as a page is saved: the
// writer output with a line break at the end.
func wideDrawerPageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// wideDrawerPropertyValue gets the text of a property of a block.
func wideDrawerPropertyValue(block *content.Block, name string) string {
	properties := block.FindProperties()
	if properties == nil {
		return ""
	}
	value := ""
	for _, node := range properties.Get(name).FilterDeep(content.IsOfType[*content.Text]()) {
		value += node.(*content.Text).Value
	}
	return value
}

// A bullet can have more than one space or a tab after the dash. The first
// content of such a block can be a logbook, a property drawer, a `\begin`
// math block or property lines. Saving the block keeps the lines of the
// construct in it, so that a second save does not close it early.
var _ = Describe("Parsing then outputting drawers and properties as the first content under a bullet with more than one space", func() {
	// Given a page with a logbook as the first content under a three-space
	// bullet, When written two times, Then both outputs are the input and
	// the second read has a logbook with one clock entry.
	It("keeps a logbook under a three-space bullet over two saves", func() {
		input := "- x\n-   :LOGBOOK:\n    CLOCK: [2024-01-01 Mon 10:00]--[2024-01-01 Mon 11:00] =>  01:00:00\n    :END:\n"
		first := wideDrawerPageText(input)
		Expect(first).To(Equal(input))
		second := wideDrawerPageText(first)
		Expect(second).To(Equal(input))

		again, err := markdown.ParseString(second, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		logbook := again.Children().FindDeep(content.IsOfType[*content.Logbook]())
		Expect(logbook).ToNot(BeNil())
		Expect(logbook.(*content.Logbook).Children().Filter(content.IsOfType[*content.LogbookEntryClock]())).To(HaveLen(1))
	})

	// Given a page with a property drawer as the first content under a
	// tab bullet, When written two times, Then the second block still has
	// the property `id` and the output has one `:END:` line.
	It("keeps a property drawer under a tab bullet over two saves", func() {
		second := wideDrawerPageText(wideDrawerPageText("- x\n-\t:PROPERTIES:\n\t:id: 6540e3c5\n\t:END:\n"))
		Expect(strings.Count(second, ":END:")).To(Equal(1), "text: %q", second)

		again, err := markdown.ParseString(second, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())
		blocks := again.Blocks()
		Expect(blocks).To(HaveLen(2))
		Expect(wideDrawerPropertyValue(blocks[1], "id")).To(Equal("6540e3c5"), "text: %q", second)
	})

	// Given property lines as the first content under a three-space
	// bullet, When written and read again, Then the second block has one
	// group of properties with `a` and `c`.
	It("keeps property lines under a three-space bullet in one group", func() {
		v := parseAndOutput("- x\n-   a:: b\n    c:: d")

		again, err := markdown.ParseString(v)
		Expect(err).ToNot(HaveOccurred())
		blocks := again.Blocks()
		Expect(blocks).To(HaveLen(2))
		groups := blocks[1].Children().Filter(content.IsOfType[*content.Properties]())
		Expect(groups).To(HaveLen(1), "text: %q", v)
		properties := groups[0].(*content.Properties)
		Expect(properties.GetAsNode("a")).ToNot(BeNil())
		Expect(properties.GetAsNode("c")).ToNot(BeNil())
	})

	// Given a page with `\begin` math as the first content under a
	// two-space bullet, When written three times, Then every output has
	// one `\end{align}` line.
	It("keeps one end line of begin math under a two-space bullet over three saves", func() {
		v := "-  \\begin{align}\n   x\n   \\end{align}\n"
		for i := 0; i < 3; i++ {
			v = wideDrawerPageText(v)
			Expect(strings.Count(v, "\\end{align}")).To(Equal(1), "save %d: %q", i+1, v)
		}
	})

	// Given a page with a logbook under a one-space bullet, When written,
	// Then the text is the same. This works now and must keep working.
	It("keeps a logbook under a one-space bullet", func() {
		input := "- :LOGBOOK:\n  CLOCK: [2024-01-01 Mon 10:00]--[2024-01-01 Mon 11:00] =>  01:00:00\n  :END:\n"
		Expect(wideDrawerPageText(input)).To(Equal(input))
	})
})
