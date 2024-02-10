package content

// Properties is a collection of Property nodes.
type Properties struct {
	baseNodeWithChildren
	previousLineAwareImpl
}

// NewProperties creates a new Properties node with the given Property children.
func NewProperties(children ...*Property) *Properties {
	p := &Properties{}
	p.self = p
	p.childValidator = allowOnlyProperties
	for _, child := range children {
		p.AddChild(child)
	}
	return p
}

func (p *Properties) WithPreviousLineType(t PreviousLineType) *Properties {
	p.previousLineType = t
	return p
}

// GetAsNode gets a Property node by name. Will return nil if no Property with the given name exists.
func (p *Properties) GetAsNode(key string) *Property {
	for _, child := range p.Children() {
		if property, ok := child.(*Property); ok && property.Name == key {
			return property
		}
	}

	return nil
}

// Get gets the value of a Property by name. Will return an empty slice if no Property with the given name exists.
func (p *Properties) Get(key string) NodeList {
	property := p.GetAsNode(key)
	if property == nil {
		return NodeList{}
	}

	return property.Children()
}

// Set a Property node by name. If a Property with the given name already exists, it will be replaced.
func (p *Properties) Set(key string, nodes ...Node) {
	property := p.GetAsNode(key)
	if property == nil {
		property = NewProperty(key)
		p.AddChild(property)
	}

	property.SetChildren(nodes...)
}

// Remove a Property node by name. If a Property with the given name does not exist this does nothing.
func (p *Properties) Remove(key string) {
	property := p.GetAsNode(key)
	if property != nil {
		p.RemoveChild(property)
	}
}

func (p *Properties) debug(p2 *debugPrinter) {
	p2.StartType("Properties")
	debugPreviousLineAware(p2, p)
	p2.Children(p)
	p2.EndType()
}

func (p *Properties) isBlock() {}

var _ Node = (*Properties)(nil)
var _ BlockNode = (*Properties)(nil)

// Property is a node that represents a property, which is a key that can have multiple values.
type Property struct {
	baseNodeWithChildren

	// Name is the name of the property.
	Name string
}

// NewProperty creates a new Property node with the given name and values.
func NewProperty(name string, children ...Node) *Property {
	property := &Property{Name: name}
	property.self = property
	property.childValidator = allowOnlyInlineNodes
	property.AddChildren(children...)
	return property
}

// WithName sets the name of the property.
func (p *Property) WithName(name string) *Property {
	p.Name = name
	return p
}

func (p *Property) debug(p2 *debugPrinter) {
	p2.StartType("Property")
	p2.Field("Name", p.Name)
	p2.Children(p)
	p2.EndType()
}

func allowOnlyProperties(node Node) bool {
	_, ok := node.(*Property)
	return ok
}
