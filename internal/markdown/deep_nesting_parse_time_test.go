package markdown_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The time to read a deeply nested outline must not grow with the cube of
// its depth.
var _ = Describe("Parse time of deeply nested outlines", func() {
	It("reads 2000 nested bullets in less than 2 seconds", func() {
		// Given 2000 bullets, each one level deeper than the one before (tabs)
		var sb strings.Builder
		for i := 0; i < 2000; i++ {
			sb.WriteString(strings.Repeat("\t", i))
			sb.WriteString(fmt.Sprintf("- b%d\n", i))
		}
		input := sb.String()

		// When it is parsed
		start := time.Now()
		_, err := markdown.ParseString(input)
		elapsed := time.Since(start)

		// Then it takes less than 2 seconds
		Expect(err).ToNot(HaveOccurred())
		Expect(elapsed).To(BeNumerically("<", 2*time.Second))
	})
})
