package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var markdownParser parser.Parser

var propertyRegex = regexp.MustCompile(`^([a-zA-Z0-9_-]+)::\s*`)

var neverMatch = regexp.MustCompile(`\A\z`)

func init() {
	markdownParser = parser.NewParser(
		parser.WithBlockParsers(parser.DefaultBlockParsers()...),
		parser.WithBlockParsers(
			util.Prioritized(&beginEndParser{}, 899),
		),
		parser.WithInlineParsers(parser.DefaultInlineParsers()...),
		parser.WithInlineParsers(
			util.Prioritized(&pageLinkParser{}, 199),
			util.Prioritized(&tagParser{}, 999),
			util.Prioritized(extension.NewLinkifyParser(
				extension.WithLinkifyEmailRegexp(neverMatch),
				extension.WithLinkifyWWWRegexp(neverMatch),
			), 998),
		),
		parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
	)
}

func Parse(src []byte) (*content.Block, error) {
	doc := markdownParser.Parse(text.NewReader(src))
	node, err := convert(src, doc)
	if err != nil {
		return nil, fmt.Errorf("Could not parse Markdown: %w", err)
	}

	if b, ok := node.(*content.Block); ok {
		return b, nil
	}

	return nil, errors.New("Could not parse Markdown")
}

func ParseString(src string) (*content.Block, error) {
	return Parse([]byte(src))
}

// convert convert from the Goldmark AST into our AST.
func convert(src []byte, in ast.Node) (content.Node, error) {
	switch node := in.(type) {
	case *ast.Document:
		return convertToBlock(src, node)
	case *ast.Heading:
		return convertHeading(src, node)
	case *ast.Paragraph:
		return convertParagraph(src, node)
	case *ast.TextBlock:
		return convertTextBlock(src, node)
	case *ast.Text:
		return convertText(src, node)
	case *ast.Emphasis:
		return convertEmphasis(src, node)
	case *ast.CodeSpan:
		return convertCodeSpan(src, node)
	case *ast.Link:
		return convertLink(src, node)
	case *ast.AutoLink:
		return content.NewAutoLink(string(node.URL(src))), nil
	case *tag:
		return content.NewHashtag(node.Page), nil
	case *pageLink:
		return content.NewPageLink(node.Page), nil
	case *ast.FencedCodeBlock:
		return convertFencedCodeBlock(src, node)
	case *ast.CodeBlock:
		return convertCodeBlock(src, node)
	case *ast.Blockquote:
		return convertBlockquote(src, node)
	case *ast.List:
		return convertList(src, node)
	case *ast.RawHTML:
		return convertRawHTML(src, node)
	case *ast.HTMLBlock:
		return convertHTMLBlock(src, node)
	case *ast.Image:
		return convertImage(src, node)
	case *ast.ThematicBreak:
		return content.NewThematicBreak(), nil
	case *beginEnd:
		return convertBeginEnd(src, node)
	}

	return nil, fmt.Errorf("Could not convert node: %T", in)
}

func unescapeString(src []byte) string {
	return string(bytes.ReplaceAll(src, []byte(`\`), []byte(``)))
}

// convertChildren converts all children of a node and adds them to the target.
func convertChildren(src []byte, node ast.Node, target content.HasChildren) error {
	var previousChild ast.Node
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		node, err := convert(src, child)
		if err != nil {
			return err
		}

		// If the previous child was a text node and the current child is a text
		// node can merge them in certain cases. This makes the output easier
		// to work with.
		if pText, pOk := previousChild.(*ast.Text); pOk {
			if text, ok := child.(*ast.Text); ok && canMergeTextNodes(pText, text) {
				previousNode := target.LastChild().(*content.Text)
				previousNode.Value += string(text.Segment.Value(src))
				previousChild = child

				previousNode.HardLineBreak = text.HardLineBreak()
				previousNode.SoftLineBreak = text.SoftLineBreak()
				continue
			}

		}

		target.AddChild(node)
		previousChild = child
	}

	return nil
}

func canMergeTextNodes(a *ast.Text, b *ast.Text) bool {
	if a.Segment.Stop != b.Segment.Start {
		return false
	}

	return true
}

// convertToBlock implements the Logseq outlining, where lists using `-` are
// converted into sub-blocks.
func convertToBlock(src []byte, node ast.Node) (*content.Block, error) {
	block := content.NewBlock()

	hasParsedBlock := false
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		// Check if the child is a list of blocks.
		if list, ok := child.(*ast.List); ok && list.Marker == '-' {
			// This is a list using the block marker, so children are parsed
			// as blocks.
			for listItem := list.FirstChild(); listItem != nil; listItem = listItem.NextSibling() {
				listItemBlock, err := convertToBlock(src, listItem)
				if err != nil {
					return nil, err
				}

				block.AddChild(listItemBlock)
			}

			hasParsedBlock = true
		} else {
			node, err := convert(src, child)
			if err != nil {
				return nil, err
			}

			if !hasParsedBlock {
				// No list of blocks encountered yet, treat nodes as content
				// to main block.
				block.AddChild(node)
			} else {
				// We have parsed blocks so this is trailing content, in Logseq
				// this seems to be added to the last block.
				if lastBlock, ok := block.LastChild().(*content.Block); ok {
					// Find the node before the first block in sub-block.
					var lastNode content.Node
					for subNode := lastBlock.FirstChild(); subNode != nil; subNode = subNode.NextSibling() {
						if _, ok := subNode.(*content.Block); ok {
							break
						}
						lastNode = subNode
					}

					if lastNode == nil {
						// Likely no content, just add as the first child.
						lastBlock.PrependChild(node)
					} else {
						// Insert after the last node.
						lastBlock.InsertAfter(node, lastNode)
					}
				} else {
					return nil, errors.New("Last node is not a block")
				}
			}
		}
	}

	// Properties can be in any paragraph in the first level of the block.
	// We check each paragraph for properties and add them to the block.
	//
	// The properties look like: `key:: value`
	//
	// Due to how we process this it means that we will only create a single
	// Properties node and any properties in later paragraphs will be added
	// to that node.
	properties := content.NewProperties()
	for child := block.FirstChild(); child != nil; child = child.NextSibling() {
		if p, ok := child.(*content.Paragraph); ok {
			// This a paragraph, let's check each text node for properties.
			var property *content.Property
			shouldLookForProperty := true

			for _, textNode := range p.Children() {
				if text, ok := textNode.(*content.Text); ok {
					if shouldLookForProperty {
						shouldLookForProperty = false

						// Check if this matches the property pattern. If it does
						// we get the key and split the text node so it can be added
						// as a property.
						if matches := propertyRegex.FindStringSubmatchIndex(text.Value); matches != nil {
							// Get the first group, which is the key.
							property = content.NewProperty(text.Value[matches[2]:matches[3]])
							properties.AddChild(property)

							if properties.Parent() == nil {
								// The properties container has to be inserted
								p.InsertBefore(properties, text)
							}

							// Remove the property key from the text node.
							text.Value = text.Value[matches[1]:]
						}
					}

					// Check if there is a newline if so the property has ended.
					if text.SoftLineBreak || text.HardLineBreak {
						if property != nil {
							// Reset line breaks as the don't make sense for properties.
							text.HardLineBreak = false
							text.SoftLineBreak = false
							property.AddChild(text)
						}

						// Reset to look for a new property.
						shouldLookForProperty = true
						property = nil
					} else if property != nil {
						// No newline, add to current property.
						property.AddChild(text)
					}
				} else {
					// Non-text nodes never start a search for new properties.
					shouldLookForProperty = false
				}
			}

			// If we have a property we can stop looking for properties.
			if property != nil {
				break
			}
		}
	}

	return block, nil
}

func convertParagraph(src []byte, node *ast.Paragraph) (*content.Paragraph, error) {
	paragraph := content.NewParagraph()
	err := convertChildren(src, node, paragraph)
	if err != nil {
		return nil, err
	}
	return paragraph, nil
}

func convertTextBlock(src []byte, node ast.Node) (*content.Paragraph, error) {
	paragraph := content.NewParagraph()
	err := convertChildren(src, node, paragraph)
	if err != nil {
		return nil, err
	}
	return paragraph, nil
}

func convertText(src []byte, node *ast.Text) (*content.Text, error) {
	value := unescapeString(node.Segment.Value(src))
	text := content.NewText(value)
	if node.SoftLineBreak() {
		text.SoftLineBreak = true
	} else if node.HardLineBreak() {
		text.HardLineBreak = true
	}
	return text, nil
}

func convertHeading(src []byte, node *ast.Heading) (*content.Heading, error) {
	heading := content.NewHeading(node.Level)
	err := convertChildren(src, node, heading)
	if err != nil {
		return nil, err
	}
	return heading, nil
}

func convertLink(src []byte, node *ast.Link) (*content.Link, error) {
	link := content.NewLink(unescapeString(node.Destination))
	err := convertChildren(src, node, link)
	if err != nil {
		return nil, err
	}
	link.Title = unescapeString(node.Title)
	return link, nil
}

func convertEmphasis(src []byte, node *ast.Emphasis) (content.HasChildren, error) {
	var result content.HasChildren
	if node.Level == 1 {
		result = content.NewEmphasis()
	} else if node.Level == 2 {
		result = content.NewStrong()
	} else {
		return nil, fmt.Errorf("Unsupported emphasis level: %d", node.Level)
	}

	err := convertChildren(src, node, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func convertCodeSpan(src []byte, node *ast.CodeSpan) (*content.CodeSpan, error) {
	code := content.NewCodeSpan(string(node.Text(src)))
	return code, nil
}

func convertFencedCodeBlock(src []byte, node *ast.FencedCodeBlock) (*content.CodeBlock, error) {
	// FencedCodeBlock contains raw data so we need to combine all the lines
	// into a single string.
	codeBuf := bytes.Buffer{}
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		_, _ = codeBuf.Write(line.Value(src))
	}

	code := content.NewCodeBlock(codeBuf.String())
	code.Language = string(node.Language(src))
	return code, nil
}

func convertCodeBlock(src []byte, node *ast.CodeBlock) (*content.CodeBlock, error) {
	// CodeBlock contains raw data so we need to combine all the lines
	// into a single string.
	codeBuf := bytes.Buffer{}
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		_, _ = codeBuf.Write(line.Value(src))
	}

	code := content.NewCodeBlock(codeBuf.String())
	return code, nil
}

func convertBlockquote(src []byte, node *ast.Blockquote) (*content.Blockquote, error) {
	blockquote := content.NewBlockquote()
	err := convertChildren(src, node, blockquote)
	if err != nil {
		return nil, err
	}
	return blockquote, nil
}

// convertList converts an ast.List into either a list or a block.
func convertList(src []byte, node *ast.List) (*content.List, error) {
	listType := content.ListTypeUnordered
	if node.Marker == '.' {
		listType = content.ListTypeOrdered
	}

	list := content.NewList(listType)
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		item, err := convertListItem(src, child)
		if err != nil {
			return nil, err
		}
		list.AddChild(item)
	}
	return list, nil
}

func convertListItem(src []byte, node ast.Node) (*content.ListItem, error) {
	item := content.NewListItem()
	err := convertChildren(src, node, item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func convertRawHTML(src []byte, node *ast.RawHTML) (*content.RawHTML, error) {
	codeBuf := bytes.Buffer{}
	for i := 0; i < node.Segments.Len(); i++ {
		segment := node.Segments.At(i)
		_, _ = codeBuf.Write(segment.Value(src))
	}

	raw := content.NewRawHTML(codeBuf.String())
	return raw, nil
}

func convertHTMLBlock(src []byte, node *ast.HTMLBlock) (*content.RawHTMLBlock, error) {
	codeBuf := bytes.Buffer{}
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		_, _ = codeBuf.Write(line.Value(src))
	}

	raw := content.NewRawHTMLBlock(codeBuf.String())
	return raw, nil
}

func convertImage(src []byte, node *ast.Image) (*content.Image, error) {
	image := content.NewImage(string(node.Destination))
	image.Title = unescapeString(node.Title)

	err := convertChildren(src, node, image)
	if err != nil {
		return nil, err
	}
	return image, nil
}

func convertBeginEnd(src []byte, node *beginEnd) (*content.AdvancedCommand, error) {
	codeBuf := bytes.Buffer{}
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		_, _ = codeBuf.Write(line.Value(src))
	}

	raw := content.NewAdvancedCommand(node.Variant, codeBuf.String())
	return raw, nil
}
