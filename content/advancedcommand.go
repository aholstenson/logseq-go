package content

// AdvancedCommand represents a command that is a BEGIN..END section within a
// block, such as `#+BEGIN_NOTE` and `#+END_NOTE`.
//
// Logseq reads the text of most commands as Markdown, for example QUOTE,
// NOTE, TIP and CENTER. For these the children of the command hold the text as
// it was read, so that the links in it are found and a rename updates them. A
// list that starts with `-` is a list there, and not a block. The text of SRC,
// EXAMPLE, COMMENT and EXPORT is not Markdown, so these have no children.
//
// See https://docs.logseq.com/#/page/advanced%20commands
type AdvancedCommand struct {
	baseNodeWithChildren
	previousLineAwareImpl

	// Type is the type of the advanced command.
	Type string

	// Value is the text between the BEGIN and END lines, as it was read. For
	// a command with children, the text is written as it is while the
	// children give the same text, and the children are written when they
	// were changed. Change the children of such a command, and not Value.
	Value string
}

// NewAdvancedCommand creates a command of the given type with the given text.
// Give children for a command whose text is Markdown, see AdvancedCommand.
func NewAdvancedCommand(variant string, value string, children ...Node) *AdvancedCommand {
	command := &AdvancedCommand{
		Type:  variant,
		Value: value,
	}
	command.self = command
	command.childValidator = allowOnlyBlockNodes
	for _, child := range children {
		command.AddChild(child)
	}
	return command
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

// WithPreviousLineType sets the type of the line before the command and returns
// it.
func (h *AdvancedCommand) WithPreviousLineType(t PreviousLineType) *AdvancedCommand {
	h.previousLineType = t
	return h
}

func (h *AdvancedCommand) debug(p *debugPrinter) {
	p.StartType("AdvancedCommand")
	p.Field("type", h.Type)
	p.Field("value", h.Value)
	debugPreviousLineAware(p, h)
	if h.FirstChild() != nil {
		p.Children(h)
	}
	p.EndType()
}

func (h *AdvancedCommand) isBlock() {}

var _ BlockNode = (*AdvancedCommand)(nil)
var _ HasChildren = (*AdvancedCommand)(nil)

// QueryCommand is a command represented by #+BEGIN_QUERY..#+END_QUERY in
// a block, which holds an advanced query of Logseq. Query holds the text
// between the two lines, which is a map of EDN. The query package reads it
// with query.ParseNode.
type QueryCommand struct {
	baseNode
	previousLineAwareImpl

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

// WithPreviousLineType sets the type of the line before the query and returns
// it.
func (h *QueryCommand) WithPreviousLineType(t PreviousLineType) *QueryCommand {
	h.previousLineType = t
	return h
}

func (h *QueryCommand) debug(p *debugPrinter) {
	p.StartType("QueryCommand")
	p.Field("query", h.Query)
	debugPreviousLineAware(p, h)
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
