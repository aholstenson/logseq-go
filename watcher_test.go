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
			time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
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
			time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
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
