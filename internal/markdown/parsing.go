package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var markdownParser parser.Parser

var neverMatch = regexp.MustCompile(`$.^`)

func init() {
	markdownParser = parser.NewParser(
		parser.WithBlockParsers(
			util.Prioritized(parser.NewThematicBreakParser(), 200),
			util.Prioritized(parser.NewListParser(), 300),
			util.Prioritized(parser.NewListItemParser(), 400),
			util.Prioritized(parser.NewCodeBlockParser(), 500),
			util.Prioritized(parser.NewATXHeadingParser(), 600),
			util.Prioritized(parser.NewFencedCodeBlockParser(), 700),
			util.Prioritized(parser.NewBlockquoteParser(), 800),
			util.Prioritized(&logbookParser{}, 898),
			util.Prioritized(&beginEndParser{}, 899),
			util.Prioritized(parser.NewHTMLBlockParser(), 900),
			util.Prioritized(parser.NewParagraphParser(), 1000),
		),
		parser.WithInlineParsers(
			util.Prioritized(parser.NewCodeSpanParser(), 100),
			util.Prioritized(&macroParser{}, 197),
			util.Prioritized(&blockRefParser{}, 198),
			util.Prioritized(&pageLinkParser{}, 199),
			util.Prioritized(&tagParser{}, 999),
			util.Prioritized(parser.NewLinkParser(), 200),
			util.Prioritized(parser.NewAutoLinkParser(), 300),
			util.Prioritized(parser.NewRawHTMLParser(), 400),
			util.Prioritized(parser.NewEmphasisParser(), 500),
			util.Prioritized(extension.NewStrikethroughParser(), 501),
			util.Prioritized(extension.NewLinkifyParser(
				extension.WithLinkifyEmailRegexp(neverMatch),
				extension.WithLinkifyWWWRegexp(neverMatch),
			), 998),
		),
		parser.WithParagraphTransformers(
			util.Prioritized(parser.LinkReferenceParagraphTransformer, 100),
		),
		parser.WithASTTransformers(
			util.Prioritized(defaultPropertiesASTTransformer, 200),
		),
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
	case *east.Strikethrough:
		return convertStrikethrough(src, node)
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
	case *blockRef:
		return content.NewBlockRef(node.ID), nil
	case *macro:
		return convertMacro(src, node)
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
	case *properties:
		return convertProperties(src, node)
	case *logbook:
		return convertLogbook(src, node)
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

			// When converting a block the first paragraph might contain a task marker
			if p, ok := node.(*content.Paragraph); ok && block.FirstChild() == nil {
				convertTaskMarker(p)
			}

			if !hasParsedBlock {
				// No list of blocks encountered yet, treat nodes as content
				// to main block.
				block.AddChild(node)

				if p, ok := node.(*content.Paragraph); ok {
					// Logseq handles a "-" without a trailing space as block
					// but Goldmark will parse it as a text item in a paragraph
					var previousNode content.Node
					var previousNodeBreak bool
					for _, paragraphNode := range p.Children() {
						if text, ok := paragraphNode.(*content.Text); ok {
							if text.Value == "-" && previousNodeBreak {
								text.RemoveSelf()

								newBlock := content.NewBlock()
								block.AddChild(newBlock)

								if previousText, ok := previousNode.(*content.Text); ok {
									previousText.SoftLineBreak = false
									previousText.HardLineBreak = false
								}
							}

							previousNodeBreak = text.HardLineBreak || text.SoftLineBreak
						} else {
							previousNodeBreak = false
						}

						previousNode = paragraphNode
					}
				}
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
						lastBlock.InsertChildAfter(node, lastNode)
					}
				} else {
					return nil, errors.New("Last node is not a block")
				}
			}
		}
	}

	return block, nil
}

func convertTaskMarker(node *content.Paragraph) {
	textNode, ok := node.FirstChild().(*content.Text)
	if !ok {
		return
	}

	potentialMarkerIdx := strings.Index(textNode.Value, " ")
	var potentialMarker string
	if potentialMarkerIdx < 0 {
		potentialMarker = textNode.Value
	} else {
		potentialMarker = textNode.Value[:potentialMarkerIdx]
	}

	var taskStatus content.TaskStatus
	switch potentialMarker {
	case "TODO":
		taskStatus = content.TaskStatusTodo
	case "DONE":
		taskStatus = content.TaskStatusDone
	case "DOING":
		taskStatus = content.TaskStatusDoing
	case "LATER":
		taskStatus = content.TaskStatusLater
	case "NOW":
		taskStatus = content.TaskStatusNow
	case "CANCELLED":
		taskStatus = content.TaskStatusCancelled
	case "CANCELED":
		taskStatus = content.TaskStatusCanceled
	case "IN-PROGRESS":
		taskStatus = content.TaskStatusInProgress
	case "WAIT":
		taskStatus = content.TaskStatusWait
	case "WAITING":
		taskStatus = content.TaskStatusWaiting
	default:
		return
	}

	textNode.Value = textNode.Value[potentialMarkerIdx+1:]
	if textNode.Value == "" {
		textNode.RemoveSelf()
	}

	node.PrependChild(content.NewTaskMarker(taskStatus))
}

func convertParagraph(src []byte, node *ast.Paragraph) (*content.Paragraph, error) {
	paragraph := content.NewParagraph()
	err := convertChildren(src, node, paragraph)
	if err != nil {
		return nil, err
	}
	updatePreviousLine(node, paragraph)
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

func convertMacro(src []byte, node *macro) (content.Node, error) {
	switch node.Name {
	case "query":
		return content.NewQuery(node.Arguments[0]), nil
	case "embed":
		// Either a [[page]] or a ((block))
		if len(node.Arguments) == 0 {
			break
		}

		arg := node.Arguments[0]
		if strings.HasPrefix(arg, "((") && strings.HasSuffix(arg, "))") {
			return content.NewBlockEmbed(arg[2 : len(arg)-2]), nil
		} else if strings.HasPrefix(arg, "[[") && strings.HasSuffix(arg, "]]") {
			return content.NewPageEmbed(arg[2 : len(arg)-2]), nil
		}
	case "cloze":
		// {{cloze answer \\ cue}} or {{cloze answer}}
		if len(node.Arguments) == 0 {
			break
		}

		arg := strings.Join(node.Arguments, ", ")
		cueIdx := strings.LastIndex(arg, "\\")
		if cueIdx < 0 {
			return content.NewCloze(
				strings.TrimSpace(arg),
			), nil
		} else {
			return content.NewClozeWithCue(
				strings.TrimSpace(arg[:cueIdx]),
				strings.TrimSpace(arg[cueIdx+1:]),
			), nil
		}
	}

	return content.NewMacro(node.Name, node.Arguments...), nil
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

func convertStrikethrough(src []byte, node *east.Strikethrough) (*content.Strikethrough, error) {
	strikethrough := content.NewStrikethrough()
	err := convertChildren(src, node, strikethrough)
	if err != nil {
		return nil, err
	}
	return strikethrough, nil
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

	updatePreviousLine(node, code)

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

	updatePreviousLine(node, code)
	return code, nil
}

func convertBlockquote(src []byte, node *ast.Blockquote) (*content.Blockquote, error) {
	blockquote := content.NewBlockquote()
	err := convertChildren(src, node, blockquote)
	if err != nil {
		return nil, err
	}

	updatePreviousLine(node, blockquote)

	return blockquote, nil
}

// convertList converts an ast.List into either a list or a block.
func convertList(src []byte, node *ast.List) (*content.List, error) {
	list := content.NewListFromMarker(node.Marker)
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		item, err := convertListItem(src, child)
		if err != nil {
			return nil, err
		}
		list.AddChild(item)
	}

	updatePreviousLine(node, list)

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

func convertProperties(src []byte, node *properties) (*content.Properties, error) {
	properties := content.NewProperties()
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		p, ok := child.(*property)
		if !ok {
			return nil, errors.New("Invalid child in properties")
		}

		prop := content.NewProperty(p.Name)
		err := convertChildren(src, p, prop)
		if err != nil {
			return nil, err
		}

		properties.AddChild(prop)
	}

	if node.HasBlankPreviousLines() {
		// The default behavior of properties is to combine with the previous
		// line so keep explicit blank line info
		properties.SetPreviousLineType(content.PreviousLineTypeBlank)
	} else {
		// Parsing didn't indicate blank lines before the node, but we might
		// be the first node on this level in which case we set the type to
		// automatic
		if node.PreviousSibling() == nil {
			properties.SetPreviousLineType(content.PreviousLineTypeAutomatic)
		} else {
			properties.SetPreviousLineType(content.PreviousLineTypeNonBlank)
		}
	}

	return properties, nil
}

func convertBeginEnd(src []byte, node *beginEnd) (content.Node, error) {
	codeBuf := bytes.Buffer{}
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		_, _ = codeBuf.Write(line.Value(src))
	}

	switch node.Variant {
	case "QUERY":
		return content.NewQueryCommand(codeBuf.String()), nil
	default:
		return content.NewAdvancedCommand(node.Variant, codeBuf.String()), nil
	}
}

func convertLogbook(src []byte, node *logbook) (content.Node, error) {
	logbook := content.NewLogbook()
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		value := strings.TrimSuffix(string(line.Value(src)), "\n")
		logbook.AddChild(content.NewLogbookEntryRaw(value))
	}

	updatePreviousLine(node, logbook)

	return logbook, nil
}

// updatePreviousLine updates the PreviousLineType of the target based on the
// node. This takes the information Goldmark parsed out and transfers it to
// our nodes.
func updatePreviousLine(node ast.Node, target content.PreviousLineAware) {
	if node.HasBlankPreviousLines() {
		target.SetPreviousLineType(content.PreviousLineTypeAutomatic)
	} else {
		// Parsing didn't indicate blank lines before the node, but we might
		// be the first node on this level in which case we set the type to
		// automatic
		if node.PreviousSibling() == nil {
			target.SetPreviousLineType(content.PreviousLineTypeAutomatic)
		} else {
			target.SetPreviousLineType(content.PreviousLineTypeNonBlank)
		}
	}
}
