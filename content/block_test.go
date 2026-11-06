package content_test

import (
	"github.com/aholstenson/logseq-go/content"
	. "github.com/aholstenson/logseq-go/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Blocks", func() {
	Describe("Properties", func() {
		It("can get properties from empty block", func() {
			block := content.NewBlock()

			properties := block.Properties()
			Expect(properties.FirstChild()).To(BeNil())

			Expect(block.FirstChild()).To(EqualNode(properties))
		})

		It("can get properties from block with properties", func() {
			block := content.NewBlock(content.NewProperties(
				content.NewProperty("key", content.NewText("value")),
			))

			properties := block.Properties()
			Expect(properties.Get("key")).To(EqualsNodes(content.NewText("value")))
		})

		It("can get properties that follow the content of the block", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("content")),
				content.NewProperties(
					content.NewProperty("key", content.NewText("value")),
				),
			)

			properties := block.Properties()
			Expect(properties.Get("key")).To(EqualsNodes(content.NewText("value")))
		})

		It("ignores properties that belong to a sub block", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("content")),
				content.NewBlock(content.NewProperties(
					content.NewProperty("key", content.NewText("value")),
				)),
			)

			Expect(block.FindProperties()).To(BeNil())
			Expect(block.Properties().Get("key")).To(BeEmpty())
		})

		It("gets a property by a name that only differs in case or in underscores", func() {
			block := content.NewBlock(content.NewProperties(
				content.NewProperty("My_Prop", content.NewText("value")),
			))

			properties := block.Properties()
			Expect(properties.Get("my-prop")).To(EqualsNodes(content.NewText("value")))
			Expect(properties.Get("MY_PROP")).To(EqualsNodes(content.NewText("value")))
		})

		It("keeps the written name when it sets a property by a normalized name", func() {
			block := content.NewBlock(content.NewProperties(
				content.NewProperty("My_Prop", content.NewText("value")),
			))

			properties := block.Properties()
			properties.Set("my-prop", content.NewText("changed"))

			Expect(properties.Children()).To(HaveLen(1))
			Expect(properties.GetAsNode("my-prop").Name).To(Equal("My_Prop"))
			Expect(properties.Get("my-prop")).To(EqualsNodes(content.NewText("changed")))
		})

		It("removes a property by a normalized name", func() {
			block := content.NewBlock(content.NewProperties(
				content.NewProperty("My_Prop", content.NewText("value")),
			))

			properties := block.Properties()
			properties.Remove("my-prop")

			Expect(properties.FirstChild()).To(BeNil())
		})

		It("creates properties after the content of the block", func() {
			paragraph := content.NewParagraph(content.NewText("content"))
			block := content.NewBlock(paragraph)

			properties := block.Properties()
			Expect(paragraph.NextSibling()).To(EqualNode(properties))
		})

		It("creates properties before the sub blocks of the block", func() {
			sub := content.NewBlock(content.NewParagraph(content.NewText("sub")))
			block := content.NewBlock(
				content.NewParagraph(content.NewText("content")),
				sub,
			)

			properties := block.Properties()
			Expect(sub.PreviousSibling()).To(EqualNode(properties))
		})
	})

	Describe("FindProperties", func() {
		It("returns nil for a block without properties", func() {
			block := content.NewBlock(content.NewParagraph(content.NewText("content")))

			Expect(block.FindProperties()).To(BeNil())
			Expect(block.Children()).To(HaveLen(1))
		})
	})

	Describe("ID", func() {
		It("returns the id property that follows the content of the block", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("content")),
				content.NewProperties(
					content.NewProperty("id", content.NewText("6500e8a3-0000-4000-8000-000000000001")),
				),
			)

			Expect(block.ID()).To(Equal("6500e8a3-0000-4000-8000-000000000001"))
		})

		It("returns an empty id for a block without properties", func() {
			block := content.NewBlock(content.NewParagraph(content.NewText("content")))

			Expect(block.ID()).To(Equal(""))
		})

		It("does not add properties to a block without them", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("content")),
				content.NewBlock(content.NewParagraph(content.NewText("child"))),
			)

			Expect(block.ID()).To(Equal(""))
			Expect(block.FindProperties()).To(BeNil())
		})

		It("returns an empty id for a block with other properties", func() {
			block := content.NewBlock(
				content.NewParagraph(content.NewText("content")),
				content.NewProperties(content.NewProperty("type", content.NewText("book"))),
			)

			Expect(block.ID()).To(Equal(""))
		})
	})
})
