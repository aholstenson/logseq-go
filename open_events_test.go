package logseq_test

import (
	"context"
	"os"
	"path/filepath"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Open events", func() {
	var dir string

	BeforeEach(func() {
		dir = setupGraph()
	})

	writePage := func(name string, data string) {
		Expect(os.WriteFile(filepath.Join(dir, "pages", name), []byte(data), 0o644)).To(Succeed())
	}

	open := func(opts ...logseq.Option) (*logseq.Graph, []logseq.OpenEvent) {
		var events []logseq.OpenEvent
		opts = append(opts, logseq.WithListener(func(event logseq.OpenEvent) {
			events = append(events, event)
		}))

		graph, err := logseq.Open(context.Background(), dir, opts...)
		Expect(err).ToNot(HaveOccurred())
		return graph, events
	}

	find := func(events []logseq.OpenEvent, match func(logseq.OpenEvent) bool) logseq.OpenEvent {
		for _, event := range events {
			if match(event) {
				return event
			}
		}

		return nil
	}

	syncStarted := func(events []logseq.OpenEvent) *logseq.SyncStarted {
		event := find(events, func(e logseq.OpenEvent) bool { _, ok := e.(*logseq.SyncStarted); return ok })
		Expect(event).ToNot(BeNil(), "expected a SyncStarted event")
		return event.(*logseq.SyncStarted)
	}

	syncCompleted := func(events []logseq.OpenEvent) *logseq.SyncCompleted {
		event := find(events, func(e logseq.OpenEvent) bool { _, ok := e.(*logseq.SyncCompleted); return ok })
		Expect(event).ToNot(BeNil(), "expected a SyncCompleted event")
		return event.(*logseq.SyncCompleted)
	}

	It("sends the events in order", func() {
		writePage("one.md", "- one\n")
		writePage("two.md", "- two\n")

		graph, events := open(logseq.WithInMemoryIndex())
		defer graph.Close()

		Expect(events).To(HaveLen(6))
		Expect(events[0]).To(BeAssignableToTypeOf(&logseq.ScanStarted{}))
		Expect(events[1]).To(Equal(&logseq.SyncStarted{Total: 2, Unchanged: 0}))
		Expect(events[2]).To(BeAssignableToTypeOf(&logseq.PageIndexed{}))
		Expect(events[3]).To(BeAssignableToTypeOf(&logseq.PageIndexed{}))
		Expect(events[4]).To(BeAssignableToTypeOf(&logseq.SyncCompleted{}))
		Expect(events[5]).To(Equal(&logseq.OpenCompleted{}))
	})

	It("counts the indexed pages in PageIndexed", func() {
		writePage("one.md", "- one\n")
		writePage("two.md", "- two\n")
		writePage("three.md", "- three\n")

		graph, events := open(logseq.WithInMemoryIndex())
		defer graph.Close()

		var done []int
		for _, event := range events {
			if indexed, ok := event.(*logseq.PageIndexed); ok {
				Expect(indexed.Total).To(Equal(3))
				done = append(done, indexed.Done)
			}
		}
		Expect(done).To(Equal([]int{1, 2, 3}))

		completed := syncCompleted(events)
		Expect(completed.Indexed).To(Equal(3))
		Expect(completed.Removed).To(Equal(0))
		Expect(completed.Duration).To(BeNumerically(">", 0))
	})

	It("does not count files that are not part of the graph", func() {
		writePage("page.md", "- page\n")
		writePage("notes.txt", "not a page\n")
		Expect(os.WriteFile(filepath.Join(dir, "journals", "not-a-date.md"), []byte("- x\n"), 0o644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(dir, "pages", "sub"), 0o755)).To(Succeed())
		writePage(filepath.Join("sub", "nested.md"), "- nested\n")

		graph, events := open(logseq.WithInMemoryIndex())
		defer graph.Close()

		Expect(syncStarted(events)).To(Equal(&logseq.SyncStarted{Total: 1, Unchanged: 0}))
	})

	It("counts pages that are up to date as unchanged", func() {
		indexDir := GinkgoT().TempDir()
		writePage("one.md", "- one\n")
		writePage("two.md", "- two\n")

		graph, _ := open(logseq.WithIndex(indexDir))
		Expect(graph.Close()).To(Succeed())

		writePage("three.md", "- three\n")

		graph, events := open(logseq.WithIndex(indexDir))
		defer graph.Close()

		Expect(syncStarted(events)).To(Equal(&logseq.SyncStarted{Total: 1, Unchanged: 2}))
		Expect(syncCompleted(events).Indexed).To(Equal(1))
	})

	It("removes pages whose files were deleted while the graph was closed", func() {
		indexDir := GinkgoT().TempDir()
		writePage("kept.md", "- kept page\n")
		writePage("deleted.md", "- deleted page\n")

		graph, _ := open(logseq.WithIndex(indexDir))
		Expect(graph.Close()).To(Succeed())

		Expect(os.Remove(filepath.Join(dir, "pages", "deleted.md"))).To(Succeed())

		graph, events := open(logseq.WithIndex(indexDir))
		defer graph.Close()

		Expect(syncCompleted(events).Removed).To(Equal(1))

		results, err := graph.SearchPages(context.Background(),
			logseq.WithQuery(logseq.ContentMatches("deleted")),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(0))

		blocks, err := graph.SearchBlocks(context.Background(),
			logseq.WithQuery(logseq.ContentMatches("deleted")),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(blocks.Size()).To(Equal(0))

		results, err = graph.SearchPages(context.Background(),
			logseq.WithQuery(logseq.ContentMatches("kept")),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(1))
	})

	It("reports a page that can not be read and continues to open", func() {
		writePage("one.md", "- one\n")
		Expect(os.WriteFile(filepath.Join(dir, "pages", "two.md"), []byte("- two\n"), 0o000)).To(Succeed())
		writePage("three.md", "- three\n")

		graph, events := open(logseq.WithInMemoryIndex())
		defer graph.Close()

		event := find(events, func(e logseq.OpenEvent) bool { _, ok := e.(*logseq.PageIndexFailed); return ok })
		Expect(event).ToNot(BeNil(), "expected a PageIndexFailed event")

		failed := event.(*logseq.PageIndexFailed)
		Expect(failed.SubPath).To(Equal(filepath.Join("pages", "two.md")))
		Expect(failed.Total).To(Equal(3))
		Expect(failed).To(MatchError(os.ErrPermission))

		var done []int
		for _, e := range events {
			switch e := e.(type) {
			case *logseq.PageIndexed:
				done = append(done, e.Done)
			case *logseq.PageIndexFailed:
				done = append(done, e.Done)
			}
		}
		Expect(done).To(Equal([]int{1, 2, 3}))

		completed := syncCompleted(events)
		Expect(completed.Indexed).To(Equal(2))
		Expect(completed.Failed).To(Equal(1))
		Expect(events[len(events)-1]).To(Equal(&logseq.OpenCompleted{}))

		results, err := graph.SearchPages(context.Background(),
			logseq.WithQuery(logseq.ContentMatches("three")),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(1))
	})

	It("tries a page that failed again the next time the graph is opened", func() {
		indexDir := GinkgoT().TempDir()
		path := filepath.Join(dir, "pages", "later.md")
		Expect(os.WriteFile(path, []byte("- readable later\n"), 0o000)).To(Succeed())

		graph, events := open(logseq.WithIndex(indexDir))
		Expect(syncCompleted(events).Failed).To(Equal(1))
		Expect(graph.Close()).To(Succeed())

		Expect(os.Chmod(path, 0o644)).To(Succeed())

		graph, events = open(logseq.WithIndex(indexDir))
		defer graph.Close()

		Expect(syncStarted(events).Total).To(Equal(1))
		Expect(syncCompleted(events).Indexed).To(Equal(1))
	})

	It("sends only OpenCompleted without an index", func() {
		writePage("page.md", "- page\n")

		graph, events := open()
		defer graph.Close()

		Expect(events).To(Equal([]logseq.OpenEvent{&logseq.OpenCompleted{}}))
	})

	It("does not send OpenCompleted when opening fails", func() {
		writePage("page.md", "- page\n")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var events []logseq.OpenEvent
		_, err := logseq.Open(ctx, dir,
			logseq.WithInMemoryIndex(),
			logseq.WithListener(func(event logseq.OpenEvent) {
				events = append(events, event)
			}),
		)
		Expect(err).To(MatchError(context.Canceled))

		Expect(find(events, func(e logseq.OpenEvent) bool { _, ok := e.(*logseq.OpenCompleted); return ok })).To(BeNil())
	})
})
