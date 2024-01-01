package content

// Properties is a collection of Property nodes.
type Properties struct {
	baseNodeWithChildren
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

// Get a Property node by name. Will return nil if no Property with the given name exists.
func (p *Properties) Get(key string) *Property {
	for _, child := range p.Children() {
		if property, ok := child.(*Property); ok && property.Name == key {
			return property
		}
	}

	return nil
}

// Set a Property node by name. If a Property with the given name already exists, it will be replaced.
func (p *Properties) Set(key string, nodes ...Node) {
	property := p.Get(key)
	if property == nil {
		property = NewProperty(key)
		p.AddChild(property)
	}

	property.SetChildren(nodes...)
}

// Remove a Property node by name. If a Property with the given name does not exist this does nothing.
func (p *Properties) Remove(key string) {
	property := p.Get(key)
	if property != nil {
		p.RemoveChild(property)
	}
}

func (p *Properties) debug(p2 *debugPrinter) {
	p2.StartType("Properties")
	p2.Children(p)
	p2.EndType()
}

var _ Node = (*Properties)(nil)

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
