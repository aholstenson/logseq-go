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

func IsLink(node Node) bool {
	_, ok := node.(*Link)
	return ok
}

func IsImage(node Node) bool {
	_, ok := node.(*Image)
	return ok
}
