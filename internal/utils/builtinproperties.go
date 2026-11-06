package utils

import "github.com/aholstenson/logseq-go/content"

// builtInProperties are the properties that Logseq uses for its own features,
// in the normalized form of their names, see content.NormalizePropertyName.
// These are the editable and hidden built-in properties of Logseq 0.10.9.
var builtInProperties = map[string]struct{}{
	// Built-in properties that users can edit.
	"title":                     {},
	"icon":                      {},
	"template":                  {},
	"template-including-parent": {},
	"public":                    {},
	"filters":                   {},
	"exclude-from-graph-view":   {},
	"logseq.query/nlp-date":     {},
	"macro":                     {},
	"filetags":                  {},
	"alias":                     {},
	"aliases":                   {},
	"tags":                      {},
	"logseq.color":              {},
	"logseq.table.version":      {},
	"logseq.table.compact":      {},
	"logseq.table.headers":      {},
	"logseq.table.hover":        {},
	"logseq.table.borders":      {},
	"logseq.table.stripes":      {},
	"logseq.table.max-width":    {},

	// Built-in properties that Logseq hides from users.
	"id":                     {},
	"custom-id":              {},
	"background-color":       {},
	"heading":                {},
	"collapsed":              {},
	"created-at":             {},
	"updated-at":             {},
	"last-modified-at":       {},
	"query-table":            {},
	"query-properties":       {},
	"query-sort-by":          {},
	"query-sort-desc":        {},
	"ls-type":                {},
	"hl-type":                {},
	"hl-page":                {},
	"hl-stamp":               {},
	"hl-color":               {},
	"logseq.macro-name":      {},
	"logseq.macro-arguments": {},
	"logseq.order-list-type": {},
	"logseq.tldraw.page":     {},
	"logseq.tldraw.shape":    {},
	"todo":                   {},
	"doing":                  {},
	"now":                    {},
	"later":                  {},
	"done":                   {},
}

// IsBuiltInProperty checks if Logseq uses the property with the given name for
// one of its own features, such as `alias`, `id` or `collapsed`.
func IsBuiltInProperty(name string) bool {
	_, ok := builtInProperties[content.NormalizePropertyName(name)]
	return ok
}
