package logseq_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A heading line after the first line of a block, such as `## TODO h` in
// "- x\n  ## TODO h", is a child heading block of `x`. Saving the page after
// changes of the heading keeps it a heading task.
var _ = Describe("Saving a heading line in the text of a block", func() {
	var graph *logseq.Graph

	AfterEach(func() {
		if graph != nil {
			graph.Close()
			graph = nil
		}
	})

	// headingTaskOf gets the child block of the first block of the page
	// whose first node is a heading, or nil.
	headingTaskOf := func(page logseq.Page) *content.Block {
		blocks := page.Blocks()
		if len(blocks) == 0 {
			return nil
		}
		for _, child := range blocks[0].Blocks() {
			if _, ok := child.FirstChild().(*content.Heading); ok {
				return child
			}
		}
		return nil
	}

	It("keeps a heading task after two changes of its text", func() {
		// Given a page with a heading task line under the text of a block
		dir := setupGraph()
		graph = openGraphWithPages(dir, map[string]string{
			"headings.md": "- x\n  ## TODO h\n- z\n",
		})

		// When the text of the heading is changed and the page is saved,
		// two times, with the page read again from the file each time
		for i := 0; i < 2; i++ {
			tx := graph.NewTransaction()
			page, err := tx.OpenPage("headings")
			Expect(err).ToNot(HaveOccurred())

			heading := headingTaskOf(page)
			Expect(heading).ToNot(BeNil(), "read %d", i+1)
			texts := heading.FirstChild().(*content.Heading).Children().FilterDeep(content.IsOfType[*content.Text]())
			Expect(texts).ToNot(BeEmpty())
			last := texts[len(texts)-1].(*content.Text)
			last.WithValue(last.Value + " E")
			Expect(tx.Save()).To(Succeed())

			graph.Close()
			var openErr error
			graph, openErr = logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
			Expect(openErr).ToNot(HaveOccurred())
		}

		// Then the third read has a level 2 heading task as a child block of x
		data, err := os.ReadFile(filepath.Join(dir, "pages", "headings.md"))
		Expect(err).ToNot(HaveOccurred())
		Expect(regexp.MustCompile(`(?m)^(\t|    )[ \t]*##`).MatchString(string(data))).To(BeFalse(), "saved text: %q", string(data))

		tx := graph.NewTransaction()
		page, err := tx.OpenPage("headings")
		Expect(err).ToNot(HaveOccurred())
		heading := headingTaskOf(page)
		Expect(heading).ToNot(BeNil(), "saved text: %q", string(data))
		Expect(heading.FirstChild().(*content.Heading).Level).To(Equal(2))
		Expect(heading.TaskStatus()).To(Equal(content.TaskStatusTodo))
	})
})
