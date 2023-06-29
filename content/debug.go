package content

import "strings"

type debugPrinter struct {
	builder strings.Builder
	level   int

	didWrite []bool
}

func newDebugPrinter() *debugPrinter {
	return &debugPrinter{
		didWrite: make([]bool, 50),
	}
}

func (p *debugPrinter) String() string {
	return p.builder.String()
}

func (p *debugPrinter) StartType(name string) {
	p.didWrite[p.level] = true
	p.builder.WriteString(name)
	p.builder.WriteString("{")
	p.level++
	p.didWrite[p.level] = false
}

func (p *debugPrinter) EndType() {
	p.level--
	if p.didWrite[p.level+1] {
		p.builder.WriteString(strings.Repeat("  ", p.level))
	}
	p.builder.WriteString("}")
}

func (p *debugPrinter) Field(name string, value string) {
	if !p.didWrite[p.level] {
		p.didWrite[p.level] = true
		p.builder.WriteString("\n")
	}

	p.builder.WriteString(strings.Repeat("  ", p.level))
	p.builder.WriteString(name)
	p.builder.WriteString("='")
	p.builder.WriteString(value)
	p.builder.WriteString("'\n")
}

func (p *debugPrinter) Children(self HasChildren) {
	if !p.didWrite[p.level] {
		p.didWrite[p.level] = true
		p.builder.WriteString("\n")
	}

	p.builder.WriteString(strings.Repeat("  ", p.level))
	p.builder.WriteString("children=[")
	p.level++
	p.didWrite[p.level] = false

	for child := self.FirstChild(); child != nil; child = child.NextSibling() {
		p.builder.WriteString("\n")
		p.builder.WriteString(strings.Repeat("  ", p.level))

		child.debug(p)
	}

	if p.didWrite[p.level] {
		p.builder.WriteString("\n")
		p.builder.WriteString(strings.Repeat("  ", p.level-1))
	}
	p.builder.WriteString("]\n")
	p.level--
}

func Debug(node Node) string {
	printer := newDebugPrinter()
	node.debug(printer)
	return printer.String()
}
