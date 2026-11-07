package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// Some editors put a UTF-8 byte order mark (BOM) at the start of a file.
// Logseq reads the BOM as text, so the first line is page text and not a
// bullet. Saving such a page keeps its text and indentation.
var _ = Describe("Parsing then outputting a page that starts with a byte order mark", func() {
	// Given a page that starts with a BOM and a task with a SCHEDULED line,
	// When written, Then the SCHEDULED line keeps its indentation.
	FullyEqual("Task with a SCHEDULED line after a byte order mark",
		"\ufeff- DONE task [[A]]\n  SCHEDULED: <2024-03-04 Mon>\n\t- child")
})
