package query_test

import (
	"testing"

	"github.com/aholstenson/logseq-go/query"
)

// FuzzParse checks that any text either fails to parse or parses into a query
// that is written as text that parses back into the same text.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"(and (task TODO LATER) (or (scheduled < today) (deadline < today)) (sort-by priority asc))",
		"(and (task TODO) (or (and (journal today) (not (scheduled))) (scheduled today)) (not [[shopping]]))",
		"(between [[Jan 1st, 2024]] -7)",
		`(page-property type "Cognitive Recipe")`,
		"(and #tag #[[a b]] \"text\" (sample 3))",
		"(property type [a b])",
		"(page-tags [book reading])",
		"hello world",
		"(and\n ; comment\n [[a]],[[b]])",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		q, err := query.Parse(text)
		if err != nil {
			return
		}

		written := q.String()
		again, err := query.Parse(written)
		if err != nil {
			t.Fatalf("parsing %q, written as %q: %v", text, written, err)
		}

		if again.String() != written {
			t.Fatalf("parsing %q, written as %q, written again as %q", text, written, again.String())
		}
	})
}
