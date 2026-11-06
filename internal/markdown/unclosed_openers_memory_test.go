package markdown_test

import (
	"runtime"
	"strings"

	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Text pieces next to each other are joined into one text node. The memory
// used to join them must grow with the length of the text, not with the
// square of the number of pieces.
var _ = Describe("Memory use for many unclosed link openers", func() {
	It("reads a line of 40 000 unclosed page link openers with memory that grows with the line", func() {
		// Given a block with one line of 40 000 `[[a ` pieces (160 KB)
		input := "- " + strings.Repeat("[[a ", 40000)

		// When it is parsed
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		_, err := markdown.ParseString(input)
		runtime.ReadMemStats(&after)
		Expect(err).ToNot(HaveOccurred())

		// Then the total memory allocated stays below 1000 times the input size
		allocated := after.TotalAlloc - before.TotalAlloc
		Expect(allocated).To(BeNumerically("<", uint64(1000*len(input))))
	})
})
