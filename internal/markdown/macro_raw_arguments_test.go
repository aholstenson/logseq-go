package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A macro gives the text of its arguments as it is in the file, with the
// spaces around the commas that the split arguments do not have. Some macros
// use all of this text as one value.
var _ = Describe("The raw arguments of a macro", func() {
	macroOf := func(input string) *content.Macro {
		root, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())
		node := root.Children().FindDeep(content.IsOfType[*content.Macro]())
		Expect(node).ToNot(BeNil())
		return node.(*content.Macro)
	}

	DescribeTable("are the text after the name in the file",
		func(input string, raw string) {
			// Given a block with the macro
			// When it is read, Then its raw arguments are the text
			Expect(macroOf(input).RawArguments()).To(Equal(raw))
		},
		Entry("spaces around commas", "- {{m a , b ,c}}", "a , b ,c"),
		Entry("more spaces after the name", "- {{m   a, b  }}", "a, b"),
		Entry("quoted arguments", `- {{m "a, b" ,  c}}`, `"a, b" ,  c`),
		Entry("three curly braces", "- {{{m a , b}}}", "a , b"),
		Entry("no arguments", "- {{m}}", ""),
	)

	It("are the text that the writer writes after the arguments are changed", func() {
		// Given a macro read from a file
		macro := macroOf("- {{m a , b}}")

		// When its arguments are changed
		macro.WithArguments("x", "y, z")

		// Then its raw arguments are the text that the writer writes
		Expect(macro.RawArguments()).To(Equal(`x, "y, z"`))
		Expect(markdown.AsString(macro)).To(Equal(`{{m x, "y, z"}}`))
	})

	It("are the text that the writer writes after one argument is changed in place", func() {
		// Given a macro read from a file
		macro := macroOf("- {{m a , b}}")

		// When an argument is changed in place
		macro.Arguments[1] = "c"

		// Then its raw arguments are the text that the writer writes
		Expect(macro.RawArguments()).To(Equal("a, c"))
	})

	DescribeTable("match the text of the written macro for a new macro",
		func(arguments ...string) {
			// Given a new macro with the arguments
			macro := content.NewMacro("m", arguments...)

			// When it is written
			output, err := markdown.AsString(macro)
			Expect(err).ToNot(HaveOccurred())

			// Then the text after the name is the raw arguments
			expected := "{{m}}"
			if len(arguments) > 0 {
				expected = "{{m " + macro.RawArguments() + "}}"
			}
			Expect(output).To(Equal(expected))
		},
		Entry("no arguments"),
		Entry("plain arguments", "a", "b"),
		Entry("an argument with a comma", "a, b"),
		Entry("an argument with a comma and a quote", `a, "b"`),
		Entry("a quoted argument", `"a, b"`),
	)
})
