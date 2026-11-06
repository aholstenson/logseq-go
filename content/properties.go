package content

import "strings"

// Properties is a collection of Property nodes.
type Properties struct {
	baseNodeWithChildren
	previousLineAwareImpl

	// Drawer is whether the properties are written in a property drawer,
	// which is the form Logseq used before `name:: value` lines and still
	// reads:
	//
	//	:PROPERTIES:
	//	:name: value
	//	:END:
	Drawer bool

	// DrawerLowerCase is whether the markers of the drawer are written in
	// lower case, as `:properties:` and `:end:`.
	DrawerLowerCase bool
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

// WithDrawer sets whether the properties are written in a property drawer.
func (p *Properties) WithDrawer(drawer bool) *Properties {
	p.Drawer = drawer
	return p
}

func (p *Properties) WithPreviousLineType(t PreviousLineType) *Properties {
	p.previousLineType = t
	return p
}

// GetAsNode gets a Property node by name. Will return nil if no Property with
// the given name exists. Names are compared the way Logseq compares them, see
// NormalizePropertyName.
func (p *Properties) GetAsNode(key string) *Property {
	key = NormalizePropertyName(key)
	for _, child := range p.Children() {
		if property, ok := child.(*Property); ok && NormalizePropertyName(property.Name) == key {
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

// Set a Property node by name. If a Property with the given name already
// exists, its value is replaced and it keeps the name it was written with.
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
	if p.Drawer {
		p2.Field("Drawer", "true")
	}
	debugPreviousLineAware(p2, p)
	p2.Children(p)
	p2.EndType()
}

func (p *Properties) isBlock() {}

var _ Node = (*Properties)(nil)
var _ BlockNode = (*Properties)(nil)

// NormalizePropertyName returns the form of a property name that Logseq uses
// to identify the property. Logseq makes the name lower case and reads an
// underscore as a dash, so `My_Prop::` and `my-prop::` are the same property.
func NormalizePropertyName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}

// Property is a node that represents a property, which is a key that can have multiple values.
type Property struct {
	baseNodeWithChildren

	// Name is the name of the property as it is written. Use
	// NormalizePropertyName to compare names.
	Name string

	// PageRefsIgnored is whether the pages named in the value of this property
	// are only text rather than references to those pages, which is what
	// `:ignored-page-references-keywords` sets up. A page is not found via the
	// value of such a property, not even where the value is written as a link.
	PageRefsIgnored bool

	// Quoted is whether the value of this property is wrapped in double
	// quotes, such as `alias:: "Smith, John"`. Logseq reads such a value as
	// text, so it names no pages and is not split on commas.
	Quoted bool

	// Directive is whether the property is written as an Org-mode directive,
	// `#+name: value`, instead of as `name:: value`. Logseq also reads
	// directives in Markdown files, and uses them as page properties.
	Directive bool
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

// WithPageRefsIgnored sets whether the pages named in the value of this
// property are only text rather than references to those pages.
func (p *Property) WithPageRefsIgnored(ignored bool) *Property {
	p.PageRefsIgnored = ignored
	return p
}

// WithQuoted sets whether the value of this property is wrapped in double
// quotes.
func (p *Property) WithQuoted(quoted bool) *Property {
	p.Quoted = quoted
	return p
}

// WithDirective sets whether the property is written as a directive,
// `#+name: value`.
func (p *Property) WithDirective(directive bool) *Property {
	p.Directive = directive
	return p
}

func (p *Property) debug(p2 *debugPrinter) {
	p2.StartType("Property")
	p2.Field("Name", p.Name)
	if p.PageRefsIgnored {
		p2.Field("PageRefsIgnored", "true")
	}
	if p.Quoted {
		p2.Field("Quoted", "true")
	}
	if p.Directive {
		p2.Field("Directive", "true")
	}
	p2.Children(p)
	p2.EndType()
}

func allowOnlyProperties(node Node) bool {
	_, ok := node.(*Property)
	return ok
}
