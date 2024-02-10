package indexing

type Query interface {
	isQuery()
}

type all struct{}

func (a *all) isQuery() {}

type none struct{}

func (n *none) isQuery() {}

type and struct {
	clauses []Query
}

func (a *and) isQuery() {}

type or struct {
	clauses []Query
}

func (o *or) isQuery() {}

type not struct {
	clause Query
}

func (n *not) isQuery() {}

type fieldMatches struct {
	field string
	text  string
}

func (f *fieldMatches) isQuery() {}

type fieldEquals struct {
	field string
	value string
}

func (f *fieldEquals) isQuery() {}

type fieldRefs struct {
	field  string
	target string
	tag    bool
}

func (f *fieldRefs) isQuery() {}

func All() *all {
	return &all{}
}

func None() *none {
	return &none{}
}

func And(clauses ...Query) *and {
	return &and{
		clauses: clauses,
	}
}

func Or(clauses ...Query) *or {
	return &or{
		clauses: clauses,
	}
}

func Not(clause Query) *not {
	return &not{
		clause: clause,
	}
}

func TitleMatches(text string) Query {
	return &fieldMatches{
		field: "title",
		text:  text,
	}
}

func ContentMatches(text string) Query {
	return &fieldMatches{
		field: "content",
		text:  text,
	}
}

func PropertyMatches(property string, text string) Query {
	return &fieldMatches{
		field: "prop:" + property,
		text:  text,
	}
}

func PropertyEquals(property string, value string) Query {
	return &fieldEquals{
		field: "prop:" + property,
		value: value,
	}
}

func PropertyReferences(property string, target string) Query {
	return &fieldRefs{
		field:  "prop:" + property,
		target: target,
	}
}

func PropertyReferencesTag(property string, target string) Query {
	return &fieldRefs{
		field:  "prop:" + property,
		target: target,
		tag:    true,
	}
}

func References(page string) Query {
	return &fieldRefs{
		field:  "pages",
		target: page,
	}
}

func ReferencesTag(tag string) Query {
	return &fieldRefs{
		field:  "pages",
		target: tag,
		tag:    true,
	}
}

func LinksToURL(url string) Query {
	return &fieldEquals{
		field: "link",
		value: url,
	}
}
