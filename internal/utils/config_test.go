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
			JournalsDir:            "journals",
			JournalPageTitleFormat: "MMM do, yyyy",
			JournalFileNameFormat:  "yyyy_MM_dd",
			PagesDir:               "pages",
			FileNameFormat:         utils.FilenameFormatTripleLowbar,
			PreferredWorkflow:      utils.PreferredWorkflowNow,
			PreferredFormat:        utils.PreferredFormatMarkdown,
			Logbook: utils.LogbookConfig{
				WithSecondSupport:          true,
				EnabledInTimestampedBlocks: true,
			},
		}))
	})

	It("parses journals directory", func() {
		c, err := utils.ParseConfig([]byte(`{:journals-directory "logs"}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.JournalsDir).To(Equal("logs"))
	})

	It("parses pages directory", func() {
		c, err := utils.ParseConfig([]byte(`{:pages-directory "notes"}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.PagesDir).To(Equal("notes"))
	})

	It("parses journal file name format", func() {
		c, err := utils.ParseConfig([]byte(`{:journal/file-name-format "yyyy-MM-dd"}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.JournalFileNameFormat).To(Equal("yyyy-MM-dd"))
	})

	It("parses journal page title format", func() {
		c, err := utils.ParseConfig([]byte(`{:journal/page-title-format "EEE do, MMM yyyy"}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.JournalPageTitleFormat).To(Equal("EEE do, MMM yyyy"))
	})

	It("falls back to the legacy date formatter for journal page titles", func() {
		c, err := utils.ParseConfig([]byte(`{:date-formatter "yyyy-MM-dd"}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.JournalPageTitleFormat).To(Equal("yyyy-MM-dd"))
	})

	It("prefers the journal page title format over the legacy date formatter", func() {
		c, err := utils.ParseConfig([]byte(`{:date-formatter "yyyy-MM-dd" :journal/page-title-format "MMM do, yyyy"}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.JournalPageTitleFormat).To(Equal("MMM do, yyyy"))
	})

	It("parses file name format written as a keyword", func() {
		c, err := utils.ParseConfig([]byte(`{:file/name-format :triple-lowbar}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.FileNameFormat).To(Equal(utils.FilenameFormatTripleLowbar))
	})

	It("parses properties separated by commas written as keywords", func() {
		c, err := utils.ParseConfig([]byte(`{:property/separated-by-commas #{:alias :tags}}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.PropertiesSeparatedByCommas).To(ConsistOf("alias", "tags"))
	})

	It("parses ignored page reference keywords written as keywords", func() {
		c, err := utils.ParseConfig([]byte(`{:ignored-page-references-keywords #{:author :website}}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.IgnoredPageReferencesKeywords).To(ConsistOf("author", "website"))
	})

	It("parses ignored page reference keywords written as strings", func() {
		c, err := utils.ParseConfig([]byte(`{:ignored-page-references-keywords ["author"]}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.IgnoredPageReferencesKeywords).To(ConsistOf("author"))
	})

	It("parses default templates", func() {
		c, err := utils.ParseConfig([]byte(`{:default-templates {:journals "test"}}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.DefaultTemplates.Journals).To(Equal("test"))
	})

	Describe("Preferred workflow", func() {
		It("parses :todo", func() {
			c, err := utils.ParseConfig([]byte(`{:preferred-workflow :todo}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.PreferredWorkflow).To(Equal(utils.PreferredWorkflowTodo))
		})

		It("parses :now", func() {
			c, err := utils.ParseConfig([]byte(`{:preferred-workflow :now}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.PreferredWorkflow).To(Equal(utils.PreferredWorkflowNow))
		})

		It("parses workflow written as a string", func() {
			c, err := utils.ParseConfig([]byte(`{:preferred-workflow "TODO"}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.PreferredWorkflow).To(Equal(utils.PreferredWorkflowTodo))
		})

		It("treats an unknown workflow as TODO/DOING", func() {
			c, err := utils.ParseConfig([]byte(`{:preferred-workflow :something-else}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.PreferredWorkflow).To(Equal(utils.PreferredWorkflowTodo))
		})

		It("treats nil as LATER/NOW", func() {
			c, err := utils.ParseConfig([]byte(`{:preferred-workflow nil}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.PreferredWorkflow).To(Equal(utils.PreferredWorkflowNow))
		})
	})

	Describe("Preferred format", func() {
		It("parses format written as a string", func() {
			c, err := utils.ParseConfig([]byte(`{:preferred-format "Markdown"}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.PreferredFormat).To(Equal(utils.PreferredFormatMarkdown))
		})

		It("parses format written as a keyword", func() {
			c, err := utils.ParseConfig([]byte(`{:preferred-format :org}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.PreferredFormat).To(Equal(utils.PreferredFormatOrg))
		})

		It("keeps an unknown format", func() {
			c, err := utils.ParseConfig([]byte(`{:preferred-format "AsciiDoc"}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.PreferredFormat).To(Equal(utils.PreferredFormat("asciidoc")))
		})
	})

	It("parses favorites", func() {
		c, err := utils.ParseConfig([]byte(`{:favorites ["Interstellar" "some page"]}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.Favorites).To(Equal([]string{"Interstellar", "some page"}))
	})

	It("parses hidden files and directories", func() {
		c, err := utils.ParseConfig([]byte(`{:hidden ["/archived" "/test.md" "../assets/archived"]}`))
		Expect(err).ToNot(HaveOccurred())

		Expect(c.Hidden).To(Equal([]string{"/archived", "/test.md", "../assets/archived"}))
	})

	Describe("IsHidden", func() {
		hidden := func(patterns string, path string) bool {
			c, err := utils.ParseConfig([]byte(`{:hidden ` + patterns + `}`))
			Expect(err).ToNot(HaveOccurred())
			return c.IsHidden(path)
		}

		It("hides nothing when there are no patterns", func() {
			c, err := utils.ParseConfig([]byte(`{}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.IsHidden("pages/test.md")).To(BeFalse())
		})

		It("hides a directory and what is in it", func() {
			Expect(hidden(`["/archived"]`, "archived")).To(BeTrue())
			Expect(hidden(`["/archived"]`, "archived/page.md")).To(BeTrue())
		})

		It("hides a single file", func() {
			Expect(hidden(`["/pages/test.md"]`, "pages/test.md")).To(BeTrue())
			Expect(hidden(`["/pages/test.md"]`, "pages/other.md")).To(BeFalse())
		})

		It("treats a pattern without a leading slash as relative to the graph", func() {
			Expect(hidden(`["logseq/bak"]`, "logseq/bak/page.md")).To(BeTrue())
		})

		It("accepts a path with a leading slash", func() {
			Expect(hidden(`["/archived"]`, "/archived/page.md")).To(BeTrue())
		})

		It("keeps paths that do not match", func() {
			Expect(hidden(`["/archived"]`, "pages/test.md")).To(BeFalse())
		})

		It("does not match a path outside the graph", func() {
			Expect(hidden(`["../assets/archived"]`, "assets/archived/file.png")).To(BeFalse())
		})
	})

	Describe("Default queries", func() {
		It("parses journal queries", func() {
			c, err := utils.ParseConfig([]byte(`{:default-queries
			  {:journals
			   [{:title "🔨 NOW"
			     :query [:find (pull ?h [*])
			             :in $ ?start ?today
			             :where
			             [?h :block/marker ?marker]]
			     :inputs [:14d :today]
			     :result-transform (fn [result] result)
			     :group-by-page? false
			     :collapsed? true}]}}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.DefaultQueries.Journals).To(HaveLen(1))

			query := c.DefaultQueries.Journals[0]
			Expect(query.Title.Text).To(Equal("🔨 NOW"))
			Expect(string(query.Title.Raw)).To(Equal(`"🔨 NOW"`))
			Expect(string(query.Query)).To(ContainSubstring("[?h :block/marker ?marker]"))
			Expect(string(query.Inputs)).To(Equal("[:14d :today]"))
			Expect(string(query.ResultTransform)).To(Equal("(fn [result] result)"))
			Expect(query.GroupByPage).To(Equal(ptr(false)))
			Expect(query.Collapsed).To(BeTrue())
		})

		It("leaves grouping undecided when the query does not say", func() {
			c, err := utils.ParseConfig([]byte(`{:default-queries {:journals [{:title "NOW"}]}}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.DefaultQueries.Journals[0].GroupByPage).To(BeNil())
			Expect(c.DefaultQueries.Journals[0].Collapsed).To(BeFalse())
		})

		It("parses a title written as Hiccup", func() {
			c, err := utils.ParseConfig([]byte(`{:default-queries {:journals [{:title [:b "NOW"]}]}}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.DefaultQueries.Journals[0].Title.Text).To(Equal(""))
			Expect(string(c.DefaultQueries.Journals[0].Title.Raw)).To(Equal(`[:b "NOW"]`))
		})
	})

	Describe("Logbook settings", func() {
		It("parses all settings", func() {
			c, err := utils.ParseConfig([]byte(`{:logbook/settings
			  {:with-second-support? false
			   :enabled-in-all-blocks true
			   :enabled-in-timestamped-blocks false}}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.Logbook).To(Equal(utils.LogbookConfig{
				WithSecondSupport:          false,
				EnabledInAllBlocks:         true,
				EnabledInTimestampedBlocks: false,
			}))
		})

		It("keeps the defaults of settings that are left out", func() {
			c, err := utils.ParseConfig([]byte(`{:logbook/settings {:enabled-in-all-blocks true}}`))
			Expect(err).ToNot(HaveOccurred())

			Expect(c.Logbook).To(Equal(utils.LogbookConfig{
				WithSecondSupport:          true,
				EnabledInAllBlocks:         true,
				EnabledInTimestampedBlocks: true,
			}))
		})
	})
})

func ptr[V any](value V) *V {
	return &value
}
