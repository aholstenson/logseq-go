package utils_test

import (
	"github.com/aholstenson/logseq-go/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filenames", func() {
	Describe("Title to filename", func() {
		It("keeps capitalization", func() {
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "Hello World")).To(Equal("Hello World"))
		})

		It("namespaces with slash converted to triple lowbar", func() {
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "Hello/World")).To(Equal("Hello___World"))
		})

		It("underscore before slash percent encoded", func() {
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "Hello_/World")).To(Equal("Hello%5F___World"))
		})

		It("underscore after slash percent encoded", func() {
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "Hello/_World")).To(Equal("Hello___%5FWorld"))
		})

		It("percent in title left unencoded", func() {
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "Hello%World")).To(Equal("Hello%World"))
		})

		It("percent encoding in title encoded", func() {
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "Hello%25World")).To(Equal("Hello%2525World"))
		})

		It("windows protected names appends triple underscores", func() {
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "CON")).To(Equal("CON___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "PRN")).To(Equal("PRN___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "AUX")).To(Equal("AUX___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "NUL")).To(Equal("NUL___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "COM1")).To(Equal("COM1___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "COM2")).To(Equal("COM2___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "COM3")).To(Equal("COM3___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "COM4")).To(Equal("COM4___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "COM5")).To(Equal("COM5___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "COM6")).To(Equal("COM6___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "COM7")).To(Equal("COM7___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "COM8")).To(Equal("COM8___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "COM9")).To(Equal("COM9___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "LPT1")).To(Equal("LPT1___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "LPT2")).To(Equal("LPT2___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "LPT3")).To(Equal("LPT3___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "LPT4")).To(Equal("LPT4___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "LPT5")).To(Equal("LPT5___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "LPT6")).To(Equal("LPT6___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "LPT7")).To(Equal("LPT7___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "LPT8")).To(Equal("LPT8___"))
			Expect(utils.TitleToFilename(utils.FilenameFormatTripleLowbar, "LPT9")).To(Equal("LPT9___"))
		})
	})

	Describe("Filename to title", func() {
		It("keeps capitalization", func() {
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "Hello World")).To(Equal("Hello World"))
		})

		It("triple lowbar to namespace with slash", func() {
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "Hello___World")).To(Equal("Hello/World"))
		})

		It("percent encoding before triple lowbar", func() {
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "Hello%5F___World")).To(Equal("Hello_/World"))
		})

		It("percent encoding after triple lowbar", func() {
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "Hello___%5FWorld")).To(Equal("Hello/_World"))
		})

		It("percent in title left as is", func() {
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "Hello%World")).To(Equal("Hello%World"))
		})

		It("percent encoding in title decoded", func() {
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "Hello%2525World")).To(Equal("Hello%25World"))
		})

		It("windows protected names appends triple underscores", func() {
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "CON___")).To(Equal("CON"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "PRN___")).To(Equal("PRN"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "AUX___")).To(Equal("AUX"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "NUL___")).To(Equal("NUL"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "COM1___")).To(Equal("COM1"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "COM2___")).To(Equal("COM2"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "COM3___")).To(Equal("COM3"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "COM4___")).To(Equal("COM4"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "COM5___")).To(Equal("COM5"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "COM6___")).To(Equal("COM6"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "COM7___")).To(Equal("COM7"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "COM8___")).To(Equal("COM8"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "COM9___")).To(Equal("COM9"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "LPT1___")).To(Equal("LPT1"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "LPT2___")).To(Equal("LPT2"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "LPT3___")).To(Equal("LPT3"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "LPT4___")).To(Equal("LPT4"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "LPT5___")).To(Equal("LPT5"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "LPT6___")).To(Equal("LPT6"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "LPT7___")).To(Equal("LPT7"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "LPT8___")).To(Equal("LPT8"))
			Expect(utils.FilenameToTitle(utils.FilenameFormatTripleLowbar, "LPT9___")).To(Equal("LPT9"))
		})
	})
})
