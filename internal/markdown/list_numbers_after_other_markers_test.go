package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// An ordered list can start on the same line as another marker: after the
// bullet of a block (`- 2. a`), after a quote marker (`> 3. a`) or after the
// marker of an other list item (`* 3. a`). Saving such a block keeps the
// numbers and the spaces after the markers as they are in the file.
var _ = Describe("Parsing then outputting ordered lists that start after another marker", func() {
	Describe("on the line of a bullet", func() {
		// Given a block whose text starts with "2. ", When written, Then
		// the number stays 2 and does not change to 1.
		FullyEqual("Number of a list as the first text of a block", "- 2. second")

		// Given a block whose text starts with a year and a dot, When
		// written, Then the year stays.
		FullyEqual("Year at the start of a block", "- 2024. was a year")

		// Given a block whose text starts with "3. ", When written, Then
		// the number stays 3.
		FullyEqual("Number three as the first text of a block", "- 3. a")

		// Given a block whose text starts with "1." and two spaces, When
		// written, Then the two spaces stay.
		FullyEqual("Two spaces after the number on the bullet line", "- 1.  a")

		// Given a block whose text starts with "2. ", When it is read, Then
		// the first item of the list has the number 2.
		It("reads the number of a list as the first text of a block", func() {
			root, err := markdown.ParseString("- 2. second")
			Expect(err).ToNot(HaveOccurred())

			item := root.Children().FindDeep(content.IsOfType[*content.ListItem]())
			Expect(item).ToNot(BeNil())
			Expect(item.(*content.ListItem).Number).To(Equal("2"))
		})
	})

	Describe("in a quote", func() {
		// Given a quote with a list that uses "1." for each item, When
		// written, Then the second item is not numbered 2.
		FullyEqual("Same numbers in a list in a quote", "- > 1. a\n  > 1. b")

		// Given a quote with a list that starts at 3 and uses ")", When
		// written, Then the numbers stay 3 and 4.
		FullyEqual("Start number of a list in a quote with a parenthesis", "> 3) a\n> 4) b")

		// Given a quote on the bullet line with a list that starts at 3,
		// When written, Then the number stays 3.
		FullyEqual("Number of a list in a quote on the bullet line", "- > 3. a")

		// Given a quote under the text of a block with a list that starts
		// at 3, When written, Then the numbers stay 3 and 4.
		FullyEqual("List that starts at three in a quote under the text of a block", "- a\n  > 3. x\n  > 4. y")

		// Given a quote at the top with "1." for each item, When written,
		// Then the numbers stay 1 and 1.
		FullyEqual("Same numbers in a list in a quote at the top", "> 1. x\n> 1. y")
	})

	Describe("after the marker of another list item", func() {
		// Given an ordered list as the first content of a star item, When
		// written, Then the number stays 3.
		FullyEqual("Ordered list as the first content of a list item", "- t\n  * 3. a")

		// Given an ordered list with the number 10 after a star marker,
		// When written, Then the number stays 10.
		FullyEqual("Number ten after a star marker", "- a\n  * 10. x")
	})

	Describe("on a line of its own", func() {
		// Given a list on its own line under the text of a block, When
		// written, Then the numbers stay. This works now and must keep
		// working.
		FullyEqual("List that starts at three on its own line", "- t\n  3. a\n  4. b")
		FullyEqual("One item that starts at three on its own line", "- a\n  3. b")
		FullyEqual("One item numbered three on its own line", "- a\n  3. x")
	})
})
