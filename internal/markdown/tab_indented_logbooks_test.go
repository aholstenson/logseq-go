package markdown_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/markdown"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// A logbook under a task can be indented with a tab. Logseq reads it as a
// logbook. Reading the page gives a logbook with its clock entries, and
// saving the page keeps it a logbook.
var _ = Describe("Parsing then outputting tab-indented logbooks", func() {
	// Given a page with a tab-indented logbook with one clock line, When
	// it is read, Then the block has a logbook with one clock entry, and the
	// written text reads back with the same logbook.
	It("reads a tab-indented logbook as a logbook", func() {
		input := "- TODO x\n\t:LOGBOOK:\n\tCLOCK: [2024-01-01 Mon 10:00:00]--[2024-01-01 Mon 11:00:00] =>  01:00:00\n\t:END:\n"
		root, err := markdown.ParseString(input, markdown.AsPage())
		Expect(err).ToNot(HaveOccurred())

		logbook := root.Children().FindDeep(content.IsOfType[*content.Logbook]())
		Expect(logbook).ToNot(BeNil())
		Expect(logbook.(*content.Logbook).Children().Filter(content.IsOfType[*content.LogbookEntryClock]())).To(HaveLen(1))

		v, err := markdown.AsString(root)
		Expect(err).ToNot(HaveOccurred())
		if !strings.HasSuffix(v, "\n") {
			v += "\n"
		}
		if v != input {
			again, err := markdown.ParseString(v, markdown.AsPage())
			Expect(err).ToNot(HaveOccurred())
			Expect(content.Debug(again)).To(Equal(content.Debug(root)), "text: %q", v)
		}
	})
})
