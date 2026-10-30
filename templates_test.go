package logseq_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// templatesPage holds the templates the specs insert.
const templatesPage = `- Daily
  template:: Daily
  id:: 6540a5f2-0000-4000-8000-000000000001
	- Plan for <% today %>
	  id:: 6540a5f2-0000-4000-8000-000000000002
	- Review <% yesterday %> and <% tomorrow %>
		- Meeting at <% time %> on <% current page %>
- Weekly
  template:: Weekly
  template-including-parent:: false
	- Ends <% next friday %>
	- Kept <% unknown %> and empty <%  %>
`

// blocksAsString writes blocks the way they would be written on a page.
func blocksAsString(blocks content.BlockList) string {
	root := content.NewBlock()
	for _, block := range blocks {
		root.AddChild(block)
	}

	text, err := logseq.AsString(root)
	Expect(err).ToNot(HaveOccurred())
	return strings.TrimSpace(text)
}

var _ = Describe("Templates", func() {
	var (
		graph *logseq.Graph
		dir   string
		ctx   context.Context

		// insertedAt is a Wednesday, so that dates before and after it are in
		// the same month.
		insertedAt time.Time
	)

	BeforeEach(func() {
		dir = setupGraph()
		ctx = context.Background()
		insertedAt = time.Date(2026, 10, 28, 15, 30, 0, 0, time.Local)

		graph = openGraphWithPages(dir, map[string]string{
			"templates.md": templatesPage,
		})
	})

	AfterEach(func() {
		graph.Close()
	})

	It("copies the template block and its children", func() {
		blocks, err := graph.Template(ctx, "Daily", logseq.WithTemplateTime(insertedAt))
		Expect(err).ToNot(HaveOccurred())
		Expect(blocks).To(HaveLen(1))
		Expect(blocksAsString(blocks)).To(Equal(strings.Join([]string{
			"- Daily",
			"\t- Plan for [[Oct 28th, 2026]]",
			"\t- Review [[Oct 27th, 2026]] and [[Oct 29th, 2026]]",
			"\t\t- Meeting at 15:30 on [[Oct 28th, 2026]]",
		}, "\n")))
	})

	It("copies only the children with template-including-parent:: false", func() {
		blocks, err := graph.Template(ctx, "Weekly", logseq.WithTemplateTime(insertedAt))
		Expect(err).ToNot(HaveOccurred())
		Expect(blocks).To(HaveLen(2))
		Expect(blocksAsString(blocks)).To(Equal(strings.Join([]string{
			"- Ends [[Nov 6th, 2026]]",
			"- Kept unknown and empty",
		}, "\n")))
	})

	It("removes the template properties and ids from the copies", func() {
		blocks, err := graph.Template(ctx, "Daily", logseq.WithTemplateTime(insertedAt))
		Expect(err).ToNot(HaveOccurred())

		Expect(blocks[0].FindProperties()).To(BeNil())
		Expect(blocks[0].Blocks()[0].FindProperties()).To(BeNil())
	})

	It("links to the page given with WithTemplatePage", func() {
		blocks, err := graph.Template(ctx, "Daily",
			logseq.WithTemplateTime(insertedAt),
			logseq.WithTemplatePage("Project"),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(blocksAsString(blocks)).To(ContainSubstring("Meeting at 15:30 on [[Project]]"))
	})

	It("links to the last day of next month when the month is shorter", func() {
		graph.Close()
		graph = openGraphWithPages(dir, map[string]string{
			"monthly.md": "- Monthly\n  template:: Monthly\n\t- Review on <% next month %>\n",
		})

		endOfJanuary := time.Date(2026, 1, 31, 9, 0, 0, 0, time.Local)
		blocks, err := graph.Template(ctx, "Monthly", logseq.WithTemplateTime(endOfJanuary))
		Expect(err).ToNot(HaveOccurred())
		Expect(blocksAsString(blocks)).To(ContainSubstring("Review on [[Feb 28th, 2026]]"))
	})

	It("matches the name without regard for case", func() {
		blocks, err := graph.Template(ctx, "daily", logseq.WithTemplateTime(insertedAt))
		Expect(err).ToNot(HaveOccurred())
		Expect(blocks).To(HaveLen(1))
	})

	It("returns the same copy each time", func() {
		first, err := graph.Template(ctx, "Daily", logseq.WithTemplateTime(insertedAt))
		Expect(err).ToNot(HaveOccurred())

		second, err := graph.Template(ctx, "Daily", logseq.WithTemplateTime(insertedAt))
		Expect(err).ToNot(HaveOccurred())

		Expect(blocksAsString(second)).To(Equal(blocksAsString(first)))
	})

	It("does not change the page of the template", func() {
		_, err := graph.Template(ctx, "Daily", logseq.WithTemplateTime(insertedAt))
		Expect(err).ToNot(HaveOccurred())

		page, err := graph.OpenPage("templates")
		Expect(err).ToNot(HaveOccurred())
		Expect(page.Blocks()).To(HaveLen(2))
		Expect(page.Blocks()[0].Blocks()).To(HaveLen(2))
	})

	It("returns ErrTemplateNotFound for an unknown name", func() {
		_, err := graph.Template(ctx, "Missing")
		Expect(err).To(MatchError(logseq.ErrTemplateNotFound))
	})

	Describe("Journal template", func() {
		// openWithJournalTemplate opens the graph again with a config that uses
		// the named template for new journals.
		openWithJournalTemplate := func(name string, opts ...logseq.Option) *logseq.Graph {
			graph.Close()

			Expect(os.WriteFile(
				filepath.Join(dir, "logseq", "config.edn"),
				[]byte(`{:default-templates {:journals "`+name+`"}}`),
				0o644,
			)).To(Succeed())

			if len(opts) == 0 {
				opts = []logseq.Option{logseq.WithInMemoryIndex()}
			}

			g, err := logseq.Open(ctx, dir, opts...)
			Expect(err).ToNot(HaveOccurred())
			return g
		}

		It("fills a new journal with the template", func() {
			graph = openWithJournalTemplate("Weekly")

			journal, err := graph.OpenJournal(insertedAt)
			Expect(err).ToNot(HaveOccurred())
			Expect(journal.IsNew()).To(BeTrue())
			Expect(blocksAsString(journal.Blocks())).To(Equal(strings.Join([]string{
				"- Ends [[Nov 6th, 2026]]",
				"- Kept unknown and empty",
			}, "\n")))
		})

		It("fills in dates relative to the day of the journal", func() {
			graph = openWithJournalTemplate("daily")

			journal, err := graph.OpenJournal(time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local))
			Expect(err).ToNot(HaveOccurred())

			text := blocksAsString(journal.Blocks())
			Expect(text).To(ContainSubstring("- Plan for [[Jan 15th, 2026]]"))
			Expect(text).To(ContainSubstring("- Review [[Jan 14th, 2026]] and [[Jan 16th, 2026]]"))
			Expect(text).To(MatchRegexp(`Meeting at \d\d:\d\d on \[\[Jan 15th, 2026\]\]`))
		})

		It("does not change a journal that exists", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", "2026_10_28.md"),
				[]byte("- Already written\n"),
				0o644,
			)).To(Succeed())

			graph = openWithJournalTemplate("Weekly")

			journal, err := graph.OpenJournal(insertedAt)
			Expect(err).ToNot(HaveOccurred())
			Expect(blocksAsString(journal.Blocks())).To(Equal("- Already written"))
		})

		It("leaves a new journal empty if the template does not exist", func() {
			graph = openWithJournalTemplate("Missing")

			journal, err := graph.OpenJournal(insertedAt)
			Expect(err).ToNot(HaveOccurred())
			Expect(journal.Blocks()).To(BeEmpty())
		})

		It("leaves a new journal empty without an index", func() {
			graph = openWithJournalTemplate("Weekly", logseq.WithRecycleDeletedPages())

			journal, err := graph.OpenJournal(insertedAt)
			Expect(err).ToNot(HaveOccurred())
			Expect(journal.Blocks()).To(BeEmpty())
		})

		It("saves the template blocks with the journal", func() {
			graph = openWithJournalTemplate("Weekly")

			tx := graph.NewTransaction()
			journal, err := tx.OpenJournal(insertedAt)
			Expect(err).ToNot(HaveOccurred())
			Expect(journal.Blocks()).To(HaveLen(2))
			Expect(tx.Save()).To(Succeed())

			data, err := os.ReadFile(filepath.Join(dir, "journals", "2026_10_28.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("- Ends [[Nov 6th, 2026]]"))
		})
	})

	It("requires an index", func() {
		withoutIndex, err := logseq.Open(ctx, dir)
		Expect(err).ToNot(HaveOccurred())
		defer withoutIndex.Close()

		_, err = withoutIndex.Template(ctx, "Daily")
		Expect(err).To(HaveOccurred())
	})
})
