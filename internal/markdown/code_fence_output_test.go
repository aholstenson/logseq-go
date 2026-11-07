package markdown_test

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A saved fenced code block must keep the full info line and the fence
// characters that the user wrote.
var _ = Describe("Code fences in saved blocks", func() {
	// firstCodeBlock parses the input and returns its first code block.
	firstCodeBlock := func(input string) *content.CodeBlock {
		block, err := markdown.ParseString(input)
		Expect(err).ToNot(HaveOccurred())

		code, ok := block.Children().FindDeep(content.IsOfType[*content.CodeBlock]()).(*content.CodeBlock)
		Expect(ok).To(BeTrue())
		return code
	}

	// writeCodeBlock writes a page with the code block as its only content.
	writeCodeBlock := func(code *content.CodeBlock) string {
		output, err := markdown.AsString(content.NewBlock(code))
		Expect(err).ToNot(HaveOccurred())
		return output
	}

	It("keeps the options after the language of a code fence", func() {
		// Given a code fence with options after the language
		input := "- c\n  ```clojure {:title \"x\"}\n  x\n  ```"

		// When it is parsed and written
		// Then the file text is unchanged
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	It("keeps a tilde code fence as tildes", func() {
		// Given a code fence made with `~~~`
		input := "- tilde\n  ~~~js\n  a\n  ~~~"

		// When it is parsed and written
		// Then it still uses `~~~`
		Expect(parseAndOutput(input)).To(Equal(input))
	})

	DescribeTable("keeps the fence line as written",
		func(input string) {
			Expect(parseAndOutput(input)).To(Equal(input))
		},
		Entry("with more spaces before the options and at the end", "- c\n  ```clojure  {:title \"x\"}  \n  x\n  ```"),
		Entry("with a space before the language", "- c\n  ``` js\n  x\n  ```"),
		Entry("with four backticks", "- c\n  ````js\n  x\n  ````"),
		Entry("with four tildes around a tilde fence in the code", "- c\n  ~~~~md\n  ~~~\n  x\n  ~~~\n  ~~~~"),
		Entry("with tildes and options", "- c\n  ~~~clojure {:title \"x\"}\n  x\n  ~~~"),
		Entry("with tildes and Windows line endings", "- c\r\n  ~~~js\r\n  x\r\n  ~~~"),
	)

	It("reads the language, the options and the fence of a code block", func() {
		// Given a tilde code fence with options after the language
		// When it is parsed
		code := firstCodeBlock("- c\n  ~~~clojure  {:title \"x\"} {:a 1}\n  x\n  ~~~")

		// Then the language, the options and the fence are on the code block
		Expect(code.Language).To(Equal("clojure"))
		Expect(code.Options).To(Equal(`{:title "x"} {:a 1}`))
		Expect(code.Fence).To(Equal("~~~"))
	})

	It("reads no fence for a fence of three backticks", func() {
		// Given a code fence of three backticks
		// When it is parsed
		code := firstCodeBlock("- c\n  ```js\n  x\n  ```")

		// Then the fence is empty, which means three backticks
		Expect(code.Fence).To(BeEmpty())
		Expect(code.Options).To(BeEmpty())
	})

	It("writes a changed language with the options of the fence line", func() {
		// Given a code fence with options, read from a file
		code := firstCodeBlock("- c\n  ```clojure  {:title \"x\"}\n  x\n  ```")

		// When its language is changed and it is written
		code.Language = "edn"

		// Then the new language is written with the options
		Expect(writeCodeBlock(code)).To(Equal("```edn {:title \"x\"}\nx\n```"))
	})

	It("writes a code block made with the API with its fence and options", func() {
		// Given a code block with tildes, a language and options
		code := content.NewCodeBlock("x\n").WithLanguage("js").WithOptions("{:a 1}").WithFence("~~~")

		// When it is written
		// Then it has the fence, the language and the options
		Expect(writeCodeBlock(code)).To(Equal("~~~js {:a 1}\nx\n~~~"))
	})

	It("does not write options without a language", func() {
		// Given a code block made with the API with options and no language
		code := content.NewCodeBlock("x\n").WithOptions("{:a 1}")

		// When it is written
		// Then the options are left out, as they would be read as the language
		Expect(writeCodeBlock(code)).To(Equal("```\nx\n```"))
	})

	It("writes a fence that is not a code fence as three backticks", func() {
		// Given a code block made with the API with two tildes as its fence
		code := content.NewCodeBlock("x\n").WithFence("~~")

		// When it is written
		// Then it uses three backticks
		Expect(writeCodeBlock(code)).To(Equal("```\nx\n```"))
	})

	It("writes a longer tilde fence when a line of the code would close it", func() {
		// Given a tilde code block whose code has a line of three tildes
		code := content.NewCodeBlock("~~~\n").WithFence("~~~")

		// When it is written and read again
		output := writeCodeBlock(code)

		// Then the fence is longer and the code is the same
		Expect(output).To(Equal("~~~~\n~~~\n~~~~"))
		Expect(firstCodeBlock(output).Code).To(Equal("~~~\n"))
	})
})
