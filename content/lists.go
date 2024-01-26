package content

type ListType int

const (
	ListTypeOrdered ListType = iota
	ListTypeUnordered
)

type List struct {
	baseNodeWithChildren
	previousLineAwareImpl

	Type   ListType
	Marker byte
}

func NewList(typ ListType, items ...*ListItem) *List {
	list := &List{
		Type: typ,
	}
	list.self = list
	for _, item := range items {
		list.AddChild(item)
	}
	switch typ {
	case ListTypeOrdered:
		list.Marker = '.'
	case ListTypeUnordered:
		list.Marker = '*'
	}
	return list
}

func NewListFromMarker(marker byte, items ...*ListItem) *List {
	var list *List
	switch marker {
	case '*', '+':
		list = NewUnorderedList(items...)
	case '-':
		list = NewUnorderedList(items...)
		marker = '*'
	case '.', ')':
		list = NewOrderedList(items...)
	default:
		list = NewUnorderedList(items...)
	}
	list.Marker = marker
	return list
}

func NewUnorderedList(items ...*ListItem) *List {
	return NewList(ListTypeUnordered, items...)
}

func NewOrderedList(items ...*ListItem) *List {
	return NewList(ListTypeOrdered, items...)
}

func (l *List) debug(p *debugPrinter) {
	p.StartType("List")
	switch l.Type {
	case ListTypeOrdered:
		p.Field("type", "ordered")
	case ListTypeUnordered:
		p.Field("type", "unordered")
	}
	p.Field("marker", string(l.Marker))
	debugPreviousLineAware(p, l)
	p.Children(l)
	p.EndType()
}

func (c *List) WithPreviousLineType(t PreviousLineType) *List {
	c.previousLineType = t
	return c
}

func (l *List) isBlock() {}

var _ Node = (*List)(nil)
var _ BlockNode = (*List)(nil)

type ListSection struct {
	baseNodeWithChildren
	marker byte
}

func (l *ListSection) debug(p *debugPrinter) {
	p.StartType("ListSection")
	p.Children(l)
	p.EndType()
}

type ListItem struct {
	baseNodeWithChildren
}

func NewListItem(items ...Node) *ListItem {
	li := &ListItem{}
	li.self = li
	li.AddChildren(addAutomaticParagraphs(items)...)
	return li
}

func (l *ListItem) debug(p *debugPrinter) {
	p.StartType("ListItem")
	p.Children(l)
	p.EndType()
}

func (l *ListItem) isBlock() {}

var _ Node = (*ListItem)(nil)
var _ BlockNode = (*ListItem)(nil)
