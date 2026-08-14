package content_test

import (
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Base nodes", func() {
	Describe("No children", func() {
		It("can remove self from parent", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)
			Expect(parent.Children()).To(HaveLen(1))
			Expect(node.Parent()).To(Equal(parent))

			node.RemoveSelf()
			Expect(parent.Children()).To(BeEmpty())
			Expect(node.Parent()).To(BeNil())
		})

		It("can replace self with another node", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)
			Expect(parent.Children()).To(HaveLen(1))
			Expect(node.Parent()).To(Equal(parent))

			node2 := content.NewText("child2")
			node.ReplaceWith(node2)
			Expect(parent.Children()).To(HaveLen(1))
			Expect(parent.Children()[0]).To(Equal(node2))
			Expect(node.Parent()).To(BeNil())
			Expect(node2.Parent()).To(Equal(parent))
		})
	})

	Describe("With children", func() {
		It("can add children", func() {
			parent := content.NewParagraph()
			Expect(parent.Children()).To(BeEmpty())

			node := content.NewText("child")
			parent.AddChild(node)
			Expect(parent.Children()).To(HaveLen(1))
			Expect(node.Parent()).To(Equal(parent))
		})

		It("can remove children", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			parent.RemoveChild(node)
			Expect(parent.Children()).To(BeEmpty())
			Expect(node.Parent()).To(BeNil())
		})

		It("can prepend children when empty", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.PrependChild(node)

			Expect(parent.Children()).To(HaveLen(1))
			Expect(parent.Children()[0]).To(Equal(node))
			Expect(node.Parent()).To(Equal(parent))
		})

		It("can prepend children when not empty", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			node2 := content.NewText("child2")
			parent.PrependChild(node2)

			Expect(parent.Children()).To(HaveLen(2))
			Expect(parent.Children()[0]).To(Equal(node2))
			Expect(parent.Children()[1]).To(Equal(node))
			Expect(node.Parent()).To(Equal(parent))
			Expect(node2.Parent()).To(Equal(parent))
		})

		It("can insert children after when last node", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			node2 := content.NewText("child2")
			parent.InsertChildAfter(node2, node)

			Expect(parent.Children()).To(HaveLen(2))
			Expect(parent.Children()[0]).To(Equal(node))
			Expect(parent.Children()[1]).To(Equal(node2))

			Expect(node.Parent()).To(Equal(parent))
			Expect(node2.Parent()).To(Equal(parent))

			Expect(node2.NextSibling()).To(BeNil())
			Expect(node.NextSibling()).To(Equal(node2))

			Expect(node.PreviousSibling()).To(BeNil())
			Expect(node2.PreviousSibling()).To(Equal(node))
		})

		It("can insert children after when not last node", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			node2 := content.NewText("child2")
			parent.InsertChildAfter(node2, node)

			node3 := content.NewText("child3")
			parent.InsertChildAfter(node3, node)

			Expect(parent.Children()).To(HaveLen(3))
			Expect(parent.Children()[0]).To(Equal(node))
			Expect(parent.Children()[1]).To(Equal(node3))
			Expect(parent.Children()[2]).To(Equal(node2))

			Expect(node.Parent()).To(Equal(parent))
			Expect(node2.Parent()).To(Equal(parent))
			Expect(node3.Parent()).To(Equal(parent))

			Expect(node.NextSibling()).To(Equal(node3))
			Expect(node3.NextSibling()).To(Equal(node2))
			Expect(node2.NextSibling()).To(BeNil())

			Expect(node.PreviousSibling()).To(BeNil())
			Expect(node3.PreviousSibling()).To(Equal(node))
			Expect(node2.PreviousSibling()).To(Equal(node3))
		})

		It("does not add children that are not allowed", func() {
			parent := content.NewParagraph()
			parent.AddChild(content.NewHeading(1))

			Expect(parent.Children()).To(BeEmpty())
		})

		It("does not prepend children that are not allowed", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			parent.PrependChild(content.NewHeading(1))

			Expect(parent.Children()).To(HaveLen(1))
			Expect(parent.Children()[0]).To(Equal(node))
		})

		It("does not insert children before that are not allowed", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			Expect(parent.InsertChildBefore(content.NewHeading(1), node)).To(BeFalse())
			Expect(parent.Children()).To(HaveLen(1))
			Expect(parent.Children()[0]).To(Equal(node))
		})

		It("does not insert children after that are not allowed", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			Expect(parent.InsertChildAfter(content.NewHeading(1), node)).To(BeFalse())
			Expect(parent.Children()).To(HaveLen(1))
			Expect(parent.Children()[0]).To(Equal(node))
		})

		It("does not replace children with ones that are not allowed", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			Expect(parent.ReplaceChild(node, content.NewHeading(1))).To(BeFalse())
			Expect(parent.Children()).To(HaveLen(1))
			Expect(parent.Children()[0]).To(Equal(node))
			Expect(node.Parent()).To(Equal(parent))
		})

		It("keeps a node it rejected free of a parent", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			other := content.NewParagraph()
			rejected := content.NewText("rejected")
			other.AddChild(rejected)

			// A paragraph is not allowed inside a paragraph, so the node it is
			// holding has to stay where it is.
			parent.PrependChild(other)

			Expect(parent.Children()).To(HaveLen(1))
			Expect(other.Children()).To(HaveLen(1))
			Expect(rejected.Parent()).To(Equal(other))
		})

		It("can set children", func() {
			parent := content.NewParagraph()
			parent.AddChildren(
				content.NewText("child"),
				content.NewText("child2"),
			)

			node := content.NewText("replacement")
			parent.SetChildren(node)

			Expect(parent.Children()).To(HaveLen(1))
			Expect(parent.Children()[0]).To(Equal(node))
		})

		It("detaches all children when setting children", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			node2 := content.NewText("child2")
			node3 := content.NewText("child3")
			parent.AddChildren(node, node2, node3)

			parent.SetChildren(content.NewText("replacement"))

			for _, removed := range []*content.Text{node, node2, node3} {
				Expect(removed.Parent()).To(BeNil())
				Expect(removed.NextSibling()).To(BeNil())
				Expect(removed.PreviousSibling()).To(BeNil())
			}
		})

		It("keeps children reachable when a replaced child is removed later", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			node2 := content.NewText("child2")
			parent.AddChildren(node, node2)

			replacement := content.NewText("replacement")
			parent.SetChildren(replacement)

			// Something still holding on to a replaced child removes it, which
			// must not disturb the children the node has now.
			node2.RemoveSelf()

			added := content.NewText("added")
			parent.AddChild(added)

			Expect(parent.Children()).To(HaveLen(2))
			Expect(parent.Children()[0]).To(Equal(replacement))
			Expect(parent.Children()[1]).To(Equal(added))
		})

		It("can set children to one of the existing children", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			node2 := content.NewText("child2")
			parent.AddChildren(node, node2)

			parent.SetChildren(node2)

			Expect(parent.Children()).To(HaveLen(1))
			Expect(parent.Children()[0]).To(Equal(node2))
			Expect(node.Parent()).To(BeNil())
			Expect(node2.Parent()).To(Equal(parent))
		})

		It("can insert children before when first node", func() {
			parent := content.NewParagraph()
			node := content.NewText("child")
			parent.AddChild(node)

			node2 := content.NewText("child2")
			parent.InsertChildBefore(node2, node)

			Expect(parent.Children()).To(HaveLen(2))
			Expect(parent.Children()[0]).To(Equal(node2))
			Expect(parent.Children()[1]).To(Equal(node))

			Expect(node.Parent()).To(Equal(parent))
			Expect(node2.Parent()).To(Equal(parent))

			Expect(node2.NextSibling()).To(Equal(node))
			Expect(node.NextSibling()).To(BeNil())

			Expect(node2.PreviousSibling()).To(BeNil())
			Expect(node.PreviousSibling()).To(Equal(node2))
		})
	})
})
