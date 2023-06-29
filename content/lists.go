package content

type ListType int

const (
	ListTypeOrdered ListType = iota
	ListTypeUnordered
)

type List struct {
	baseNodeWithChildren

	Type ListType
}

func NewList(typ ListType, items ...*ListItem) *List {
	list := &List{
		Type: typ,
	}
	list.self = list
	for _, item := range items {
		list.AddChild(item)
	}
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
	p.Children(l)
	p.EndType()
}

func (l *List) isBlock() {}

var _ Node = (*List)(nil)
var _ BlockNode = (*List)(nil)

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