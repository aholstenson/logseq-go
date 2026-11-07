package markdown

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// findPropertyName finds the name of a property at the start of a line, which
// is written as `name::`. It returns the name and where the text after the `::`
// starts.
//
// Logseq finds a property in two steps, and both are done here. First its
// parser, mldoc, finds the name, see findPropertyKey. Then Logseq keeps the
// property only if its name is a valid property name, see validPropertyName.
func findPropertyName(line string) (name string, end int, ok bool) {
	name, end, ok = findPropertyKey(line)
	if !ok || !validPropertyName(name) {
		return "", 0, false
	}

	return name, end, true
}

// findPropertyKey finds the name of a property at the start of a line in the
// same way as mldoc, the parser of Logseq: the name is all the text up to
// `::` that is not a colon, a space or a line break. The name is not checked,
// so it can be a name that Logseq does not keep, such as `#tag` or `a(b`.
// It returns the name and where the text after the `::` starts.
func findPropertyKey(line string) (name string, end int, ok bool) {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ':':
			if i == 0 || !strings.HasPrefix(line[i:], "::") {
				return "", 0, false
			}

			return line[:i], i + 2, true
		case ' ', '\t', '\r', '\n', '\f', '\x1a':
			return "", 0, false
		}
	}

	return "", 0, false
}

// validPropertyName checks if Logseq keeps a property with the given name.
// Logseq turns the name into a keyword, so the name must be one that reads as
// a keyword in EDN, the data format of Logseq, see validEDNKeyword. A name can
// hold letters of any language, digits and most punctuation, but some
// characters are not allowed.
func validPropertyName(name string) bool {
	name = normalizePropertyName(name)
	if name == "" || name[0] == '#' {
		// A name that starts with # is a tag
		return false
	}

	// The characters that Logseq does not allow before it reads the name
	if strings.ContainsAny(name, "\"|^(){}") {
		return false
	}

	return validEDNKeyword(name)
}

var propertiesKind = ast.NewNodeKind("Properties")

type properties struct {
	ast.BaseBlock

	// drawer is whether the properties are written in a `:PROPERTIES:`
	// drawer, and drawerLowerCase whether its markers are in lower case.
	drawer          bool
	drawerLowerCase bool
}

func (*properties) Kind() ast.NodeKind {
	return propertiesKind
}

func (n *properties) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

var propertyKind = ast.NewNodeKind("Property")

type property struct {
	ast.BaseBlock
	Name string

	// PageRefsIgnored is whether the pages named in the value are read as text
	// instead of as references, also where they are written as links. Logseq
	// does this for some of its built-in properties, see
	// builtInPropertiesWithoutPageReferences.
	PageRefsIgnored bool

	// Quoted is whether the value is wrapped in double quotes, which Logseq
	// reads as text.
	Quoted bool

	// Directive is whether the property is written as `#+name: value`.
	Directive bool

	// InvalidName is whether Logseq does not keep the property, as its name
	// is not a valid property name, see validPropertyName.
	InvalidName bool

	// SpaceBeforeValue is the spaces and tabs between the separator and the
	// value that are not part of the value. See skipSpaceBeforeValue.
	SpaceBeforeValue string

	// TrailingSpace and NoSpaceAfterSeparator keep the spaces at the end of
	// the line of the property. See propertyLineSpace.
	TrailingSpace         string
	NoSpaceAfterSeparator bool
}

func (*property) Kind() ast.NodeKind {
	return propertyKind
}

func (n *property) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

var pageRefTextKind = ast.NewNodeKind("PageRefText")

// pageRefText is a page referenced by its title alone, without any markup
// around it, which is how the values of the properties that hold a list of
// pages are written.
type pageRefText struct {
	ast.BaseInline

	// Segment is where the title of the page is in the source.
	Segment text.Segment
}

func (*pageRefText) Kind() ast.NodeKind {
	return pageRefTextKind
}

func (n *pageRefText) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type propertiesASTTransformer struct {
}

var defaultPropertiesASTTransformer = &propertiesASTTransformer{}

// Transform paragraphs by looking for properties. A property in Logseq can
// interrupt a paragraph and looks like `key:: value`.
//
// We go through text nodes and handle those that are on a new line and contain
// a property name and then adopt all of the following text nodes until we find
// a line break.
//
// If a property is found just after another property, it is considered to be
// part of the same properties block.
func (t *propertiesASTTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	// The paragraphs are found before they are changed, as a paragraph that
	// only holds properties is removed, and a removed node has no next sibling
	// for the walk to continue to.
	var paragraphs []ast.Node
	_ = ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && (node.Kind() == ast.KindParagraph || node.Kind() == ast.KindTextBlock) {
			paragraphs = append(paragraphs, node)
		}

		return ast.WalkContinue, nil
	})

	for _, paragraph := range paragraphs {
		t.transformTextBlockOrParagraph(paragraph, reader)
	}

	// How the value of a property is read depends on the settings of the graph,
	// which are applied once the properties have been found.
	options, _ := pc.Get(parseOptionsKey).(*parseOptions)
	if options == nil {
		defaults := defaultParseOptions()
		options = &defaults
	}

	_ = ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if properties, ok := node.(*properties); ok && entering {
			applyOptionsToProperties(properties, reader, options)
		}

		return ast.WalkContinue, nil
	})
}

// applyOptionsToProperties reads the values of a block of properties the way
// the settings of the graph say they should be read.
func applyOptionsToProperties(node *properties, reader text.Reader, options *parseOptions) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		property, ok := child.(*property)
		if !ok {
			continue
		}

		if isQuotedPropertyValue(property, reader.Source()) {
			// Logseq reads a value in quotes as text, which also keeps the
			// commas in it from splitting it.
			property.Quoted = true
			continue
		}

		if options.hasNoPageReferences(property.Name) {
			// The value of this property is only text, so it is left as it was
			// parsed and marked as not pointing at any page.
			property.PageRefsIgnored = true
			continue
		}

		if options.ignoresPageReferences(property.Name) {
			// Logseq does not split the value at commas, but it still reads
			// the links and tags in it, so the value is left as it was parsed.
			continue
		}

		if options.isSeparatedByCommas(property.Name) {
			splitValueIntoPageRefs(property, reader)
		}
	}
}

// isQuotedPropertyValue checks if the value of a property is wrapped in
// double quotes, in the same way as Logseq: the value starts and ends with a
// quote. A quote that is escaped with a backslash does not count.
func isQuotedPropertyValue(node *property, source []byte) bool {
	first, ok := node.FirstChild().(*ast.Text)
	if !ok {
		return false
	}

	last, ok := node.LastChild().(*ast.Text)
	if !ok {
		return false
	}

	start := bytes.TrimLeft(first.Segment.Value(source), " \t")
	end := bytes.TrimRight(last.Segment.Value(source), " \t\r\n")
	if len(start) == 0 || start[0] != '"' || len(end) == 0 || end[len(end)-1] != '"' {
		return false
	}

	// A value of a single quote is not wrapped in quotes.
	return first != last || len(bytes.TrimSpace(start)) >= 2
}

// subSegment is the part of a segment between two offsets into its text.
func subSegment(segment text.Segment, start int, stop int) text.Segment {
	part := segment.WithStart(segment.Start + start)
	return part.WithStop(segment.Start + stop)
}

// isValueSeparator checks if a rune separates the values of a property that
// holds a list, which are the two commas Logseq splits such a value on.
func isValueSeparator(r rune) bool {
	return r == ',' || r == '，'
}

// splitValueIntoPageRefs turns the text of a property value into references to
// the pages it names. Only the text is split, so pages that are written as
// links are left as they are, as are the separators and the space around the
// titles.
func splitValueIntoPageRefs(node *property, reader text.Reader) {
	source := reader.Source()

	// A value is parsed into several text nodes, so the text that follows on
	// from the text before it is gathered up and split as one, letting a title
	// span more than one of them.
	var run []ast.Node
	var segment text.Segment

	split := func() {
		if len(run) == 0 {
			return
		}

		for _, replacement := range pageRefsInValue(segment, source) {
			node.InsertBefore(node, run[0], replacement)
		}

		for _, textNode := range run {
			node.RemoveChild(node, textNode)
		}

		run = nil
	}

	var next ast.Node
	for child := node.FirstChild(); child != nil; child = next {
		next = child.NextSibling()

		textNode, ok := child.(*ast.Text)
		if !ok {
			split()
			continue
		}

		if len(run) > 0 && segment.Stop == textNode.Segment.Start {
			segment = segment.WithStop(textNode.Segment.Stop)
			run = append(run, child)
			continue
		}

		split()

		run = []ast.Node{child}
		segment = textNode.Segment
	}

	split()
}

// pageRefsInValue splits the text of a property value on the separators,
// turning each of the values into a reference to the page it names. The
// separators and the space around the titles are kept as text, so that the
// value is written back the way it was read.
//
// The value of a property ends with the line it starts on, so the line breaks
// of its text have already been taken off and only the text of the segment is
// left to split.
func pageRefsInValue(segment text.Segment, source []byte) []ast.Node {
	value := segment.Value(source)

	nodes := make([]ast.Node, 0)

	// keepText keeps a part of the text as it was written.
	keepText := func(start int, stop int) {
		if start >= stop {
			return
		}

		nodes = append(nodes, ast.NewTextSegment(subSegment(segment, start, stop)))
	}

	// takePageRef turns one of the values into a reference to the page it
	// names, leaving the space that surrounds the title out of the title.
	takePageRef := func(start int, stop int) {
		titleStart := start
		for titleStart < stop && util.IsSpace(value[titleStart]) {
			titleStart++
		}

		titleStop := stop
		for titleStop > titleStart && util.IsSpace(value[titleStop-1]) {
			titleStop--
		}

		keepText(start, titleStart)

		if titleStop > titleStart {
			ref := &pageRefText{}
			ref.Segment = subSegment(segment, titleStart, titleStop)
			nodes = append(nodes, ref)
		}

		keepText(titleStop, stop)
	}

	valueStart := 0
	for i := 0; i < len(value); {
		r, size := utf8.DecodeRune(value[i:])
		if !isValueSeparator(r) {
			i += size
			continue
		}

		takePageRef(valueStart, i)
		keepText(i, i+size)

		i += size
		valueStart = i
	}

	takePageRef(valueStart, len(value))

	return nodes
}

func (t *propertiesASTTransformer) transformTextBlockOrParagraph(node ast.Node, reader text.Reader) {
	wasPreviousLinebreak := true
	var currentProperties *properties
	var currentProperty *property

	// The lines of the paragraph are kept, as the paragraph can be split, and
	// they are needed to find where a line starts that does not start with
	// text. lineBreakAt is where the text of the line before ends, or -1 on
	// the first line.
	lines := node.Lines()
	lineBreakAt := -1

	var next ast.Node
	for child := node.FirstChild(); child != nil; child = next {
		next = child.NextSibling()

		if currentProperty != nil {
			// Currently reading the value of a property
			textNode, isText := child.(*ast.Text)
			if isText {
				if !textNode.Segment.IsEmpty() {
					currentProperty.AppendChild(currentProperty, child)
				} else {
					textNode.Parent().RemoveChild(textNode.Parent(), child)
				}
			} else {
				currentProperty.AppendChild(currentProperty, child)
			}

			if isText && (textNode.HardLineBreak() || textNode.SoftLineBreak()) {
				// End of the property value due to a line break
				currentProperty = nil

				textNode.SetHardLineBreak(false)
				textNode.SetSoftLineBreak(false)

				wasPreviousLinebreak = true
				lineBreakAt = textNode.Segment.Stop
			}
		} else {
			textNode, isText := child.(*ast.Text)
			if !isText && !wasPreviousLinebreak {
				node = maybeSplitParagraph(node, currentProperties, child)

				currentProperties = nil
				wasPreviousLinebreak = false
				continue
			}

			if wasPreviousLinebreak {
				// Potentially a new property. The name is found in the source
				// of the line, as Goldmark can split the name into several
				// nodes, such as at an underscore, or start the line with a
				// node that is not text, such as the tag in `#tag:: x`.
				lineStart := -1
				if isText {
					lineStart = textNode.Segment.Start
				} else if start, found := lineStartAfter(lines, reader.Source(), lineBreakAt); found {
					lineStart = start
				}

				name, valueStart, ok := "", 0, false
				if lineStart >= 0 {
					name, valueStart, ok = findPropertyAt(reader.Source(), lineStart)
				}

				directive := false
				if !ok && isText && (currentProperties != nil || atPageStart(node)) {
					// A directive is a property before the first bullet of
					// a page, and in a group of properties.
					name, valueStart, ok = findDirectiveAt(reader.Source(), lineStart)
					directive = ok
				}

				var rest ast.Node
				var spaceBeforeValue string
				if ok {
					nameEnd := lineStart + len(name)
					valueStart, spaceBeforeValue = skipSpaceBeforeValue(reader.Source(), valueStart)
					rest, ok = nodeAfterName(child, lineStart, nameEnd, valueStart)
				}

				if !ok {
					// Not a property
					node = maybeSplitParagraph(node, currentProperties, child)
					currentProperties = nil
					wasPreviousLinebreak = isText && (textNode.HardLineBreak() || textNode.SoftLineBreak())
					if wasPreviousLinebreak {
						lineBreakAt = textNode.Segment.Stop
					}
					continue
				}

				if currentProperties == nil {
					// This is a new block of properties that splits the paragraph
					currentProperties = &properties{}
					node.Parent().InsertAfter(node.Parent(), node, currentProperties)

					// If the properties are the first child of the paragraph,
					// they take the place of the paragraph, so the blank lines
					// before the paragraph move to them. This information is
					// only set if the paragraph is not the first child at its
					// level. Otherwise the paragraph keeps its text before the
					// properties, and the blank lines before it.
					if child.PreviousSibling() == nil {
						if node.PreviousSibling() != nil {
							currentProperties.SetBlankPreviousLines(node.HasBlankPreviousLines())
						}

						node.SetBlankPreviousLines(false)
					}
				}

				currentProperty = &property{}
				currentProperty.Name = name
				currentProperty.Directive = directive
				currentProperty.InvalidName = !directive && !validPropertyName(name)
				currentProperty.SpaceBeforeValue = spaceBeforeValue
				currentProperty.TrailingSpace, currentProperty.NoSpaceAfterSeparator =
					propertyLineSpace(reader.Source(), lineStart, directive)

				currentProperties.AppendChild(currentProperties, currentProperty)

				// The previous node no longer has a line break
				if previousTextNode, ok := child.PreviousSibling().(*ast.Text); ok {
					previousTextNode.SetHardLineBreak(false)
					previousTextNode.SetSoftLineBreak(false)
				}

				// The nodes that hold the name are taken out, and the value of
				// the property continues with the node after them.
				lineEnded := false
				for nameNode := child; nameNode != rest; {
					following := nameNode.NextSibling()
					if nameText, ok := nameNode.(*ast.Text); ok && (nameText.HardLineBreak() || nameText.SoftLineBreak()) {
						lineEnded = true
						lineBreakAt = nameText.Segment.Stop
					}

					node.RemoveChild(node, nameNode)
					nameNode = following
				}

				// A text node can hold both the end of the name and the start of
				// the value, so only the value is kept.
				if restText, ok := rest.(*ast.Text); ok && restText.Segment.Start < valueStart {
					restText.Segment = restText.Segment.WithStart(valueStart)
				}

				next = rest

				if lineEnded {
					// Nothing follows the name on the line, so the property has
					// an empty value.
					currentProperty = nil
					wasPreviousLinebreak = true
				}
			} else {
				wasPreviousLinebreak = textNode.HardLineBreak() || textNode.SoftLineBreak()
				if wasPreviousLinebreak {
					lineBreakAt = textNode.Segment.Stop
				}
			}
		}
	}

	if node.FirstChild() == nil {
		// The paragraph is now empty
		node.Parent().RemoveChild(node.Parent(), node)
	}
}

// findPropertyAt finds the name of a property at the start of the line that
// starts at the given offset into the source. It returns the name and the
// offset where the value of the property starts.
//
// The name is followed by `::` and a space, after which the value starts. If
// nothing follows the `::` on the line the property has an empty value.
//
// The name is found in the same way as mldoc, see findPropertyKey, so it can
// be a name that Logseq does not keep. Use validPropertyName to check it.
func findPropertyAt(source []byte, start int) (name string, valueStart int, ok bool) {
	line := source[start:]
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = bytes.TrimSuffix(line, []byte("\r"))

	name, end, ok := findPropertyKey(string(line))
	if !ok {
		return "", 0, false
	}

	rest := line[end:]
	switch {
	case len(rest) > 0 && rest[0] == ' ':
		return name, start + end + 1, true
	case len(bytes.Trim(rest, " \t")) == 0:
		return name, start + len(line), true
	default:
		// The space after the :: is missing
		return "", 0, false
	}
}

// propertyLineSpace gets the spaces and tabs at the end of the line of a
// property, which starts at the given offset into the source. If the value of
// the property is empty, they are the spaces and tabs after the separator,
// without the one space that is written after `::` by default. Then noSpace
// is true if no space follows the `::`. See content.Property.TrailingSpace.
func propertyLineSpace(source []byte, start int, directive bool) (trailing string, noSpace bool) {
	line := source[start:]
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = bytes.TrimSuffix(line, []byte("\r"))

	separator := []byte("::")
	if directive {
		separator = []byte(":")
	}

	i := bytes.Index(line, separator)
	if i < 0 {
		return "", false
	}

	rest := line[i+len(separator):]
	value := bytes.TrimRight(rest, " \t")
	if len(value) > 0 {
		return string(rest[len(value):]), false
	}

	if directive {
		return string(rest), false
	}

	if len(rest) > 0 && rest[0] == ' ' {
		return string(rest[1:]), false
	}

	return string(rest), true
}

// skipSpaceBeforeValue skips the spaces and tabs at the start of the value of
// a property, which starts at the given offset into the source. Logseq does
// not read them as part of the value, so `id::  <uuid>` has the value
// `<uuid>`. It returns the offset where the value starts after them, and the
// spaces and tabs that it skipped, so that the text can be written back
// unchanged. If only spaces and tabs follow on the line, the value is empty
// and nothing is skipped.
func skipSpaceBeforeValue(source []byte, valueStart int) (int, string) {
	end := valueStart
	for end < len(source) && (source[end] == ' ' || source[end] == '\t') {
		end++
	}

	if end == len(source) || source[end] == '\n' || source[end] == '\r' {
		return valueStart, ""
	}

	return end, string(source[valueStart:end])
}

// nodeAfterName finds the first node that holds a part of the value of a
// property, when the name of the property starts with the given node. The
// line starts at lineStart, the name ends at nameEnd, and the value starts at
// valueStart, which are offsets into the source. It returns nil if the value
// is empty, or if the line ends before the value.
//
// A node such as a tag or a link to a page has no text of its own. Such a
// node is a part of the name if the node after it starts before the end of
// the name, as with the tag in `#tag:: x`.
//
// It is not possible to take the name out when a node that is not text holds
// both a part of the name and a part of the value, such as emphasis that
// starts in the name, so then ok is false.
func nodeAfterName(first ast.Node, lineStart int, nameEnd int, valueStart int) (rest ast.Node, ok bool) {
	// covered is where the text of the nodes before this one ends.
	covered := lineStart
	for n := first; n != nil; n = n.NextSibling() {
		start, stop, ok := nodeRange(n)
		if !ok {
			// If the name ends before the node, the node is the start of the
			// value.
			if covered >= valueStart {
				return n, true
			}

			if followingStart, found := startOfNodeAfter(n); found && followingStart <= nameEnd {
				continue
			}

			return nil, false
		}

		covered = stop

		if stop <= valueStart {
			if t, isText := n.(*ast.Text); isText && (t.HardLineBreak() || t.SoftLineBreak()) {
				// The line ends with the name
				return n.NextSibling(), true
			}

			continue
		}

		if start >= valueStart {
			return n, true
		}

		if _, isText := n.(*ast.Text); isText {
			// The text holds the end of the name and the start of the value
			return n, true
		}

		return nil, false
	}

	return nil, true
}

// startOfNodeAfter finds where the text of the first node after the given
// node starts in the source, skipping the nodes that have no text of their
// own. It returns false if no node after it has text.
func startOfNodeAfter(n ast.Node) (int, bool) {
	for following := n.NextSibling(); following != nil; following = following.NextSibling() {
		if start, _, ok := nodeRange(following); ok {
			return start, true
		}
	}

	return 0, false
}

// lineStartAfter finds where the text of the first line of a paragraph that
// starts after the given offset into the source begins, after its spaces and
// tabs. Use -1 as the offset to find the first line.
func lineStartAfter(lines *text.Segments, source []byte, offset int) (int, bool) {
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		if line.Start <= offset {
			continue
		}

		start := line.Start
		for start < line.Stop && (source[start] == ' ' || source[start] == '\t') {
			start++
		}

		return start, true
	}

	return 0, false
}

// nodeRange finds the part of the source that the text in a node comes from.
// The markup around the text, such as the * of emphasis, is not included. It
// is not known for nodes that hold no text from the source.
func nodeRange(n ast.Node) (start int, stop int, ok bool) {
	switch n := n.(type) {
	case *ast.Text:
		return n.Segment.Start, n.Segment.Stop, true
	case *ast.RawHTML:
		if n.Segments.Len() == 0 {
			return 0, 0, false
		}

		return n.Segments.At(0).Start, n.Segments.At(n.Segments.Len() - 1).Stop, true
	}

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		childStart, childStop, childOk := nodeRange(child)
		if !childOk {
			return 0, 0, false
		}

		if !ok {
			start = childStart
			ok = true
		}

		stop = childStop
	}

	return start, stop, ok
}

func maybeSplitParagraph(node ast.Node, divider *properties, firstChildOfNewParagraph ast.Node) ast.Node {
	if divider == nil {
		return node
	}

	newParagraph := &ast.Paragraph{}
	node.Parent().InsertAfter(node.Parent(), divider, newParagraph)

	for child := firstChildOfNewParagraph; child != nil; {
		next := child.NextSibling()
		newParagraph.AppendChild(newParagraph, child)
		child = next
	}

	if node.FirstChild() == nil {
		// The paragraph is now empty
		node.Parent().RemoveChild(node.Parent(), node)
	}

	return newParagraph
}
