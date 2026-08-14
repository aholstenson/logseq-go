package markdown

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var propertyRegex = regexp.MustCompile(`^([a-zA-Z0-9_-]+)::`)

var propertiesKind = ast.NewNodeKind("Properties")

type properties struct {
	ast.BaseBlock
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
	// instead of as references, as set by `:ignored-page-references-keywords`.
	PageRefsIgnored bool
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
	ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && (node.Kind() == ast.KindParagraph || node.Kind() == ast.KindTextBlock) {
			t.transformTextBlockOrParagraph(node, reader)
		}

		return ast.WalkContinue, nil
	})

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

		if options.ignoresPageReferences(property.Name) {
			// The value of this property is only text, so it is left as it was
			// parsed and marked as not pointing at any page.
			property.PageRefsIgnored = true
			continue
		}

		if options.isSeparatedByCommas(property.Name) {
			splitValueIntoPageRefs(property, reader)
		}
	}
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
			}
		} else {
			textNode, isText := child.(*ast.Text)
			if !isText {
				node = maybeSplitParagraph(node, currentProperties, child)

				currentProperties = nil
				wasPreviousLinebreak = false
				continue
			}

			if wasPreviousLinebreak {
				// Potentially a new property
				potentialName := string(reader.Value(textNode.Segment))

				// In Goldmark the space after :: will either be part of the
				// current text node or the next one.
				matches := propertyRegex.FindStringSubmatchIndex(potentialName)
				if matches == nil {
					// Not a property
					node = maybeSplitParagraph(node, currentProperties, child)
					currentProperties = nil
					wasPreviousLinebreak = textNode.HardLineBreak() || textNode.SoftLineBreak()
					continue
				}

				// The value of the property starts after the name and the space
				// that separates the two.
				valueStart := matches[1]

				// Check if there is a space after the ::
				if strings.HasPrefix(potentialName[valueStart:], " ") {
					valueStart++
				} else {
					// There isn't a space after :: in the current text node
					nextTextNode, _ := next.(*ast.Text)
					if startsWithSpace(nextTextNode, reader) {
						// The text node has a space, update it to remove the space
						nextTextNode.Segment = nextTextNode.Segment.WithStart(nextTextNode.Segment.Start + 1)
					} else {
						// The space is missing, not parsing as property
						node = maybeSplitParagraph(node, currentProperties, child)
						currentProperties = nil
						wasPreviousLinebreak = textNode.HardLineBreak() || textNode.SoftLineBreak()
						continue
					}
				}

				if currentProperties == nil {
					// This is a new block of properties that splits the paragraph
					currentProperties = &properties{}
					node.Parent().InsertAfter(node.Parent(), node, currentProperties)

					// If the properties is the first child of the paragraph we
					// set explicit blank line information if the paragraph
					// is not the first child at its level
					if child.PreviousSibling() == nil && node.PreviousSibling() != nil {
						currentProperties.SetBlankPreviousLines(node.HasBlankPreviousLines())
					}

					// The paragraph does not have any blank lines before it
					node.SetBlankPreviousLines(false)
				}

				currentProperty = &property{}
				currentProperty.Name = potentialName[matches[2]:matches[3]]

				currentProperties.AppendChild(currentProperties, currentProperty)

				// The previous node no longer has a line break
				if previousTextNode, ok := textNode.PreviousSibling().(*ast.Text); ok {
					previousTextNode.SetHardLineBreak(false)
					previousTextNode.SetSoftLineBreak(false)
				}

				// Only the name is taken out of the text node, as anything that
				// follows it on the line is the start of the value.
				textNode.Segment = textNode.Segment.WithStart(textNode.Segment.Start + valueStart)

				if textNode.Segment.IsEmpty() {
					node.RemoveChild(node, child)
				} else {
					currentProperty.AppendChild(currentProperty, child)
				}

				if textNode.HardLineBreak() || textNode.SoftLineBreak() {
					// The value of the property ends with the line it starts on
					currentProperty = nil

					textNode.SetHardLineBreak(false)
					textNode.SetSoftLineBreak(false)

					wasPreviousLinebreak = true
				}
			} else {
				wasPreviousLinebreak = textNode.HardLineBreak() || textNode.SoftLineBreak()
			}
		}
	}

	if node.FirstChild() == nil {
		// The paragraph is now empty
		node.Parent().RemoveChild(node.Parent(), node)
	}
}

func startsWithSpace(node *ast.Text, reader text.Reader) bool {
	if node == nil {
		return false
	}

	value := string(reader.Value(node.Segment))
	return strings.HasPrefix(value, " ")
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
