package utils_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LocationOf", func() {
	It("finds the location of a block on its page", func() {
		grandchild := content.NewBlock(content.NewText("grandchild"))
		first := content.NewBlock(content.NewText("first"), content.NewBlock(content.NewText("child")), content.NewBlock(content.NewText("second child"), grandchild))
		root := content.NewBlock(content.NewBlock(content.NewText("pre")), first)

		location, onPage := utils.LocationOf(grandchild)
		Expect(onPage).To(BeTrue())
		Expect(location).To(Equal([]int{1, 1, 0}))

		location, _ = utils.LocationOf(first)
		Expect(location).To(Equal([]int{1}))

		_, onPage = utils.LocationOf(root)
		Expect(onPage).To(BeFalse())
	})
})
