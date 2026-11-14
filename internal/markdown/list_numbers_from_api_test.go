package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A program can make an ordered list that does not start at one, by setting
// the number of its first item. The list must read back as a list with the
// same start number, also when an item has a number that a list item cannot
// have, or when the list comes directly after a line of text.
var _ = Describe("Numbers of an ordered list made with the API", func() {
	item := func(number string, value string) *content.ListItem {
		i := content.NewListItem(content.NewParagraph(content.NewText(value)))
		i.Number = number
		return i
	}

	numbersOf := func(root *content.Block) []string {
		var numbers []string
		for _, n := range root.Children().FilterDeep(content.IsOfType[*content.ListItem]()) {
			numbers = append(numbers, n.(*content.ListItem).Number)
		}

		return numbers
	}

	write := func(list *content.List) (string, *content.Block) {
		root := content.NewBlock(content.NewBlock(content.NewParagraph(content.NewText("x")), list))
		output, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		again, err := markdown.ParseString(output)
		Expect(err).ToNot(HaveOccurred())
		return output, again
	}

	It("numbers the items after a first item with a number", func() {
		// Given a list where only the first item has the number 3
		list := content.NewOrderedList(item("3", "a"), item("", "b"), item("", "c"))

		// When it is written and read again
		output, root := write(list)

		// Then the items are numbered from 3
		Expect(output).To(Equal("- x\n\n  3. a\n  4. b\n  5. c"))
		Expect(numbersOf(root)).To(Equal([]string{"3", "4", "5"}))
	})

	It("keeps the leading zeros of the first number", func() {
		// Given a list where only the first item has the number 007
		list := content.NewOrderedList(item("007", "a"), item("", "b"))

		// When it is written, Then the first number is kept
		output, _ := write(list)
		Expect(output).To(Equal("- x\n\n  007. a\n  8. b"))
	})

	DescribeTable("numbers the items from one when a number cannot be a list number",
		func(number string) {
			// Given a list with an item with the number
			list := content.NewOrderedList(item(number, "a"))

			// When it is written and read again
			output, root := write(list)

			// Then it is a list that starts at 1
			Expect(root.Children().FindDeep(content.IsOfType[*content.List]())).ToNot(BeNil(), "output: %q", output)
			Expect(numbersOf(root)).To(Equal([]string{"1"}), "output: %q", output)
		},
		Entry("ten digits", "1234567890"),
		Entry("a negative number", "-1"),
		Entry("a letter", "a"),
	)

	It("keeps the start number of a list directly after a line of text", func() {
		// Given a paragraph and, on the next line, a list that starts at 3
		list := content.NewOrderedList(item("3", "a"), item("", "b"))
		list.SetPreviousLineType(content.PreviousLineTypeNonBlank)

		// When it is written and read again
		output, root := write(list)

		// Then a blank line comes before the list, and it starts at 3
		Expect(output).To(Equal("- x\n\n  3. a\n  4. b"))
		Expect(numbersOf(root)).To(Equal([]string{"3", "4"}))
	})

	It("writes a list that starts at 1 on the line after text", func() {
		// Given a paragraph and, on the next line, a list that starts at 1
		list := content.NewOrderedList(item("", "a"), item("", "b"))
		list.SetPreviousLineType(content.PreviousLineTypeNonBlank)

		// When it is written, Then no blank line comes before the list
		output, _ := write(list)
		Expect(output).To(Equal("- x\n  1. a\n  2. b"))
	})
})
