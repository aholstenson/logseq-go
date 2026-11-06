package textmerge

import (
	"math/rand"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Merge", func() {
	merge := func(base string, original string, changed string) string {
		merged, ok := Merge(base, original, changed)
		Expect(ok).To(BeTrue())
		return merged
	}

	It("keeps the original when nothing changed", func() {
		Expect(merge("a\nB\nc\n", "a\nb\nc\n", "a\nB\nc\n")).To(Equal("a\nb\nc\n"))
	})

	It("applies a changed line and keeps the other lines of the original", func() {
		// Given an original that differs from base on line 2
		// When line 1 changes
		// Then line 1 is changed and line 2 stays as in the original
		Expect(merge("a\nB\nc\n", "a\nb\nc\n", "A\nB\nc\n")).To(Equal("A\nb\nc\n"))
	})

	It("applies inserted and removed lines", func() {
		Expect(merge("a\nB\nc\n", "a\nb\nc\n", "a\nnew\nB\n")).To(Equal("a\nnew\nb\n"))
	})

	It("uses the changed lines where both changed the same line", func() {
		Expect(merge("a\nB\nc\n", "a\nb\nc\n", "a\nX\nc\n")).To(Equal("a\nX\nc\n"))
	})

	It("uses the changed lines where both inserted lines at the same place", func() {
		Expect(merge("a\nc\n", "a\nb\nc\n", "a\nx\nc\n")).To(Equal("a\nx\nc\n"))
	})

	It("keeps lines the original added next to a changed line", func() {
		Expect(merge("a\nc\n", "a\nb\nc\n", "a\nC\n")).To(Equal("a\nb\nC\n"))
	})

	It("handles texts without a line break at the end", func() {
		Expect(merge("a\nB", "a\nb", "A\nB")).To(Equal("A\nb"))
	})

	It("handles empty texts", func() {
		Expect(merge("", "", "a\n")).To(Equal("a\n"))
		Expect(merge("a\n", "a\n", "")).To(Equal(""))
	})

	It("gives up when the texts differ too much", func() {
		a := strings.Builder{}
		b := strings.Builder{}
		for i := 0; i < maxEdits+10; i++ {
			a.WriteString("a\n")
			b.WriteString("b\n")
		}

		_, ok := Merge(a.String(), a.String(), b.String())
		Expect(ok).To(BeFalse())
	})

	Describe("with random texts", func() {
		// The seed is fixed so that a failure can be repeated.
		const seed = 20261106
		random := rand.New(rand.NewSource(seed))

		randomText := func() string {
			lines := random.Intn(12)
			out := strings.Builder{}
			for i := 0; i < lines; i++ {
				out.WriteString(string(rune('a' + random.Intn(4))))
				out.WriteString("\n")
			}
			return out.String()
		}

		It("finds hunks that change one text into the other", func() {
			for i := 0; i < 2000; i++ {
				a, b := randomText(), randomText()
				hunks, ok := diff(splitLines(a), splitLines(b))
				Expect(ok).To(BeTrue())

				out := strings.Builder{}
				writeHunks(&out, splitLines(a), 0, len(splitLines(a)), hunks)
				Expect(out.String()).To(Equal(b), "a: %q, b: %q, seed: %d", a, b, seed)
			}
		})

		It("gives the changed text when the original is the base", func() {
			for i := 0; i < 2000; i++ {
				base, changed := randomText(), randomText()
				Expect(merge(base, base, changed)).To(Equal(changed), "base: %q, changed: %q, seed: %d", base, changed, seed)
			}
		})

		It("gives the original when nothing changed", func() {
			for i := 0; i < 2000; i++ {
				base, original := randomText(), randomText()
				Expect(merge(base, original, base)).To(Equal(original), "base: %q, original: %q, seed: %d", base, original, seed)
			}
		})
	})
})
