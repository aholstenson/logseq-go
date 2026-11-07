package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// The lines of a code fence can have more indentation than the text of the
// block, and the closing fence can be longer than the opening fence.
// Saving the block keeps the fence lines as they are in the file.
var _ = Describe("Parsing then outputting the whitespace of code fence lines", func() {
	// Given a closing fence that is longer than the opening fence, When
	// written, Then the closing fence keeps its length.
	FullyEqual("Closing fence longer than the opening fence", "- ```\n  x\n  ````")

	// Given a fence that is indented one column more than the block text,
	// When written, Then the fence lines keep their indentation.
	FullyEqual("Fence indented one column more than the block text", "- a\n   ```js\n   x\n   ```")
})
