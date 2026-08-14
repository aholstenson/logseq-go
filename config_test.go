package logseq_test

import (
	"context"
	"os"
	"path/filepath"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var dir string

	BeforeEach(func() {
		dir = setupGraph()
	})

	// withConfig replaces the config of the graph and opens it.
	withConfig := func(config string, opts ...logseq.Option) (*logseq.Graph, error) {
		Expect(os.WriteFile(
			filepath.Join(dir, "logseq", "config.edn"),
			[]byte(config),
			0o644,
		)).To(Succeed())

		return logseq.Open(context.Background(), dir, opts...)
	}

	openWithConfig := func(config string, opts ...logseq.Option) *logseq.Graph {
		graph, err := withConfig(config, opts...)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(graph.Close)
		return graph
	}

	Describe("Preferred format", func() {
		It("opens a Markdown graph", func() {
			graph := openWithConfig(`{:preferred-format "Markdown"}`)
			Expect(graph).ToNot(BeNil())
		})

		It("refuses to open an Org graph", func() {
			_, err := withConfig(`{:preferred-format :org}`)
			Expect(err).To(MatchError(ContainSubstring("only Markdown graphs are supported")))
		})
	})

	Describe("Journal formats", func() {
		It("uses the format of the journal file name", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", "2025-06-15.md"),
				[]byte("- journal entry\n"),
				0o644,
			)).To(Succeed())

			graph := openWithConfig(`{:journal/file-name-format "yyyy-MM-dd"}`)

			page, err := graph.OpenJournal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local))
			Expect(err).ToNot(HaveOccurred())
			Expect(page.IsNew()).To(BeFalse())
			Expect(page.Blocks()).To(HaveLen(1))
		})

		It("uses the format of the journal page title", func() {
			graph := openWithConfig(`{:journal/page-title-format "yyyy-MM-dd"}`)

			page, err := graph.OpenJournal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local))
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Title()).To(Equal("2025-06-15"))
		})

		It("uses the legacy date formatter for the journal page title", func() {
			graph := openWithConfig(`{:date-formatter "yyyy-MM-dd"}`)

			page, err := graph.OpenJournal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local))
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Title()).To(Equal("2025-06-15"))
		})

		It("writes an ordinal day in the journal page title", func() {
			graph := openWithConfig(`{}`)

			page, err := graph.OpenJournal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local))
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Title()).To(Equal("Jun 15th, 2025"))
		})

		It("reads a journal file name with an ordinal day", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", "2025_06_15th.md"),
				[]byte("- journal entry\n"),
				0o644,
			)).To(Succeed())

			graph := openWithConfig(`{:journal/file-name-format "yyyy_MM_do"}`)

			page, err := graph.OpenJournal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local))
			Expect(err).ToNot(HaveOccurred())
			Expect(page.IsNew()).To(BeFalse())
			Expect(page.Blocks()).To(HaveLen(1))
		})
	})

	Describe("Hidden files", func() {
		It("leaves hidden pages out of the index", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "visible.md"),
				[]byte("- visible\n"),
				0o644,
			)).To(Succeed())
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "secret.md"),
				[]byte("- secret\n"),
				0o644,
			)).To(Succeed())

			var indexed []string
			openWithConfig(`{:hidden ["/pages/secret.md"]}`,
				logseq.WithInMemoryIndex(),
				logseq.WithListener(func(event logseq.OpenEvent) {
					if page, ok := event.(*logseq.PageIndexed); ok {
						indexed = append(indexed, page.SubPath)
					}
				}),
			)

			Expect(indexed).To(ContainElement(filepath.Join("pages", "visible.md")))
			Expect(indexed).ToNot(ContainElement(filepath.Join("pages", "secret.md")))
		})

		It("does not report changes to hidden pages", func() {
			graph := openWithConfig(`{:hidden ["/pages/secret.md"]}`, logseq.WithInMemoryIndex())

			watcher := graph.Watch()
			DeferCleanup(watcher.Close)

			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "secret.md"),
				[]byte("- secret\n"),
				0o644,
			)).To(Succeed())
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "visible.md"),
				[]byte("- visible\n"),
				0o644,
			)).To(Succeed())

			// The visible page is the only change that should come through, so
			// receiving it means the hidden one was skipped rather than late.
			var event logseq.ChangeEvent
			Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
			Expect(event).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))
			Expect(event.(*logseq.PageUpdated).Page.Title()).To(Equal("visible"))
			Consistently(watcher.Events(), 2*time.Second).ShouldNot(Receive())
		})

		It("skips hidden directories when indexing", func() {
			Expect(os.MkdirAll(filepath.Join(dir, "pages", "archived"), 0o755)).To(Succeed())
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", "archived", "old.md"),
				[]byte("- archived\n"),
				0o644,
			)).To(Succeed())

			graph, err := withConfig(`{:hidden ["/pages/archived"]}`, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())
			Expect(graph.Close()).To(Succeed())
		})

		It("reports whether a path is hidden", func() {
			graph := openWithConfig(`{:hidden ["/pages/archived"]}`)

			Expect(graph.IsHidden(filepath.Join("pages", "archived", "old.md"))).To(BeTrue())
			Expect(graph.IsHidden(filepath.Join("pages", "visible.md"))).To(BeFalse())
		})
	})

	Describe("Workflow", func() {
		It("defaults to LATER/NOW", func() {
			graph := openWithConfig(`{}`)

			Expect(graph.Workflow()).To(Equal(logseq.WorkflowNow))
			Expect(graph.Workflow().Todo()).To(Equal(content.TaskStatusLater))
			Expect(graph.Workflow().Doing()).To(Equal(content.TaskStatusNow))
		})

		It("uses TODO/DOING when the graph prefers it", func() {
			graph := openWithConfig(`{:preferred-workflow :todo}`)

			Expect(graph.Workflow()).To(Equal(logseq.WorkflowTodo))
			Expect(graph.Workflow().Todo()).To(Equal(content.TaskStatusTodo))
			Expect(graph.Workflow().Doing()).To(Equal(content.TaskStatusDoing))
		})
	})

	Describe("Favorites", func() {
		It("returns the favorited pages", func() {
			graph := openWithConfig(`{:favorites ["Interstellar" "some page"]}`)

			Expect(graph.Favorites()).To(Equal([]string{"Interstellar", "some page"}))
		})

		It("returns an empty list when there are no favorites", func() {
			graph := openWithConfig(`{}`)

			Expect(graph.Favorites()).To(BeEmpty())
		})
	})

	Describe("LogbookSettings", func() {
		It("returns the defaults of Logseq", func() {
			graph := openWithConfig(`{}`)

			Expect(graph.LogbookSettings()).To(Equal(logseq.LogbookSettings{
				WithSeconds:                true,
				EnabledInAllBlocks:         false,
				EnabledInTimestampedBlocks: true,
			}))
		})

		It("returns the settings of the graph", func() {
			graph := openWithConfig(`{:logbook/settings {:with-second-support? false :enabled-in-all-blocks true}}`)

			Expect(graph.LogbookSettings()).To(Equal(logseq.LogbookSettings{
				WithSeconds:                false,
				EnabledInAllBlocks:         true,
				EnabledInTimestampedBlocks: true,
			}))
		})

		// logbookBlock is a task with a two minute clock entry on it.
		logbookBlock := func() *content.Block {
			return content.NewBlock(
				content.NewParagraph(
					content.NewTaskMarker(content.TaskStatusTodo),
					content.NewText("Task"),
				),
				content.NewLogbook(content.NewLogbookEntryClock(
					time.Date(2023, time.June, 26, 17, 25, 56, 0, time.Local),
					time.Date(2023, time.June, 26, 17, 27, 58, 0, time.Local),
				)).WithPreviousLineType(content.PreviousLineTypeNonBlank),
			)
		}

		It("writes logbooks with seconds by default", func() {
			graph := openWithConfig(`{}`)

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("clocked")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(logbookBlock())
			Expect(tx.Save()).To(Succeed())

			data, err := os.ReadFile(filepath.Join(dir, "pages", "clocked.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(
				"CLOCK: [2023-06-26 Mon 17:25:56]--[2023-06-26 Mon 17:27:58] =>  00:02:02",
			))
		})

		It("writes logbooks without seconds when the graph is set up that way", func() {
			graph := openWithConfig(`{:logbook/settings {:with-second-support? false}}`)

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("clocked")
			Expect(err).ToNot(HaveOccurred())
			page.AddBlock(logbookBlock())
			Expect(tx.Save()).To(Succeed())

			data, err := os.ReadFile(filepath.Join(dir, "pages", "clocked.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(
				"CLOCK: [2023-06-26 Mon 17:25]--[2023-06-26 Mon 17:27] =>  00:02",
			))
		})

		It("converts a node to Markdown with the settings of the graph", func() {
			graph := openWithConfig(`{:logbook/settings {:with-second-support? false}}`)

			Expect(graph.AsString(logbookBlock())).To(Equal(
				"TODO Task\n:LOGBOOK:\nCLOCK: [2023-06-26 Mon 17:25]--[2023-06-26 Mon 17:27] =>  00:02\n:END:",
			))
		})
	})

	Describe("DefaultJournalQueries", func() {
		It("returns no queries when the graph has none", func() {
			graph := openWithConfig(`{}`)

			Expect(graph.DefaultJournalQueries()).To(BeEmpty())
		})

		It("returns the queries of the journal page", func() {
			graph := openWithConfig(`{:default-queries
			  {:journals
			   [{:title "🔨 NOW"
			     :query [:find (pull ?h [*]) :where [?h :block/marker ?marker]]
			     :inputs [:14d :today]
			     :group-by-page? false
			     :collapsed? true}]}}`)

			queries := graph.DefaultJournalQueries()
			Expect(queries).To(HaveLen(1))
			Expect(queries[0].Title).To(Equal("🔨 NOW"))
			Expect(queries[0].Query).To(Equal("[:find (pull ?h [*]) :where [?h :block/marker ?marker]]"))
			Expect(queries[0].Inputs).To(Equal("[:14d :today]"))
			Expect(queries[0].GroupByPage).ToNot(BeNil())
			Expect(*queries[0].GroupByPage).To(BeFalse())
			Expect(queries[0].Collapsed).To(BeTrue())
		})
	})
})
