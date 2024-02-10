package content

type ListType int

const (
	ListTypeOrdered ListType = iota
	ListTypeUnordered
)

type List struct {
	baseNodeWithChildren
	previousLineAwareImpl

	// Type is the type of the list.
	Type ListType

	// Marker is the marker of the list.
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

// WithType sets the type of the list.
func (l *List) WithType(typ ListType) *List {
	l.Type = typ

	// If the marker is not valid for the type, change it to the default.
	switch typ {
	case ListTypeOrdered:
		if l.Marker != '.' && l.Marker != ')' {
			l.Marker = '.'
		}
	case ListTypeUnordered:
		if l.Marker != '*' && l.Marker != '+' {
			l.Marker = '*'
		}
	}

	return l
}

// WithMarker sets the marker of the list.
func (l *List) WithMarker(marker byte) *List {
	// Update the type to make sure it's valid for the marker.
	switch marker {
	case '*', '+':
		l.Type = ListTypeUnordered
		l.Marker = marker
	case '-':
		l.Type = ListTypeUnordered
		l.Marker = '*'
	case '.', ')':
		l.Type = ListTypeOrdered
		l.Marker = marker
	default:
		// Unknown type of marker, default to unordered.
		l.Type = ListTypeUnordered
		l.Marker = '*'
	}
	return l
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
	li.AddChildren(AddAutomaticParagraphs(items)...)
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
