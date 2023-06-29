package content_test

import (
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nodes with children", func() {
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
})
