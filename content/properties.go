package content

type Properties struct {
	baseNodeWithChildren
}

func NewProperties(children ...*Property) *Properties {
	p := &Properties{}
	p.self = p
	p.childValidator = allowOnlyProperties
	for _, child := range children {
		p.AddChild(child)
	}
	return p
}

func (p *Properties) Get(key string) *Property {
	for _, child := range p.Children() {
		if property, ok := child.(*Property); ok && property.Name == key {
			return property
		}
	}

	return nil
}

func (p *Properties) Set(key string, nodes ...Node) {
	property := p.Get(key)
	if property == nil {
		property = NewProperty(key)
		p.AddChild(property)
	}

	property.SetChildren(nodes...)
}

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

type Property struct {
	baseNodeWithChildren

	Name string
}

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
