package logseq_test

import (
	"context"
	"os"
	"path/filepath"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Graph", func() {
	var dir string

	BeforeEach(func() {
		dir = setupGraph()
	})

	Describe("Open", func() {
		It("opens a graph with minimal config", func() {
			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(graph).ToNot(BeNil())
			Expect(graph.Close()).To(Succeed())
		})

		It("returns the directory", func() {
			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			Expect(graph.Directory()).To(Equal(dir))
		})

		It("fails when directory does not exist", func() {
			_, err := logseq.Open(context.Background(), filepath.Join(dir, "nonexistent"))
			Expect(err).To(HaveOccurred())
		})

		It("fails when config.edn is missing", func() {
			Expect(os.Remove(filepath.Join(dir, "logseq", "config.edn"))).To(Succeed())

			_, err := logseq.Open(context.Background(), dir)
			Expect(err).To(HaveOccurred())
		})

		It("invokes the listener during sync", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "listened.md"),
				[]byte("- content\n"),
				0o644,
			)).To(Succeed())

			var events []logseq.OpenEvent
			graph, err := logseq.Open(context.Background(), dir,
				logseq.WithInMemoryIndex(),
				logseq.WithListener(func(event logseq.OpenEvent) {
					events = append(events, event)
				}),
			)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			Expect(events).ToNot(BeEmpty())

			found := false
			for _, e := range events {
				if indexed, ok := e.(*logseq.PageIndexed); ok {
					if indexed.SubPath == filepath.Join("pages", "listened.md") {
						found = true
					}
				}
			}
			Expect(found).To(BeTrue(), "expected PageIndexed event for pages/listened.md")
		})
	})

	Describe("OpenPage", func() {
		It("opens an existing page", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "mypage.md"),
				[]byte("- hello from mypage\n"),
				0o644,
			)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			page, err := graph.OpenPage("mypage")
			Expect(err).ToNot(HaveOccurred())
			Expect(page).ToNot(BeNil())
			Expect(page.Title()).To(Equal("mypage"))
			Expect(page.Type()).To(Equal(logseq.PageTypeDedicated))
			Expect(page.IsNew()).To(BeFalse())
			Expect(page.Blocks()).To(HaveLen(1))
		})

		It("creates a new page when file does not exist", func() {
			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			page, err := graph.OpenPage("brandnew")
			Expect(err).ToNot(HaveOccurred())
			Expect(page).ToNot(BeNil())
			Expect(page.Title()).To(Equal("brandnew"))
			Expect(page.IsNew()).To(BeTrue())
		})

		It("opens a page with special characters in the title", func() {
			// Logseq uses ___  for / in filenames
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "parent___child.md"),
				[]byte("- nested page content\n"),
				0o644,
			)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			page, err := graph.OpenPage("parent/child")
			Expect(err).ToNot(HaveOccurred())
			Expect(page).ToNot(BeNil())
			Expect(page.Title()).To(Equal("parent/child"))
			Expect(page.IsNew()).To(BeFalse())
		})
	})

	Describe("OpenJournal", func() {
		// writeJournalFile creates a journal file using the default
		// yyyy_MM_dd format with midnight local time.
		writeJournalFile := func(dir string, year int, month time.Month, day int, body string) {
			date := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
			fname := date.Format("2006_01_02") + ".md"
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", fname),
				[]byte(body),
				0o644,
			)).To(Succeed())
		}

		It("opens an existing journal", func() {
			writeJournalFile(dir, 2025, 6, 15, "- journal entry\n")

			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			date := time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local)
			page, err := graph.OpenJournal(date)
			Expect(err).ToNot(HaveOccurred())
			Expect(page).ToNot(BeNil())
			Expect(page.Type()).To(Equal(logseq.PageTypeJournal))
			Expect(page.IsNew()).To(BeFalse())
			Expect(page.Blocks()).To(HaveLen(1))
		})

		It("creates a new journal when file does not exist", func() {
			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local)
			page, err := graph.OpenJournal(date)
			Expect(err).ToNot(HaveOccurred())
			Expect(page).ToNot(BeNil())
			Expect(page.IsNew()).To(BeTrue())
			Expect(page.Type()).To(Equal(logseq.PageTypeJournal))
		})

		It("truncates time to date", func() {
			writeJournalFile(dir, 2025, 6, 15, "- entry\n")

			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			// Open with a different hour — should still find the same journal
			dateWithTime := time.Date(2025, 6, 15, 23, 59, 0, 0, time.Local)
			page, err := graph.OpenJournal(dateWithTime)
			Expect(err).ToNot(HaveOccurred())
			Expect(page).ToNot(BeNil())
			Expect(page.IsNew()).To(BeFalse())
		})

		It("opens the correct journal at midnight local time", func() {
			writeJournalFile(dir, 2025, 6, 15, "- midnight entry\n")

			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			// Midnight local — previously broken due to Truncate(24h) using UTC
			date := time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local)
			page, err := graph.OpenJournal(date)
			Expect(err).ToNot(HaveOccurred())
			Expect(page).ToNot(BeNil())
			Expect(page.IsNew()).To(BeFalse())
			Expect(page.Blocks()).To(HaveLen(1))
		})
	})

	Describe("Close", func() {
		It("can close a graph without index", func() {
			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(graph.Close()).To(Succeed())
		})

		It("can close a graph with index", func() {
			graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())
			Expect(graph.Close()).To(Succeed())
		})
	})

	Describe("Transaction", func() {
		It("opens and saves a new page", func() {
			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("txpage")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.IsNew()).To(BeTrue())

			Expect(tx.Save()).To(Succeed())

			// Verify the file was written
			_, err = os.Stat(filepath.Join(dir, "pages", "txpage.md"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("saves modifications to an existing page", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "existing.md"),
				[]byte("- original\n"),
				0o644,
			)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("existing")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Blocks()).To(HaveLen(1))

			Expect(tx.Save()).To(Succeed())

			// File should still exist
			_, err = os.Stat(filepath.Join(dir, "pages", "existing.md"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the same page instance for duplicate opens", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "dup.md"),
				[]byte("- content\n"),
				0o644,
			)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			tx := graph.NewTransaction()
			page1, err := tx.OpenPage("dup")
			Expect(err).ToNot(HaveOccurred())

			page2, err := tx.OpenPage("dup")
			Expect(err).ToNot(HaveOccurred())

			// Should be the exact same instance
			Expect(page1).To(BeIdenticalTo(page2))
		})

		It("returns the same journal instance for duplicate opens", func() {
			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			tx := graph.NewTransaction()
			date := time.Date(2025, 3, 1, 0, 0, 0, 0, time.Local)

			j1, err := tx.OpenJournal(date)
			Expect(err).ToNot(HaveOccurred())

			j2, err := tx.OpenJournal(date)
			Expect(err).ToNot(HaveOccurred())

			Expect(j1).To(BeIdenticalTo(j2))
		})

		It("detects concurrent modification", func() {
			pagePath := filepath.Join(dir, "pages", "conflict.md")
			Expect(os.WriteFile(pagePath, []byte("- original\n"), 0o644)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			tx := graph.NewTransaction()
			_, err = tx.OpenPage("conflict")
			Expect(err).ToNot(HaveOccurred())

			// Simulate external modification
			time.Sleep(10 * time.Millisecond)
			Expect(os.WriteFile(pagePath, []byte("- modified externally\n"), 0o644)).To(Succeed())

			err = tx.Save()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("modified since"))
		})
	})

	Describe("Indexing", func() {
		It("indexes pages on open", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "indexed.md"),
				[]byte("- indexed content\n"),
				0o644,
			)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			results, err := graph.SearchPages(context.Background(),
				logseq.WithQuery(logseq.TitleMatches("indexed")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))
		})

		It("indexes journals on open", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", "2025_03_15.md"),
				[]byte("- journal indexed content\n"),
				0o644,
			)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			results, err := graph.SearchPages(context.Background(),
				logseq.WithQuery(logseq.ContentMatches("journal indexed")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))
			Expect(results.Results()[0].Type()).To(Equal(logseq.PageTypeJournal))
		})

		It("skips non-markdown files during sync", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "readme.txt"),
				[]byte("not markdown"),
				0o644,
			)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			results, err := graph.SearchPages(context.Background(),
				logseq.WithQuery(logseq.ContentMatches("not markdown")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(0))
		})

		It("returns error when searching without index", func() {
			graph, err := logseq.Open(context.Background(), dir)
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			_, err = graph.SearchPages(context.Background(),
				logseq.WithQuery(logseq.TitleMatches("anything")),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("indexing is not enabled"))
		})

		It("reflects file changes via the watcher in the index", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "initial.md"),
				[]byte("- initial content\n"),
				0o644,
			)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			// Start watching to trigger index updates
			watcher := graph.Watch()
			defer watcher.Close()

			// Add a new page
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "dynamic.md"),
				[]byte("- dynamically added uniquetoken123\n"),
				0o644,
			)).To(Succeed())

			// Wait for the watcher to process the change
			var event logseq.ChangeEvent
			Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))

			// The new page should now be searchable
			results, err := graph.SearchPages(context.Background(),
				logseq.WithQuery(logseq.ContentMatches("uniquetoken123")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))
		})

		It("removes deleted pages from the index", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "todelete.md"),
				[]byte("- deletable uniquetoken456\n"),
				0o644,
			)).To(Succeed())

			graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())
			defer graph.Close()

			// Verify it's indexed
			results, err := graph.SearchPages(context.Background(),
				logseq.WithQuery(logseq.ContentMatches("uniquetoken456")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(1))

			// Start watching
			watcher := graph.Watch()
			defer watcher.Close()

			// Delete the page
			Expect(os.Remove(filepath.Join(dir, "pages", "todelete.md"))).To(Succeed())

			// Wait for the delete event
			var event logseq.ChangeEvent
			Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
			Expect(event).To(BeAssignableToTypeOf(&logseq.PageDeleted{}))

			// The page should no longer be searchable
			results, err = graph.SearchPages(context.Background(),
				logseq.WithQuery(logseq.ContentMatches("uniquetoken456")),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(results.Size()).To(Equal(0))
		})
	})
})
