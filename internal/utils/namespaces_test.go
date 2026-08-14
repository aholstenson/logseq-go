package utils_test

import (
	"github.com/aholstenson/logseq-go/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Namespaces", func() {
	Describe("NamespaceOf", func() {
		It("has no namespace for a title without one", func() {
			Expect(utils.NamespaceOf("Page")).To(Equal(""))
		})

		It("takes the part before the last slash", func() {
			Expect(utils.NamespaceOf("Parent/Child")).To(Equal("Parent"))
		})

		It("keeps the parts of a nested namespace", func() {
			Expect(utils.NamespaceOf("Parent/Child/Grandchild")).To(Equal("Parent/Child"))
		})

		It("ignores slashes at the start and end of a title", func() {
			Expect(utils.NamespaceOf("/Page/")).To(Equal(""))
			Expect(utils.NamespaceOf("/Parent/Child/")).To(Equal("Parent"))
		})

		It("ignores empty parts of a title", func() {
			Expect(utils.NamespaceOf("Parent//Child")).To(Equal("Parent"))
		})
	})

	Describe("NamespacesOf", func() {
		It("has no namespaces for a title without one", func() {
			Expect(utils.NamespacesOf("Page")).To(BeEmpty())
		})

		It("has the namespace of a title in one", func() {
			Expect(utils.NamespacesOf("Parent/Child")).To(Equal([]string{"Parent"}))
		})

		It("has every namespace of a nested title, closest one first", func() {
			Expect(utils.NamespacesOf("Parent/Child/Grandchild")).To(Equal([]string{
				"Parent/Child",
				"Parent",
			}))
		})
	})
})
