package content_test

import (
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lists", func() {
	Describe("NewListFromMarker", func() {
		It("keeps unordered markers", func() {
			for _, marker := range []byte{'*', '+'} {
				list := content.NewListFromMarker(marker)
				Expect(list.Type).To(Equal(content.ListTypeUnordered))
				Expect(list.Marker).To(Equal(marker))
			}
		})

		It("keeps ordered markers", func() {
			for _, marker := range []byte{'.', ')'} {
				list := content.NewListFromMarker(marker)
				Expect(list.Type).To(Equal(content.ListTypeOrdered))
				Expect(list.Marker).To(Equal(marker))
			}
		})

		It("turns the block marker into an unordered list", func() {
			list := content.NewListFromMarker('-')
			Expect(list.Type).To(Equal(content.ListTypeUnordered))
			Expect(list.Marker).To(Equal(byte('*')))
		})

		It("turns an unknown marker into an unordered list", func() {
			list := content.NewListFromMarker('x')
			Expect(list.Type).To(Equal(content.ListTypeUnordered))
			Expect(list.Marker).To(Equal(byte('*')))
		})

		It("keeps the items it is given", func() {
			item := content.NewListItem(content.NewText("item"))
			list := content.NewListFromMarker('x', item)

			Expect(list.Children()).To(HaveLen(1))
			Expect(list.Children()[0]).To(Equal(item))
		})
	})

	Describe("WithMarker", func() {
		It("turns an unknown marker into an unordered list", func() {
			list := content.NewOrderedList().WithMarker('x')
			Expect(list.Type).To(Equal(content.ListTypeUnordered))
			Expect(list.Marker).To(Equal(byte('*')))
		})
	})

	Describe("WithType", func() {
		It("replaces a marker that does not fit the type", func() {
			list := content.NewUnorderedList().WithType(content.ListTypeOrdered)
			Expect(list.Marker).To(Equal(byte('.')))
		})

		It("keeps a marker that fits the type", func() {
			list := content.NewListFromMarker(')').WithType(content.ListTypeOrdered)
			Expect(list.Marker).To(Equal(byte(')')))
		})
	})
})
