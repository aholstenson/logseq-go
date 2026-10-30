package indexing_test

import (
	"context"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/indexing"
	"github.com/blugelabs/bluge"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Index version", func() {
	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
	})

	open := func(version int) *indexing.BlugeIndex {
		idx, err := indexing.NewBlugeIndexWithVersion(dir, version)
		Expect(err).ToNot(HaveOccurred())
		return idx
	}

	indexed := func(idx *indexing.BlugeIndex) []string {
		subPaths, err := idx.PageSubPaths(context.Background())
		Expect(err).ToNot(HaveOccurred())
		return subPaths
	}

	It("keeps the pages when the version is the same", func() {
		idx := open(indexing.IndexVersion)
		indexPage(idx, "pages/a.md", "Page A", content.NewBlock(content.NewParagraph(content.NewText("a"))))
		Expect(idx.Close()).To(Succeed())

		idx = open(indexing.IndexVersion)
		defer idx.Close()

		Expect(indexed(idx)).To(ConsistOf("pages/a.md"))
	})

	It("removes all pages and blocks when the version changes", func() {
		idx := open(indexing.IndexVersion)
		indexPage(idx, "pages/a.md", "Page A", content.NewBlock(content.NewParagraph(content.NewText("a"))))
		Expect(idx.Close()).To(Succeed())

		idx = open(indexing.IndexVersion + 1)
		defer idx.Close()

		Expect(indexed(idx)).To(BeEmpty())
		Expect(searchBlocks(idx, indexing.All())).To(BeEmpty())

		state, err := idx.GetFileState(context.Background(), "pages/a.md")
		Expect(err).ToNot(HaveOccurred())
		Expect(state).To(Equal(indexing.FileState{}))
	})

	It("keeps the pages indexed after a version change", func() {
		idx := open(indexing.IndexVersion)
		Expect(idx.Close()).To(Succeed())

		idx = open(indexing.IndexVersion + 1)
		indexPage(idx, "pages/a.md", "Page A", content.NewBlock(content.NewParagraph(content.NewText("a"))))
		Expect(idx.Close()).To(Succeed())

		idx = open(indexing.IndexVersion + 1)
		defer idx.Close()

		Expect(indexed(idx)).To(ConsistOf("pages/a.md"))
	})

	It("removes the pages of an index that has no version", func() {
		// An index built before the index stored its version.
		writer, err := bluge.OpenWriter(bluge.DefaultConfig(dir))
		Expect(err).ToNot(HaveOccurred())
		Expect(writer.Update(bluge.Identifier("pages/a.md"), bluge.NewDocument("pages/a.md").
			AddField(bluge.NewKeywordField("type", "page").StoreValue()).
			AddField(bluge.NewDateTimeField("lastModified", time.Now()).StoreValue()),
		)).To(Succeed())
		Expect(writer.Close()).To(Succeed())

		idx := open(indexing.IndexVersion)
		defer idx.Close()

		Expect(indexed(idx)).To(BeEmpty())
	})

	It("does not return the version in searches", func() {
		idx := open(indexing.IndexVersion)
		defer idx.Close()

		Expect(searchPages(idx, indexing.All())).To(BeEmpty())
		Expect(searchBlocks(idx, indexing.All())).To(BeEmpty())
	})
})
