package content

type AdvancedComamnd struct {
	baseNode

	Type  string
	Value string
}

func NewAdvancedCommand(variant string, value string) *AdvancedComamnd {
	return &AdvancedComamnd{
		Type:  variant,
		Value: value,
	}
}

func (h *AdvancedComamnd) debug(p *debugPrinter) {
	p.StartType("AdvancedCommand")
	p.Field("type", h.Type)
	p.Field("value", h.Value)
	p.EndType()
}

func (h *AdvancedComamnd) isBlock() {}

var _ BlockNode = (*AdvancedComamnd)(nil)
