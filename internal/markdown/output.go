package markdown

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/aholstenson/logseq-go/content"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var urlRegexp = regexp.MustCompile(`^(?:http|https|ftp)://[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-z]+(?::\d+)?(?:[/#?][-a-zA-Z0-9@:%_+.~#$!?&/=\(\);,'">\^{}\[\]` + "`" + `]*)?`)

// EscapeFunc escapes a string so that reading it back gives the same value.
type EscapeFunc func(string) string

func EscapeNone(str string) string {
	return str
}

// EscapePotentialMarkdown escapes characters in text that would otherwise be
// read back as formatting.
//
// Square brackets are only escaped where they would be read back as a link, a
// page link or a footnote. Logseq reads `\[` up
// to a later `\]` as a display math formula, so `[` is never escaped. Where a
// bracket must be escaped, the escape goes on the `]` or on the `^` of a
// footnote instead.
//
// The markers of emphasis, strikethrough and highlights are only escaped
// where they could open or close formatting, so that text such as
// `snake_case` or `2 * 3` is written as it is.
//
// The `#` of a tag, the `((` of a block reference, the `{{` of a macro, the
// backticks of code and the `<` of HTML are escaped where they would be read
// back as that syntax. A backslash is written as `\\` where it would otherwise
// escape the character after it, so `C:\Users` is written as it is.
func EscapePotentialMarkdown(str string) string {
	return escapeText(str, unknownRune, unknownRune)
}

// unknownRune stands for a character next to a text that is not known.
const unknownRune = rune(-1)

// lineEdgeRune stands for the start or the end of the line or of the block
// that a text is in, such as the end of a paragraph.
const lineEdgeRune = rune(-2)

// escapeText is EscapePotentialMarkdown for a text with known characters
// directly before and after it. Either can be unknownRune or lineEdgeRune.
func escapeText(str string, before rune, after rune) string {
	return escapeTextIn(str, before, after, false)
}

// escapeTextIn is escapeText for a text that can be inside formatting, such
// as emphasis. The text of formatting has no tags when it is read back, so a
// `#` in it is not escaped.
func escapeTextIn(str string, before rune, after rune, inFormatting bool) string {
	escapes := map[int]bool{}
	mathEscapes(str, escapes)
	bracketEscapes(str, escapes)
	delimiterEscapes(str, before, after, escapes)
	markupEscapes(str, before, !inFormatting, escapes)
	backslashEscapes(str, after, escapes)

	out := strings.Builder{}
	for i, r := range str {
		if escapes[i] {
			out.WriteRune('\\')
		}

		out.WriteRune(r)
	}

	return out.String()
}

// asCharacter gets the character that a rune next to a text acts as for the
// rules of the reader. The edge of a line acts as a line break.
func asCharacter(r rune) rune {
	if r == lineEdgeRune {
		return '\n'
	}

	return r
}

// isASCIIPunctuation returns true for the characters that a backslash
// escapes, which are the ASCII punctuation characters.
func isASCIIPunctuation(r rune) bool {
	return r < utf8.RuneSelf && util.IsPunct(byte(r))
}

// backslashEscapes finds the backslashes in the text that are written as
// `\\`. A backslash before ASCII punctuation must be, as it would otherwise
// escape that character. A backslash before a space is also written as `\\`.
// `\ ` and `\\ ` read the same, and this keeps the text `a \\ b` as it is. A
// backslash at the end of the text is written as `\\` when a line break,
// ASCII punctuation or something not known comes after it, as a backslash
// before a line break is a hard line break.
func backslashEscapes(str string, after rune, escapes map[int]bool) {
	for i := 0; i < len(str); i++ {
		if str[i] != '\\' {
			continue
		}

		if i+1 < len(str) {
			next, _ := utf8.DecodeRuneInString(str[i+1:])
			escapes[i] = isASCIIPunctuation(next) || unicode.IsSpace(next)
			continue
		}

		escapes[i] = after == unknownRune || after == '\n' || isASCIIPunctuation(after)
	}
}

// markupEscapes finds the characters in the text that would start a tag, a
// heading, a block reference, a macro, code or HTML when read back. Tags are
// only looked for if tags is true.
func markupEscapes(str string, before rune, tags bool, escapes map[int]bool) {
	if !strings.ContainsAny(str, "#({`<") {
		return
	}

	for i := 0; i < len(str); i++ {
		switch str[i] {
		case '#':
			previous := before
			if i > 0 {
				previous, _ = utf8.DecodeLastRuneInString(str[:i])
			}

			if (tags && startsTag(str, i, previous)) || startsHeading(str, i, previous) {
				escapes[i] = true
			}
		case '(':
			if blockRefLength([]byte(str[i:])) > 0 {
				escapes[i] = true
			}
		case '{':
			if strings.HasPrefix(str[i:], "{{") && strings.Contains(str[i+2:], "}}") {
				escapes[i] = true
			}
		case '`':
			end := i
			for end < len(str) && str[end] == '`' {
				end++
			}

			if backtickRunAt(str, end, end-i) >= 0 {
				// A backslash does not escape a backtick inside code, so an
				// escaped backtick can still close code that a backtick
				// that is not escaped opens. All of them are escaped.
				for k := i; k < len(str); k++ {
					if str[k] == '`' {
						escapes[k] = true
					}
				}
			}

			i = end - 1
		case '<':
			if startsHTMLOrAutoLink(str[i+1:]) {
				escapes[i] = true
			}
		}
	}
}

// startsTag returns true if the `#` at the position would be read back as the
// start of a tag, using the rules of the tag parser.
func startsTag(str string, i int, previous rune) bool {
	if previous != unknownRune && !unicode.IsSpace(asCharacter(previous)) &&
		!strings.ContainsRune(tagPrefixes, previous) {
		return false
	}

	if previous == '[' && i > 0 && isPriorityText(str[i-1:]) {
		// The priority parser reads `[#A]` and `[#word]` as a whole, so the
		// tag parser never sees the `#`.
		return false
	}

	rest := str[i+1:]
	if strings.HasPrefix(rest, "#") {
		return false
	}

	if strings.HasPrefix(rest, "[[") {
		return strings.Contains(rest[2:], "]]")
	}

	return tagNameLength([]byte(rest)) > 0
}

// isPriorityText returns true if the text starts with `[#` and a closing `]`
// that has no space or `[` before it, which the priority parser reads as a
// priority or claims as text.
func isPriorityText(str string) bool {
	if len(str) < 4 || !strings.HasPrefix(str, "[#") {
		return false
	}

	for i := 2; i < len(str); i++ {
		if str[i] == ']' {
			return i > 2
		}

		if str[i] == '[' || util.IsSpace(str[i]) {
			return false
		}
	}

	return false
}

// startsHeading returns true if the `#` at the position starts a line and
// would be read back as the marker of a heading.
func startsHeading(str string, i int, previous rune) bool {
	if previous != unknownRune && previous != lineEdgeRune && previous != '\n' {
		return false
	}

	end := i
	for end < len(str) && str[end] == '#' {
		end++
	}

	return end-i <= 6 && (end == len(str) || str[end] == ' ' || str[end] == '\t')
}

// backtickRunAt finds the first run of exactly the given number of
// backticks from the position, which would close code that a run of the same
// length opens. Returns -1 if there is no such run.
func backtickRunAt(str string, from int, length int) int {
	for i := from; i < len(str); {
		if str[i] != '`' {
			i++
			continue
		}

		end := i
		for end < len(str) && str[end] == '`' {
			end++
		}

		if end-i == length {
			return i
		}

		i = end
	}

	return -1
}

// startsHTMLOrAutoLink returns true if the text after a `<` could make the
// `<` the start of an HTML tag or an autolink such as `<https://a.com>`.
func startsHTMLOrAutoLink(rest string) bool {
	if rest == "" || !strings.Contains(rest, ">") {
		return false
	}

	c := rest[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '/' || c == '!' || c == '?'
}

// delimiterEscapes finds the positions in the text that need a backslash so
// that the runs of `*`, `_`, `~` and `^` in it are read back as text. Every
// `*` and `_` of a run is escaped. For `~~` and `^^` the first marker is kept,
// as a single `~` or `^` is not a marker.
func delimiterEscapes(str string, before rune, after rune, escapes map[int]bool) {
	if !strings.ContainsAny(str, "*_~^") {
		return
	}

	for i := 0; i < len(str); {
		processor, min := delimiterProcessorFor(str[i])
		if processor == nil {
			i++
			continue
		}

		end := i
		for end < len(str) && str[end] == str[i] {
			end++
		}

		if end-i >= min && canFormat(str, i, end, min, processor, before, after) {
			for k := i; k < end; k++ {
				if min == 1 || k > i {
					escapes[k] = true
				}
			}
		}

		i = end
	}
}

// delimiterProcessorFor gets the processor and the shortest run of markers
// for the formatting that the character is a marker of.
func delimiterProcessorFor(c byte) (parser.DelimiterProcessor, int) {
	switch c {
	case '*', '_':
		return defaultEmphasisDelimiterProcessor, 1
	case '~':
		return defaultStrikethroughDelimiterProcessor, 2
	case '^':
		return defaultHighlightDelimiterProcessor, 2
	}

	return nil, 0
}

// canFormat returns true if the run of markers between start and end can
// open or close formatting when it is read back. A run at the start or the
// end of the text uses the given character before or after the text. When
// that character is not known, each kind of character that changes the result
// is tried there.
func canFormat(str string, start int, end int, min int, processor parser.DelimiterProcessor, before rune, after rune) bool {
	befores := []rune{' ', 'a', '.'}
	if start > 0 {
		previous, _ := utf8.DecodeLastRuneInString(str[:start])
		befores = []rune{previous}
	} else if before != unknownRune {
		befores = []rune{asCharacter(before)}
	}

	afters := []string{""}
	if end == len(str) {
		if after == lineEdgeRune {
			afters = []string{""}
		} else if after != unknownRune {
			afters = []string{string(after)}
		} else {
			afters = []string{"", "a", "."}
		}
	}

	for _, before := range befores {
		for _, after := range afters {
			node := scanDelimiter([]byte(str[start:]+after), before, min, processor)
			if node != nil && (node.CanOpen || node.CanClose) {
				return true
			}
		}
	}

	return false
}

// bracketEscapes finds the positions in the text that need a backslash so
// that the square brackets in it are read back as text.
func bracketEscapes(str string, escapes map[int]bool) {
	if !strings.ContainsAny(str, "[]") {
		return
	}

	for i := 0; i < len(str); i++ {
		switch str[i] {
		case '[':
			if i+1 >= len(str) {
				continue
			}

			if str[i+1] == '[' && (i == 0 || str[i-1] != '!') {
				// A page link. After `![` the reader starts an image, so
				// `![[Page]]` is text and is not changed.
				if end := pageLinkEnd(str, i+2, escapes); end >= 0 {
					escapes[end+1] = true
				}
			} else if str[i+1] == '^' && footnoteLabelEnd([]byte(str[i:])) >= 0 {
				// A footnote reference or the start of a footnote definition.
				escapes[i+1] = true
			}
		case ']':
			if escapes[i] || i+1 >= len(str) {
				continue
			}

			// Logseq has no reference links or link reference definitions,
			// so only a `]` that ends the text of an inline link is escaped.
			if str[i+1] == '(' && closesLink(str[i+1:]) {
				escapes[i] = true
			}
		}
	}
}

// pageLinkEnd finds the `]]` that ends a page link that starts before the
// given position, in the same way as the page link parser. Returns the
// position of the first `]`, or -1 if the page link does not end on the same
// line.
func pageLinkEnd(str string, from int, escapes map[int]bool) int {
	for j := from; j+1 < len(str) && str[j] != '\n'; j++ {
		if str[j] == ']' && str[j+1] == ']' && !escapes[j] && !escapes[j+1] {
			return j
		}
	}

	return -1
}

// closesLink returns true if a `]` that is directly followed by rest would end
// the text of a link. The reader decides this, so that a `]` before something
// that is not a valid destination, such as `(Page name)`, is not escaped.
func closesLink(rest string) bool {
	src := []byte("[x]" + rest)

	options := defaultParseOptions()
	context := parser.NewContext()
	context.Set(parseOptionsKey, &options)
	doc := markdownParser.Parse(text.NewReader(src), parser.WithContext(context))

	found := false
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if link, ok := n.(*ast.Link); ok {
			label, isText := link.FirstChild().(*ast.Text)
			found = isText && label.Segment.Start == 1
			return ast.WalkStop, nil
		}

		return ast.WalkContinue, nil
	})

	return found
}

// mathEscapes finds the dollar signs of the parts of a text that would be
// read back as a LaTeX formula, so that text such as `$x$` stays text. The
// dollar signs within a formula are escaped as well, as leaving one of them
// alone would let it pair up with something later in the text.
func mathEscapes(str string, escapes map[int]bool) {
	if !strings.Contains(str, "$") {
		return
	}

	src := []byte(str)
	for i := 0; i < len(src); {
		if src[i] != '$' {
			i++
			continue
		}

		_, length, _, ok := mathAt(src[i:])
		if !ok {
			i++
			continue
		}

		for k := i; k < i+length; k++ {
			if src[k] == '$' {
				escapes[k] = true
			}
		}

		i += length
	}
}

func EscapeLinkURL(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == '(' || r == ')'
	})
}

func EscapeLinkTitle(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == '"' || r == '\'' || r == ')'
	})
}

func EscapeWikiLink(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == ']'
	})
}

// EscapePageLinkTitle escapes the title of a page link that is written as
// `[[title]]`. The reader ends a page link at the first `]]`, so only a `]`
// that is followed by another `]`, or that is the last character, is escaped.
// Other brackets are kept, so that `[[file:./a.md][A]]` is written as it is.
func EscapePageLinkTitle(str string) string {
	out := strings.Builder{}
	for i := 0; i < len(str); i++ {
		if str[i] == ']' && (i+1 == len(str) || str[i+1] == ']') {
			out.WriteByte('\\')
		}

		out.WriteByte(str[i])
	}

	return out.String()
}

// EscapeFootnoteLabel escapes the label of a footnote, which is read up to the
// first closing bracket in it.
func EscapeFootnoteLabel(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == ']' || r == '['
	})
}

func EscapeBlockRef(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == ')'
	})
}

func EscapeMacroQuotedArgument(str string) string {
	return escapeRunes(str, func(prev rune, r rune) bool {
		return r == '"'
	})
}

// escapeRunes puts a backslash in front of every rune the given function picks
// out, passing it the rune before it for the cases that need that context.
func escapeRunes(str string, f func(prev rune, r rune) bool) string {
	out := strings.Builder{}
	p := rune(0)
	for _, r := range str {
		if f(p, r) {
			out.WriteRune('\\')
		}

		out.WriteRune(r)
		p = r
	}

	return out.String()
}

// Option changes how Markdown is written for the parts of the syntax that a
// graph configures the shape of.
type Option func(*outputOptions)

// outputOptions are the settings that the writer takes from the graph. The
// zero value is not usable, use defaultOutputOptions to get the defaults of
// Logseq.
type outputOptions struct {
	// logbookWithSeconds is whether the times in logbook entries are written
	// with seconds.
	logbookWithSeconds bool
}

// defaultOutputOptions are what Logseq does for a graph that does not
// configure anything else.
func defaultOutputOptions() outputOptions {
	return outputOptions{
		logbookWithSeconds: true,
	}
}

// WithLogbookSeconds sets whether the times in logbook entries are written
// with seconds, which is `:with-second-support?` of `:logbook/settings`.
func WithLogbookSeconds(withSeconds bool) Option {
	return func(o *outputOptions) {
		o.logbookWithSeconds = withSeconds
	}
}

// Output is used to write Markdown to an output buffer. It will help keep
// track of list indentation and when to add newlines.
type Output struct {
	out  *writer
	opts outputOptions

	// isolated is set when a node is written on its own, to find the
	// characters around another node. Text is then escaped without looking
	// at the nodes around it, which also keeps two text nodes from writing
	// each other without end.
	isolated bool
}

// NewWriter creates a new Markdown writer.
func NewWriter(out io.Writer, opts ...Option) *Output {
	options := defaultOutputOptions()
	for _, opt := range opts {
		opt(&options)
	}

	return &Output{
		out:  newWriter(out),
		opts: options,
	}
}

func AsString(n content.Node, opts ...Option) (string, error) {
	out := strings.Builder{}
	w := NewWriter(&out, opts...)
	if err := w.Write(n); err != nil {
		return "", err
	}

	return out.String(), nil
}

func Write(n content.Node, out io.Writer, opts ...Option) error {
	w := NewWriter(out, opts...)
	return w.Write(n)
}

func (w *Output) Write(n content.Node) error {
	switch node := n.(type) {
	case *content.RawHTML:
		return w.writeRaw(node.HTML)
	case *content.Text:
		return w.writeText(node)
	case *content.RawText:
		// Raw text is already Markdown, so it is written as it is.
		return w.writeRaw(node.Value)
	case *content.Emphasis:
		return w.writeEmphasis(node)
	case *content.Strong:
		return w.writeStrong(node)
	case *content.Strikethrough:
		return w.writeStrikethrough(node)
	case *content.Highlight:
		return w.writeHighlight(node)
	case *content.FootnoteRef:
		return w.writeFootnoteRef(node)
	case *content.FootnoteDefinition:
		return w.writeFootnoteDefinition(node)
	case *content.Math:
		return w.writeMath(node)
	case *content.MathBlock:
		return w.writeMathBlock(node)
	case *content.CodeSpan:
		return w.writeCodeSpan(node)
	case *content.Link:
		return w.writeLink(node)
	case *content.AutoLink:
		return w.writeAutoLink(node)
	case *content.PageLink:
		return w.writePageLink(node)
	case *content.PageRefText:
		return w.writePageRefText(node)
	case *content.Hashtag:
		return w.writeHashtag(node)
	case *content.BlockRef:
		return w.writeBlockRef(node)
	case *content.Image:
		return w.writeImage(node)
	case *content.Macro:
		return w.writeMacro(node, node.Name, node.Arguments)
	case *content.Query:
		return w.writeMacroText("query", node.Query)
	case *content.PageEmbed:
		return w.writeMacroText("embed", "[["+EscapeWikiLink(node.To)+"]]")
	case *content.BlockEmbed:
		return w.writeMacroText("embed", "(("+EscapeBlockRef(node.ID)+"))")
	case *content.Cloze:
		return w.writeCloze(node)
	case *content.Heading:
		return w.writeHeading(node)
	case *content.RawHTMLBlock:
		return w.writeRawHTMLBlock(node)
	case *content.Paragraph:
		return w.writeParagraph(node)
	case *content.List:
		return w.writeList(node)
	case *content.Table:
		return w.writeTable(node)
	case *content.Blockquote:
		return w.writeBlockquote(node)
	case *content.CodeBlock:
		return w.writeCodeBlock(node)
	case *content.ThematicBreak:
		return w.writeThematicBreak(node)
	case *content.Block:
		return w.writeBlock(node)
	case *content.Properties:
		return w.writeProperties(node)
	case *content.AdvancedCommand:
		return w.writeAdvancedCommand(node)
	case *content.QueryCommand:
		return w.writeBeginEnd(node, "QUERY", node.Query)
	case *content.TaskMarker:
		return w.writeTaskMarker(node)
	case *content.TaskPriority:
		return w.writeTaskPriority(node)
	case *content.TaskDate:
		return w.writeTaskDate(node)
	case *content.Logbook:
		return w.writeLogbook(node)
	default:
		return fmt.Errorf("unsupported node: %T", node)
	}
}

func (w *Output) writeRaw(s string) error {
	return w.out.WriteString(s)
}

func (w *Output) write(s string, escapeFunc EscapeFunc) error {
	return w.writeRaw(escapeFunc(s))
}

func (w *Output) startBlock(node content.BlockNode, marker string) error {
	return w.startBlockWithAutomaticBehavior(node, marker, true)
}

func (w *Output) startBlockWithAutomaticBehavior(node content.BlockNode, marker string, doubleNewLineForAutomatic bool) error {
	if w.out.HasWrittenAtCurrentIndent() {
		var prefix string
		if pl, ok := node.(content.PreviousLineAware); ok {
			switch pl.PreviousLineType() {
			case content.PreviousLineTypeBlank:
				prefix = "\n\n"
			case content.PreviousLineTypeNonBlank:
				prefix = "\n"
			case content.PreviousLineTypeAutomatic:
				if doubleNewLineForAutomatic {
					prefix = "\n\n"
				} else {
					prefix = "\n"
				}
			default:
				return fmt.Errorf("unknown previous line type: %d", pl.PreviousLineType())
			}
		} else {
			if doubleNewLineForAutomatic {
				prefix = "\n\n"
			} else {
				prefix = "\n"
			}
		}

		err := w.out.WriteString(prefix)
		if err != nil {
			return err
		}
	}

	w.out.PushIndentation(marker)
	return nil
}

func (w *Output) endBlock() {
	w.out.PopIndentation()
}

func (w *Output) writeChildren(node content.HasChildren) error {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		err := w.Write(child)
		if err != nil {
			return err
		}
	}

	return nil
}

// writePageRefText writes the title of a page that a property value names
// without markup. The title is written the same way text is.
func (w *Output) writePageRefText(node *content.PageRefText) error {
	before, after := unknownRune, unknownRune
	if !w.isolated {
		before, after = w.textSurroundings(node, false)
	}

	if r := after; r != unknownRune && r != lineEdgeRune && isValueSeparator(r) {
		// The value is split on the separator before the titles are read, so
		// the separator acts as the end of the title.
		after = lineEdgeRune
	}

	return w.writeRaw(escapeText(node.To, before, after))
}

func (w *Output) writeText(node *content.Text) error {
	before, after := unknownRune, unknownRune
	if !w.isolated {
		before, after = w.textSurroundings(node, node.SoftLineBreak || node.HardLineBreak)
	}

	value := escapeTextIn(node.Value, before, after, isInFormatting(node))
	if strings.HasSuffix(value, "]") && !strings.HasSuffix(value, `\]`) && (after == '(' || after == '[') {
		// The next node starts with `(` or `[`, which together with the
		// closing bracket would be read back as a link.
		value = value[:len(value)-1] + `\]`
	}

	err := w.writeRaw(value)
	if err != nil {
		return err
	}

	if node.SoftLineBreak {
		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	} else if node.HardLineBreak {
		err = w.writeRaw("\\\n")
		if err != nil {
			return err
		}
	}

	return nil
}

// isInFormatting returns true if the node is inside emphasis, strong text,
// strikethrough or a highlight.
func isInFormatting(node content.Node) bool {
	for p := node.Parent(); p != nil; p = p.Parent() {
		switch p.(type) {
		case *content.Emphasis, *content.Strong, *content.Strikethrough, *content.Highlight:
			return true
		}
	}

	return false
}

// textSurroundings finds the characters that are written directly before
// and after a text node, as these decide if the characters in the text can be
// read back as markup. Returns unknownRune for a side that is not known.
func (w *Output) textSurroundings(node content.Node, lineBreak bool) (rune, rune) {
	before, after := edgeRunesOf(node.Parent())

	if prev := node.PreviousSibling(); prev != nil {
		before = unknownRune
		if text, ok := prev.(*content.Text); ok && (text.SoftLineBreak || text.HardLineBreak) {
			before = '\n'
		} else if written := w.writeIsolated(prev); written != "" {
			before, _ = utf8.DecodeLastRuneInString(written)
		}
	}

	if lineBreak {
		after = '\n'
	} else if next := node.NextSibling(); next != nil {
		after = unknownRune
		if written := w.writeIsolated(next); written != "" {
			after, _ = utf8.DecodeRuneInString(written)
		}
	}

	return before, after
}

// edgeRunesOf gets characters that are like the ones written directly before
// the first child and after the last child of the node. Returns unknownRune
// if the node is not one of the known kinds.
func edgeRunesOf(node content.HasChildren) (rune, rune) {
	switch node.(type) {
	case *content.Paragraph:
		return lineEdgeRune, lineEdgeRune
	case *content.Heading, *content.Property:
		// After the `# ` of a heading or the `:: ` of a property, and at the
		// end of the line.
		return ' ', lineEdgeRune
	case *content.Emphasis, *content.Strong, *content.Strikethrough, *content.Highlight,
		*content.Link, *content.PageLink, *content.BlockRef:
		// The markers around formatting and the brackets around a label.
		return '.', '.'
	}

	return unknownRune, unknownRune
}

// writeIsolated writes a node on its own, to find the characters at its
// start and end. Returns an empty string if the node cannot be written.
func (w *Output) writeIsolated(node content.Node) string {
	out := strings.Builder{}
	isolated := &Output{out: newWriter(&out), opts: w.opts, isolated: true}
	if err := isolated.Write(node); err != nil {
		return ""
	}

	return out.String()
}

func (w *Output) writeEmphasis(node *content.Emphasis) error {
	if _, ok := node.PreviousSibling().(*content.Emphasis); ok {
		// Writing two emphasis nodes next to each other is not valid Markdown,
		// so we add a space between them as a compromise.
		err := w.writeRaw(" ")
		if err != nil {
			return err
		}
	}

	err := w.writeRaw("*")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("*")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeStrong(node *content.Strong) error {
	if _, ok := node.PreviousSibling().(*content.Strong); ok {
		// Writing two strong nodes next to each other is not valid Markdown,
		// so we add a space between them as a compromise.
		err := w.writeRaw(" ")
		if err != nil {
			return err
		}
	}

	err := w.writeRaw("**")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("**")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeStrikethrough(node *content.Strikethrough) error {
	err := w.writeRaw("~~")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("~~")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeHighlight(node *content.Highlight) error {
	err := w.writeRaw("^^")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("^^")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeFootnoteRef(node *content.FootnoteRef) error {
	return w.writeRaw("[^" + EscapeFootnoteLabel(node.Label) + "]")
}

func (w *Output) writeFootnoteDefinition(node *content.FootnoteDefinition) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw("[^" + EscapeFootnoteLabel(node.Label) + "]: ")
	if err != nil {
		return err
	}

	// The content of a footnote carries on over the lines that are indented
	// under the definition.
	w.out.PushIndentation("    ")

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	w.out.PopIndentation()

	w.endBlock()
	return nil
}

func (w *Output) writeMath(node *content.Math) error {
	marker := "$"
	if node.Displayed {
		marker = "$$"
	}

	err := w.writeRaw(marker)
	if err != nil {
		return err
	}

	err = w.writeRaw(node.Value)
	if err != nil {
		return err
	}

	return w.writeRaw(marker)
}

func (w *Output) writeMathBlock(node *content.MathBlock) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw("$$\n")
	if err != nil {
		return err
	}

	err = w.writeRaw(node.Value)
	if err != nil {
		return err
	}

	// The line that ends the block needs a line of its own.
	if !strings.HasSuffix(node.Value, "\n") {
		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw("$$")
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeCodeSpan(node *content.CodeSpan) error {
	// The marker has to be longer than the longest run of backticks in the
	// value, or it would end the code span early.
	longestRun := 0
	currentRun := 0
	for i := 0; i < len(node.Value); i++ {
		if node.Value[i] == '`' {
			currentRun++
			if currentRun > longestRun {
				longestRun = currentRun
			}
		} else {
			currentRun = 0
		}
	}
	marker := strings.Repeat("`", longestRun+1)

	// A code span that starts or ends with a backtick needs to be padded with
	// spaces to keep the backtick apart from the marker. The same padding is
	// needed for a value surrounded by spaces, as a reader strips one space
	// from each end when both are present.
	value := node.Value
	if needsCodeSpanPadding(value) {
		value = " " + value + " "
	}

	err := w.writeRaw(marker)
	if err != nil {
		return err
	}

	err = w.writeRaw(value)
	if err != nil {
		return err
	}

	err = w.writeRaw(marker)
	if err != nil {
		return err
	}
	return nil
}

// needsCodeSpanPadding checks if a code span value has to be written with a
// space at each end to survive being read back.
func needsCodeSpanPadding(value string) bool {
	if strings.HasPrefix(value, "`") || strings.HasSuffix(value, "`") {
		return true
	}

	// A value of only spaces is kept as is by a reader, and padding it would
	// make it longer instead.
	if strings.Trim(value, " ") == "" {
		return false
	}

	return strings.HasPrefix(value, " ") && strings.HasSuffix(value, " ")
}

func (w *Output) writeLink(node *content.Link) error {
	err := w.writeRaw("[")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("](")
	if err != nil {
		return err
	}

	err = w.write(node.URL, EscapeLinkURL)
	if err != nil {
		return err
	}

	if node.Title != "" {
		err = w.writeRaw(" '")
		if err != nil {
			return err
		}

		err = w.write(node.Title, EscapeLinkTitle)
		if err != nil {
			return err
		}

		err = w.writeRaw("'")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw(")")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeAutoLink(node *content.AutoLink) error {
	if urlRegexp.Match([]byte(node.URL)) {
		// No need for brackets, Logseq will automatically linkify the URL.
		return w.writeRaw(node.URL)
	}

	err := w.writeRaw("<")
	if err != nil {
		return err
	}

	err = w.writeRaw(node.URL)
	if err != nil {
		return err
	}

	err = w.writeRaw(">")
	if err != nil {
		return err
	}

	return nil
}

// writePageLink writes *content.PageLink as `[[to]]`, or as `[label]([[to]])`
// if the link has a label.
func (w *Output) writePageLink(node *content.PageLink) error {
	if node.HasLabel() {
		err := w.writeLabelStart(node)
		if err != nil {
			return err
		}
	}

	err := w.writeRaw("[[")
	if err != nil {
		return err
	}

	err = w.write(node.To, EscapePageLinkTitle)
	if err != nil {
		return err
	}

	err = w.writeRaw("]]")
	if err != nil {
		return err
	}

	if node.HasLabel() {
		return w.writeRaw(")")
	}

	return nil
}

// writeLabelStart writes the label of a page link or block reference as
// `[label](`. The target and the closing `)` are written after it.
func (w *Output) writeLabelStart(node content.HasChildren) error {
	err := w.writeRaw("[")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	return w.writeRaw("](")
}

// writeHashtag writes *content.PageLink as `#to` or `#[[to]]`. The extended
// syntax is used if the target contains whitespace.
func (w *Output) writeHashtag(node *content.Hashtag) error {
	err := w.writeRaw("#")
	if err != nil {
		return err
	}

	writeExtended := false
	for _, r := range node.To {
		if unicode.IsSpace(r) {
			writeExtended = true
			break
		}
	}

	if writeExtended {
		err = w.writeRaw("[[")
		if err != nil {
			return err
		}
	}

	err = w.write(node.To, EscapeWikiLink)
	if err != nil {
		return err
	}

	if writeExtended {
		err = w.writeRaw("]]")
		if err != nil {
			return err
		}
	}

	return nil
}

// writeBlockRef writes *content.BlockRef as `((id))`, or as `[label](((id)))`
// if the reference has a label.
func (w *Output) writeBlockRef(node *content.BlockRef) error {
	if node.HasLabel() {
		err := w.writeLabelStart(node)
		if err != nil {
			return err
		}
	}

	err := w.writeRaw("((")
	if err != nil {
		return err
	}

	err = w.write(node.ID, EscapeWikiLink)
	if err != nil {
		return err
	}

	err = w.writeRaw("))")
	if err != nil {
		return err
	}

	if node.HasLabel() {
		return w.writeRaw(")")
	}

	return nil
}

func (w *Output) writeImage(node *content.Image) error {
	err := w.writeRaw("![")
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	err = w.writeRaw("](")
	if err != nil {
		return err
	}

	err = w.write(node.URL, EscapeLinkURL)
	if err != nil {
		return err
	}

	if node.Title != "" {
		err = w.writeRaw(" '")
		if err != nil {
			return err
		}

		err = w.write(node.Title, EscapeLinkTitle)
		if err != nil {
			return err
		}

		err = w.writeRaw("'")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw(")")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeCloze(node *content.Cloze) error {
	cue := strings.TrimSpace(node.Cue)
	answer := strings.TrimSpace(node.Answer)
	if cue != "" {
		return w.writeMacro(node, "cloze", []string{answer + " \\ " + cue})
	} else {
		return w.writeMacro(node, "cloze", []string{answer})
	}
}

// writeMacroText writes a macro with the text after the name as it is.
// Logseq joins the arguments of the `query` and `embed` macros again, so a
// comma in the text needs no quotes.
func (w *Output) writeMacroText(name string, text string) error {
	return w.writeRaw("{{" + name + " " + text + "}}")
}

func (w *Output) writeMacro(node content.Node, name string, arguments []string) error {
	err := w.writeRaw("{{")
	if err != nil {
		return err
	}

	// Validate the macro name, it can not contain whitespace.
	for _, r := range name {
		if unicode.IsSpace(r) {
			return fmt.Errorf("macro name can not contain whitespace")
		}
	}

	err = w.writeRaw(name)
	if err != nil {
		return err
	}

	if arguments != nil {
		for i, arg := range arguments {
			if i == 0 {
				err = w.writeRaw(" ")
				if err != nil {
					return err
				}
			} else {
				err = w.writeRaw(", ")
				if err != nil {
					return err
				}
			}

			// Check if the argument contains a comma, if so we need to quote
			// the argument.
			quoted := false
			for _, r := range arg {
				if r == ',' {
					quoted = true
					break
				}
			}

			if quoted {
				err = w.writeRaw("\"")
				if err != nil {
					return err
				}

				err = w.write(arg, EscapeMacroQuotedArgument)
				if err != nil {
					return err
				}

				err = w.writeRaw("\"")
				if err != nil {
					return err
				}
			} else {
				err = w.write(arg, EscapeNone)
				if err != nil {
					return err
				}
			}
		}
	}

	err = w.writeRaw("}}")
	if err != nil {
		return err
	}

	return nil
}

func (w *Output) writeRawHTMLBlock(node *content.RawHTMLBlock) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	// The HTML of a block read from a file ends with the line break of its
	// last line. The line break before the next node is written by that node.
	err = w.writeRaw(strings.TrimSuffix(node.HTML, "\n"))
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeHeading(node *content.Heading) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	// The marks are written as text and not as indentation, as indentation
	// is only written after a line break. A heading block starts on the line
	// of its bullet, as in `- ## Section`.
	marks := strings.Repeat("#", node.Level)
	if node.FirstChild() != nil {
		marks += " "
	}

	err = w.writeRaw(marks)
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeParagraph(node *content.Paragraph) error {
	doubleNewLine := true
	if _, previousProperties := node.PreviousSibling().(*content.Properties); previousProperties {
		doubleNewLine = false
	}

	err := w.startBlockWithAutomaticBehavior(node, "", doubleNewLine)
	if err != nil {
		return err
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeList(node *content.List) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	i := 0
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*content.ListItem); !ok {
			return fmt.Errorf("unsupported list child: %T", child)
		}

		i++
		var marker string
		if node.Type == content.ListTypeOrdered {
			marker = fmt.Sprintf("%d", i) + string(node.Marker)
		} else {
			marker = string(node.Marker)
		}

		err := w.out.WriteString(marker + " ")
		if err != nil {
			return err
		}

		w.out.PushIndentation(strings.Repeat(" ", len(marker)+1))

		err = w.writeChildren(child.(content.HasChildren))
		if err != nil {
			return err
		}

		if child.NextSibling() != nil {
			err = w.writeRaw("\n")
			if err != nil {
				return err
			}
		}

		w.out.PopIndentation()
	}

	w.endBlock()
	return nil
}

// writeTable writes a table in the GitHub Flavored Markdown syntax, padding
// the cells so that the pipes of every row line up in a text editor.
func (w *Output) writeTable(node *content.Table) error {
	rows, err := w.tableRows(node)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return fmt.Errorf("table has no rows")
	}

	widths := tableColumnWidths(rows)
	for len(widths) < len(node.Alignments) {
		// A column that the table sets the alignment of, but that no row has a
		// cell for, is still written out.
		widths = append(widths, 3)
	}

	err = w.startBlock(node, "")
	if err != nil {
		return err
	}

	// The header names the columns and is followed by the row that sets how
	// the text of each column lines up.
	err = w.writeRaw(tableRow(rows[0], widths))
	if err != nil {
		return err
	}

	err = w.writeRaw("\n" + tableDelimiterRow(node, widths))
	if err != nil {
		return err
	}

	for _, row := range rows[1:] {
		err = w.writeRaw("\n" + tableRow(row, widths))
		if err != nil {
			return err
		}
	}

	w.endBlock()
	return nil
}

// tableRows writes out the cells of every row, so that the width of each
// column is known before any of it is written.
func (w *Output) tableRows(node *content.Table) ([][]string, error) {
	rows := make([][]string, 0)
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		row, ok := child.(*content.TableRow)
		if !ok {
			return nil, fmt.Errorf("unsupported table child: %T", child)
		}

		cells := make([]string, 0)
		for cellNode := row.FirstChild(); cellNode != nil; cellNode = cellNode.NextSibling() {
			cell, ok := cellNode.(*content.TableCell)
			if !ok {
				return nil, fmt.Errorf("unsupported table row child: %T", cellNode)
			}

			value, err := w.tableCell(cell)
			if err != nil {
				return nil, err
			}

			cells = append(cells, value)
		}

		rows = append(rows, cells)
	}

	return rows, nil
}

// tableCell writes the content of a cell to the text that goes between the
// pipes of a row. A row is a single line, so line breaks become spaces, and
// the pipes in the content are escaped to keep them out of the shape of the
// table.
func (w *Output) tableCell(node *content.TableCell) (string, error) {
	out := strings.Builder{}
	cellWriter := &Output{
		out:  newWriter(&out),
		opts: w.opts,
	}

	err := cellWriter.writeChildren(node)
	if err != nil {
		return "", err
	}

	value := out.String()
	value = strings.ReplaceAll(value, "\\\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.ReplaceAll(value, "|", "\\|"), nil
}

// tableColumnWidths measures how wide each column has to be for the content of
// every cell in it to fit.
func tableColumnWidths(rows [][]string) []int {
	widths := make([]int, 0)
	for _, cells := range rows {
		for i, cell := range cells {
			// Three is the shortest a column can be, as that is what the row
			// that sets the alignment of a centered column needs.
			for i >= len(widths) {
				widths = append(widths, 3)
			}

			if width := utf8.RuneCountInString(cell); width > widths[i] {
				widths[i] = width
			}
		}
	}

	return widths
}

// tableRow writes one row of a table, padding every cell to the width of its
// column. Rows that are missing cells get empty ones, as a row that is shorter
// than the header is read back with the columns it does not have left empty.
func tableRow(cells []string, widths []int) string {
	row := strings.Builder{}
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}

		row.WriteString("| ")
		row.WriteString(cell)
		row.WriteString(strings.Repeat(" ", width-utf8.RuneCountInString(cell)))
		row.WriteString(" ")
	}

	row.WriteString("|")
	return row.String()
}

// tableDelimiterRow writes the row below the header, which sets how the text
// of each column lines up.
func tableDelimiterRow(node *content.Table, widths []int) string {
	row := strings.Builder{}
	for i, width := range widths {
		row.WriteString("| ")

		switch node.AlignmentOf(i) {
		case content.TableAlignmentLeft:
			row.WriteString(":" + strings.Repeat("-", width-1))
		case content.TableAlignmentRight:
			row.WriteString(strings.Repeat("-", width-1) + ":")
		case content.TableAlignmentCenter:
			row.WriteString(":" + strings.Repeat("-", width-2) + ":")
		default:
			row.WriteString(strings.Repeat("-", width))
		}

		row.WriteString(" ")
	}

	row.WriteString("|")
	return row.String()
}

func (w *Output) writeBlockquote(node *content.Blockquote) error {
	err := w.startBlock(node, "> ")
	if err != nil {
		return err
	}

	if !w.out.lastWasLineBreak {
		// This is a hack to make sure that the indicator is written in lists
		// if the blockquote is the first item in a list item.
		_, err = w.out.output.Write([]byte{'>', ' '})
		if err != nil {
			return err
		}
	}

	err = w.writeChildren(node)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeCodeBlock(node *content.CodeBlock) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw("```")
	if err != nil {
		return err
	}

	if node.Language != "" {
		err = w.writeRaw(node.Language)
		if err != nil {
			return err
		}
	}

	err = w.writeRaw("\n")
	if err != nil {
		return err
	}

	err = w.writeRaw(node.Code)
	if err != nil {
		return err
	}

	// If the code does not end with a blank line, we add a newline
	if !strings.HasSuffix(node.Code, "\n") {
		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw("```")
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeThematicBreak(node *content.ThematicBreak) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw("---")
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeBlock(node *content.Block) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	// Write the content first
	for _, child := range node.Content() {
		err := w.Write(child)
		if err != nil {
			return err
		}
	}

	w.endBlock()

	hasParentBlock := false
	if _, ok := node.Parent().(*content.Block); ok && !writesWithoutBullet(node) {
		hasParentBlock = true
	}

	previousIndent := ""
	if hasParentBlock && w.out.IndentationLevel() > 0 {
		// As Logseq uses tabs for indentation of blocks we pop the current
		// indentation which is the two spaces to align content with "- " of
		// the list item. This allows the indentation to be only tabs for
		// blocks
		previousIndent = w.out.PopIndentation()
	}

	// Output the sub blocks
	blocks := node.Blocks()
	if len(blocks) > 0 {
		hasContent := w.out.HasWrittenAtCurrentIndent()
		if hasContent {
			err := w.out.WriteString("\n")
			if err != nil {
				return err
			}
		}

		if hasParentBlock || writesWithoutBullet(node) {
			// The children of the pre-block are indented with a tab so that
			// they are read back as its children and not as top-level blocks.
			w.out.PushIndentation("\t")
		} else {
			w.out.PushIndentation("")
		}

		i := 0
		for _, child := range blocks {
			i++

			if writesWithoutBullet(child) {
				// The pre-block is the content before the first bullet of the
				// page, so it is written as is with no marker to indent under.
				err := w.Write(child)
				if err != nil {
					return err
				}

				if child.NextSibling() != nil {
					err = w.writeRaw("\n")
					if err != nil {
						return err
					}
				}

				continue
			}

			if child.PreviousLineType() == content.PreviousLineTypeBlank && (i > 1 || hasContent) {
				// Keep the blank line between the block and what is written
				// before it.
				err := w.out.WriteString("\n")
				if err != nil {
					return err
				}
			}

			err := w.out.WriteString("- ")
			if err != nil {
				return err
			}

			w.out.PushIndentation("  ")

			err = w.Write(child)
			if err != nil {
				return err
			}

			if child.NextSibling() != nil {
				err = w.writeRaw("\n")
				if err != nil {
					return err
				}
			}

			w.out.PopIndentation()
		}

		w.out.PopIndentation()
	}

	if hasParentBlock {
		// Push the previous indentation back on the stack
		w.out.PushIndentation(previousIndent)
	}

	return nil
}

// writesWithoutBullet checks if a block is a pre-block in a position where the
// bullet-less form parses back into the same structure, which is as the first
// block of a page.
func writesWithoutBullet(node *content.Block) bool {
	if !node.IsPreBlock() || node.PreviousSibling() != nil {
		return false
	}

	parent, ok := node.Parent().(*content.Block)
	return ok && parent.Parent() == nil
}

func (w *Output) writeProperties(node *content.Properties) error {
	err := w.startBlockWithAutomaticBehavior(node, "", false)
	if err != nil {
		return err
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*content.Property); !ok {
			return fmt.Errorf("unsupported properties child: %T", child)
		}

		property := child.(*content.Property)
		err := w.writeRaw(property.Name)
		if err != nil {
			return err
		}

		err = w.writeRaw(":: ")
		if err != nil {
			return err
		}

		err = w.writeChildren(property)
		if err != nil {
			return err
		}

		if child.NextSibling() != nil {
			err = w.writeRaw("\n")
			if err != nil {
				return err
			}
		}
	}

	w.endBlock()
	return nil
}

func (w *Output) writeAdvancedCommand(node *content.AdvancedCommand) error {
	return w.writeBeginEnd(node, node.Type, node.Value)
}

func (w *Output) writeBeginEnd(node content.BlockNode, variant string, value string) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw("#+BEGIN_" + variant + "\n")
	if err != nil {
		return err
	}

	err = w.writeRaw(value)
	if err != nil {
		return err
	}

	if !w.out.lastWasLineBreak {
		err = w.writeRaw("\n")
		if err != nil {
			return err
		}
	}

	err = w.writeRaw("#+END_" + variant)
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

func (w *Output) writeTaskMarker(node *content.TaskMarker) error {
	if node.Status == content.TaskStatusNone {
		return nil
	}

	marker := node.Status.String()
	if marker == "" {
		return fmt.Errorf("unsupported task status: %d", node.Status)
	}

	err := w.writeRaw(marker)
	if err != nil {
		return err
	}

	if node.NextSibling() != nil {
		err = w.writeRaw(" ")
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Output) writeTaskPriority(node *content.TaskPriority) error {
	var err error
	switch node.Priority {
	case content.PriorityNone:
		return nil
	case content.PriorityA:
		err = w.writeRaw("[#A]")
	case content.PriorityB:
		err = w.writeRaw("[#B]")
	case content.PriorityC:
		err = w.writeRaw("[#C]")
	default:
		return fmt.Errorf("unsupported priority: %d", node.Priority)
	}

	if err != nil {
		return err
	}

	if node.NextSibling() != nil {
		err = w.writeRaw(" ")
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *Output) writeTaskDate(node *content.TaskDate) error {
	var keyword string
	switch node.Type {
	case content.TaskDateTypeScheduled:
		keyword = "SCHEDULED"
	case content.TaskDateTypeDeadline:
		keyword = "DEADLINE"
	default:
		return fmt.Errorf("unsupported task date type: %d", node.Type)
	}

	value := strings.Builder{}
	value.WriteString(keyword)
	value.WriteString(": <")
	value.WriteString(node.Date.Format("2006-01-02 Mon"))

	if node.HasTime {
		value.WriteString(node.Date.Format(" 15:04"))
	}

	if node.Repeater != nil {
		repeater := node.Repeater.String()
		if repeater == "" {
			return fmt.Errorf("unsupported repeater: %+v", *node.Repeater)
		}

		value.WriteString(" ")
		value.WriteString(repeater)
	}

	value.WriteString(">")

	// Task dates belong to the line above them, so they never get a blank line
	// of their own when the previous line type is automatic.
	err := w.startBlockWithAutomaticBehavior(node, "", false)
	if err != nil {
		return err
	}

	err = w.writeRaw(value.String())
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}

// logbookLayout is the timestamp format of the graph, which leaves the seconds
// out when the graph is set up without second support.
func (w *Output) logbookLayout() string {
	if w.opts.logbookWithSeconds {
		return logbookTimeLayout
	}

	return logbookTimeLayoutWithoutSeconds
}

// logbookClock formats a clock entry, deriving the duration from the two
// times so it always matches them.
func (w *Output) logbookClock(node *content.LogbookEntryClock) (string, error) {
	layout := w.logbookLayout()

	value := "CLOCK: [" + node.Start.Format(layout) + "]"
	if node.IsRunning() {
		return value, nil
	}

	if node.Duration() < 0 {
		return "", fmt.Errorf("logbook clock ends before it starts: %s", value)
	}

	value += "--[" + node.End.Format(layout) + "] =>  "

	if !w.opts.logbookWithSeconds {
		// The duration is measured between the times as they are written, so
		// that it adds up for whoever reads the entry.
		duration := node.End.Truncate(time.Minute).Sub(node.Start.Truncate(time.Minute))
		return value + fmt.Sprintf(
			"%02d:%02d",
			int(duration/time.Hour),
			int(duration/time.Minute)%60,
		), nil
	}

	duration := node.Duration()
	return value + fmt.Sprintf(
		"%02d:%02d:%02d",
		int(duration/time.Hour),
		int(duration/time.Minute)%60,
		int(duration/time.Second)%60,
	), nil
}

func (w *Output) logbookStateChange(node *content.LogbookEntryStateChange) (string, error) {
	to := node.To.String()
	if to == "" {
		return "", fmt.Errorf("unsupported task status: %d", node.To)
	}

	value := `* State "` + to + `"`

	if node.From != content.TaskStatusNone {
		from := node.From.String()
		if from == "" {
			return "", fmt.Errorf("unsupported task status: %d", node.From)
		}

		value += ` from "` + from + `"`
	}

	return value + " [" + node.Time.Format(w.logbookLayout()) + "]", nil
}

func (w *Output) writeLogbook(node *content.Logbook) error {
	err := w.startBlock(node, "")
	if err != nil {
		return err
	}

	err = w.writeRaw(":LOGBOOK:\n")
	if err != nil {
		return err
	}

	for _, entry := range node.Children() {
		var value string
		switch e := entry.(type) {
		case *content.LogbookEntryRaw:
			value = e.Value
		case *content.LogbookEntryClock:
			value, err = w.logbookClock(e)
		case *content.LogbookEntryStateChange:
			value, err = w.logbookStateChange(e)
		default:
			return fmt.Errorf("unsupported logbook entry: %T", entry)
		}

		if err != nil {
			return err
		}

		err = w.writeRaw(value)
		if err != nil {
			return err
		}

		if !strings.HasSuffix(value, "\n") {
			err = w.writeRaw("\n")
			if err != nil {
				return err
			}
		}
	}

	err = w.writeRaw(":END:")
	if err != nil {
		return err
	}

	w.endBlock()
	return nil
}
