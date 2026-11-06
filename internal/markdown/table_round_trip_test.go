package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Logseq keeps a table as the user typed it, with all cells of each row.
// Saving a page keeps the text of its tables.
var _ = Describe("Parsing then outputting tables in blocks", func() {
	// Given a row with more cells than the header, When written, Then all
	// cells of the row are kept.
	It("keeps all cells of a row that has more cells than the header", func() {
		v := parseAndOutput("- | a | b |\n  |---|---|\n  | 1 | 2 | 3 |")
		Expect(v).To(MatchRegexp(`\|\s*1\s*\|\s*2\s*\|\s*3\s*\|`))
	})

	// Given a line of text followed by a delimiter row, When written, Then
	// the text is not made into a table header.
	FullyEqual("Text followed by a delimiter row", "- plain text\n  |---|---|")

	// Given a table without padding, When written, Then the text of the
	// table is the same.
	FullyEqual("Table in a block without padding", "- | a | b |\n  |---|---|\n  | 1 | 2 |")
	FullyEqual("Table in a block with references without padding", "- | a | b |\n  |---|---|\n  | [[x]] | #y |")
})
