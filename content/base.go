package content

// Node is the basic building block of a document. Nodes represent a single
// element in the document, such as a paragraph, heading, list or text.
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
		n.parent.InsertChildBefore(node, n)
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

	// FirstChild gets the first child of this node. Can be nil if there are
	// no children. Can be the same as LastChild if there is only one child.
	FirstChild() Node

	// LastChild gets the last child of this node. Can be nil if there are no
	// children. Can be the same as FirstChild if there is only one child.
	LastChild() Node

	// SetChildren sets the children of this node, this will remove any
	// existing children.
	SetChildren(node ...Node)

	// AddChild adds a child to this node. The child will be added to the end
	// of the list of children. Nodes can only have one parent, if the node
	// already has a parent it will be removed from that parent.
	AddChild(node Node)

	// AddChildren adds multiple children to this node. The children will be
	// added to the end of the list of children. Nodes can only have one
	// parent, if any of the nodes already have a parent they will be removed
	AddChildren(nodes ...Node)

	// PrependChild adds a child to the start of this node. Nodes can only
	// have one parent, if the node already has a parent it will be removed
	// from that parent.
	PrependChild(node Node)

	// PrependChildren adds multiple children to the start of this node. Nodes
	// can only have one parent, if any of the nodes already have a parent
	// they will be removed from that parent.
	PrependChildren(nodes ...Node)

	// RemoveChild removes a child from this node. Returns true if the child
	// was removed, false if the child was not a child of this node.
	RemoveChild(node Node) bool

	// RemoveChildren removes multiple children from this node. Returns true
	// if all of the children were removed, false if any of the children were
	// not a child of this node.
	RemoveChildren(nodes ...Node)

	// ReplaceChild replaces a child of this node with another node. Returns
	// true if the child was replaced, false if the old node was not a child
	// of this node.
	ReplaceChild(oldNode Node, newNode Node) bool

	// InsertChildBefore inserts a node before another node. Returns true if
	// the node was inserted, false if the before node was not a child of
	// this node.
	InsertChildBefore(node Node, before Node) bool

	// InsertChildBefore inserts a node before another node. Returns true if
	// the node was inserted, false if the before node was not a child of
	// this node.
	InsertChildAfter(node Node, after Node) bool
}

// PreviousLineType contains information about the previous line, commonly
// saved while parsing.
type PreviousLineType int

const (
	// PreviousLineTypeAutomatic is the default value, which lets block nodes
	// pick what kind of previous line type they want.
	PreviousLineTypeAutomatic PreviousLineType = iota
	// PreviousLineTypeBlank is used when the previous line was blank.
	PreviousLineTypeBlank
	// PreviousLineTypeNonBlank is used when the previous line was not blank.
	// Such as when something interrupts as paragraph.
	PreviousLineTypeNonBlank
)

// PreviousLineAware is an interface that can be implemented by block nodes
// to get information about the previous line.
//
// This information is used when generating Markdown output to keep the
// Markdown close to the original input.
//
// Most implementations of this interface are encouraged to also provide a
// helper method `WithPreviousLineType` that sets the previous line type
// and returns the node.
type PreviousLineAware interface {
	// PreviousLineType gets the type of the previous line.
	PreviousLineType() PreviousLineType

	// SetPreviousLineType sets the type of the previous line.
	SetPreviousLineType(PreviousLineType)
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

func (c *baseNodeWithChildren) InsertChildBefore(node Node, before Node) bool {
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

func (c *baseNodeWithChildren) InsertChildAfter(node Node, after Node) bool {
	if after.Parent() != c.self {
		return false
	}

	// If the node is attached somewhere else, remove it
	if node.Parent() != nil {
		node.Parent().RemoveChild(node)
	}

	node.setParent(c.self)

	next := after.NextSibling()
	if next == nil {
		c.lastChild = node
		node.setNextSibling(nil)
	} else {
		next.setPreviousSibling(node)
		node.setNextSibling(next)
	}

	after.setNextSibling(node)
	node.setPreviousSibling(after)

	return true
}

var _ HasChildren = (*baseNodeWithChildren)(nil)

type previousLineAwareImpl struct {
	previousLineType PreviousLineType
}

func (p *previousLineAwareImpl) PreviousLineType() PreviousLineType {
	return p.previousLineType
}

func (p *previousLineAwareImpl) SetPreviousLineType(previousLineType PreviousLineType) {
	p.previousLineType = previousLineType
}

var _ PreviousLineAware = (*previousLineAwareImpl)(nil)

func debugPreviousLineAware(p *debugPrinter, node PreviousLineAware) {
	switch node.PreviousLineType() {
	case PreviousLineTypeAutomatic:
		p.Field("previousLineType", "automatic")
	case PreviousLineTypeBlank:
		p.Field("previousLineType", "blank")
	case PreviousLineTypeNonBlank:
		p.Field("previousLineType", "non-blank")
	}
}

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
