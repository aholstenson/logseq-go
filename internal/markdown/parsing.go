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
			util.Prioritized(newBlockThematicBreakParser(), 200),
			util.Prioritized(newBlockListParser(), 300),
			util.Prioritized(newBlockListItemParser(), 400),
			util.Prioritized(&indentedBlocksParser{}, 450),
			util.Prioritized(parser.NewCodeBlockParser(), 500),
			util.Prioritized(parser.NewATXHeadingParser(), 600),
			util.Prioritized(parser.NewFencedCodeBlockParser(), 700),
			util.Prioritized(parser.NewBlockquoteParser(), 800),
			util.Prioritized(&mathBlockParser{}, 896),
			util.Prioritized(&taskDateParser{}, 897),
			util.Prioritized(&logbookParser{}, 898),
			util.Prioritized(&beginEndParser{}, 899),
			util.Prioritized(newHTMLBlockParser(), 900),
			util.Prioritized(&footnoteDefinitionParser{}, 999),
			util.Prioritized(newPropertyLineParser(), 999),
			util.Prioritized(parser.NewParagraphParser(), 1000),
		),
		parser.WithInlineParsers(
			util.Prioritized(parser.NewCodeSpanParser(), 100),
			util.Prioritized(&footnoteRefParser{}, 194),
			util.Prioritized(&mathParser{}, 195),
			util.Prioritized(&priorityParser{}, 196),
			util.Prioritized(&macroParser{}, 197),
			util.Prioritized(&blockRefParser{}, 198),
			util.Prioritized(&pageLinkParser{}, 199),
			util.Prioritized(&tagParser{}, 999),
			util.Prioritized(newLabelledRefLinkParser(), 200),
			util.Prioritized(parser.NewAutoLinkParser(), 300),
			util.Prioritized(parser.NewRawHTMLParser(), 400),
			util.Prioritized(&emphasisParser{}, 500),
			util.Prioritized(&strikethroughParser{}, 501),
			util.Prioritized(&highlightParser{}, 502),
			util.Prioritized(extension.NewLinkifyParser(
				extension.WithLinkifyEmailRegexp(neverMatch),
				extension.WithLinkifyWWWRegexp(neverMatch),
			), 998),
		),
		// Logseq has no link reference definitions, so a line such as
		// `[1]: https://example.com` is text. Goldmark's paragraph transformer
		// for them is not used.
		parser.WithParagraphTransformers(
			util.Prioritized(newTableParagraphTransformer(extension.NewTableParagraphTransformer()), 200),
		),
		parser.WithASTTransformers(
			util.Prioritized(extension.NewTableASTTransformer(), 0),
			util.Prioritized(defaultTagsInFormattingASTTransformer, 100),
			util.Prioritized(defaultPropertiesASTTransformer, 200),
		),
	)
}

// ParseOption changes how Markdown is read for the parts of the syntax that a
// graph configures the shape of.
type ParseOption func(*parseOptions)

// parseOptionsKey is where the options of the current parse are kept, so that
// the parts of parsing that Goldmark drives can reach them.
var parseOptionsKey = parser.NewContextKey()

// builtInPropertiesSeparatedByCommas are the properties whose value Logseq
// always reads as a list of pages separated by commas, no matter what the
// graph configures.
var builtInPropertiesSeparatedByCommas = []string{"alias", "aliases", "tags"}

// parseOptions are the settings that parsing takes from the graph. The zero
// value is not usable, use defaultParseOptions to get the defaults of Logseq.
type parseOptions struct {
	// propertiesSeparatedByCommas are the properties whose value is a list of
	// pages separated by commas.
	propertiesSeparatedByCommas map[string]struct{}

	// ignoredPageReferences are the properties whose value never points at a
	// page, even where it is written as a link.
	ignoredPageReferences map[string]struct{}
}

// defaultParseOptions are what Logseq does for a graph that does not configure
// anything else.
func defaultParseOptions() parseOptions {
	options := parseOptions{
		propertiesSeparatedByCommas: make(map[string]struct{}),
		ignoredPageReferences:       make(map[string]struct{}),
	}

	for _, name := range builtInPropertiesSeparatedByCommas {
		options.propertiesSeparatedByCommas[name] = struct{}{}
	}

	return options
}

// WithPropertiesSeparatedByCommas adds properties whose value is a list of
// pages separated by commas, which is `:property/separated-by-commas`. The
// properties Logseq always reads that way are included without being listed.
func WithPropertiesSeparatedByCommas(names ...string) ParseOption {
	return func(o *parseOptions) {
		for _, name := range names {
			o.propertiesSeparatedByCommas[normalizePropertyName(name)] = struct{}{}
		}
	}
}

// WithIgnoredPageReferences adds properties whose value does not point at a
// page, which is `:ignored-page-references-keywords`.
func WithIgnoredPageReferences(names ...string) ParseOption {
	return func(o *parseOptions) {
		for _, name := range names {
			o.ignoredPageReferences[normalizePropertyName(name)] = struct{}{}
		}
	}
}

// isSeparatedByCommas checks if the value of the property with the given name
// is a list of pages separated by commas.
func (o *parseOptions) isSeparatedByCommas(name string) bool {
	_, ok := o.propertiesSeparatedByCommas[normalizePropertyName(name)]
	return ok
}

// ignoresPageReferences checks if the value of the property with the given
// name points at pages or is only text.
func (o *parseOptions) ignoresPageReferences(name string) bool {
	_, ok := o.ignoredPageReferences[normalizePropertyName(name)]
	return ok
}

// normalizePropertyName brings a property name into the form it is matched in,
// as Logseq does not distinguish between property names that only differ in
// case or in the use of an underscore or a dash.
func normalizePropertyName(name string) string {
	return content.NormalizePropertyName(name)
}

func Parse(src []byte, opts ...ParseOption) (*content.Block, error) {
	options := defaultParseOptions()
	for _, opt := range opts {
		opt(&options)
	}

	context := parser.NewContext()
	context.Set(parseOptionsKey, &options)

	doc := markdownParser.Parse(text.NewReader(src), parser.WithContext(context))
	node, err := convert(src, doc)
	if err != nil {
		return nil, fmt.Errorf("Could not parse Markdown: %w", err)
	}

	if b, ok := node.(*content.Block); ok {
		return b, nil
	}

	return nil, errors.New("Could not parse Markdown")
}

func ParseString(src string, opts ...ParseOption) (*content.Block, error) {
	return Parse([]byte(src), opts...)
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
	case *highlight:
		return convertHighlight(src, node)
	case *footnoteRef:
		return content.NewFootnoteRef(node.Label), nil
	case *footnoteDefinition:
		return convertFootnoteDefinition(src, node)
	case *math:
		return content.NewMath(node.Value).WithDisplayed(node.Displayed), nil
	case *mathBlock:
		return convertMathBlock(src, node)
	case *ast.CodeSpan:
		return convertCodeSpan(src, node)
	case *ast.Link:
		return convertLink(src, node)
	case *ast.AutoLink:
		return content.NewAutoLink(string(node.URL(src))), nil
	case *tag:
		return content.NewHashtag(node.Page).WithBracketed(node.Extended), nil
	case *pageLink:
		return content.NewPageLink(node.Page), nil
	case *pageRefText:
		return content.NewPageRefText(unescapePunctuation(node.Segment.Value(src))), nil
	case *blockRef:
		return content.NewBlockRef(node.ID), nil
	case *priority:
		return content.NewTaskPriority(node.Priority), nil
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
	case *east.Table:
		return convertTable(src, node)
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
	case *taskDate:
		return convertTaskDate(node)
	}

	return nil, fmt.Errorf("Could not convert node: %T", in)
}

// unescapeString removes every backslash. It is used for the quoted arguments
// of macros, which have their own rules for backslashes.
func unescapeString(src []byte) string {
	return string(bytes.ReplaceAll(src, []byte(`\`), []byte(``)))
}

// unescapePunctuation removes the backslashes that escape ASCII punctuation,
// in the same way as CommonMark, so `\*` is read as `*` and `\\` as `\`. Other
// backslashes are text and are kept, so `C:\Users` stays as it is.
func unescapePunctuation(src []byte) string {
	return string(util.UnescapePunctuations(src))
}

// convertChildren converts all children of a node and adds them to the target.
func convertChildren(src []byte, node ast.Node, target content.HasChildren) error {
	var previousChild ast.Node
	// mergedStart is where the source of the last text node starts, so that
	// text nodes that are merged are unescaped as one. A backslash at the end
	// of one of them can escape the first character of the next.
	mergedStart := 0
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
				previousNode.Value = unescapePunctuation(src[mergedStart:text.Segment.Stop])
				previousChild = child

				previousNode.HardLineBreak = text.HardLineBreak()
				previousNode.SoftLineBreak = text.SoftLineBreak()
				continue
			}

		}

		if text, ok := child.(*ast.Text); ok {
			mergedStart = text.Segment.Start
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

	// target is the block that content before the first list of blocks goes
	// into. It changes to a pre-block when the content of a page has child
	// blocks of its own.
	target := block

	hasParsedBlock := false
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if indented, ok := child.(*indentedBlocks); ok {
			nested, err := convertToBlock(src, indented)
			if err != nil {
				return nil, err
			}

			parent := target
			if hasParsedBlock {
				// The bullets are indented under the last block.
				lastBlock, ok := block.LastChild().(*content.Block)
				if !ok {
					return nil, errors.New("Last node is not a block")
				}

				parent = lastBlock
			} else if _, isDocument := node.(*ast.Document); isDocument && target == block {
				// The content before the first bullet of a page is the
				// pre-block, so the indented bullets are its children.
				target = content.NewPreBlock()
				for _, existing := range block.Content() {
					target.AddChild(existing)
				}

				block.PrependChild(target)
				parent = target
			}

			for _, nestedBlock := range nested.Blocks() {
				parent.AddChild(nestedBlock)
			}

			continue
		}

		// Check if the child is a list of blocks.
		if list, ok := child.(*ast.List); ok && list.Marker == '-' {
			// This is a list using the block marker, so children are parsed
			// as blocks.
			for listItem := list.FirstChild(); listItem != nil; listItem = listItem.NextSibling() {
				listItemBlock, err := convertToBlock(src, listItem)
				if err != nil {
					return nil, err
				}

				_, isDocument := node.(*ast.Document)
				if isDocument && !hasParsedBlock && listItem == list.FirstChild() &&
					list.HasBlankPreviousLines() && block.FirstChild() != nil {
					// Keep the blank line between the content before the
					// first bullet of a page, such as page properties, and
					// the first bullet.
					listItemBlock.SetPreviousLineType(content.PreviousLineTypeBlank)
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

			if target != block && !hasParsedBlock {
				// Content after the child blocks of the pre-block belongs to
				// the block that comes last in the file.
				addBeforeBlocks(deepestLastBlock(target), node)
			} else if !hasParsedBlock {
				// No list of blocks encountered yet, treat nodes as content
				// to main block.
				block.AddChild(node)
			} else {
				// We have parsed blocks so this is trailing content. Logseq
				// adds it to the block that comes last in the file, which can
				// be a child of the last block.
				if lastBlock, ok := block.LastChild().(*content.Block); ok {
					addBeforeBlocks(deepestLastBlock(lastBlock), node)
				} else {
					return nil, errors.New("Last node is not a block")
				}
			}
		}
	}

	return block, nil
}

// deepestLastBlock follows the last child blocks down from the block, and
// returns the block that comes last in the file.
func deepestLastBlock(block *content.Block) *content.Block {
	for {
		last, ok := block.LastChild().(*content.Block)
		if !ok {
			return block
		}

		block = last
	}
}

// addBeforeBlocks adds a node to the end of the content of a block, which is
// before the first child block.
func addBeforeBlocks(block *content.Block, node content.Node) {
	var lastNode content.Node
	for subNode := block.FirstChild(); subNode != nil; subNode = subNode.NextSibling() {
		if _, ok := subNode.(*content.Block); ok {
			break
		}
		lastNode = subNode
	}

	if lastNode == nil {
		// No content yet, so the node is the first child.
		block.PrependChild(node)
	} else {
		block.InsertChildAfter(node, lastNode)
	}
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

	taskStatus := taskStatusFor(potentialMarker)
	if taskStatus == content.TaskStatusNone {
		return
	}

	if potentialMarkerIdx < 0 {
		// The marker is the entire text, so there is nothing left of it.
		textNode.Value = ""
	} else {
		textNode.Value = textNode.Value[potentialMarkerIdx+1:]
	}

	if textNode.Value == "" {
		textNode.RemoveSelf()
	}

	node.PrependChild(content.NewTaskMarker(taskStatus))
}

// taskStatusFor maps a task marker to its status, returning TaskStatusNone if
// the word is not a marker.
func taskStatusFor(marker string) content.TaskStatus {
	switch marker {
	case "TODO":
		return content.TaskStatusTodo
	case "DONE":
		return content.TaskStatusDone
	case "DOING":
		return content.TaskStatusDoing
	case "LATER":
		return content.TaskStatusLater
	case "NOW":
		return content.TaskStatusNow
	case "CANCELLED":
		return content.TaskStatusCancelled
	case "CANCELED":
		return content.TaskStatusCanceled
	case "IN-PROGRESS":
		return content.TaskStatusInProgress
	case "WAIT":
		return content.TaskStatusWait
	case "WAITING":
		return content.TaskStatusWaiting
	}

	return content.TaskStatusNone
}

func convertParagraph(src []byte, node *ast.Paragraph) (*content.Paragraph, error) {
	paragraph := content.NewParagraph()
	err := convertChildren(src, node, paragraph)
	if err != nil {
		return nil, err
	}
	if _, afterProperties := node.PreviousSibling().(*properties); afterProperties {
		// A paragraph directly after properties is written on the next line,
		// so a blank line between them has to be recorded explicitly.
		updatePreviousLineKeepingBlank(node, paragraph)
	} else {
		updatePreviousLine(node, paragraph)
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
	value := unescapePunctuation(node.Segment.Value(src))
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

// convertLink converts a link. Logseq reads `[label]([[Page]])` as a link to
// the page and `[label](((uuid)))` as a reference to the block, both with the
// text of the link as the label. A link with a title is kept as a link, as
// there is no place for the title in a page link or block reference.
func convertLink(src []byte, node *ast.Link) (content.Node, error) {
	destination := node.Destination
	if len(node.Title) == 0 {
		if length := pageRefLength(destination); length > 0 && length == len(destination) {
			link := content.NewPageLink(unescapePunctuation(destination[2 : length-2]))
			return link, convertChildren(src, node, link)
		}

		if length := blockRefLength(destination); length > 0 && length == len(destination) {
			ref := content.NewBlockRef(string(destination[2 : length-2]))
			return ref, convertChildren(src, node, ref)
		}
	}

	link := content.NewLink(unescapePunctuation(destination))
	err := convertChildren(src, node, link)
	if err != nil {
		return nil, err
	}
	link.Title = unescapePunctuation(node.Title)
	return link, nil
}

func convertMacro(src []byte, node *macro) (content.Node, error) {
	switch node.Name {
	case "query":
		// A query without a query string is not a query, keep it as a macro so
		// it is written back the way it was found.
		if len(node.Arguments) == 0 {
			break
		}

		// Logseq joins the arguments again, so the query is all of the text
		// after the name, also when it has commas.
		return content.NewQuery(node.RawArguments), nil
	case "embed":
		// Either a [[page]] or a ((block)). Logseq joins the arguments again,
		// so a page title with a comma is one target.
		if len(node.Arguments) == 0 {
			break
		}

		arg := node.RawArguments
		if strings.HasPrefix(arg, "((") && strings.HasSuffix(arg, "))") && !strings.Contains(arg[2:len(arg)-2], "))") {
			return content.NewBlockEmbed(arg[2 : len(arg)-2]), nil
		} else if page, ok := wholePageLink(arg); ok {
			return content.NewPageEmbed(page), nil
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

func convertHighlight(src []byte, node *highlight) (*content.Highlight, error) {
	highlight := content.NewHighlight()
	err := convertChildren(src, node, highlight)
	if err != nil {
		return nil, err
	}
	return highlight, nil
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

func convertFootnoteDefinition(src []byte, node *footnoteDefinition) (*content.FootnoteDefinition, error) {
	definition := content.NewFootnoteDefinition(unescapePunctuation([]byte(node.Label)))
	err := convertChildren(src, node, definition)
	if err != nil {
		return nil, err
	}

	updatePreviousLine(node, definition)

	return definition, nil
}

func convertMathBlock(src []byte, node *mathBlock) (*content.MathBlock, error) {
	// The lines of the formula are raw data, so they are put back together
	// into the LaTeX between the lines that mark the block out.
	valueBuf := bytes.Buffer{}
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		_, _ = valueBuf.Write(line.Value(src))
	}

	value := content.NewMathBlock(valueBuf.String())

	updatePreviousLine(node, value)

	return value, nil
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

func convertTable(src []byte, node *east.Table) (*content.Table, error) {
	table := content.NewTable()
	for _, alignment := range node.Alignments {
		table.Alignments = append(table.Alignments, tableAlignmentFor(alignment))
	}

	// The header of the table is its own type in Goldmark, but both it and the
	// rows below it are a row of cells.
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		row, err := convertTableRow(src, child)
		if err != nil {
			return nil, err
		}
		table.AddChild(row)
	}

	updatePreviousLine(node, table)

	return table, nil
}

func convertTableRow(src []byte, node ast.Node) (*content.TableRow, error) {
	row := content.NewTableRow()
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*east.TableCell); !ok {
			return nil, fmt.Errorf("Could not convert node: %T", child)
		}

		cell := content.NewTableCell()
		err := convertChildren(src, child, cell)
		if err != nil {
			return nil, err
		}
		row.AddChild(cell)
	}
	return row, nil
}

func tableAlignmentFor(alignment east.Alignment) content.TableAlignment {
	switch alignment {
	case east.AlignLeft:
		return content.TableAlignmentLeft
	case east.AlignRight:
		return content.TableAlignmentRight
	case east.AlignCenter:
		return content.TableAlignmentCenter
	}

	return content.TableAlignmentNone
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
	image.Title = unescapePunctuation(node.Title)

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

		prop := content.NewProperty(p.Name).WithPageRefsIgnored(p.PageRefsIgnored)
		err := convertChildren(src, p, prop)
		if err != nil {
			return nil, err
		}

		properties.AddChild(prop)
	}

	updatePreviousLineKeepingBlank(node, properties)

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
		logbook.AddChild(parseLogbookEntry(value))
	}

	updatePreviousLine(node, logbook)

	return logbook, nil
}

func convertTaskDate(node *taskDate) (content.Node, error) {
	var date *content.TaskDate
	if node.HasTime {
		date = content.NewTaskDateWithTime(node.DateType, node.Date)
	} else {
		date = content.NewTaskDate(node.DateType, node.Date)
	}
	date.Repeater = node.Repeater

	updatePreviousLineKeepingBlank(node, date)

	return date, nil
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

// updatePreviousLineKeepingBlank is updatePreviousLine for nodes that combine
// with the previous line by default. For those a blank line before the node
// has to be recorded explicitly, as the automatic behavior would drop it.
func updatePreviousLineKeepingBlank(node ast.Node, target content.PreviousLineAware) {
	if node.HasBlankPreviousLines() {
		target.SetPreviousLineType(content.PreviousLineTypeBlank)
	} else if node.PreviousSibling() == nil {
		// Parsing didn't indicate blank lines before the node, but we might
		// be the first node on this level in which case we set the type to
		// automatic
		target.SetPreviousLineType(content.PreviousLineTypeAutomatic)
	} else {
		target.SetPreviousLineType(content.PreviousLineTypeNonBlank)
	}
}
