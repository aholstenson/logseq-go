package content

import "strings"

type Macro struct {
	baseNode

	// Name gets the name of this macro.
	Name string

	// Arguments gets the arguments of this macro.
	Arguments []string
}

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
