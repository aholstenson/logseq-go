package logseq_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// setupGraph creates a temporary Logseq graph directory with the minimal
// structure needed for Open to succeed.
func setupGraph() string {
	dir := GinkgoT().TempDir()

	// Create the required directory structure
	Expect(os.MkdirAll(filepath.Join(dir, "logseq"), 0o755)).To(Succeed())
	Expect(os.MkdirAll(filepath.Join(dir, "pages"), 0o755)).To(Succeed())
	Expect(os.MkdirAll(filepath.Join(dir, "journals"), 0o755)).To(Succeed())

	// Write a minimal config.edn
	Expect(os.WriteFile(
		filepath.Join(dir, "logseq", "config.edn"),
		[]byte("{}"),
		0o644,
	)).To(Succeed())

	return dir
}

var _ = Describe("Watcher", func() {
	var (
		graph   *logseq.Graph
		watcher *logseq.Watcher
		dir     string
	)

	BeforeEach(func() {
		dir = setupGraph()

		// Create an initial page so the graph has something to index
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "test.md"),
			[]byte("- hello world\n"),
			0o644,
		)).To(Succeed())
	})

	JustBeforeEach(func() {
		var err error
		graph, err = logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
		Expect(err).ToNot(HaveOccurred())

		watcher = graph.Watch()
	})

	AfterEach(func() {
		if watcher != nil {
			watcher.Close()
		}
	})

	It("emits PageUpdated when a page file is modified", func() {
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "test.md"),
			[]byte("- updated content\n"),
			0o644,
		)).To(Succeed())

		var event logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))

		updated := event.(*logseq.PageUpdated)
		Expect(updated.Page).ToNot(BeNil())
		Expect(updated.Page.Title()).To(Equal("test"))
		Expect(updated.Page.Type()).To(Equal(logseq.PageTypeDedicated))
	})

	It("emits PageDeleted when a page file is removed", func() {
		Expect(os.Remove(filepath.Join(dir, "pages", "test.md"))).To(Succeed())

		var event logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageDeleted{}))

		deleted := event.(*logseq.PageDeleted)
		Expect(deleted.Title).To(Equal("test"))
		Expect(deleted.Type).To(Equal(logseq.PageTypeDedicated))
	})

	It("emits PageUpdated when a new page file is created", func() {
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "new___page.md"),
			[]byte("- brand new page\n"),
			0o644,
		)).To(Succeed())

		var event logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))

		updated := event.(*logseq.PageUpdated)
		Expect(updated.Page).ToNot(BeNil())
		Expect(updated.Page.Title()).To(Equal("new/page"))
		Expect(updated.Page.Type()).To(Equal(logseq.PageTypeDedicated))
	})

	It("emits PageUpdated when a journal file is created", func() {
		// Default journal filename format: yyyy_MM_dd -> 2006_01_02
		Expect(os.WriteFile(
			filepath.Join(dir, "journals", "2025_06_15.md"),
			[]byte("- journal entry\n"),
			0o644,
		)).To(Succeed())

		var event logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))

		updated := event.(*logseq.PageUpdated)
		Expect(updated.Page).ToNot(BeNil())
		Expect(updated.Page.Type()).To(Equal(logseq.PageTypeJournal))
		Expect(updated.Page.Date()).To(Equal(
			time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local),
		))
	})

	It("emits PageDeleted when a journal file is removed", func() {
		// Create a journal file before opening the graph
		journalPath := filepath.Join(dir, "journals", "2025_06_15.md")
		Expect(os.WriteFile(journalPath, []byte("- entry\n"), 0o644)).To(Succeed())

		var createEvent logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&createEvent))

		// Now delete
		Expect(os.Remove(journalPath)).To(Succeed())

		var event logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageDeleted{}))

		deleted := event.(*logseq.PageDeleted)
		Expect(deleted.Type).To(Equal(logseq.PageTypeJournal))
		Expect(deleted.Date).To(Equal(
			time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local),
		))
	})

	It("does not emit events for non-markdown files", func() {
		// Create a non-markdown file
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "notes.txt"),
			[]byte("not markdown"),
			0o644,
		)).To(Succeed())

		// Should not receive any event
		Consistently(watcher.Events(), 3*time.Second).ShouldNot(Receive())
	})

	It("reflects updated content in the PageUpdated event", func() {
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "test.md"),
			[]byte("- first block\n- second block\n"),
			0o644,
		)).To(Succeed())

		var event logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))

		updated := event.(*logseq.PageUpdated)
		Expect(updated.Page.Blocks()).To(HaveLen(2))
	})

	It("notifies multiple watchers of the same event", func() {
		watcher2 := graph.Watch()
		defer watcher2.Close()

		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "test.md"),
			[]byte("- changed\n"),
			0o644,
		)).To(Succeed())

		var event1, event2 logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event1))
		Eventually(watcher2.Events(), 5*time.Second).Should(Receive(&event2))

		Expect(event1).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))
		Expect(event2).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))
	})
})

var _ = Describe("Watcher renames", func() {
	var (
		graph   *logseq.Graph
		watcher *logseq.Watcher
		dir     string
	)

	BeforeEach(func() {
		dir = setupGraph()

		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "old.md"),
			[]byte("- hello world\n"),
			0o644,
		)).To(Succeed())
	})

	JustBeforeEach(func() {
		var err error
		graph, err = logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
		Expect(err).ToNot(HaveOccurred())

		watcher = graph.Watch()
	})

	AfterEach(func() {
		watcher.Close()
		graph.Close()
	})

	// receive collects the given number of events, and then checks that no
	// more events arrive. Both halves of a rename are handled within the same
	// debounce window, so a short wait is enough to see a missing pairing.
	receive := func(count int) []logseq.ChangeEvent {
		var events []logseq.ChangeEvent
		for len(events) < count {
			var event logseq.ChangeEvent
			Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
			events = append(events, event)
		}

		Consistently(watcher.Events(), 300*time.Millisecond).ShouldNot(Receive())
		return events
	}

	It("emits PageRenamed when a page file is moved", func() {
		Expect(os.Rename(
			filepath.Join(dir, "pages", "old.md"),
			filepath.Join(dir, "pages", "new.md"),
		)).To(Succeed())

		events := receive(1)
		Expect(events[0]).To(BeAssignableToTypeOf(&logseq.PageRenamed{}))

		renamed := events[0].(*logseq.PageRenamed)
		Expect(renamed.OldType).To(Equal(logseq.PageTypeDedicated))
		Expect(renamed.OldTitle).To(Equal("old"))
		Expect(renamed.Page.Title()).To(Equal("new"))
		Expect(renamed.Revision).To(Equal(renamed.Page.Revision()))
		Expect(renamed.OwnWrite).To(BeFalse())
	})

	It("updates the index when a page file is moved", func() {
		Expect(os.Rename(
			filepath.Join(dir, "pages", "old.md"),
			filepath.Join(dir, "pages", "new.md"),
		)).To(Succeed())

		receive(1)

		results, err := graph.SearchPages(context.Background(),
			logseq.WithQuery(logseq.TitleMatches("old")),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(0))

		results, err = graph.SearchPages(context.Background(),
			logseq.WithQuery(logseq.TitleMatches("new")),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(results.Size()).To(Equal(1))
	})

	It("emits PageRenamed when a file is removed and created with the same content", func() {
		Expect(os.Remove(filepath.Join(dir, "pages", "old.md"))).To(Succeed())

		// The create comes after the removal, but before the removal is
		// handled.
		time.Sleep(300 * time.Millisecond)
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "new.md"),
			[]byte("- hello world\n"),
			0o644,
		)).To(Succeed())

		events := receive(1)
		Expect(events[0]).To(BeAssignableToTypeOf(&logseq.PageRenamed{}))
		Expect(events[0].(*logseq.PageRenamed).OldTitle).To(Equal("old"))
	})

	It("emits PageRenamed when a journal file is moved", func() {
		journalPath := filepath.Join(dir, "journals", "2025_06_15.md")
		Expect(os.WriteFile(journalPath, []byte("- entry\n"), 0o644)).To(Succeed())
		receive(1)

		Expect(os.Rename(journalPath, filepath.Join(dir, "journals", "2025_06_16.md"))).To(Succeed())

		events := receive(1)
		Expect(events[0]).To(BeAssignableToTypeOf(&logseq.PageRenamed{}))

		renamed := events[0].(*logseq.PageRenamed)
		Expect(renamed.OldType).To(Equal(logseq.PageTypeJournal))
		Expect(renamed.OldDate).To(Equal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local)))
		Expect(renamed.Page.Date()).To(Equal(time.Date(2025, 6, 16, 0, 0, 0, 0, time.Local)))
	})

	It("emits PageDeleted and PageUpdated when the content changes", func() {
		Expect(os.Remove(filepath.Join(dir, "pages", "old.md"))).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "new.md"),
			[]byte("- other content\n"),
			0o644,
		)).To(Succeed())

		events := receive(2)
		Expect(events).To(ConsistOf(
			BeAssignableToTypeOf(&logseq.PageDeleted{}),
			BeAssignableToTypeOf(&logseq.PageUpdated{}),
		))
	})

	It("emits PageDeleted and PageUpdated when more than one file matches", func() {
		Expect(os.Remove(filepath.Join(dir, "pages", "old.md"))).To(Succeed())
		for _, name := range []string{"first.md", "second.md"} {
			Expect(os.WriteFile(
				filepath.Join(dir, "pages", name),
				[]byte("- hello world\n"),
				0o644,
			)).To(Succeed())
		}

		events := receive(3)
		Expect(events).To(ConsistOf(
			BeAssignableToTypeOf(&logseq.PageDeleted{}),
			BeAssignableToTypeOf(&logseq.PageUpdated{}),
			BeAssignableToTypeOf(&logseq.PageUpdated{}),
		))
	})

	It("marks a page renamed by the graph as its own write", func() {
		tx := graph.NewTransaction()
		Expect(tx.RenamePage(context.Background(), "old", "new")).To(Succeed())
		Expect(tx.Save()).To(Succeed())

		events := receive(1)
		Expect(events[0]).To(BeAssignableToTypeOf(&logseq.PageRenamed{}))
		Expect(events[0].(*logseq.PageRenamed).OwnWrite).To(BeTrue())
	})

	It("emits PageDeleted and PageUpdated without an index", func() {
		withoutIndex, err := logseq.Open(context.Background(), dir)
		Expect(err).ToNot(HaveOccurred())
		defer withoutIndex.Close()

		other := withoutIndex.Watch()
		defer other.Close()

		Expect(os.Rename(
			filepath.Join(dir, "pages", "old.md"),
			filepath.Join(dir, "pages", "new.md"),
		)).To(Succeed())

		var events []logseq.ChangeEvent
		for len(events) < 2 {
			var event logseq.ChangeEvent
			Eventually(other.Events(), 5*time.Second).Should(Receive(&event))
			events = append(events, event)
		}

		Expect(events).To(ConsistOf(
			BeAssignableToTypeOf(&logseq.PageDeleted{}),
			BeAssignableToTypeOf(&logseq.PageUpdated{}),
		))
	})
})

var _ = Describe("Watcher lifecycle", func() {
	var dir string

	BeforeEach(func() {
		dir = setupGraph()
	})

	It("can be closed more than once", func() {
		graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
		Expect(err).ToNot(HaveOccurred())
		defer graph.Close()

		watcher := graph.Watch()
		Expect(watcher.Close()).To(Succeed())
		Expect(watcher.Close()).To(Succeed())

		Expect(watcher.Done()).To(BeClosed())
	})

	It("closes done when the watcher is closed", func() {
		graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
		Expect(err).ToNot(HaveOccurred())
		defer graph.Close()

		watcher := graph.Watch()
		other := graph.Watch()
		defer other.Close()

		Expect(watcher.Done()).ToNot(BeClosed())
		Expect(watcher.Close()).To(Succeed())

		Expect(watcher.Done()).To(BeClosed())
		Expect(other.Done()).ToNot(BeClosed())
	})

	It("keeps indexing changes after a watcher is closed", func() {
		graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
		Expect(err).ToNot(HaveOccurred())
		defer graph.Close()

		watcher := graph.Watch()
		Expect(watcher.Close()).To(Succeed())

		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "later.md"),
			[]byte("- indexed after the watcher was closed\n"),
			0o644,
		)).To(Succeed())

		Eventually(func() int {
			results, err := graph.SearchPages(context.Background(),
				logseq.WithQuery(logseq.TitleMatches("later")),
			)
			Expect(err).ToNot(HaveOccurred())
			return results.Size()
		}, 5*time.Second).Should(Equal(1))
	})

	It("watches again after all watchers have been closed without an index", func() {
		graph, err := logseq.Open(context.Background(), dir)
		Expect(err).ToNot(HaveOccurred())
		defer graph.Close()

		Expect(graph.Watch().Close()).To(Succeed())

		watcher := graph.Watch()
		defer watcher.Close()

		Expect(os.WriteFile(
			filepath.Join(dir, "pages", "again.md"),
			[]byte("- content\n"),
			0o644,
		)).To(Succeed())

		var event logseq.ChangeEvent
		Eventually(watcher.Events(), 5*time.Second).Should(Receive(&event))
		Expect(event).To(BeAssignableToTypeOf(&logseq.PageUpdated{}))
	})
})

var _ = Describe("Change watching shutdown", func() {
	It("does not panic when the graph is closed while changes are debounced", func() {
		// Changes are debounced for a second before they are indexed, so each
		// round closes the graph right around the point where the timers for a
		// batch of changed files fire.
		for i := 0; i < 4; i++ {
			dir := setupGraph()

			graph, err := logseq.Open(context.Background(), dir, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())

			for j := 0; j < 10; j++ {
				Expect(os.WriteFile(
					filepath.Join(dir, "pages", fmt.Sprintf("debounced-%d.md", j)),
					[]byte("- hello\n"),
					0o644,
				)).To(Succeed())
			}

			time.Sleep(time.Duration(995+i*3) * time.Millisecond)
			Expect(graph.Close()).To(Succeed())
		}

		// Give timers that fired during the last close a chance to run.
		time.Sleep(100 * time.Millisecond)
	})
})
