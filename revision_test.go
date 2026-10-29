package logseq_test

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Revisions", func() {
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

	pagePath := func(name string) string {
		return filepath.Join(dir, "pages", name)
	}

	readPage := func(name string) string {
		data, err := os.ReadFile(pagePath(name))
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	writePage := func(name string, data string) {
		Expect(os.WriteFile(pagePath(name), []byte(data), 0o644)).To(Succeed())
	}

	Describe("Page.Revision", func() {
		It("is empty for a page without a file", func() {
			graph = openGraphWithPages(dir, map[string]string{})

			page, err := graph.OpenPage("missing")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Revision()).To(BeEmpty())
		})

		It("is the same when the content is the same", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			first, err := graph.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			second, err := graph.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())

			Expect(first.Revision()).ToNot(BeEmpty())
			Expect(second.Revision()).To(Equal(first.Revision()))
		})

		It("changes when the content changes, even with the same modification time", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			before, err := graph.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())

			info, err := os.Stat(pagePath("page.md"))
			Expect(err).ToNot(HaveOccurred())

			writePage("page.md", "- changed\n")
			Expect(os.Chtimes(pagePath("page.md"), info.ModTime(), info.ModTime())).To(Succeed())

			after, err := graph.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())

			Expect(after.LastModified()).To(Equal(before.LastModified()))
			Expect(after.Revision()).ToNot(Equal(before.Revision()))
		})
	})

	Describe("Transaction.Save", func() {
		It("fails with ErrConflict when the page changed after it was opened", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("from transaction"))

			info, err := os.Stat(pagePath("page.md"))
			Expect(err).ToNot(HaveOccurred())

			// Keep the modification time, as two quick writes can share it.
			writePage("page.md", "- outside\n")
			Expect(os.Chtimes(pagePath("page.md"), info.ModTime(), info.ModTime())).To(Succeed())

			err = tx.Save()
			Expect(errors.Is(err, logseq.ErrConflict)).To(BeTrue(), "error was %v", err)
			Expect(readPage("page.md")).To(Equal("- outside\n"))
		})

		It("fails with ErrConflict when the page was removed after it was opened", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("from transaction"))

			Expect(os.Remove(pagePath("page.md"))).To(Succeed())

			err = tx.Save()
			Expect(errors.Is(err, logseq.ErrConflict)).To(BeTrue(), "error was %v", err)
			Expect(pagePath("page.md")).ToNot(BeAnExistingFile())
		})

		It("fails with ErrConflict when a new page was created after it was opened", func() {
			graph = openGraphWithPages(dir, map[string]string{})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("from transaction"))

			writePage("page.md", "- outside\n")

			err = tx.Save()
			Expect(errors.Is(err, logseq.ErrConflict)).To(BeTrue(), "error was %v", err)
			Expect(readPage("page.md")).To(Equal("- outside\n"))
		})

		It("writes nothing when one of the pages has a conflict", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"a.md": "- a\n",
				"b.md": "- b\n",
			})

			tx := graph.NewTransaction()
			a, err := tx.OpenPage("a")
			Expect(err).ToNot(HaveOccurred())
			a.AddBlock(textBlock("added"))
			b, err := tx.OpenPage("b")
			Expect(err).ToNot(HaveOccurred())
			b.AddBlock(textBlock("added"))

			writePage("b.md", "- outside\n")

			Expect(errors.Is(tx.Save(), logseq.ErrConflict)).To(BeTrue())
			Expect(readPage("a.md")).To(Equal("- a\n"))
		})

		It("can save the same transaction more than once", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())

			page.AddBlock(textBlock("first"))
			Expect(tx.Save()).To(Succeed())

			page.AddBlock(textBlock("second"))
			Expect(tx.Save()).To(Succeed())

			Expect(readPage("page.md")).To(Equal("- content\n- first\n- second\n"))
		})

		It("updates the revision of the page to the content that was written", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("added"))
			Expect(tx.Save()).To(Succeed())

			reopened, err := graph.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Revision()).To(Equal(reopened.Revision()))
			Expect(page.IsNew()).To(BeFalse())
		})
	})

	Describe("WithExpectedRevision", func() {
		It("saves when the page is at the expected revision", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			editor, err := graph.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("added"))

			Expect(tx.Save(logseq.WithExpectedRevision(page, editor.Revision()))).To(Succeed())
			Expect(readPage("page.md")).To(Equal("- content\n- added\n"))
		})

		It("fails with ErrConflict when the page changed after the expected revision", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			// The editor read the page before the change from outside, while the
			// transaction is opened after it.
			editor, err := graph.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())

			writePage("page.md", "- outside\n")

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("added"))

			err = tx.Save(logseq.WithExpectedRevision(page, editor.Revision()))
			Expect(errors.Is(err, logseq.ErrConflict)).To(BeTrue(), "error was %v", err)
			Expect(readPage("page.md")).To(Equal("- outside\n"))
		})

		It("checks pages that are not part of the transaction", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md":  "- content\n",
				"other.md": "- other\n",
			})

			other, err := graph.OpenPage("other")
			Expect(err).ToNot(HaveOccurred())

			writePage("other.md", "- outside\n")

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("added"))

			err = tx.Save(logseq.WithExpectedRevision(other, other.Revision()))
			Expect(errors.Is(err, logseq.ErrConflict)).To(BeTrue(), "error was %v", err)
			Expect(readPage("page.md")).To(Equal("- content\n"))
			Expect(readPage("other.md")).To(Equal("- outside\n"))
		})

		It("expects no file for the empty revision", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"page.md": "- content\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("page")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(textBlock("added"))

			err = tx.Save(logseq.WithExpectedRevision(page, ""))
			Expect(errors.Is(err, logseq.ErrConflict)).To(BeTrue(), "error was %v", err)
		})
	})
})

var _ = Describe("Own writes", func() {
	var (
		graph   *logseq.Graph
		watcher *logseq.Watcher
		dir     string
	)

	BeforeEach(func() {
		dir = setupGraph()
		graph = openGraphWithPages(dir, map[string]string{
			"test.md": "- hello world\n",
		})
		watcher = graph.Watch()
	})

	AfterEach(func() {
		watcher.Close()
		graph.Close()
	})

	receive := func() logseq.ChangeEvent {
		var event logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
		return event
	}

	It("marks a page saved by the graph as its own write", func() {
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("test")
		Expect(err).ToNot(HaveOccurred())
		page.AddBlock(textBlock("added"))
		Expect(tx.Save()).To(Succeed())

		event := receive()
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))

		updated := event.(*logseq.PageUpdated)
		Expect(updated.OwnWrite).To(BeTrue())
		Expect(updated.Revision).To(Equal(page.Revision()))
		Expect(updated.Page.Revision()).To(Equal(updated.Revision))
	})

	It("does not mark a change made by another program as an own write", func() {
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "test.md"),
			[]byte("- outside\n"),
			0o644,
		)).To(Succeed())

		event := receive()
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))

		updated := event.(*logseq.PageUpdated)
		Expect(updated.OwnWrite).To(BeFalse())
		Expect(updated.Revision).ToNot(BeEmpty())
	})

	It("does not mark a change made over an own write as an own write", func() {
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("test")
		Expect(err).ToNot(HaveOccurred())
		page.AddBlock(textBlock("added"))
		Expect(tx.Save()).To(Succeed())

		// The change comes before the debounced event for the save, so a
		// single event covers both writes.
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "test.md"),
			[]byte("- outside\n"),
			0o644,
		)).To(Succeed())

		event := receive()
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))
		Expect(event.(*logseq.PageUpdated).OwnWrite).To(BeFalse())
	})

	It("marks a page deleted by the graph as its own write", func() {
		tx := graph.NewTransaction()
		Expect(tx.DeletePage("test")).To(Succeed())
		Expect(tx.Save()).To(Succeed())

		event := receive()
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageDeleted{}))
		Expect(event.(*logseq.PageDeleted).OwnWrite).To(BeTrue())
	})

	It("does not mark a page deleted by another program as an own write", func() {
		Expect(os.Remove(filepath.Join(dir, "pages", "test.md"))).To(Succeed())

		event := receive()
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageDeleted{}))
		Expect(event.(*logseq.PageDeleted).OwnWrite).To(BeFalse())
	})
})
