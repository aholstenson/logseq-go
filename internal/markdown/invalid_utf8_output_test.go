package markdown_test

import (
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Text can hold bytes that are not valid UTF-8. Writing the text must keep
// those bytes or return an error.
var _ = Describe("Parsing then outputting text with bytes that are not valid UTF-8", func() {
	// Given blocks with bytes that are not valid UTF-8, When parsed and
	// written, Then the bytes are the same or writing returns an error.
	It("keeps bytes that are not valid UTF-8 or returns an error", func() {
		input := "- a\xff\xfeb [[p\xc3]] #t\xe2\x82\n- c"

		block, err := markdown.ParseString(input)
		if err != nil {
			return
		}
		v, err := markdown.AsString(block)
		if err != nil {
			return
		}
		Expect([]byte(v)).To(Equal([]byte(input)))
	})
})
