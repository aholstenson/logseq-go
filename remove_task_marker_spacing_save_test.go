package logseq_test

import (
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The spaces after a task marker are kept as they are in the file. When the
// marker is removed, its spaces must be removed too, so that the next part
// of the block starts right after the bullet.
var _ = Describe("Removing a task marker with more than one space after it", func() {
	var (
		graph *logseq.Graph
		dir   string
	)

	BeforeEach(func() {
		dir = setupGraph()
	})

	AfterEach(func() {
		if graph != nil {
			graph.Close()
			graph = nil
		}
	})

	readPage := func(name string) string {
		data, err := os.ReadFile(filepath.Join(dir, "pages", name))
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	It("removes the spaces after the marker", func() {
		// Given a page with a task with two spaces after the marker and after the priority
		graph = openGraphWithPages(dir, map[string]string{
			"task.md": "- TODO  [#A]  x\n",
		})

		// When the task marker is removed and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("task")
		Expect(err).ToNot(HaveOccurred())
		page.Blocks()[0].SetTaskStatus(content.TaskStatusNone)
		Expect(tx.Save()).To(Succeed())

		// Then the priority is right after the bullet, and its own spaces are kept
		Expect(readPage("task.md")).To(Equal("- [#A]  x\n"))
	})
})
