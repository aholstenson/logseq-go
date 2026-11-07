package markdown_test

import (
	"regexp"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The numbers of an ordered list are kept as they are in the file. When a
// program adds, removes or moves items, the written numbers must still make
// one list that starts at the same number and has the same style.
var _ = Describe("Numbers of an ordered list after its items change", func() {
	parse := func(input string) *content.Block {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		return root
	}

	write := func(root *content.Block) string {
		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		return v
	}

	item := func(text string) *content.ListItem {
		return content.NewListItem(content.NewParagraph(content.NewText(text)))
	}

	firstList := func(root *content.Block) *content.List {
		list := root.Children().FindDeep(content.IsOfType[*content.List]())
		Expect(list).ToNot(BeNil())
		return list.(*content.List)
	}

	items := func(root *content.Block) []*content.ListItem {
		result := []*content.ListItem{}
		for _, n := range root.Children().FilterDeep(content.IsOfType[*content.ListItem]()) {
			result = append(result, n.(*content.ListItem))
		}

		return result
	}

	textsOf := func(node content.HasChildren) []string {
		result := []string{}
		for _, n := range node.Children().FilterDeep(content.IsOfType[*content.Text]()) {
			result = append(result, n.(*content.Text).Value)
		}

		return result
	}

	numberPattern := regexp.MustCompile(`(?m)^(?:- |\s+)(\d+)\. `)
	numbersOf := func(output string) []string {
		numbers := []string{}
		for _, m := range numberPattern.FindAllStringSubmatch(output, -1) {
			numbers = append(numbers, m[1])
		}

		return numbers
	}

	It("keeps a list when its first item is removed", func() {
		// Given a block with a paragraph and a list 1, 2, 3
		root := parse("- a\n  1. x\n  2. y\n  3. z")

		// When the first item is removed and the block is written and read again
		items(root)[0].RemoveSelf()
		output := write(root)
		again := parse(output)

		// Then the block has the paragraph `a` and a list with the items `y` and `z`
		Expect(again.Blocks()).To(HaveLen(1), "output: %q", output)
		block := again.Blocks()[0]
		paragraph := block.Children().Find(content.IsOfType[*content.Paragraph]())
		Expect(paragraph).ToNot(BeNil(), "output: %q", output)
		Expect(textsOf(paragraph.(*content.Paragraph))).To(Equal([]string{"a"}), "output: %q", output)

		list := block.Children().Find(content.IsOfType[*content.List]())
		Expect(list).ToNot(BeNil(), "output: %q", output)
		Expect(textsOf(list.(*content.List))).To(Equal([]string{"y", "z"}), "output: %q", output)
	})

	It("does not write a number two times when an item is inserted", func() {
		// Given a list 1, 2, 3, 4
		root := parse("- 1. a\n  2. b\n  3. c\n  4. d")

		// When an item is inserted after `c` and the block is written
		firstList(root).InsertChildAfter(item("new"), items(root)[2])
		output := write(root)

		// Then the numbers are 1, 2, 3, 4, 5
		Expect(numbersOf(output)).To(Equal([]string{"1", "2", "3", "4", "5"}), "output: %q", output)
	})

	It("keeps the style of a list with the same number on each item when an item is added", func() {
		// Given a list where each item has the number 1
		root := parse("- a\n  1. x\n  1. y")

		// When an item is added at the end and the block is written
		firstList(root).AddChild(item("z"))

		// Then the new item also has the number 1
		Expect(write(root)).To(Equal("- a\n  1. x\n  1. y\n  1. z"))
	})

	It("keeps the start number of a list when an item is put first", func() {
		// Given a list that starts at 3
		root := parse("- a\n\n  3. x\n  4. y")

		// When an item is put first and the block is written
		firstList(root).PrependChild(item("new"))
		output := write(root)

		// Then the numbers are 3, 4, 5
		Expect(numbersOf(output)).To(Equal([]string{"3", "4", "5"}), "output: %q", output)
	})

	It("keeps the numbers of a list whose items are not added, removed or moved", func() {
		// Given a list with the numbers 3 and 5
		root := parse("- a\n\n  3. x\n  5. y")

		// When only the text of an item is changed and the block is written
		text := root.Children().FindDeep(func(n content.Node) bool {
			t, ok := n.(*content.Text)
			return ok && t.Value == "x"
		})
		Expect(text).ToNot(BeNil())
		text.(*content.Text).WithValue("X")

		// Then the numbers are not changed
		Expect(write(root)).To(Equal("- a\n\n  3. X\n  5. y"))
	})
})
