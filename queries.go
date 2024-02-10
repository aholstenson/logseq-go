package logseq

import "github.com/aholstenson/logseq-go/internal/indexing"

type Query = indexing.Query

func All() Query {
	return indexing.All()
}

func None() Query {
	return indexing.None()
}

func And(queries ...Query) Query {
	return indexing.And(queries...)
}

func Or(queries ...Query) Query {
	return indexing.Or(queries...)
}

func Not(query Query) Query {
	return indexing.Not(query)
}

func TitleMatches(text string) Query {
	return indexing.TitleMatches(text)
}

func ContentMatches(text string) Query {
	return indexing.ContentMatches(text)
}

func PropertyMatches(property, text string) Query {
	return indexing.PropertyMatches(property, text)
}

func PropertyEquals(property, value string) Query {
	return indexing.PropertyEquals(property, value)
}

func PropertyReferences(property, target string) Query {
	return indexing.PropertyReferences(property, target)
}

func PropertyReferencesTag(property, tag string) Query {
	return indexing.PropertyReferencesTag(property, tag)
}

func References(page string) Query {
	return indexing.References(page)
}

func ReferencesTag(page string) Query {
	return indexing.ReferencesTag(page)
}

func LinksToURL(url string) Query {
	return indexing.LinksToURL(url)
}
