package content

import (
	"strings"
	"unicode"
)

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

	// PageRefsIgnored is whether the pages named in the values of these
	// properties are only text rather than references to those pages.
	// Logseq reads only the first group of properties of a block, so the
	// groups after it in the same block are read with this set, unless the
	// text is read with markdown.WithLaterPropertyGroupRefs.
	PageRefsIgnored bool
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

// WithPageRefsIgnored sets whether the pages named in the values of these
// properties are only text, see PageRefsIgnored.
func (p *Properties) WithPageRefsIgnored(ignored bool) *Properties {
	p.PageRefsIgnored = ignored
	return p
}

func (p *Properties) WithPreviousLineType(t PreviousLineType) *Properties {
	p.previousLineType = t
	return p
}

// GetAsNode gets a Property node by name. Will return nil if no Property with
// the given name exists. Names are compared the way Logseq compares them, see
// NormalizePropertyName.
//
// If the name is written more than once, the last Property with the name is
// returned, as Logseq uses the last value.
func (p *Properties) GetAsNode(key string) *Property {
	return p.lastNamed(key)
}

// lastNamed gets the last Property that has one of the given names. Names are
// compared the way Logseq compares them, and a Property with a name that
// Logseq does not keep is skipped, see Property.InvalidName.
func (p *Properties) lastNamed(keys ...string) *Property {
	for i := range keys {
		keys[i] = NormalizePropertyName(keys[i])
	}

	var last *Property
	for _, child := range p.Children() {
		property, ok := child.(*Property)
		if !ok || property.InvalidName {
			continue
		}

		name := NormalizePropertyName(property.Name)
		for _, key := range keys {
			if name == key {
				last = property
			}
		}
	}

	return last
}

// IsOverridden checks if a later Property in the same properties has the same
// name as the given Property, so that Logseq uses the value of the later one
// instead. Names are compared the way Logseq compares them.
func (p *Properties) IsOverridden(property *Property) bool {
	return p.lastNamed(property.Name) != property
}

// Get gets the value of a Property by name. Will return an empty slice if no
// Property with the given name exists. If the name is written more than once,
// the value of the last Property with the name is returned.
func (p *Properties) Get(key string) NodeList {
	property := p.GetAsNode(key)
	if property == nil {
		return NodeList{}
	}

	return property.Children()
}

// Set a Property node by name. If a Property with the given name already
// exists, its value is replaced and it keeps the name it was written with. If
// the name is written more than once, the value of the last one is replaced.
func (p *Properties) Set(key string, nodes ...Node) {
	property := p.GetAsNode(key)
	if property == nil {
		property = NewProperty(key)
		p.AddChild(property)
	}

	property.SetChildren(nodes...)
}

// Remove a Property node by name. If the name is written more than once, all
// the Property nodes with the name are removed. If a Property with the given
// name does not exist this does nothing.
func (p *Properties) Remove(key string) {
	for property := p.GetAsNode(key); property != nil; property = p.GetAsNode(key) {
		p.RemoveChild(property)
	}
}

func (p *Properties) debug(p2 *debugPrinter) {
	p2.StartType("Properties")
	if p.Drawer {
		p2.Field("Drawer", "true")
	}
	if p.PageRefsIgnored {
		p2.Field("PageRefsIgnored", "true")
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
//
// The name is made lower case in the same way as in JavaScript, where Logseq
// does it. That is different from strings.ToLower for two letters: `İ`
// becomes `i̇`, an `i` with a combining dot, and a `Σ` at the end of a word
// becomes the final form `ς`.
func NormalizePropertyName(name string) string {
	return strings.ReplaceAll(lowerCaseAsJavaScript(name), "_", "-")
}

// lowerCaseAsJavaScript makes the text lower case in the same way as
// toLowerCase in JavaScript. It uses the mappings of Unicode that do not
// depend on the language, which are the simple mappings of strings.ToLower,
// the mapping of `İ` to two characters, and the final form of sigma.
func lowerCaseAsJavaScript(s string) string {
	if !strings.ContainsAny(s, "\u0130\u03a3") {
		return strings.ToLower(s)
	}

	runes := []rune(s)
	var lower strings.Builder
	for i, r := range runes {
		switch {
		case r == '\u0130':
			lower.WriteString("i\u0307")
		case r == '\u03a3' && isFinalSigma(runes, i):
			lower.WriteRune('\u03c2')
		default:
			lower.WriteRune(unicode.ToLower(r))
		}
	}

	return lower.String()
}

// isFinalSigma checks if the sigma at the given index is at the end of a
// word, which is the Final_Sigma condition of Unicode: a cased letter comes
// before it and no cased letter comes after it, when the characters that
// case ignores are skipped.
func isFinalSigma(runes []rune, i int) bool {
	before := false
	for j := i - 1; j >= 0; j-- {
		if !isCaseIgnorable(runes[j]) {
			before = isCased(runes[j])
			break
		}
	}

	if !before {
		return false
	}

	for j := i + 1; j < len(runes); j++ {
		if !isCaseIgnorable(runes[j]) {
			return !isCased(runes[j])
		}
	}

	return true
}

// isCased checks if a character is cased in Unicode: it is an upper case, a
// lower case or a title case letter.
func isCased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) ||
		unicode.In(r, unicode.Other_Lowercase, unicode.Other_Uppercase)
}

// isCaseIgnorable checks if case ignores a character in Unicode, such as
// marks, format characters, modifiers and the apostrophe, the period and
// the colon.
func isCaseIgnorable(r rune) bool {
	switch r {
	case '\'', '.', ':', '\u00b7', '\u0387', '\u055f', '\u05f4', '\u2018', '\u2019',
		'\u2024', '\u2027', '\ufe13', '\ufe52', '\ufe55', '\uff07', '\uff0e', '\uff1a':
		return true
	}

	return unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk)
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

	// InvalidName is whether the name of this property is one that Logseq
	// does not keep, such as `#tag` or `a(b` in `a(b:: c`. Logseq reads such a
	// line as a part of the group of properties, so the properties after it
	// are still in the same group, but it drops the property and the pages
	// named in its value. Such a property is not found by name with
	// Properties.Get, and its value refers to no page. It is kept so that the
	// line does not change when it is written.
	InvalidName bool

	// SpaceBeforeValue is the spaces and tabs that are written after the
	// separator and its one space, before the value, such as the second space
	// in `id::  <uuid>`. Logseq does not read them as part of the value. They
	// are kept so that the text of the property does not change when it is
	// written.
	SpaceBeforeValue string

	// TrailingSpace is the spaces and tabs at the end of the line of the
	// property. For a property with a value, they come after the value. For a
	// property with an empty value, they come after the separator and the
	// space that is written after it, see NoSpaceAfterSeparator. Logseq does
	// not read them as part of the value. They are kept so that the text of
	// the property does not change when it is written.
	TrailingSpace string

	// NoSpaceAfterSeparator is whether a property with an empty value is
	// written as `name::`, with no space after the separator. By default it
	// is written as `name:: `. Logseq reads both in the same way. It is not
	// used for a property with a value, or for a directive, which is written
	// as `#+name:` when its value is empty.
	NoSpaceAfterSeparator bool
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

// WithInvalidName sets whether the name of this property is one that Logseq
// does not keep, see InvalidName.
func (p *Property) WithInvalidName(invalid bool) *Property {
	p.InvalidName = invalid
	return p
}

// WithSpaceBeforeValue sets the spaces and tabs that are written before the
// value, after the separator and its one space.
func (p *Property) WithSpaceBeforeValue(space string) *Property {
	p.SpaceBeforeValue = space
	return p
}

// WithTrailingSpace sets the spaces and tabs at the end of the line of the
// property.
func (p *Property) WithTrailingSpace(space string) *Property {
	p.TrailingSpace = space
	return p
}

// WithNoSpaceAfterSeparator sets whether a property with an empty value is
// written as `name::` instead of `name:: `.
func (p *Property) WithNoSpaceAfterSeparator(noSpace bool) *Property {
	p.NoSpaceAfterSeparator = noSpace
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
	if p.InvalidName {
		p2.Field("InvalidName", "true")
	}
	if p.SpaceBeforeValue != "" {
		p2.Field("SpaceBeforeValue", p.SpaceBeforeValue)
	}
	if p.TrailingSpace != "" {
		p2.Field("TrailingSpace", p.TrailingSpace)
	}
	if p.NoSpaceAfterSeparator {
		p2.Field("NoSpaceAfterSeparator", "true")
	}
	p2.Children(p)
	p2.EndType()
}

func allowOnlyProperties(node Node) bool {
	_, ok := node.(*Property)
	return ok
}
