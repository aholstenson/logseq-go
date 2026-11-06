package content

type NodePredicate func(node Node) bool

type NodeList []Node

func (n NodeList) Find(predicate NodePredicate) Node {
	for _, node := range n {
		if predicate(node) {
			return node
		}
	}

	return nil
}

func (n NodeList) FindDeep(predicate NodePredicate) Node {
	for _, node := range n {
		if predicate(node) {
			return node
		}

		if children, ok := node.(HasChildren); ok {
			if found := children.Children().FindDeep(predicate); found != nil {
				return found
			}
		}
	}

	return nil
}

func (n NodeList) Filter(predicate NodePredicate) NodeList {
	filtered := make([]Node, 0)
	for _, node := range n {
		if predicate(node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

func (n NodeList) FilterDeep(predicate NodePredicate) NodeList {
	filtered := make([]Node, 0)
	for _, node := range n {
		if predicate(node) {
			filtered = append(filtered, node)
		}

		if children, ok := node.(HasChildren); ok {
			filtered = append(filtered, children.Children().FilterDeep(predicate)...)
		}
	}

	return filtered
}

func (n NodeList) Map(mapper func(node Node) Node) NodeList {
	mapped := make([]Node, len(n))
	for i, node := range n {
		mapped[i] = mapper(node)
	}

	return mapped
}

func IsOfType[T Node]() NodePredicate {
	return func(node Node) bool {
		_, ok := node.(T)
		return ok
	}
}

func IsEither(a NodePredicate, b NodePredicate) NodePredicate {
	return func(node Node) bool {
		return a(node) || b(node)
	}
}

func IsBoth(a NodePredicate, b NodePredicate) NodePredicate {
	return func(node Node) bool {
		return a(node) && b(node)
	}
}

// IsPageReference matches nodes that point at a page by its title, such as
// page links, hashtags and page embeds.
func IsPageReference() NodePredicate {
	return IsOfType[PageRef]()
}

// PageReferences finds every reference to a page in these nodes and the nodes
// below them, which is what makes a page show up in the linked references of
// another. A page link in the title of a page link or tag is a reference too,
// see NestedPageRef. The values of the properties that ignore references are left out,
// as Logseq reads those as text rather than as pointing at a page. The same
// goes for a value that is wrapped in quotes.
//
// The text after a SCHEDULED or DEADLINE line is left out too, until a node
// that is not text. Logseq reads that text and the date as one paragraph that
// starts with a date, and it does not take references from such a paragraph.
func (n NodeList) PageReferences() NodeList {
	refs := make(NodeList, 0)
	afterTaskDate := false
	for _, node := range n {
		if property, ok := node.(*Property); ok && (property.PageRefsIgnored || property.Quoted) {
			continue
		}

		switch node.(type) {
		case *TaskDate:
			afterTaskDate = true
		case *Paragraph:
			if afterTaskDate {
				continue
			}
		default:
			afterTaskDate = false
		}

		if IsPageReference()(node) {
			refs = append(refs, node)
			refs = append(refs, nestedPageRefsOf(node.(PageRef))...)
		}

		if children, ok := node.(HasChildren); ok {
			refs = append(refs, children.Children().PageReferences()...)
		}
	}

	return refs
}

// IsQuery matches the nodes that hold a Logseq query: a simple query written
// as `{{query ...}}`, which is a *Query, and an advanced query written between
// `#+BEGIN_QUERY` and `#+END_QUERY`, which is a *QueryCommand. The query
// package reads both with query.ParseNode.
func IsQuery() NodePredicate {
	return IsEither(IsOfType[*Query](), IsOfType[*QueryCommand]())
}

// BlockOf returns the closest block that holds the node, or the node itself
// if it is a block. Returns nil if the node is not in a block.
func BlockOf(node Node) *Block {
	for node != nil {
		if block, ok := node.(*Block); ok {
			return block
		}

		parent := node.Parent()
		if parent == nil {
			return nil
		}
		node = parent
	}

	return nil
}
