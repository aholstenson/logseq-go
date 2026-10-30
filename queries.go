package logseq

import "github.com/aholstenson/logseq-go/internal/indexing"

type Query = indexing.Query

// All returns a query that matches everything.
func All() Query {
	return indexing.All()
}

// None returns a query that matches nothing.
func None() Query {
	return indexing.None()
}

// And returns a query that matches all of the given queries.
func And(queries ...Query) Query {
	return indexing.And(queries...)
}

// Or returns a query that matches any of the given queries.
func Or(queries ...Query) Query {
	return indexing.Or(queries...)
}

// Not returns a query that matches anything that does not match the given query.
func Not(query Query) Query {
	return indexing.Not(query)
}

// TitleMatches returns a query that matches the title of a page against a
// text.
func TitleMatches(text string) Query {
	return indexing.TitleMatches(text)
}

// ContentMatches returns a query that matches the content of a page or a
// block against a text.
func ContentMatches(text string) Query {
	return indexing.ContentMatches(text)
}

// BlockIDEquals matches the block with the given id, which is the identifier
// that block references such as `((id))` point at.
func BlockIDEquals(id string) Query {
	return indexing.BlockIDEquals(id)
}

// PropertyMatches returns a query that matches the given text against the
// value of a property.
func PropertyMatches(property, text string) Query {
	return indexing.PropertyMatches(property, text)
}

// PropertyEquals returns a query that checks that a property has a specific
// value.
func PropertyEquals(property, value string) Query {
	return indexing.PropertyEquals(property, value)
}

// HasProperty returns a query that matches pages and blocks that have the
// given property, whatever its value is. A property with an empty value, such
// as `tags:: `, also matches. Property names are matched without regard for
// case, so `template` also matches `Template::`.
func HasProperty(property string) Query {
	return indexing.HasProperty(property)
}

// PropertyReferences checks if the given property references a specific
// page, as both a link and a tag.
func PropertyReferences(property, target string) Query {
	return indexing.PropertyReferences(property, target)
}

// PropertyReferencesTag checks if the given property references a
// specific page as a tag.
func PropertyReferencesTag(property, tag string) Query {
	return indexing.PropertyReferencesTag(property, tag)
}

// InNamespace matches the pages that are directly in a namespace, so `Parent`
// matches the page `Parent/Child` but not `Parent/Child/Grandchild`.
func InNamespace(namespace string) Query {
	return indexing.InNamespace(namespace)
}

// UnderNamespace matches the pages anywhere in a namespace, so `Parent` matches
// both `Parent/Child` and `Parent/Child/Grandchild`.
func UnderNamespace(namespace string) Query {
	return indexing.UnderNamespace(namespace)
}

// HasAlias matches the page that has the given title as one of the aliases in
// its `alias` property.
func HasAlias(alias string) Query {
	return indexing.HasAlias(alias)
}

// References returns a query that matches if the given page is referenced
// both as a link and a tag.
func References(page string) Query {
	return indexing.References(page)
}

// ReferencesTag returns a query that matches if the given page is
// referenced as a tag.
func ReferencesTag(page string) Query {
	return indexing.ReferencesTag(page)
}

// LinksToURL returns a query that checks if a block or page links to a
// URL.
func LinksToURL(url string) Query {
	return indexing.LinksToURL(url)
}
