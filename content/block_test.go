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
	})
})
