package content

// AdvancedCommand represents an unknown command that is a BEGIN..END section
// within a block.
//
// See https://docs.logseq.com/#/page/advanced%20commands
type AdvancedCommand struct {
	baseNode

	// Type is the type of the advanced command.
	Type string

	// Value is the value of the advanced command. Specific to the type of
	// command.
	Value string
}

func NewAdvancedCommand(variant string, value string) *AdvancedCommand {
	return &AdvancedCommand{
		Type:  variant,
		Value: value,
	}
}

// WithType sets the type of the advanced command.
func (h *AdvancedCommand) WithType(variant string) *AdvancedCommand {
	h.Type = variant
	return h
}

// WithValue sets the value of the advanced command.
func (h *AdvancedCommand) WithValue(value string) *AdvancedCommand {
	h.Value = value
	return h
}

func (h *AdvancedCommand) debug(p *debugPrinter) {
	p.StartType("AdvancedCommand")
	p.Field("type", h.Type)
	p.Field("value", h.Value)
	p.EndType()
}

func (h *AdvancedCommand) isBlock() {}

var _ BlockNode = (*AdvancedCommand)(nil)

// QueryCommand is a command represented by #+BEGIN_QUERY..#+END_QUERY in
// a block.
type QueryCommand struct {
	baseNode

	Query string
}

func NewQueryCommand(query string) *QueryCommand {
	return &QueryCommand{
		Query: query,
	}
}

// WithQuery sets the query for the command.
func (h *QueryCommand) WithQuery(query string) *QueryCommand {
	h.Query = query
	return h
}

func (h *QueryCommand) debug(p *debugPrinter) {
	p.StartType("QueryCommand")
	p.Field("query", h.Query)
	p.EndType()
}

func (h *QueryCommand) isBlock() {}

var _ BlockNode = (*QueryCommand)(nil)

// QuoteCommand is a command represented by #+BEGIN_QUOTE..#+END_QUOTE in
// a block.
type QuoteCommand struct {
	baseNode

	Quote string
}
