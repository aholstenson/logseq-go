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

		It("makes the name lower case in the same way as Logseq", func() {
			// Given names with a capital sigma and a dotted capital I
			// Then the lower case form is the one that JavaScript makes
			Expect(content.NormalizePropertyName("ΑΣ")).To(Equal("ας"))
			Expect(content.NormalizePropertyName("ΣΑ")).To(Equal("σα"))
			Expect(content.NormalizePropertyName("İ")).To(Equal("i\u0307"))

			block := content.NewBlock(content.NewProperties(
				content.NewProperty("ΟΔΟΣ", content.NewText("value")),
			))
			Expect(block.Properties().Get("οδος")).To(EqualsNodes(content.NewText("value")))
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

		It("gets and sets the last property when a name is written twice", func() {
			block := content.NewBlock(content.NewProperties(
				content.NewProperty("foo", content.NewText("1")),
				content.NewProperty("FOO", content.NewText("2")),
			))

			properties := block.Properties()
			Expect(properties.Get("foo")).To(EqualsNodes(content.NewText("2")))
			Expect(properties.IsOverridden(properties.Children()[0].(*content.Property))).To(BeTrue())
			Expect(properties.IsOverridden(properties.Children()[1].(*content.Property))).To(BeFalse())

			properties.Set("foo", content.NewText("3"))
			Expect(properties.Children()).To(HaveLen(2))
			Expect(properties.Children()[0].(*content.Property).Children()).To(EqualsNodes(content.NewText("1")))
			Expect(properties.Get("foo")).To(EqualsNodes(content.NewText("3")))
		})

		It("removes all the properties with a name that is written twice", func() {
			block := content.NewBlock(content.NewProperties(
				content.NewProperty("foo", content.NewText("1")),
				content.NewProperty("other", content.NewText("x")),
				content.NewProperty("foo", content.NewText("2")),
			))

			properties := block.Properties()
			properties.Remove("foo")

			Expect(properties.Get("foo")).To(BeEmpty())
			Expect(properties.Children()).To(HaveLen(1))
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

		It("gives a new id to the id property that Logseq reads", func() {
			// Given an id that is a uuid followed by a custom-id that is not
			block := content.NewBlock(
				content.NewParagraph(content.NewText("content")),
				content.NewProperties(
					content.NewProperty("id", content.NewText("6500e8a3-0000-4000-8000-000000000001")),
					content.NewProperty("custom-id", content.NewText("not-a-uuid")),
				),
			)
			Expect(block.ID()).To(BeEmpty())

			// When the block gets an id
			block.WithID()

			// Then the custom-id gets the new id, as Logseq reads it last
			Expect(block.ID()).ToNot(BeEmpty())
			Expect(block.ID()).ToNot(Equal("6500e8a3-0000-4000-8000-000000000001"))
			Expect(block.FindProperties().Children()).To(HaveLen(2))
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
