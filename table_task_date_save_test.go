package logseq_test

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A block can start with a table. When such a block is scheduled with the
// API and the page is saved, the date must read back as a date of the block,
// and the table must stay the same.
var _ = Describe("Scheduling a block that starts with a table", func() {
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

	It("keeps the date as a date and the table as a table", func() {
		// Given a page with a block that starts with a table
		graph = openGraphWithPages(dir, map[string]string{
			"table.md": "- | a |\n  |---|\n  | 1 |\n",
		})

		// When the block is scheduled and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("table")
		Expect(err).ToNot(HaveOccurred())
		page.Blocks()[0].SetScheduled(content.NewTaskDate(
			content.TaskDateTypeScheduled,
			time.Date(2024, 1, 2, 0, 0, 0, 0, time.Local),
		))
		Expect(tx.Save()).To(Succeed())
		saved := readPage("table.md")

		// Then no line of the file is the date on the bullet line
		Expect(strings.Split(saved, "\n")).ToNot(ContainElement("- SCHEDULED: <2024-01-02 Tue>"))

		// And the block reads back with the date and the same table
		again, err := graph.NewTransaction().OpenPage("table")
		Expect(err).ToNot(HaveOccurred())
		Expect(again.Blocks()).To(HaveLen(1), "saved: %q", saved)
		block := again.Blocks()[0]

		date := block.Scheduled()
		Expect(date).ToNot(BeNil(), "saved: %q", saved)
		Expect(date.Date.Format("2006-01-02")).To(Equal("2024-01-02"))

		table := block.Children().FindDeep(content.IsOfType[*content.Table]())
		Expect(table).ToNot(BeNil(), "saved: %q", saved)
		Expect(saved).To(ContainSubstring("| a |\n  |---|\n  | 1 |\n"))
	})
})
