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
	"github.com/yuin/goldmark/util"
)

var markdownParser parser.Parser

// neverMatch is a regexp that matches no text. Logseq does not make links
// from bare e-mail and www addresses, so the linkify parser gets it for them.
// It is anchored at the start of the text. The parser calls it at each space
// of a line, and a regexp that is not anchored reads to the end of the line
// each time, which makes a long line slow to read.
var neverMatch = regexp.MustCompile(`\A[^\x00-\x{10FFFF}]`)

func init() {
	markdownParser = parser.NewParser(
		parser.WithBlockParsers(
			util.Prioritized(&frontMatterParser{}, 100),
			util.Prioritized(newBlockThematicBreakParser(), 200),
			util.Prioritized(newBlockListParser(), 300),
			util.Prioritized(newBlockListItemParser(), 400),
			util.Prioritized(&indentedBlocksParser{}, 450),
			util.Prioritized(&indentedParagraphParser{}, 500),
			util.Prioritized(newATXHeadingParser(), 600),
			util.Prioritized(newFencedCodeBlockParser(), 700),
			util.Prioritized(newQuoteParser(), 800),
			util.Prioritized(&mathBlockParser{}, 896),
			util.Prioritized(&latexEnvironmentParser{}, 896),
			util.Prioritized(&taskDateParser{}, 897),
			util.Prioritized(&logbookParser{}, 898),
			util.Prioritized(&propertyDrawerParser{}, 898),
			util.Prioritized(&beginEndParser{}, 899),
			util.Prioritized(&drawerParser{}, 899),
			util.Prioritized(newHTMLBlockParser(), 900),
			util.Prioritized(&footnoteDefinitionParser{}, 999),
			util.Prioritized(newPropertyLineParser(), 999),
			util.Prioritized(newParagraphParser(), 1000),
		),
		parser.WithInlineParsers(inlineParsers()...),
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
			util.Prioritized(&taskDatePositions{}, 300),
		),
	)
}

// inlineParsers are the parsers for the text inside a block, which are the
// same for all text that Logseq reads as Markdown.
func inlineParsers() []util.PrioritizedValue {
	return []util.PrioritizedValue{
		util.Prioritized(newCodeSpanSourceParser(), 100),
		util.Prioritized(&footnoteRefParser{}, 194),
		util.Prioritized(&mathParser{}, 195),
		util.Prioritized(&priorityParser{}, 196),
		util.Prioritized(&imageMetadataParser{}, 190),
		util.Prioritized(&macroParser{}, 197),
		util.Prioritized(&blockRefParser{}, 198),
		util.Prioritized(&pageLinkParser{}, 199),
		util.Prioritized(&tagParser{}, 999),
		util.Prioritized(newLabelledRefLinkParser(), 200),
		util.Prioritized(&htmlElementParser{}, 250),
		util.Prioritized(newBracketedAutoLinkParser(), 300),
		util.Prioritized(parser.NewRawHTMLParser(), 400),
		util.Prioritized(&emphasisParser{}, 500),
		util.Prioritized(&strikethroughParser{}, 501),
		util.Prioritized(&highlightParser{}, 502),
		util.Prioritized(extension.NewLinkifyParser(
			extension.WithLinkifyEmailRegexp(neverMatch),
			extension.WithLinkifyWWWRegexp(neverMatch),
		), 998),
	}
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

// builtInPropertiesWithoutPageReferences are the properties that Logseq uses
// itself and whose value it never reads as references to pages: the built-in
// properties that the user can edit, except alias, aliases and tags, and the
// built-in properties that Logseq hides. A link in the value of one of them,
// such as `template:: [[x]]`, is only text.
var builtInPropertiesWithoutPageReferences = []string{
	// The built-in properties that the user can edit.
	"title", "icon", "template", "template-including-parent", "public",
	"filters", "exclude-from-graph-view", "logseq.query/nlp-date", "macro",
	"filetags",

	// The built-in properties that Logseq hides.
	"id", "custom-id", "background-color", "heading", "collapsed",
	"created-at", "updated-at", "last-modified-at", "query-table",
	"query-properties", "query-sort-by", "query-sort-desc", "ls-type",
	"hl-type", "hl-page", "hl-stamp", "hl-color", "logseq.macro-name",
	"logseq.macro-arguments", "logseq.order-list-type", "logseq.tldraw.page",
	"logseq.tldraw.shape", "logseq.color",

	// The task markers, which Logseq also hides as properties.
	"todo", "doing", "now", "later", "done",
}

// parseOptions are the settings that parsing takes from the graph. The zero
// value is not usable, use defaultParseOptions to get the defaults of Logseq.
type parseOptions struct {
	// propertiesSeparatedByCommas are the properties whose value is a list of
	// pages separated by commas.
	propertiesSeparatedByCommas map[string]struct{}

	// propertiesWithoutPageReferences are the properties whose value never
	// points at a page, even where it is written as a link. These are the
	// built-in properties of Logseq, see builtInPropertiesWithoutPageReferences.
	propertiesWithoutPageReferences map[string]struct{}

	// ignoredPageReferences are the properties of
	// `:ignored-page-references-keywords`, whose value is not split into pages
	// at commas, see WithIgnoredPageReferences.
	ignoredPageReferences map[string]struct{}

	// page is whether the text is the text of a whole page, see AsPage.
	page bool

	// relaxedTaskDates is whether task dates are read also in the forms and
	// places that Logseq ignores, see WithRelaxedTaskDates.
	relaxedTaskDates bool

	// laterPropertyGroupRefs is whether the groups of properties of a block
	// after the first make references, see WithLaterPropertyGroupRefs.
	laterPropertyGroupRefs bool
}

// defaultParseOptions are what Logseq does for a graph that does not configure
// anything else.
func defaultParseOptions() parseOptions {
	options := parseOptions{
		propertiesSeparatedByCommas:     make(map[string]struct{}),
		propertiesWithoutPageReferences: make(map[string]struct{}),
		ignoredPageReferences:           make(map[string]struct{}),
	}

	for _, name := range builtInPropertiesSeparatedByCommas {
		options.propertiesSeparatedByCommas[name] = struct{}{}
	}

	for _, name := range builtInPropertiesWithoutPageReferences {
		options.propertiesWithoutPageReferences[normalizePropertyName(name)] = struct{}{}
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

// WithIgnoredPageReferences adds properties whose value is only text where it
// is not written as a link, which is `:ignored-page-references-keywords`. The
// value of such a property is not split into pages at commas, also when the
// property is separated by commas, so `tags:: a, b` points at no page. Logseq
// 0.10.9 still reads the links and tags in the value, so `author:: [[Bob]]`
// points at the page Bob. The built-in properties whose value Logseq never
// reads as pages, not even as links, are not set here, see
// builtInPropertiesWithoutPageReferences.
func WithIgnoredPageReferences(names ...string) ParseOption {
	return func(o *parseOptions) {
		for _, name := range names {
			o.ignoredPageReferences[normalizePropertyName(name)] = struct{}{}
		}
	}
}

// AsPage reads the text as the text of a whole page, instead of as the text
// of one block. In a page, Logseq reads each heading line that is not in a
// bullet, such as `# Title`, as the start of a top-level block, see
// content.NewHeadingBlock. In the text of a block, such a line before the
// first bullet is part of the block. After the first bullet, it starts a
// block in both, as the text is then an outline of blocks.
func AsPage() ParseOption {
	return func(o *parseOptions) {
		o.page = true
	}
}

// WithRelaxedTaskDates sets whether `SCHEDULED:` and `DEADLINE:` lines are
// read as dates also in the forms and places that Logseq ignores, such as
// `<2024-01-01>` without a weekday, or a date after the text of a block. By
// default a date is read only where Logseq reads it, see taskDatePositions.
func WithRelaxedTaskDates(relaxed bool) ParseOption {
	return func(o *parseOptions) {
		o.relaxedTaskDates = relaxed
	}
}

// WithLaterPropertyGroupRefs sets whether the pages named in the groups of
// properties of a block after its first group are references. A block can
// have more than one group when its properties are split by a text line or a
// blank line, as in "- a\n  b:: c\n  text\n  d:: [[e]]". Logseq reads only
// the first group, so by default the groups after it are read with
// content.Properties.PageRefsIgnored set, and `e` is not a reference.
func WithLaterPropertyGroupRefs(enabled bool) ParseOption {
	return func(o *parseOptions) {
		o.laterPropertyGroupRefs = enabled
	}
}

// isSeparatedByCommas checks if the value of the property with the given name
// is a list of pages separated by commas.
func (o *parseOptions) isSeparatedByCommas(name string) bool {
	_, ok := o.propertiesSeparatedByCommas[normalizePropertyName(name)]
	return ok
}

// hasNoPageReferences checks if the value of the property with the given name
// is only text, also where it is written as a link.
func (o *parseOptions) hasNoPageReferences(name string) bool {
	_, ok := o.propertiesWithoutPageReferences[normalizePropertyName(name)]
	return ok
}

// ignoresPageReferences checks if the value of the property with the given
// name is in `:ignored-page-references-keywords`, see
// WithIgnoredPageReferences.
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

	return parseWithOptions(src, &options)
}

// parseWithOptions parses Markdown with options that are already set up.
//
// A text with Windows line endings (CRLF), or with a lone carriage return
// (CR) as a line break, is read as a text with "\n" line breaks. Logseq reads
// a lone CR as a line break too. The line break of most lines is kept on the
// returned block, see content.Block.LineEnding.
func parseWithOptions(src []byte, options *parseOptions) (*content.Block, error) {
	lineEnding := ""
	if bytes.IndexByte(src, '\r') >= 0 {
		lineEnding = lineEndingOf(src)
		src = normalizeLineEndings(src)
	}

	context := parser.NewContext()
	context.Set(parseOptionsKey, options)

	doc := markdownParser.Parse(newLineCachingReader(src), parser.WithContext(context))
	if lines, ok := context.Get(lazyFootnoteLinesKey).(lazyFootnoteLines); ok {
		doc.SetAttribute(lazyFootnoteLinesAttribute, lines)
	}
	if lines, ok := context.Get(lazyBlockLinesKey).(map[int]bool); ok {
		doc.SetAttribute(lazyBlockLinesAttribute, lines)
	}
	if indentation, ok := context.Get(lineIndentationKey).(lineIndentation); ok {
		doc.SetAttribute(lineIndentationAttribute, indentation)
	}

	levels := blockLevels{}
	block, err := convertOutline(src, doc, options.page, levels)
	if err != nil {
		return nil, fmt.Errorf("Could not parse Markdown: %w", err)
	}

	nestByLevels(block, levels)

	if !options.laterPropertyGroupRefs {
		ignoreRefsOfLaterPropertyGroups(block)
	}

	if lineEnding != "\n" {
		block.WithLineEnding(lineEnding)
	}

	return block, nil
}

// ignoreRefsOfLaterPropertyGroups marks the groups of properties of each
// block after its first group as not making references, as Logseq reads only
// the first group, see WithLaterPropertyGroupRefs. The first group is the one
// that content.Block.FindProperties finds.
func ignoreRefsOfLaterPropertyGroups(block *content.Block) {
	first := block.FindProperties()
	for node := block.FirstChild(); node != nil; node = node.NextSibling() {
		switch n := node.(type) {
		case *content.Properties:
			if n != first {
				n.PageRefsIgnored = true
			}
		case *content.Block:
			ignoreRefsOfLaterPropertyGroups(n)
		}
	}
}

// lineEndingOf finds the line break that most lines of the text end with:
// "\r\n", "\r" or "\n". If two are as common, "\n" comes before "\r\n", and
// "\r\n" before "\r".
func lineEndingOf(src []byte) string {
	crlf, cr, lf := 0, 0, 0
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '\r':
			if i+1 < len(src) && src[i+1] == '\n' {
				crlf++
				i++
			} else {
				cr++
			}
		case '\n':
			lf++
		}
	}

	if lf >= crlf && lf >= cr {
		return "\n"
	}

	if crlf >= cr {
		return "\r\n"
	}

	return "\r"
}

// normalizeLineEndings changes each "\r\n" and each lone "\r" in the text to
// "\n".
func normalizeLineEndings(src []byte) []byte {
	out := make([]byte, 0, len(src))
	for i := 0; i < len(src); i++ {
		if src[i] != '\r' {
			out = append(out, src[i])
			continue
		}

		out = append(out, '\n')
		if i+1 < len(src) && src[i+1] == '\n' {
			i++
		}
	}

	return out
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
	case *blockRefText:
		return content.NewText(string(node.Segment.Value(src))), nil
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
		return content.NewMath(node.Value).WithDisplayed(node.Displayed).WithBackslashes(node.Backslashes), nil
	case *mathBlock:
		return convertMathBlock(src, node)
	case *ast.CodeSpan:
		return convertCodeSpan(src, node)
	case *ast.Link:
		return convertLink(src, node)
	case *ast.AutoLink:
		_, bracketed := node.AttributeString(bracketedAttribute)
		return content.NewAutoLink(string(node.URL(src))).WithBracketed(bracketed), nil
	case *tag:
		hashtag := content.NewHashtag(node.Page).WithBracketed(node.Extended)
		hashtag.SetToSource(node.Source)
		return hashtag, nil
	case *pageLink:
		link := content.NewPageLink(node.Page)
		link.SetToSource(node.Source)
		return link, nil
	case *pageRefText:
		return content.NewPageRefText(unescapePunctuation(node.Segment.Value(src))), nil
	case *blockRef:
		return content.NewBlockRef(node.ID), nil
	case *priority:
		result := content.NewTaskPriority(node.Priority)
		result.SetSpaceAfter(node.SpaceAfter)
		return result, nil
	case *macro:
		return convertMacro(src, node)
	case *ast.FencedCodeBlock:
		return convertFencedCodeBlock(src, node)
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
		thematicBreak := content.NewThematicBreak()
		if marker, ok := node.AttributeString(thematicBreakMarkerAttribute); ok && marker != "---" {
			thematicBreak.Marker = marker.(string)
		}
		if indentation, ok := node.AttributeString(thematicBreakIndentationAttribute); ok {
			thematicBreak.Indentation = indentation.(string)
		}
		updatePreviousLine(src, node, thematicBreak)
		return thematicBreak, nil
	case *beginEnd:
		return convertBeginEnd(src, node)
	case *properties:
		return convertProperties(src, node)
	case *logbook:
		return convertLogbook(src, node)
	case *drawer:
		return convertDrawer(src, node)
	case *taskDate:
		return convertTaskDate(src, node)
	case *frontMatter:
		return convertFrontMatter(src, node)
	}

	return nil, fmt.Errorf("Could not convert node: %T", in)
}

// unescapePunctuation removes the backslashes that escape ASCII punctuation,
// in the same way as CommonMark, so `\*` is read as `*` and `\\` as `\`. Other
// backslashes are text and are kept, so `C:\Users` stays as it is.
func unescapePunctuation(src []byte) string {
	return string(util.UnescapePunctuations(src))
}

// setTextValue sets the value of a text from its source in the file. If
// reading removes backslashes, the source is kept on the text, so that the
// backslashes are written back, see content.Text.TextSource.
func setTextValue(text *content.Text, raw []byte) {
	text.Value = unescapePunctuation(raw)
	if len(raw) != len(text.Value) {
		text.SetTextSource(string(raw))
	} else {
		text.SetTextSource("")
	}
}

// convertChildren converts all children of a node and adds them to the target.
func convertChildren(src []byte, node ast.Node, target content.HasChildren) error {
	var previousChild ast.Node
	// mergedStart is where the source of the last text node starts, so that
	// text nodes that are merged are unescaped as one. A backslash at the end
	// of one of them can escape the first character of the next.
	mergedStart := 0

	// merged is the text node that the text nodes after it are merged into,
	// and mergedStop is where the source of the last of them stops. The
	// value is set once, when no more text nodes are merged into it, so that
	// a line of many text nodes is read in a time that grows with its
	// length.
	var merged *content.Text
	mergedStop := 0
	setMergedValue := func() {
		if merged != nil {
			setTextValue(merged, src[mergedStart:mergedStop])
			merged = nil
		}
	}

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
				merged = target.LastChild().(*content.Text)
				mergedStop = text.Segment.Stop
				previousChild = child

				merged.HardLineBreak = text.HardLineBreak()
				merged.SoftLineBreak = text.SoftLineBreak()
				merged.TrailingSpace = trailingSpaceOf(src, text)
				merged.BackslashLineBreak = isBackslashLineBreak(src, text)
				merged.LazyLine, merged.LazyLineMarkers = lazyLineAfter(src, text)
				merged.LazyLine = merged.LazyLine || isLazyListItemLineAfter(src, text)
				merged.LazyBlockLine = isLazyBlockLineAfter(src, text)
				merged.NextLineIndentation = nextLineIndentationOf(src, text)
				continue
			}

		}

		setMergedValue()

		if text, ok := child.(*ast.Text); ok {
			mergedStart = text.Segment.Start
		}

		if aware, ok := node.(content.PreviousLineAware); ok {
			keepBlankQuoteLineAfterText(src, child, aware)
		}

		target.AddChild(node)
		previousChild = child
	}

	setMergedValue()
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
	return convertOutline(src, node, false, nil)
}

// convertOutline converts a page, a block or indented bullets into a block.
// If headingBlocks is true, the node is a page and its heading lines start
// top-level blocks, see AsPage. In the text of a block, only the heading lines
// after the first bullet start top-level blocks, the same as in a page. If
// levels is not nil, it gets the level of
// each block, see nestByLevels.
func convertOutline(src []byte, node ast.Node, headingBlocks bool, levels blockLevels) (*content.Block, error) {
	block := content.NewBlock()
	if item, ok := node.(*ast.ListItem); ok {
		if space, empty := emptyBulletLine(src, item); empty {
			block.WithEmptyFirstLine(true, space)
		} else {
			block.WithBulletSpace(bulletSpace(src, item))
		}
	}

	// target is the block that content before the first list of blocks goes
	// into. It changes to a pre-block when the content of a page has child
	// blocks of its own.
	target := block

	hasParsedBlock := false
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if indented, ok := child.(*indentedBlocks); ok {
			nested, err := convertOutline(src, indented, false, levels)
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

		if heading, ok := child.(*ast.Heading); ok && (headingBlocks || ((hasParsedBlock || target != block) && isATXHeading(src, heading))) {
			if _, isDocument := node.(*ast.Document); isDocument {
				// A heading line of a page starts a top-level block. The
				// content after it goes to the block that comes last in the
				// file, which is this block until it gets children.
				headingNode, err := convertHeading(src, heading)
				if err != nil {
					return nil, err
				}

				convertTaskMarker(headingNode, src, heading)

				headingBlock := content.NewHeadingBlock(headingNode)
				levels.set(headingBlock, 1)
				if heading.HasBlankPreviousLines() && block.FirstChild() != nil {
					headingBlock.SetPreviousLineType(content.PreviousLineTypeBlank)
				}

				block.AddChild(headingBlock)
				hasParsedBlock = true
				continue
			}
		}

		if heading, ok := child.(*ast.Heading); ok && !isOnBulletLine(src, heading, block) && isATXHeading(src, heading) {
			if _, isListItem := node.(*ast.ListItem); isListItem {
				// A heading line after the first line of a block starts a
				// child block, the same as Logseq. Its level comes from the
				// indentation of the line, in the same way as for a bullet,
				// so the heading is a sibling of the bullets at the same
				// indentation. The content after it goes to the block that
				// comes last in the file.
				headingNode, err := convertHeading(src, heading)
				if err != nil {
					return nil, err
				}

				convertTaskMarker(headingNode, src, heading)

				headingBlock := content.NewHeadingBlock(headingNode)
				if heading.HasBlankPreviousLines() {
					headingBlock.SetPreviousLineType(content.PreviousLineTypeBlank)
				}
				if indent, ok := headingIndent(src, heading); ok {
					levels.set(headingBlock, indent+1)
				} else if indent, ok := bulletIndent(node); ok {
					// The heading has no text to find its line from, so it
					// is read as a child of the block.
					levels.set(headingBlock, indent+2)
				}

				block.AddChild(headingBlock)
				hasParsedBlock = true
				continue
			}
		}

		// Check if the child is a list of blocks.
		if list, ok := child.(*ast.List); ok && list.Marker == '-' {
			// This is a list using the block marker, so children are parsed
			// as blocks.
			for listItem := list.FirstChild(); listItem != nil; listItem = listItem.NextSibling() {
				listItemBlock, err := convertOutline(src, listItem, false, levels)
				if err != nil {
					return nil, err
				}

				if indent, ok := bulletIndent(listItem); ok {
					levels.set(listItemBlock, indent+1)
				}

				_, isDocument := node.(*ast.Document)
				if isDocument && listItem == list.FirstChild() &&
					list.HasBlankPreviousLines() && block.FirstChild() != nil &&
					(!hasParsedBlock || isHeadingBlock(block.LastChild())) {
					// Keep the blank line between the content before the
					// first bullet of a page, such as page properties, and
					// the first bullet.
					listItemBlock.SetPreviousLineType(content.PreviousLineTypeBlank)
				} else if hasBlankLineBeforeBullet(src, listItem) {
					// Keep the blank line between this block and the block or
					// content before it.
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

			// The first paragraph of a block can start with a task marker.
			// Logseq also reads a marker after the `#` marks of a heading
			// that starts the block, as in `- ## TODO Project`. The marker
			// must be on the line of the bullet.
			if block.FirstChild() == nil && !block.HasEmptyFirstLine() {
				switch n := node.(type) {
				case *content.Paragraph:
					convertTaskMarker(n, src, child)
				case *content.Heading:
					convertTaskMarker(n, src, child)
				}
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

// isATXHeading checks if a heading is written with `#` marks, such as
// `## Section`. Logseq reads only such headings. A setext heading, which is
// text followed by a line of `=` or `-`, is not a heading in Logseq.
func isATXHeading(src []byte, heading *ast.Heading) bool {
	if heading.Lines().Len() == 0 {
		// An ATX heading can be empty, such as `##`. A setext heading
		// always has text.
		return true
	}

	start := lineStart(src, heading.Lines().At(0).Start)
	line := bytes.TrimLeft(src[start:], " \t")
	return len(line) > 0 && line[0] == '#'
}

// isOnBulletLine checks if a heading is on the line of the bullet of its
// block, as in `- ## Section`. A heading with no text, such as `##`, is on
// that line if it is the first thing in the block.
func isOnBulletLine(src []byte, heading *ast.Heading, block *content.Block) bool {
	if heading.Lines().Len() == 0 {
		return block.FirstChild() == nil
	}

	line := bytes.TrimLeft(src[lineStart(src, heading.Lines().At(0).Start):], " \t")
	return isBulletLine(line)
}

// headingIndent gets the number of spaces and tabs before the `#` marks of a
// heading, from the start of its line. It is not known for a heading with no
// text.
func headingIndent(src []byte, heading *ast.Heading) (int, bool) {
	if heading.Lines().Len() == 0 {
		return 0, false
	}

	count := 0
	for i := lineStart(src, heading.Lines().At(0).Start); i < len(src) && (src[i] == ' ' || src[i] == '\t'); i++ {
		count++
	}

	return count, true
}

// isHeadingBlock checks if a node is a block read from a heading line.
func isHeadingBlock(node content.Node) bool {
	block, ok := node.(*content.Block)
	return ok && block.IsHeadingBlock()
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

// convertTaskMarker turns the task marker at the start of a paragraph or
// heading, such as `TODO`, into a *content.TaskMarker. The source is the node
// before it was converted.
//
// Logseq reads a marker only if a space follows it, or if it is the end of the
// file. Thus `TODO` followed by a line break is text. When a space and a line
// break follow a marker with no title, as in "- TODO \n  text", an empty text
// with the line break is kept after the marker, so that the line break is
// written back.
func convertTaskMarker(node content.HasChildren, src []byte, source ast.Node) {
	textNode, ok := node.FirstChild().(*content.Text)
	if !ok {
		return
	}

	keepEmptyText := false
	if !strings.Contains(textNode.Value, " ") {
		first, ok := source.FirstChild().(*ast.Text)
		if !ok {
			return
		}

		// The segment of the text does not hold the spaces at the end of
		// its line, so the source after it is checked.
		end := first.Segment.Stop
		switch {
		case end >= len(src):
			// The marker ends the file.
		case src[end] == ' ' || src[end] == '\t':
			// A line break after the space must be written back.
			keepEmptyText = textNode.SoftLineBreak || textNode.HardLineBreak
		default:
			return
		}
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
		textNode.SetTextSource("")
	} else {
		textNode.Value = textNode.Value[potentialMarkerIdx+1:]

		// A marker has no backslashes, so the source starts with it too.
		source := textNode.TextSource()
		textNode.SetTextSource(strings.TrimPrefix(source, potentialMarker+" "))
	}

	if textNode.Value == "" && !keepEmptyText && potentialMarkerIdx < 0 {
		textNode.RemoveSelf()

		// If the marker ends the line, the writer writes the space after it,
		// which is the first of the spaces at the end of the line.
		switch n := node.(type) {
		case *content.Paragraph:
			n.TrailingSpace = trimFirstByte(n.TrailingSpace)
		case *content.Heading:
			n.TrailingSpace = trimFirstByte(n.TrailingSpace)
		}
	} else if textNode.Value == "" && !keepEmptyText {
		textNode.RemoveSelf()
	} else if potentialMarkerIdx < 0 && textNode.TrailingSpace != "" {
		// The marker is written with a space after it, which is the first
		// space at the end of the line.
		textNode.TrailingSpace = textNode.TrailingSpace[1:]
	}

	node.PrependChild(content.NewTaskMarker(taskStatus))
}

// trimFirstByte removes the first byte of a text, if it has one.
func trimFirstByte(text string) string {
	if text == "" {
		return ""
	}

	return text[1:]
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
	case "STARTED":
		return content.TaskStatusStarted
	}

	return content.TaskStatusNone
}

func convertParagraph(src []byte, node *ast.Paragraph) (*content.Paragraph, error) {
	paragraph := content.NewParagraph()
	paragraph.Indentation = paragraphIndentation(node)
	paragraph.TrailingSpace = trailingSpaceOfLines(src, node)
	paragraph.FirstLineIndentation = firstLineIndentationOf(src, node)
	err := convertChildren(src, node, paragraph)
	if err != nil {
		return nil, err
	}
	keepLineStartSources(paragraph)
	if _, afterProperties := node.PreviousSibling().(*properties); afterProperties {
		// A paragraph directly after properties is written on the next line,
		// so a blank line between them has to be recorded explicitly.
		updatePreviousLineKeepingBlank(src, node, paragraph)
	} else {
		updatePreviousLine(src, node, paragraph)
	}
	keepBlankQuoteLineBefore(src, node, paragraph)
	return paragraph, nil
}

// keepBlankQuoteLineBefore records a blank line before a node in a quote,
// which Goldmark does not mark, see hasBlankQuoteLineBefore. Without it, two
// paragraphs of a quote would be written as one.
func keepBlankQuoteLineBefore(src []byte, node ast.Node, target content.PreviousLineAware) {
	if node.PreviousSibling() != nil && target.PreviousLineType() == content.PreviousLineTypeNonBlank && hasBlankQuoteLineBefore(src, node) {
		target.SetPreviousLineType(content.PreviousLineTypeBlank)
	}
}

// keepBlankQuoteLineAfterText records a blank line before a node in a quote
// that comes after a paragraph, such as the list in "> a\n>\n> * b". Goldmark
// does not mark the `>` line as a blank line. The line after the last line of
// the paragraph is checked, as the first line of a node such as a list or a
// math block is not in its lines.
func keepBlankQuoteLineAfterText(src []byte, node ast.Node, target content.PreviousLineAware) {
	if target.PreviousLineType() != content.PreviousLineTypeNonBlank || !inQuote(node) {
		return
	}

	switch node.(type) {
	case *ast.Paragraph, *ast.TextBlock:
		// See keepBlankQuoteLineBefore.
		return
	}

	switch previous := node.PreviousSibling().(type) {
	case *ast.Paragraph, *ast.TextBlock:
		lines := previous.Lines()
		if lines.Len() == 0 {
			return
		}

		end := lines.At(lines.Len() - 1).Stop
		if end < len(src) && src[end-1] != '\n' {
			if i := bytes.IndexByte(src[end:], '\n'); i >= 0 {
				end += i + 1
			} else {
				return
			}
		}

		next := src[end:]
		if i := bytes.IndexByte(next, '\n'); i >= 0 {
			next = next[:i]
		}

		if end < len(src) && isBlankLine(next, true) {
			target.SetPreviousLineType(content.PreviousLineTypeBlank)
		}
	}
}

func convertTextBlock(src []byte, node ast.Node) (*content.Paragraph, error) {
	paragraph := content.NewParagraph()
	paragraph.TrailingSpace = trailingSpaceOfLines(src, node)
	paragraph.FirstLineIndentation = firstLineIndentationOf(src, node)
	err := convertChildren(src, node, paragraph)
	if err != nil {
		return nil, err
	}
	keepLineStartSources(paragraph)

	updatePreviousLine(src, node, paragraph)
	keepBlankQuoteLineBefore(src, node, paragraph)
	return paragraph, nil
}

// keepLineStartSources sets the source of each text that starts a line of
// the paragraph and looks like block syntax, such as `3. a` on the second
// line of "- a\n  3. a", where it is text. The writer escapes such text when
// its value is changed, see lineStartEscapes, but text that still has the
// value it was read with is written as it was.
func keepLineStartSources(paragraph *content.Paragraph) {
	lineStart := true
	for child := paragraph.FirstChild(); child != nil; child = child.NextSibling() {
		text, ok := child.(*content.Text)
		if ok && lineStart && text.TextSource() == "" && lineStartEscape(text.Value, unknownRune) >= 0 {
			text.SetTextSource(text.Value)
		}

		lineStart = ok && (text.SoftLineBreak || text.HardLineBreak)
	}
}

func convertText(src []byte, node *ast.Text) (*content.Text, error) {
	text := content.NewText("")
	setTextValue(text, node.Segment.Value(src))
	if node.SoftLineBreak() {
		text.SoftLineBreak = true
	} else if node.HardLineBreak() {
		text.HardLineBreak = true
	}
	text.TrailingSpace = trailingSpaceOf(src, node)
	text.BackslashLineBreak = isBackslashLineBreak(src, node)
	text.LazyLine, text.LazyLineMarkers = lazyLineAfter(src, node)
	text.LazyLine = text.LazyLine || isLazyFootnoteLineAfter(src, node) || isLazyListItemLineAfter(src, node)
	text.LazyBlockLine = isLazyBlockLineAfter(src, node)
	text.NextLineIndentation = nextLineIndentationOf(src, node)
	return text, nil
}

// isLazyListItemLineAfter checks if the line after the line break of a text
// in a list item is indented less than the text of the item, as `b` is in
// "* a\nb", see content.Text.LazyLine. A text in a quote is left to
// lazyLineAfter.
func isLazyListItemLineAfter(src []byte, node *ast.Text) bool {
	if !node.SoftLineBreak() && !node.HardLineBreak() {
		return false
	}

	var item *ast.ListItem
	for p := node.Parent(); p != nil; p = p.Parent() {
		switch n := p.(type) {
		case *ast.Blockquote:
			return false
		case *ast.ListItem:
			if item == nil && !isBlockList(n.Parent()) {
				item = n
			}
		}
	}

	if item == nil {
		return false
	}

	value, ok := item.Attribute(listMarkerStartAttribute)
	markerAt, isInt := value.(int)
	if !ok || !isInt {
		return false
	}

	marker, space := listMarkerSpace(src, item)
	if marker == "" {
		return false
	}

	start := lineStart(src, markerAt)
	column := 0
	for i := start; i < markerAt; i++ {
		column = nextColumn(column, src[i])
	}

	textColumn := column + len(marker)
	if space == "" {
		textColumn++
	}
	for i := 0; i < len(space); i++ {
		textColumn = nextColumn(textColumn, space[i])
	}

	i := bytes.IndexByte(src[node.Segment.Stop:], '\n')
	if i < 0 {
		return false
	}

	return indentationWidth(src, node.Segment.Stop+i+1) < textColumn
}

// lazyLineAfter checks if the line after the line break of a text in a
// quote is written with fewer `>` markers than the quotes that hold the text,
// see content.Text.LazyLine. It also returns the number of markers that the
// line has, see content.Text.LazyLineMarkers.
func lazyLineAfter(src []byte, node *ast.Text) (bool, int) {
	if !node.SoftLineBreak() && !node.HardLineBreak() {
		return false, 0
	}

	depth := 0
	for p := node.Parent(); p != nil; p = p.Parent() {
		if _, ok := p.(*ast.Blockquote); ok {
			depth++
		}
	}

	if depth == 0 {
		return false, 0
	}

	i := bytes.IndexByte(src[node.Segment.Stop:], '\n')
	if i < 0 {
		return false, 0
	}

	next := src[node.Segment.Stop+i+1:]
	markers := 0
	for {
		next = bytes.TrimLeft(next, " \t")
		if len(next) == 0 || next[0] != '>' {
			break
		}

		markers++
		next = next[1:]
	}

	if len(next) == 0 || next[0] == '\n' || next[0] == '\r' || markers >= depth {
		return false, 0
	}

	return true, markers
}

// trailingSpaceOfLines returns the spaces and tabs at the end of the last line
// of a paragraph. The segments of the paragraph do not hold them, and its
// lines can also hold the lines of properties that were split from it, so the
// line of the last text in the paragraph is read. If the paragraph has no text,
// or its last text has a line break, the paragraph ends with a node that is
// not text, and the last line of the paragraph is read instead. That line is
// not known when properties were split from the paragraph, and then no spaces
// are found.
func trailingSpaceOfLines(src []byte, node ast.Node) string {
	var last *ast.Text
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if text, ok := n.(*ast.Text); ok && entering {
			last = text
		}
		return ast.WalkContinue, nil
	})

	if last == nil || last.SoftLineBreak() || last.HardLineBreak() {
		lines := node.Lines()
		if _, split := node.NextSibling().(*properties); split || lines.Len() == 0 {
			return ""
		}

		return string(spacesAndTabs(src[lines.At(lines.Len()-1).Stop:]))
	}

	end := last.Segment.Stop
	for end < len(src) && src[end] != '\n' && src[end] != '\r' {
		end++
	}

	line := src[last.Segment.Stop:end]
	return string(line[len(bytes.TrimRight(line, " \t")):])
}

// trailingSpaceOf returns the spaces and tabs between a text and its line
// break. The segment of the text does not hold them, so the source after the
// segment is read. A backslash hard line break has no trailing space, as the
// spaces before the backslash are part of the segment.
// isBackslashLineBreak checks if the hard line break of a text is written as
// a backslash at the end of the line.
func isBackslashLineBreak(src []byte, node *ast.Text) bool {
	return node.HardLineBreak() && node.Segment.Stop < len(src) && src[node.Segment.Stop] == '\\'
}

func trailingSpaceOf(src []byte, node *ast.Text) string {
	if !node.SoftLineBreak() && !node.HardLineBreak() {
		return ""
	}

	end := node.Segment.Stop
	for end < len(src) && (src[end] == ' ' || src[end] == '\t') {
		end++
	}

	return string(src[node.Segment.Stop:end])
}

func convertHeading(src []byte, node *ast.Heading) (*content.Heading, error) {
	heading := content.NewHeading(node.Level)
	if space, ok := node.AttributeString(headingMarksSpaceAttribute); ok {
		heading.MarksSpace = space.(string)
	}
	if space, ok := node.AttributeString(headingTrailingSpaceAttribute); ok {
		heading.TrailingSpace = space.(string)
	}
	if space, ok := node.AttributeString(headingIndentationAttribute); ok {
		heading.Indentation = space.(string)
	}
	err := convertChildren(src, node, heading)
	if err != nil {
		return nil, err
	}
	return heading, nil
}

// convertLink converts a link. Logseq reads `[label]([[Page]])` as a link to
// the page and `[label](((uuid)))` as a reference to the block, both with the
// text of the link as the label. Logseq also reads them with a title in double
// quotes, as `[label]([[Page]] "title")`. A link with a title in another form,
// such as `'title'`, or with an empty title is kept as a link, as Logseq does
// not read it as a reference, see refLinkTitle.
func convertLink(src []byte, node *ast.Link) (content.Node, error) {
	destination := node.Destination
	source, hasSource := node.AttributeString(destinationSourceAttribute)
	sourceText, _ := source.(string)

	if isRefTitleAccepted(node, sourceText, hasSource) {
		if length := pageRefLength(destination); length > 0 && length == len(destination) {
			link := content.NewPageLink(unescapePunctuation(destination[2 : length-2]))
			link.Title = unescapePunctuation(node.Title)
			if hasSource {
				link.SetDestinationSource(sourceText)
				if end := pageLinkClose([]byte(sourceText)); end > 0 && strings.HasPrefix(sourceText, "[[") {
					link.SetToSource(sourceText[2:end])
				}
			}
			return link, convertChildren(src, node, link)
		}

		if length := blockRefLength(destination); length > 0 && length == len(destination) {
			ref := content.NewBlockRef(string(destination[2 : length-2]))
			ref.Title = unescapePunctuation(node.Title)
			if hasSource {
				ref.SetDestinationSource(sourceText)
			}
			return ref, convertChildren(src, node, ref)
		}
	}

	link := content.NewLink(unescapePunctuation(destination))
	err := convertChildren(src, node, link)
	if err != nil {
		return nil, err
	}
	link.Title = unescapePunctuation(node.Title)
	if hasSource {
		link.SetDestinationSource(sourceText)
	}
	return link, nil
}

// isRefTitleAccepted checks if the title of a link, if it has one, is in a
// form that Logseq reads as the title of a page link or block reference. A
// link that is read from more than one line has no source text, and is a
// reference only without a title.
func isRefTitleAccepted(node *ast.Link, source string, hasSource bool) bool {
	if !hasSource {
		return len(node.Title) == 0
	}

	_, ok := refLinkTitle(refTitleSource(source))
	return ok
}

// convertMacro converts a macro, and keeps its text so that it can be
// written back unchanged.
func convertMacro(src []byte, node *macro) (content.Node, error) {
	converted := macroNodeOf(node)
	if aware, ok := converted.(content.MacroSourceAware); ok {
		aware.SetMacroSource(node.Source)
	}

	return converted, nil
}

// macroNodeOf returns the node for a macro. Built-in macros, such as `query`
// and `embed`, get their own node types.
func macroNodeOf(node *macro) content.Node {
	switch node.Name {
	case "query":
		// A query without a query string is not a query, keep it as a macro so
		// it is written back the way it was found.
		if len(node.Arguments) == 0 {
			break
		}

		// Logseq joins the arguments again, so the query is all of the text
		// after the name, also when it has commas.
		return content.NewQuery(node.RawArguments)
	case "embed":
		// Either a [[page]] or a ((block)). Logseq joins the arguments again,
		// so a page title with a comma is one target.
		if len(node.Arguments) == 0 {
			break
		}

		// A block is embedded only by its uuid, see isBlockID.
		arg := node.RawArguments
		if strings.HasPrefix(arg, "((") && strings.HasSuffix(arg, "))") && isBlockID([]byte(arg[2:len(arg)-2])) {
			return content.NewBlockEmbed(arg[2 : len(arg)-2])
		} else if page, ok := wholePageLink(arg); ok {
			return content.NewPageEmbed(page)
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
			)
		} else {
			return content.NewClozeWithCue(
				strings.TrimSpace(arg[:cueIdx]),
				strings.TrimSpace(arg[cueIdx+1:]),
			)
		}
	}

	return content.NewMacro(node.Name, node.Arguments...)
}

func convertEmphasis(src []byte, node *ast.Emphasis) (content.HasChildren, error) {
	_, underscore := node.AttributeString(emphasisUnderscoreAttribute)

	var result content.HasChildren
	if node.Level == 1 {
		result = content.NewEmphasis().WithUnderscore(underscore)
	} else if node.Level == 2 {
		result = content.NewStrong().WithUnderscore(underscore)
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
	highlight := content.NewHighlight().WithEqualSigns(node.EqualSigns)
	err := convertChildren(src, node, highlight)
	if err != nil {
		return nil, err
	}
	return highlight, nil
}

func convertCodeSpan(src []byte, node *ast.CodeSpan) (*content.CodeSpan, error) {
	code := content.NewCodeSpan(string(node.Text(src)))
	if source, ok := node.AttributeString(codeSpanSourceAttribute); ok {
		code.SetCodeSpanSource(source.(string))
	}
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
	if node.Info != nil {
		code.Language, code.Options = splitCodeInfo(string(node.Info.Segment.Value(src)))
	}
	if fence, ok := node.AttributeString(codeFenceAttribute); ok {
		code.Fence = fence.(string)
	}
	if source, ok := node.AttributeString(codeInfoSourceAttribute); ok {
		code.SetInfoSource(source.(string))
	}
	if indentation, ok := node.AttributeString(codeIndentationAttribute); ok {
		code.Indentation = indentation.(string)
	}
	if value, ok := node.Attribute(codeSourceAttribute); ok {
		if buf, ok := value.(*bytes.Buffer); ok {
			code.SetCodeSource(buf.String())
		}
	}
	if closing, ok := node.AttributeString(closingFenceAttribute); ok {
		code.SetClosingFenceSource(closing.(string))
	}

	updatePreviousLine(src, node, code)

	return code, nil
}

// splitCodeInfo splits the text after the opening fence of a code block into
// the language and the options after it, the same way Goldmark finds the
// language: the language ends at the first space.
func splitCodeInfo(info string) (string, string) {
	info = strings.TrimFunc(info, isCodeInfoSpace)
	language, options, _ := strings.Cut(info, " ")
	return language, strings.TrimLeftFunc(options, isCodeInfoSpace)
}

// isCodeInfoSpace checks if a character is a space that Goldmark trims from
// the text after the opening fence of a code block.
func isCodeInfoSpace(r rune) bool {
	return r < 0x80 && util.IsSpace(byte(r))
}

func convertFootnoteDefinition(src []byte, node *footnoteDefinition) (*content.FootnoteDefinition, error) {
	definition := content.NewFootnoteDefinition(unescapePunctuation([]byte(node.Label)))
	definition.LabelSpace = node.LabelSpace
	definition.EmptyFirstLine = node.EmptyFirstLine
	err := convertChildren(src, node, definition)
	if err != nil {
		return nil, err
	}

	updatePreviousLine(src, node, definition)

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

	value := content.NewMathBlock(valueBuf.String()).WithEnvironment(node.environment)

	updatePreviousLine(src, node, value)

	return value, nil
}

func convertBlockquote(src []byte, node *ast.Blockquote) (*content.Blockquote, error) {
	blockquote := content.NewBlockquote()
	if _, ok := node.AttributeString(quoteNoSpaceAttribute); ok {
		blockquote.NoSpaceAfterMarker = true
	}
	if node.FirstChild() != nil {
		_, blockquote.BlankFirstLine = node.AttributeString(quoteBlankFirstLineAttribute)
		if last, ok := node.AttributeString(quoteBlankLastLineAttribute); ok {
			blockquote.BlankLastLine = last.(bool)
		}
	}
	err := convertChildren(src, node, blockquote)
	if err != nil {
		return nil, err
	}

	updatePreviousLine(src, node, blockquote)

	return blockquote, nil
}

// convertList converts an ast.List into either a list or a block.
func convertList(src []byte, node *ast.List) (*content.List, error) {
	list := content.NewListFromMarker(node.Marker)
	if node.Marker == '-' {
		// A `-` list is not a list of blocks here, such as in a quote, so
		// it keeps its marker.
		list.Marker = '-'
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		item, err := convertListItem(src, child)
		if err != nil {
			return nil, err
		}
		list.AddChild(item)
	}

	updatePreviousLine(src, node, list)

	return list, nil
}

// hasEmptyMarkerLine checks if the line of the marker of a list item has no
// text after the marker.
func hasEmptyMarkerLine(src []byte, node ast.Node, marker string) bool {
	value, ok := node.Attribute(listMarkerStartAttribute)
	start, isInt := value.(int)
	if !ok || !isInt || marker == "" || start+len(marker) > len(src) {
		return false
	}

	rest := src[start+len(marker):]
	if i := bytes.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[:i]
	}

	return util.IsBlank(rest)
}

func convertListItem(src []byte, node ast.Node) (*content.ListItem, error) {
	item := content.NewListItem()
	marker, space := listMarkerSpace(src, node)
	item.MarkerSpace = space
	item.EmptyFirstLine = node.FirstChild() != nil && hasEmptyMarkerLine(src, node, marker)
	if node.PreviousSibling() != nil && hasBlankLineBeforeBullet(src, node) {
		item.SetPreviousLineType(content.PreviousLineTypeBlank)
	}
	if len(marker) > 1 {
		// The number of an item of an ordered list, without the `.` or `)`.
		item.Number = marker[:len(marker)-1]
	}

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

	if source, ok := node.AttributeString(tableSourceAttribute); ok {
		table.SetSourceText(string(source.([]byte)))
	}

	updatePreviousLine(src, node, table)

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

	// Goldmark keeps the line that ends the block, such as `-->` or
	// `</pre>`, out of the lines of the block.
	if node.HasClosure() {
		_, _ = codeBuf.Write(node.ClosureLine.Value(src))
	}

	raw := content.NewRawHTMLBlock(codeBuf.String())
	updatePreviousLine(src, node, raw)
	return raw, nil
}

func convertImage(src []byte, node *ast.Image) (*content.Image, error) {
	image := content.NewImage(string(node.Destination))
	image.Title = unescapePunctuation(node.Title)
	if metadata, ok := node.AttributeString(imageMetadataAttribute); ok {
		image.Metadata = metadata.(string)
	}
	if source, ok := node.AttributeString(destinationSourceAttribute); ok {
		image.SetDestinationSource(source.(string))
	}

	err := convertChildren(src, node, image)
	if err != nil {
		return nil, err
	}
	return image, nil
}

func convertProperties(src []byte, node *properties) (*content.Properties, error) {
	properties := content.NewProperties().WithDrawer(node.drawer)
	properties.DrawerLowerCase = node.drawerLowerCase
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		p, ok := child.(*property)
		if !ok {
			return nil, errors.New("Invalid child in properties")
		}

		prop := content.NewProperty(p.Name).
			WithPageRefsIgnored(p.PageRefsIgnored).
			WithQuoted(p.Quoted).
			WithDirective(p.Directive).
			WithInvalidName(p.InvalidName).
			WithSpaceBeforeValue(p.SpaceBeforeValue).
			WithTrailingSpace(p.TrailingSpace).
			WithNoSpaceAfterSeparator(p.NoSpaceAfterSeparator)
		prop.Indentation = p.Indentation
		err := convertChildren(src, p, prop)
		if err != nil {
			return nil, err
		}

		properties.AddChild(prop)
	}

	updatePreviousLineKeepingBlank(src, node, properties)

	return properties, nil
}

func convertBeginEnd(src []byte, node *beginEnd) (content.Node, error) {
	codeBuf := bytes.Buffer{}
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		_, _ = codeBuf.Write(line.Value(src))
	}

	switch strings.ToUpper(node.Variant) {
	case "QUERY":
		query := content.NewQueryCommand(codeBuf.String()).
			WithParameters(node.Parameters).
			WithLowerCase(node.LowerCase)
		query.SetBeginLineSource(node.beginLine)
		query.SetEndLineSource(node.endLine)
		updatePreviousLine(src, node, query)
		return query, nil
	default:
		value := codeBuf.String()
		var children []content.Node
		if isMarkdownCommand(node.Variant) {
			var err error
			children, err = parseCommandBody(value, node.options)
			if err != nil {
				return nil, err
			}
		}

		command := content.NewAdvancedCommand(node.Variant, value, children...).
			WithParameters(node.Parameters).
			WithLowerCase(node.LowerCase)
		command.SetBeginLineSource(node.beginLine)
		command.SetEndLineSource(node.endLine)
		updatePreviousLine(src, node, command)
		return command, nil
	}
}

func convertLogbook(src []byte, node *logbook) (content.Node, error) {
	logbook := content.NewLogbook()
	logbook.StartLineText = node.StartLineText
	logbook.EndLine = node.EndLine
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		value := strings.TrimSuffix(string(line.Value(src)), "\n")
		logbook.AddChild(parseLogbookEntry(value))
	}

	updatePreviousLine(src, node, logbook)

	return logbook, nil
}

func convertDrawer(src []byte, node *drawer) (content.Node, error) {
	var buf bytes.Buffer
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		_, _ = buf.Write(line.Value(src))
	}

	result := content.NewDrawer(node.Name, buf.String())
	if node.EndLine != ":END:" {
		result.EndLine = node.EndLine
	}

	updatePreviousLine(src, node, result)
	return result, nil
}

func convertTaskDate(src []byte, node *taskDate) (content.Node, error) {
	var date *content.TaskDate
	if node.HasTime {
		date = content.NewTaskDateWithTime(node.DateType, node.Date)
	} else {
		date = content.NewTaskDate(node.DateType, node.Date)
	}
	date.Repeater = node.Repeater
	date.Indentation = node.indentation
	if item, ok := blockItemOf(node.Parent()); ok {
		// The date of a block has its indentation from the column of the
		// `-` of the block, as tabs are kept in this way.
		lineStart := bytes.LastIndexByte(src[:node.Segment.Start], '\n') + 1
		date.Indentation = blockDashIndentation(src, item, lineStart)
	}
	date.SetDateSource(string(node.Segment.Value(src)))

	err := convertChildren(src, node, date)
	if err != nil {
		return nil, err
	}

	updatePreviousLineKeepingBlank(src, node, date)

	return date, nil
}

// updatePreviousLine updates the PreviousLineType of the target based on the
// node. This takes the information Goldmark parsed out and transfers it to
// our nodes.
func updatePreviousLine(src []byte, node ast.Node, target content.PreviousLineAware) {
	if node.HasBlankPreviousLines() {
		keepBlankLineSpace(src, node, target)
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

// keepBlankLineSpace records the spaces and tabs of the blank line before a
// node, when one line of only spaces and tabs comes before it, see
// content.BlankLineSpaceAware.
func keepBlankLineSpace(src []byte, node ast.Node, target content.PreviousLineAware) {
	aware, ok := target.(content.BlankLineSpaceAware)
	if !ok {
		return
	}

	start, ok := nodeStart(node)
	if !ok {
		return
	}

	lineStart := bytes.LastIndexByte(src[:start], '\n') + 1
	if lineStart < 2 {
		return
	}

	blankStart := bytes.LastIndexByte(src[:lineStart-1], '\n') + 1
	blank := bytes.TrimSuffix(src[blankStart:lineStart-1], []byte("\r"))
	if len(blank) == 0 || len(bytes.Trim(blank, " \t")) > 0 {
		return
	}

	// Only one blank line is written back, so the space is kept only when
	// the line before it has text.
	if blankStart > 0 {
		before := bytes.LastIndexByte(src[:blankStart-1], '\n') + 1
		if len(bytes.Trim(src[before:blankStart-1], " \t\r")) == 0 {
			return
		}
	}

	aware.SetBlankLineSpace(string(blank))
}

// nodeStart gets the offset in the source of the first line of a block node,
// from the lines of the node or of its first child.
func nodeStart(node ast.Node) (int, bool) {
	for n := node; n != nil; n = n.FirstChild() {
		if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
			return n.Lines().At(0).Start, true
		}

		if text, ok := n.(*ast.Text); ok {
			return text.Segment.Start, true
		}
	}

	return 0, false
}

// updatePreviousLineKeepingBlank is updatePreviousLine for nodes that combine
// with the previous line by default. For those a blank line before the node
// has to be recorded explicitly, as the automatic behavior would drop it.
func updatePreviousLineKeepingBlank(src []byte, node ast.Node, target content.PreviousLineAware) {
	if node.HasBlankPreviousLines() {
		keepBlankLineSpace(src, node, target)
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
