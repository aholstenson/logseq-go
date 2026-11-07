package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// taskDateLinePageText reads a page and writes it as a page is saved: the
// writer output with a line break at the end.
func taskDateLinePageText(input string) string {
	root, err := markdown.ParseString(input, markdown.AsPage())
	Expect(err).ToNot(HaveOccurred())
	v, err := markdown.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	if v != "" && !strings.HasSuffix(v, "\n") {
		v += "\n"
	}
	return v
}

// A SCHEDULED or DEADLINE line that Logseq reads as text, such as one in the
// text before the first block of a page, can be indented. Saving keeps its
// indentation.
var _ = Describe("Parsing then outputting indented SCHEDULED lines that are text", func() {
	// Given a page that starts with a byte order mark and a task with a
	// SCHEDULED line, When written, Then the SCHEDULED line keeps its
	// indentation.
	It("keeps the indentation of a SCHEDULED line after a byte order mark in a page", func() {
		input := "\ufeff- DONE task [[A]]\n  SCHEDULED: <2024-03-04 Mon>\n\t- child\n"
		Expect(taskDateLinePageText(input)).To(Equal(input))
	})

	// Given a page with an indented SCHEDULED line in the page text, When
	// written, Then the line keeps its indentation.
	It("keeps the indentation of a SCHEDULED line in the page text", func() {
		input := "text\n   SCHEDULED: <2024-03-04 Mon>\n"
		Expect(taskDateLinePageText(input)).To(Equal(input))
	})

	// Given a SCHEDULED line after a blank line in a block that is
	// indented one more space, When written, Then the line keeps its
	// indentation.
	FullyEqual("SCHEDULED line after a blank line in a block", "- a\n  b\n  c\n\n   SCHEDULED: <2024-03-04 Mon>")
})
