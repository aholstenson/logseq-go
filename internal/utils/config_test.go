package utils_test

import (
	"github.com/aholstenson/logseq-go/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	It("parses empty config", func() {
		c, err := utils.ParseConfig([]byte("{}"))
		Expect(err).ToNot(HaveOccurred())

		Expect(c).To(Equal(&utils.GraphConfig{
			JournalsDir: "journals",
			Journal: utils.JournalConfig{
				FileNameFormat: "yyyy_MM_dd",
			},
			File: utils.FileConfig{
				NameFormat: utils.FilenameFormatTripleLowbar,
			},
		}))
	})

	It("parses journals directory", func() {
		c, err := utils.ParseConfig([]byte(`{:journals-directory "journals"}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.JournalsDir).To(Equal("journals"))
	})

	It("parses journal file name format", func() {
		c, err := utils.ParseConfig([]byte(`{:journal {:file-name-format "yyyy_MM_dd"}}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.Journal.FileNameFormat).To(Equal("yyyy_MM_dd"))
	})

	It("parses default templates", func() {
		c, err := utils.ParseConfig([]byte(`{:default-templates {:journals "test"}}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.DefaultTemplates.Journals).To(Equal("test"))
	})
})
