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
})
