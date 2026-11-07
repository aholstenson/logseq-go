package markdown_test

import (
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A #+BEGIN_ line can be inside an unclosed drawer, and its #+END_ line can be
// after the end of the drawer. Reading such text must not stop the program.
var _ = Describe("Parsing a begin block that starts inside a logbook", func() {
	// Given a #+BEGIN_QUOTE line between :LOGBOOK: and :END:, and the
	// #+END_QUOTE line after :END:, When parsed, Then parsing does not panic.
	It("does not panic", func() {
		input := "        - TODO b\n \t:LOGBOOK:\n   #+BEGIN_QUOTE\n        :END:\n \t#+END_QUOTE\n"
		Expect(func() {
			_, _ = markdown.ParseString(input)
		}).ToNot(Panic())
	})
})
