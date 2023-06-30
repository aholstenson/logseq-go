package content

type AdvancedCommand struct {
	baseNode

	Type  string
	Value string
}

func NewAdvancedCommand(variant string, value string) *AdvancedCommand {
	return &AdvancedCommand{
		Type:  variant,
		Value: value,
	}
}

func (h *AdvancedCommand) debug(p *debugPrinter) {
	p.StartType("AdvancedCommand")
	p.Field("type", h.Type)
	p.Field("value", h.Value)
	p.EndType()
}

func (h *AdvancedCommand) isBlock() {}

var _ BlockNode = (*AdvancedCommand)(nil)
