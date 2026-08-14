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
// another. The values of the properties that ignore references are left out,
// as Logseq reads those as text rather than as pointing at a page.
func (n NodeList) PageReferences() NodeList {
	refs := make(NodeList, 0)
	for _, node := range n {
		if property, ok := node.(*Property); ok && property.PageRefsIgnored {
			continue
		}

		if IsPageReference()(node) {
			refs = append(refs, node)
		}

		if children, ok := node.(HasChildren); ok {
			refs = append(refs, children.Children().PageReferences()...)
		}
	}

	return refs
}
