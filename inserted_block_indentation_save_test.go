package logseq_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	logseq "github.com/aholstenson/logseq-go"
	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Save keeps the lines of a page that did not change. A new block that is
// put between blocks with space indentation must not get another kind of
// indentation than its siblings, as Logseq gives a level to each
// indentation unit. A tab line next to space lines can make Logseq read the
// next sibling as a child of the new block.
var _ = Describe("Indentation of a new block between space-indented siblings", func() {
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

	// siblingIndentation gives the indentation of each line that starts
	// one of the named child blocks.
	siblingIndentation := func(saved string, names ...string) []string {
		pattern := regexp.MustCompile(`^(\s+)- (` + strings.Join(names, "|") + `)$`)
		indentation := []string{}
		for _, line := range strings.Split(saved, "\n") {
			if m := pattern.FindStringSubmatch(line); m != nil {
				indentation = append(indentation, m[1])
			}
		}

		return indentation
	}

	insertAfterFirstChild := func(original string) string {
		graph = openGraphWithPages(dir, map[string]string{
			"outline.md": original,
		})

		tx := graph.NewTransaction()
		page, err := tx.OpenPage("outline")
		Expect(err).ToNot(HaveOccurred())
		a := page.Blocks()[0]
		b := a.Blocks()[0]
		a.InsertChildAfter(content.NewBlock(content.NewParagraph(content.NewText("new"))), b)
		Expect(tx.Save()).To(Succeed())

		return readPage("outline.md")
	}

	It("uses the same indentation for a new block as for its siblings in a page with two spaces", func() {
		// Given a page with two children of `a`, two spaces of indentation and a blank line between them
		// When a new block is put after `b` and the page is saved
		saved := insertAfterFirstChild("- a\n  - b\n\n  - c\n")

		// Then the file has no tab-indented `- new` line between the space lines
		Expect(saved).ToNot(ContainSubstring("  - b\n\t- new\n\n  - c"))

		// And every line that starts a sibling of `b` has the same indentation
		indentation := siblingIndentation(saved, "b", "new", "c")
		Expect(indentation).To(HaveLen(3), "saved: %q", saved)
		Expect(indentation).To(HaveEach(indentation[0]), "saved: %q", saved)
	})

	It("uses the same indentation for a new block as for its siblings in a page with four spaces", func() {
		// Given a page with two children of `a`, four spaces of indentation and a blank line between them
		// When a new block is put after `b` and the page is saved
		saved := insertAfterFirstChild("- a\n    - b\n\n    - c\n")

		// Then every line that starts a sibling of `b` has the same indentation
		indentation := siblingIndentation(saved, "b", "new", "c")
		Expect(indentation).To(HaveLen(3), "saved: %q", saved)
		Expect(indentation).To(HaveEach(indentation[0]), "saved: %q", saved)
	})

	It("does not mix indentation between siblings when an edit adds no line", func() {
		// Given a page with two children of `a`, two spaces of indentation and a blank line between them
		graph = openGraphWithPages(dir, map[string]string{
			"outline.md": "- a\n  - b\n\n  - c\n",
		})

		// When the text of `c` is changed and the page is saved
		tx := graph.NewTransaction()
		page, err := tx.OpenPage("outline")
		Expect(err).ToNot(HaveOccurred())
		c := page.Blocks()[0].Blocks()[1]
		text := c.Children().FindDeep(content.IsOfType[*content.Text]())
		Expect(text).ToNot(BeNil())
		text.(*content.Text).WithValue("C")
		Expect(tx.Save()).To(Succeed())
		saved := readPage("outline.md")

		// Then the siblings `b` and `C` have the same indentation
		Expect(saved).To(BeElementOf(
			"- a\n  - b\n\n  - C\n",
			"- a\n\t- b\n\n\t- C\n",
		))
	})
})
