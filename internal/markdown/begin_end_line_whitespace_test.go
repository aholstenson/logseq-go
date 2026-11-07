package markdown_test

import (
	. "github.com/onsi/ginkgo/v2"
)

// The `#+BEGIN_` and `#+END_` lines of a command can have extra spaces, as
// in "#+BEGIN_SRC   python" or an indented `#+END_NOTE`. Saving the block
// keeps these lines as they are in the file.
var _ = Describe("Parsing then outputting the whitespace of BEGIN and END lines", func() {
	// Given spaces after `#+BEGIN_SRC`, When written, Then the spaces stay.
	FullyEqual("Spaces after BEGIN_SRC", "- #+BEGIN_SRC   python   :a  b\n  print(1)\n  #+END_SRC")

	// Given an indented END line, When written, Then the indentation stays.
	FullyEqual("Indented END line", "- #+BEGIN_NOTE\n  a\n    #+END_NOTE")
})
