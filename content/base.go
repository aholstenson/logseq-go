package content

type Node interface {
	// Parent gets the parent of this node.
	Parent() HasChildren

	setParent(node HasChildren)

	// NextSibling gets the next sibling of this node.
	NextSibling() Node

	setNextSibling(node Node)

	// PreviousSibling gets the previous sibling of this node.
	PreviousSibling() Node

	setPreviousSibling(node Node)

	// RemoveSelf this node from its parent.
	RemoveSelf()

	// ReplaceWith replaces this node with the given node.
	ReplaceWith(node Node)

	debug(*debugPrinter)
}

type baseNode struct {
	parent          HasChildren
	nextSibling     Node
	previousSibling Node
}

func (n *baseNode) Parent() HasChildren {
	return n.parent
}

func (n *baseNode) setParent(node HasChildren) {
	n.parent = node
}

func (n *baseNode) NextSibling() Node {
	return n.nextSibling
}

func (n *baseNode) setNextSibling(node Node) {
	n.nextSibling = node
}

func (n *baseNode) PreviousSibling() Node {
	return n.previousSibling
}

func (n *baseNode) setPreviousSibling(node Node) {
	n.previousSibling = node
}

func (n *baseNode) RemoveSelf() {
	if n.parent != nil {
		n.parent.RemoveChild(n)
	}
}

func (n *baseNode) ReplaceWith(node Node) {
	if n.parent != nil {
		n.parent.InsertBefore(node, n)
		n.RemoveSelf()
	}
}

func (n *baseNode) debug(p *debugPrinter) {
	p.StartType("Unknown")
	p.EndType()
}

var _ Node = (*baseNode)(nil)

type InlineNode interface {
	Node

	isInline()
}

type BlockNode interface {
	Node

	isBlock()
}

type HasChildren interface {
	Node

	// Children gets all the direct children of this node.
	Children() NodeList

	// FirstChild gets the first child of this node.
	FirstChild() Node

	// LastChild gets the last child of this node.
	LastChild() Node

	// SetChildren sets the children of this node, this will remove any
	// existing children.
	SetChildren(node ...Node)

	// AddChild adds a child to this node.
	AddChild(node Node)

	// AddChildren adds multiple children to this node.
	AddChildren(nodes ...Node)

	// PrependChild adds a child to the start of this node.
	PrependChild(node Node)

	// PrependChildren adds multiple children to the start of this node.
	PrependChildren(nodes ...Node)

	// RemoveChild removes a child from this node.
	RemoveChild(node Node) bool

	// RemoveChildren removes multiple children from this node.
	RemoveChildren(nodes ...Node)

	// ReplaceChild replaces a child of this node with another node.
	ReplaceChild(oldNode Node, newNode Node) bool

	// InsertBefore inserts a node before another node.
	InsertBefore(node Node, before Node) bool

	// InsertBefore inserts a node before another node.
	InsertAfter(node Node, after Node) bool
}

type baseNodeWithChildren struct {
	baseNode

	firstChild Node
	lastChild  Node

	self           HasChildren
	childValidator func(Node) bool
}

func (c *baseNodeWithChildren) Children() NodeList {
	children := make([]Node, 0)
	for child := c.firstChild; child != nil; child = child.NextSibling() {
		children = append(children, child)
	}
	return children
}

func (c *baseNodeWithChildren) FirstChild() Node {
	return c.firstChild
}

func (c *baseNodeWithChildren) LastChild() Node {
	return c.lastChild
}

func (c *baseNodeWithChildren) SetChildren(nodes ...Node) {
	// Remove all of the children
	for child := c.firstChild; child != nil; child = child.NextSibling() {
		child.setParent(nil)
		child.setPreviousSibling(nil)
		child.setNextSibling(nil)
	}

	// Add the new children
	c.firstChild = nil
	c.lastChild = nil
	for _, node := range nodes {
		c.AddChild(node)
	}
}

func (c *baseNodeWithChildren) AddChild(node Node) {
	// TODO: Should this return an error instead?
	if c.childValidator != nil && !c.childValidator(node) {
		return
	}

	// If the node is attached somewhere else, remove it
	if node.Parent() != nil {
		node.Parent().RemoveChild(node)
	}

	node.setParent(c.self)

	if c.firstChild == nil {
		c.firstChild = node
		node.setPreviousSibling(nil)
	} else {
		c.lastChild.setNextSibling(node)
		node.setPreviousSibling(c.lastChild)
	}

	c.lastChild = node
	node.setNextSibling(nil)
}

func (c *baseNodeWithChildren) AddChildren(nodes ...Node) {
	for _, node := range nodes {
		c.AddChild(node)
	}
}

func (c *baseNodeWithChildren) PrependChild(node Node) {
	// If the node is attached somewhere else, remove it
	if node.Parent() != nil {
		node.Parent().RemoveChild(node)
	}

	node.setParent(c.self)

	if c.lastChild == nil {
		c.lastChild = node
		node.setNextSibling(nil)
	} else {
		c.firstChild.setPreviousSibling(node)
		node.setNextSibling(c.firstChild)
	}

	c.firstChild = node
	node.setPreviousSibling(nil)
}

func (c *baseNodeWithChildren) PrependChildren(nodes ...Node) {
	// Prepend in reverse order.
	for i := len(nodes) - 1; i >= 0; i-- {
		c.PrependChild(nodes[i])
	}
}

func (c *baseNodeWithChildren) RemoveChild(node Node) bool {
	if node.Parent() != c.self {
		return false
	}

	if node.PreviousSibling() == nil {
		c.firstChild = node.NextSibling()
	} else {
		node.PreviousSibling().setNextSibling(node.NextSibling())
	}

	if node.NextSibling() == nil {
		c.lastChild = node.PreviousSibling()
	} else {
		node.NextSibling().setPreviousSibling(node.PreviousSibling())
	}

	node.setParent(nil)
	node.setNextSibling(nil)
	node.setPreviousSibling(nil)
	return true
}

func (c *baseNodeWithChildren) RemoveChildren(nodes ...Node) {
	for _, node := range nodes {
		c.RemoveChild(node)
	}
}

func (c *baseNodeWithChildren) ReplaceChild(oldNode Node, newNode Node) bool {
	if oldNode.Parent() != c.self {
		return false
	}

	// If the node is attached somewhere else, remove it
	if newNode.Parent() != nil {
		newNode.Parent().RemoveChild(newNode)
	}

	newNode.setParent(c.self)

	if oldNode.PreviousSibling() == nil {
		c.firstChild = newNode
	} else {
		oldNode.PreviousSibling().setNextSibling(newNode)
	}

	if oldNode.NextSibling() == nil {
		c.lastChild = newNode
	} else {
		oldNode.NextSibling().setPreviousSibling(newNode)
	}

	newNode.setPreviousSibling(oldNode.PreviousSibling())
	newNode.setNextSibling(oldNode.NextSibling())

	oldNode.setParent(nil)
	oldNode.setNextSibling(nil)
	oldNode.setPreviousSibling(nil)
	return true
}

func (c *baseNodeWithChildren) InsertBefore(node Node, before Node) bool {
	if before.Parent() != c.self {
		return false
	}

	// If the node is attached somewhere else, remove it
	if node.Parent() != nil {
		node.Parent().RemoveChild(node)
	}

	node.setParent(c.self)

	if before.PreviousSibling() == nil {
		c.firstChild = node
		node.setPreviousSibling(nil)
	} else {
		before.PreviousSibling().setNextSibling(node)
		node.setPreviousSibling(before.PreviousSibling())
	}

	before.setPreviousSibling(node)
	node.setNextSibling(before)

	return true
}

func (c *baseNodeWithChildren) InsertAfter(node Node, after Node) bool {
	if after.Parent() != c.self {
		return false
	}

	// If the node is attached somewhere else, remove it
	if node.Parent() != nil {
		node.Parent().RemoveChild(node)
	}

	node.setParent(c.self)

	if after.NextSibling() == nil {
		c.lastChild = node
		node.setNextSibling(nil)
	} else {
		after.NextSibling().setPreviousSibling(node)
		node.setNextSibling(after.NextSibling())
	}

	after.setNextSibling(node)
	node.setPreviousSibling(after)

	return true
}

var _ HasChildren = (*baseNodeWithChildren)(nil)

func allowOnlyInlineNodes(node Node) bool {
	_, ok := node.(InlineNode)
	return ok
}

func allowOnlyBlockNodes(node Node) bool {
	_, ok := node.(BlockNode)
	return ok
}

func allowInlineNodesAndProperties(node Node) bool {
	_, ok := node.(InlineNode)
	if ok {
		return true
	}

	_, ok = node.(*Properties)
	return ok
}
