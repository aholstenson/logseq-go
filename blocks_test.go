package logseq_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Blocks", func() {
	var (
		graph *logseq.Graph
		dir   string
		ctx   context.Context
	)

	BeforeEach(func() {
		dir = setupGraph()
		ctx = context.Background()
	})

	AfterEach(func() {
		if graph != nil {
			graph.Close()
			graph = nil
		}
	})

	Describe("OpenBlock", func() {
		It("opens a block by its id", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "- referenced block\n  id:: 65e0a3f6-0000-4000-8000-000000000001\n",
			})

			block, page, err := graph.OpenBlock(ctx, "65e0a3f6-0000-4000-8000-000000000001")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Title()).To(Equal("target"))
			Expect(block.ID()).To(Equal("65e0a3f6-0000-4000-8000-000000000001"))

			text := block.Content().FindDeep(content.IsOfType[*content.Text]())
			Expect(text.(*content.Text).Value).To(Equal("referenced block"))
		})

		It("opens a block nested inside another block", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "- parent\n\t- child\n\t  id:: 65e0a3f6-0000-4000-8000-000000000002\n",
			})

			block, _, err := graph.OpenBlock(ctx, "65e0a3f6-0000-4000-8000-000000000002")
			Expect(err).ToNot(HaveOccurred())

			text := block.Content().FindDeep(content.IsOfType[*content.Text]())
			Expect(text.(*content.Text).Value).To(Equal("child"))
		})

		It("opens a block in a journal", func() {
			Expect(os.WriteFile(
				filepath.Join(dir, "journals", "2025_06_15.md"),
				[]byte("- journal block\n  id:: 65e0a3f6-0000-4000-8000-000000000003\n"),
				0o644,
			)).To(Succeed())

			var err error
			graph, err = logseq.Open(ctx, dir, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())

			block, page, err := graph.OpenBlock(ctx, "65e0a3f6-0000-4000-8000-000000000003")
			Expect(err).ToNot(HaveOccurred())
			Expect(page.Type()).To(Equal(logseq.PageTypeJournal))
			Expect(page.Date()).To(Equal(time.Date(2025, 6, 15, 0, 0, 0, 0, time.Local)))
			Expect(block.ID()).To(Equal("65e0a3f6-0000-4000-8000-000000000003"))
		})

		It("resolves the target of a block reference", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "- referenced block\n  id:: 65e0a3f6-0000-4000-8000-000000000004\n",
				"source.md": "- see ((65e0a3f6-0000-4000-8000-000000000004))\n",
			})

			page, err := graph.OpenPage("source")
			Expect(err).ToNot(HaveOccurred())

			ref := page.Blocks().FindDeep(func(block *content.Block) bool {
				return block.Content().FindDeep(content.IsOfType[*content.BlockRef]()) != nil
			})
			Expect(ref).ToNot(BeNil())

			node := ref.Content().FindDeep(content.IsOfType[*content.BlockRef]())
			block, _, err := graph.OpenBlock(ctx, node.(*content.BlockRef).ID)
			Expect(err).ToNot(HaveOccurred())

			text := block.Content().FindDeep(content.IsOfType[*content.Text]())
			Expect(text.(*content.Text).Value).To(Equal("referenced block"))
		})

		It("fails when no block has the id", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "- a block without an id\n",
			})

			_, _, err := graph.OpenBlock(ctx, "65e0a3f6-0000-4000-8000-000000000005")
			Expect(errors.Is(err, logseq.ErrBlockNotFound)).To(BeTrue())
		})

		It("fails when the id is empty", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "- a block without an id\n",
			})

			_, _, err := graph.OpenBlock(ctx, "")
			Expect(errors.Is(err, logseq.ErrBlockNotFound)).To(BeTrue())
		})

		It("fails when indexing is not enabled", func() {
			var err error
			graph, err = logseq.Open(ctx, dir)
			Expect(err).ToNot(HaveOccurred())

			_, _, err = graph.OpenBlock(ctx, "65e0a3f6-0000-4000-8000-000000000006")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, logseq.ErrBlockNotFound)).To(BeFalse())
		})

		It("saves changes made to a block opened in a transaction", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "- referenced block\n  id:: 65e0a3f6-0000-4000-8000-000000000007\n",
			})

			tx := graph.NewTransaction()
			block, _, err := tx.OpenBlock(ctx, "65e0a3f6-0000-4000-8000-000000000007")
			Expect(err).ToNot(HaveOccurred())

			block.AddChild(textBlock("added to the block"))
			Expect(tx.Save()).To(Succeed())

			data, err := os.ReadFile(filepath.Join(dir, "pages", "target.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal(
				"- referenced block\n  id:: 65e0a3f6-0000-4000-8000-000000000007\n\t- added to the block\n",
			))
		})

		It("finds a block that had an id added to it", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "- a block without an id\n",
			})

			tx := graph.NewTransaction()
			page, err := tx.OpenPage("target")
			Expect(err).ToNot(HaveOccurred())

			id := page.Blocks()[0].WithID().ID()
			Expect(id).ToNot(BeEmpty())
			Expect(tx.Save()).To(Succeed())

			// Logseq writes the id on the line after the content of the block
			data, err := os.ReadFile(filepath.Join(dir, "pages", "target.md"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("- a block without an id\n  id:: " + id + "\n"))

			// Reopening the graph indexes the id, making the block resolvable
			Expect(graph.Close()).To(Succeed())
			graph, err = logseq.Open(ctx, dir, logseq.WithInMemoryIndex())
			Expect(err).ToNot(HaveOccurred())

			block, _, err := graph.OpenBlock(ctx, id)
			Expect(err).ToNot(HaveOccurred())
			Expect(block.ID()).To(Equal(id))
		})

		It("returns the page the block is on for further changes", func() {
			graph = openGraphWithPages(dir, map[string]string{
				"target.md": "- referenced block\n  id:: 65e0a3f6-0000-4000-8000-000000000008\n",
			})

			tx := graph.NewTransaction()
			_, page, err := tx.OpenBlock(ctx, "65e0a3f6-0000-4000-8000-000000000008")
			Expect(err).ToNot(HaveOccurred())

			opened, err := tx.OpenPage("target")
			Expect(err).ToNot(HaveOccurred())
			Expect(opened).To(BeIdenticalTo(page))
		})
	})
})
