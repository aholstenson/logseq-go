package markdown

import (
	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// commandBodyParser reads the text of a BEGIN..END command that Logseq reads
// as Markdown, such as QUOTE or NOTE. Logseq reads less of the syntax there
// than in a block: quotes, code fences, tables, lists that start with `*`, `+`
// or a number, and other commands are read, while headings, properties and
// indented code are text. A line that starts with `-` is text in Logseq. Here
// it is read as a list, which gives the same references and the same text.
var commandBodyParser = parser.NewParser(
	parser.WithBlockParsers(
		util.Prioritized(parser.NewThematicBreakParser(), 200),
		util.Prioritized(parser.NewListParser(), 300),
		util.Prioritized(parser.NewListItemParser(), 400),
		util.Prioritized(newFencedCodeBlockParser(), 700),
		util.Prioritized(parser.NewBlockquoteParser(), 800),
		util.Prioritized(&mathBlockParser{}, 896),
		util.Prioritized(&beginEndParser{}, 899),
		util.Prioritized(newHTMLBlockParser(), 900),
		util.Prioritized(parser.NewParagraphParser(), 1000),
	),
	parser.WithInlineParsers(inlineParsers()...),
	parser.WithParagraphTransformers(
		util.Prioritized(newTableParagraphTransformer(extension.NewTableParagraphTransformer()), 200),
	),
	parser.WithASTTransformers(
		util.Prioritized(extension.NewTableASTTransformer(), 0),
		util.Prioritized(defaultTagsInFormattingASTTransformer, 100),
	),
)

// verbatimCommands are the BEGIN..END commands whose text Logseq does not read
// as Markdown.
var verbatimCommands = map[string]struct{}{
	"SRC":     {},
	"EXAMPLE": {},
	"COMMENT": {},
	"EXPORT":  {},
}

// isMarkdownCommand checks if Logseq reads the text of a BEGIN..END command of
// the given type as Markdown.
func isMarkdownCommand(variant string) bool {
	_, verbatim := verbatimCommands[variant]
	return !verbatim
}

// parseCommandBody reads the text of a BEGIN..END command as Markdown and
// returns the nodes it holds.
func parseCommandBody(value string, options *parseOptions) ([]content.Node, error) {
	if options == nil {
		defaults := defaultParseOptions()
		options = &defaults
	}

	src := []byte(value)
	context := parser.NewContext()
	context.Set(parseOptionsKey, options)
	doc := commandBodyParser.Parse(text.NewReader(src), parser.WithContext(context))

	nodes := make([]content.Node, 0)
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		// A list is converted as a list, also when it uses `-`, as the text of
		// a command has no blocks.
		node, err := convert(src, child)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}
