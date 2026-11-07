package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// A line of code can have less indentation than its fence. The code then
// starts at that column. Saving the block keeps the columns of the code
// lines relative to each other, so that indented code (such as Python or
// YAML) does not change.
var _ = Describe("Parsing then outputting code lines with less indentation than their fence", func() {
	// Given a code line with less indentation than the fence, When
	// written, Then the lines keep their columns.
	FullyEqual("Code line with less indentation than the fence", "- a\n    ```\n    x\n  y\n    ```")
})
