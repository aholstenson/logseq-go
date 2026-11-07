package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A bullet can have more than one space after the dash (`-   x`). Saving a
// page after a change of a code block under such a bullet keeps the code
// block.
var _ = Describe("Saving a code block under a bullet with more than one space", func() {
	var graph *logseq.Graph

	AfterEach(func() {
		if graph != nil {
			graph.Close()
			graph = nil
		}
	})

	It("keeps the code block when its code is changed", func() {
		// Given a page with a code fence under a bullet with three spaces
		dir := setupGraph()
		graph = openGraphWithPages(dir, map[string]string{
			"code.md": "-   ```\n    x\n    ```\n- other\n",
		})

		// When the code is changed to two lines and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("code")
		Expect(err).ToNot(HaveOccurred())

		code := page.Blocks()[0].Children().FindDeep(content.IsOfType[*content.CodeBlock]())
		Expect(code).ToNot(BeNil())
		code.(*content.CodeBlock).Code = "x\ny\n"
		Expect(tx.Save()).To(Succeed())

		// Then the saved page reads back with one code block with the new code
		data, err := os.ReadFile(filepath.Join(dir, "pages", "code.md"))
		Expect(err).ToNot(HaveOccurred())

		block, err := logseq.ParseBlock(string(data))
		Expect(err).ToNot(HaveOccurred())

		blocks := block.Children().FilterDeep(content.IsOfType[*content.CodeBlock]())
		Expect(blocks).To(HaveLen(1), "saved text: %q", string(data))
		Expect(blocks[0].(*content.CodeBlock).Code).To(Equal("x\ny\n"), "saved text: %q", string(data))
	})
})
