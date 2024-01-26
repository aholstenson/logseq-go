package content

import "strings"

// Macro represents a macro, which in the source is on the form `{{macro-name arg1, arg2, ..., argN}}`.
//
// Macros share their syntax with some built-ins, such as `{{query}}` and `{{embed}}`,
// but are otherwise user-defined. If a supported built-in is found it will be parsed
// as its own node type, such as `Query`, `PageEmbed` or `BlockEmbed`.
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
