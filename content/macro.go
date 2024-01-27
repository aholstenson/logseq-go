package content

import "strings"

// Macro represents a macro, which in the source is on the form `{{macro-name arg1, arg2, ..., argN}}`.
//
// Macros share their syntax with some built-ins, such as `{{query}}` and `{{embed}}`,
// but are otherwise user-defined. If a supported built-in is found it will be parsed
// as its own node type, such as `Query`, `PageEmbed` or `BlockEmbed`.
//
// In Logseq arguments are optional, but must be comma separated. Arguments can be quoted,
// in which case they can contain commas. In this library arguments will be
// normalized, if a comma is in the argument it will be quoted in the output.
type Macro struct {
	baseNode

	// Name gets the name of this macro.
	Name string

	// Arguments gets the arguments of this macro.
	Arguments []string
}

// NewMacro creates a new macro with the given name and arguments.
func NewMacro(name string, args ...string) *Macro {
	return &Macro{
		Name:      name,
		Arguments: args,
	}
}

func (m *Macro) debug(p *debugPrinter) {
	p.StartType("Macro")
	p.Field("name", m.Name)
	p.Field("arguments", strings.Join(m.Arguments, ", "))
	p.EndType()
}

func (m *Macro) isInline() {}

var _ InlineNode = (*Macro)(nil)

// Query represents a simple query in the source, on the form `{{query datalog query}}`.
type Query struct {
	baseNode

	Query string
}

func NewQuery(query string) *Query {
	return &Query{
		Query: query,
	}
}

func (q *Query) debug(p *debugPrinter) {
	p.StartType("Query")
	p.Field("query", q.Query)
	p.EndType()
}

func (q *Query) isInline() {}

var _ InlineNode = (*Query)(nil)

type PageEmbed struct {
	baseNode

	To string
}

func NewPageEmbed(to string) *PageEmbed {
	return &PageEmbed{
		To: to,
	}
}

func (p *PageEmbed) debug(pr *debugPrinter) {
	pr.StartType("PageEmbed")
	pr.Field("to", p.To)
	pr.EndType()
}

func (p *PageEmbed) isInline() {}

var _ InlineNode = (*PageEmbed)(nil)

type BlockEmbed struct {
	baseNode

	ID string
}

func NewBlockEmbed(id string) *BlockEmbed {
	return &BlockEmbed{
		ID: id,
	}
}

func (b *BlockEmbed) debug(pr *debugPrinter) {
	pr.StartType("BlockEmbed")
	pr.Field("id", b.ID)
	pr.EndType()
}

func (b *BlockEmbed) isInline() {}

var _ InlineNode = (*BlockEmbed)(nil)
